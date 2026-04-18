package httpapi

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"repoforge/internal/config"
	"repoforge/internal/db"
	"repoforge/internal/repoindex"
	"repoforge/internal/storage"
)

var slugRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type API struct {
	cfg      config.Config
	store    *db.Store
	fs       *storage.FS
	indexers *repoindex.Set
}

type createRepoBody struct {
	Name   string          `json:"name"`
	Slug   string          `json:"slug"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

func (a *API) postRepository(w http.ResponseWriter, r *http.Request) {
	var body createRepoBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Slug = strings.TrimSpace(body.Slug)
	body.Type = strings.TrimSpace(body.Type)
	if body.Name == "" || body.Slug == "" || body.Type == "" {
		http.Error(w, `{"error":"name, slug, type required"}`, http.StatusBadRequest)
		return
	}
	if !slugRe.MatchString(body.Slug) {
		http.Error(w, `{"error":"invalid slug"}`, http.StatusBadRequest)
		return
	}
	typ := db.RepoType(body.Type)
	switch typ {
	case db.RepoDeb, db.RepoRpm, db.RepoFile:
	default:
		http.Error(w, `{"error":"type must be deb|rpm|file"}`, http.StatusBadRequest)
		return
	}
	var cfg any
	switch typ {
	case db.RepoDeb:
		raw := body.Config
		if len(raw) == 0 || string(raw) == "null" {
			raw = []byte("{}")
		}
		c, err := db.ParseDebConfig(raw)
		if err != nil {
			http.Error(w, `{"error":"invalid deb config"}`, http.StatusBadRequest)
			return
		}
		cfg = c
	default:
		if len(body.Config) > 0 && string(body.Config) != "null" {
			var raw map[string]any
			if err := json.Unmarshal(body.Config, &raw); err != nil {
				http.Error(w, `{"error":"invalid config"}`, http.StatusBadRequest)
				return
			}
			cfg = raw
		}
	}
	repo, err := a.store.CreateRepository(r.Context(), body.Name, body.Slug, typ, cfg)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") ||
			strings.Contains(strings.ToLower(err.Error()), "unique") {
			http.Error(w, `{"error":"slug already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, repoJSON(repo))
}

func (a *API) listRepositories(w http.ResponseWriter, r *http.Request) {
	list, err := a.store.ListRepositories(r.Context())
	if err != nil {
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}
	out := make([]any, 0, len(list))
	for _, x := range list {
		out = append(out, repoJSON(x))
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositories": out})
}

func (a *API) getRepository(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	repo, err := a.store.GetRepositoryBySlug(r.Context(), slug)
	if err != nil {
		if err == db.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, repoJSON(repo))
}

func (a *API) postUpload(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	repo, err := a.store.GetRepositoryBySlug(r.Context(), slug)
	if err != nil {
		if err == db.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}
	if err := r.ParseMultipartForm(a.cfg.MaxUploadBytes + (1 << 20)); err != nil {
		http.Error(w, `{"error":"multipart parse failed"}`, http.StatusBadRequest)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file field required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()
	baseName := filepath.Base(strings.TrimSpace(hdr.Filename))
	if baseName == "." || baseName == "/" || baseName == "" {
		http.Error(w, `{"error":"invalid filename"}`, http.StatusBadRequest)
		return
	}

	tmpPath, sha256sum, size, err := a.fs.WriteTempAndHash(file, a.cfg.MaxUploadBytes)
	if err != nil {
		_ = os.Remove(tmpPath)
		http.Error(w, `{"error":"upload failed"}`, http.StatusBadRequest)
		return
	}

	logicalPath, destAbs, err := a.resolveUploadPaths(repo, tmpPath, baseName, r.FormValue("path"))
	if err != nil {
		_ = os.Remove(tmpPath)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if err := a.fs.EnsureRepoDir(repo.Slug); err != nil {
		_ = os.Remove(tmpPath)
		http.Error(w, `{"error":"storage"}`, http.StatusInternalServerError)
		return
	}
	if err := a.fs.RenameInto(destAbs, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		http.Error(w, `{"error":"storage"}`, http.StatusInternalServerError)
		return
	}

	ct := hdr.Header.Get("Content-Type")
	if ct == "" {
		ct = mime.TypeByExtension(strings.ToLower(filepath.Ext(baseName)))
	}
	art, err := a.store.CreateArtifact(r.Context(), repo.ID, logicalPath, sha256sum, ct, size)
	if err != nil {
		_ = os.Remove(destAbs)
		http.Error(w, `{"error":"artifact conflict"}`, http.StatusConflict)
		return
	}

	runID, _ := a.store.StartIndexRun(r.Context(), repo.ID)
	idxErr := a.indexers.Reindex(r.Context(), repo, a.fs.RepoRoot(repo.Slug))
	if runID != 0 {
		if idxErr != nil {
			_ = a.store.FinishIndexRun(r.Context(), runID, false, idxErr.Error())
		} else {
			_ = a.store.FinishIndexRun(r.Context(), runID, true, "")
		}
	}
	if idxErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":    "index failed",
			"detail":   idxErr.Error(),
			"artifact": artifactJSON(art),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"artifact": artifactJSON(art)})
}

func (a *API) resolveUploadPaths(repo db.Repository, tmpPath, baseName, pathOverride string) (logicalPath string, destAbs string, err error) {
	root := a.fs.RepoRoot(repo.Slug)
	switch repo.Type {
	case db.RepoFile:
		rel := baseName
		if strings.TrimSpace(pathOverride) != "" {
			rel = pathOverride
		}
		safe, e := storage.SafeLogicalPath(rel)
		if e != nil {
			return "", "", e
		}
		logicalPath = filepath.ToSlash(filepath.Join("files", safe))
		return logicalPath, filepath.Join(root, filepath.FromSlash(logicalPath)), nil
	case db.RepoRpm:
		if !strings.HasSuffix(strings.ToLower(baseName), ".rpm") {
			return "", "", errInvalid("rpm uploads must use .rpm extension")
		}
		logicalPath = filepath.ToSlash(filepath.Join("rpms", baseName))
		return logicalPath, filepath.Join(root, filepath.FromSlash(logicalPath)), nil
	case db.RepoDeb:
		if !strings.HasSuffix(strings.ToLower(baseName), ".deb") {
			return "", "", errInvalid("deb uploads must use .deb extension")
		}
		lp, e := repoindex.DebUploadLogicalPath(tmpPath, baseName)
		if e != nil {
			return "", "", e
		}
		return lp, filepath.Join(root, filepath.FromSlash(lp)), nil
	default:
		return "", "", errInvalid("unknown repo type")
	}
}

type invalidErr string

func (s invalidErr) Error() string { return string(s) }

func errInvalid(msg string) error { return invalidErr(msg) }

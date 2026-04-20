package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

func (a *API) listArtifacts(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if _, err := a.store.GetRepositoryBySlug(r.Context(), slug); err != nil {
		if err == db.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}
	list, err := a.store.ListArtifactsByRepositorySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}
	out := make([]any, 0, len(list))
	for _, x := range list {
		out = append(out, artifactJSON(x))
	}
	writeJSON(w, http.StatusOK, map[string]any{"artifacts": out})
}

func (a *API) deleteRepository(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if _, err := a.store.GetRepositoryBySlug(r.Context(), slug); err != nil {
		if err == db.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}
	if err := a.store.DeleteRepositoryBySlug(r.Context(), slug); err != nil {
		if err == db.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}
	if err := a.fs.RemoveRepo(slug); err != nil {
		http.Error(w, `{"error":"storage cleanup failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (a *API) deleteArtifact(w http.ResponseWriter, r *http.Request) {
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
	artifactID, err := strconv.ParseInt(chi.URLParam(r, "artifactID"), 10, 64)
	if err != nil || artifactID <= 0 {
		http.Error(w, `{"error":"invalid artifact id"}`, http.StatusBadRequest)
		return
	}
	art, err := a.store.GetArtifactByIDAndRepositorySlug(r.Context(), slug, artifactID)
	if err != nil {
		if err == db.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}

	root := filepath.Clean(a.fs.RepoRoot(slug))
	abs := filepath.Join(root, filepath.FromSlash(art.LogicalPath))
	abs = filepath.Clean(abs)
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		http.Error(w, `{"error":"invalid artifact path"}`, http.StatusInternalServerError)
		return
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		http.Error(w, `{"error":"artifact file delete failed"}`, http.StatusInternalServerError)
		return
	}
	if err := a.store.DeleteArtifactByID(r.Context(), artifactID); err != nil {
		if err == db.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
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
			"error":  "index failed",
			"detail": idxErr.Error(),
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (a *API) installRepoScript(w http.ResponseWriter, r *http.Request) {
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
	if repo.Type == db.RepoFile {
		http.Error(w, "install script is not available for file repositories\n", http.StatusBadRequest)
		return
	}
	baseURL := "http://" + r.Host
	if r.TLS != nil {
		baseURL = "https://" + r.Host
	}
	body := installScriptForRepo(repo, baseURL)
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(body))
}

func installScriptForRepo(repo db.Repository, baseURL string) string {
	repoName := repo.Name
	if repoName == "" {
		repoName = repo.Slug
	}
	switch repo.Type {
	case db.RepoDeb:
		cfg, _ := db.ParseDebConfig(repo.ConfigJSON)
		return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
if ! command -v apt-get >/dev/null 2>&1; then
  echo "apt-get not found; this script supports Debian/Ubuntu systems." >&2
  exit 1
fi
if [[ "$(id -u)" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi
BASE_URL=%q
SLUG=%q
CODENAME=%q
COMPONENT=%q
LIST_FILE="/etc/apt/sources.list.d/repoforge-${SLUG}.list"
echo "deb [trusted=yes] ${BASE_URL}/repo/${SLUG} ${CODENAME} ${COMPONENT}" | ${SUDO} tee "${LIST_FILE}" >/dev/null
${SUDO} apt-get update
echo "Configured APT source in ${LIST_FILE}"
`, baseURL, repo.Slug, cfg.Codename, cfg.Component)
	case db.RepoRpm:
		return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
if ! command -v dnf >/dev/null 2>&1 && ! command -v yum >/dev/null 2>&1; then
  echo "dnf/yum not found; this script supports RPM-based systems." >&2
  exit 1
fi
if [[ "$(id -u)" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi
BASE_URL=%q
SLUG=%q
REPO_FILE="/etc/yum.repos.d/repoforge-${SLUG}.repo"
cat <<EOF | ${SUDO} tee "${REPO_FILE}" >/dev/null
[repoforge-${SLUG}]
name=%s
baseurl=${BASE_URL}/repo/${SLUG}/rpms
enabled=1
gpgcheck=0
EOF
if command -v dnf >/dev/null 2>&1; then
  ${SUDO} dnf clean all
  ${SUDO} dnf makecache
else
  ${SUDO} yum clean all
  ${SUDO} yum makecache
fi
echo "Configured RPM repository in ${REPO_FILE}"
`, baseURL, repo.Slug, repoName)
	default:
		return "#!/usr/bin/env bash\nexit 1\n"
	}
}

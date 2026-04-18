package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"repoforge/internal/db"
	"repoforge/internal/storage"
)

func (a *API) serveRepoFile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	rel := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if rel == "" {
		http.NotFound(w, r)
		return
	}
	if _, err := a.store.GetRepositoryBySlug(r.Context(), slug); err != nil {
		if err == db.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, `{"error":"database"}`, http.StatusInternalServerError)
		return
	}
	safe, err := storage.SafeLogicalPath(rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	root := filepath.Clean(a.fs.RepoRoot(slug))
	abs := filepath.Join(root, filepath.FromSlash(safe))
	abs = filepath.Clean(abs)
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	st, err := os.Stat(abs)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if st.IsDir() {
		if tryRedirectRepodataDir(w, r, abs) {
			return
		}
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	http.ServeContent(w, r, filepath.Base(abs), st.ModTime(), f)
}

// tryRedirectRepodataDir sends browsers and mistaken directory URLs to repomd.xml.
// DNF uses concrete files under repodata/; this handler otherwise returns 404 for directories.
func tryRedirectRepodataDir(w http.ResponseWriter, r *http.Request, absDir string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if !strings.EqualFold(filepath.Base(absDir), "repodata") {
		return false
	}
	repomd := filepath.Join(absDir, "repomd.xml")
	st, err := os.Stat(repomd)
	if err != nil || st.IsDir() {
		return false
	}
	base := strings.TrimSuffix(r.URL.Path, "/")
	loc := path.Clean(base + "/repomd.xml")
	if r.URL.RawQuery != "" {
		loc += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, loc, http.StatusFound)
	return true
}

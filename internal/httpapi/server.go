package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"repoforge/internal/config"
	"repoforge/internal/db"
	"repoforge/internal/repoindex"
	"repoforge/internal/storage"
)

func New(cfg config.Config, store *db.Store, fs *storage.FS, indexers *repoindex.Set) http.Handler {
	a := &API{cfg: cfg, store: store, fs: fs, indexers: indexers}
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	r.Get("/readyz", a.readyz)
	r.Route("/v1", func(r chi.Router) {
		if stringsNotEmpty(cfg.BearerToken) {
			r.Use(bearerAuth(cfg.BearerToken))
		}
		r.Post("/repositories", a.postRepository)
		r.Get("/repositories", a.listRepositories)
		r.Get("/repositories/{slug}", a.getRepository)
		r.Delete("/repositories/{slug}", a.deleteRepository)
		r.Get("/repositories/{slug}/artifacts", a.listArtifacts)
		r.Post("/repositories/{slug}/uploads", a.postUpload)
		r.Delete("/repositories/{slug}/artifacts/{artifactID}", a.deleteArtifact)
		r.Post("/system/install-repo-tooling", a.postInstallRepoTooling)
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		})
	})
	r.Route("/repo", func(r chi.Router) {
		r.Get("/{slug}/install.sh", a.installRepoScript)
		// Trailing "/*" is required: chi's "/{path:*}" matches only one segment, which breaks
		// nested repo paths and /assets/... from the embedded Vite build.
		r.Get("/{slug}/*", a.serveRepoFile)
	})
	r.Get("/*", a.serveWebUI)
	return r
}

func stringsNotEmpty(s string) bool { return s != "" }

func (a *API) readyz(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Ping(r.Context()); err != nil {
		http.Error(w, "not ready\n", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

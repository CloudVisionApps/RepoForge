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
	r.Route("/repo", func(r chi.Router) {
		r.Get("/{slug}/{path:*}", a.serveRepoFile)
	})
	r.Route("/v1", func(r chi.Router) {
		if stringsNotEmpty(cfg.BearerToken) {
			r.Use(bearerAuth(cfg.BearerToken))
		}
		r.Post("/repositories", a.postRepository)
		r.Get("/repositories", a.listRepositories)
		r.Get("/repositories/{slug}", a.getRepository)
		r.Post("/repositories/{slug}/uploads", a.postUpload)
	})
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

package httpapi

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed all:webui/dist
var webuiDist embed.FS

func (a *API) serveWebUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
		return
	}
	sub, err := fs.Sub(webuiDist, "webui/dist")
	if err != nil {
		http.Error(w, "web ui filesystem\n", http.StatusInternalServerError)
		return
	}
	p := strings.Trim(strings.TrimPrefix(chi.URLParam(r, "*"), "/"), "/")
	if strings.Contains(p, "..") {
		http.NotFound(w, r)
		return
	}
	if p == "" {
		p = "index.html"
	}
	b, err := fs.ReadFile(sub, p)
	if err != nil {
		if path.Ext(p) != "" {
			http.NotFound(w, r)
			return
		}
		b, err = fs.ReadFile(sub, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		p = "index.html"
	}
	ext := path.Ext(p)
	switch ext {
	case ".js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".html", "":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	default:
		if ct := mime.TypeByExtension(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		} else {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}
	}
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

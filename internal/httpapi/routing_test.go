package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Regression: chi's "/{name:*}" matches a single path segment only. Embedded UI and repo
// trees need a trailing "/*" wildcard so multi-segment paths resolve.
func TestChiWildcardTrailingStar(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/repo", func(r chi.Router) {
		r.Get("/{slug}/*", func(w http.ResponseWriter, req *http.Request) {
			_, _ = io.WriteString(w, chi.URLParam(req, "slug")+"|"+chi.URLParam(req, "*"))
		})
	})
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = io.WriteString(w, chi.URLParam(req, "*"))
	})
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/repo/deb/pool/a/b.deb")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if got, want := string(body), "deb|pool/a/b.deb"; got != want {
		t.Fatalf("repo nested path: got %q want %q", got, want)
	}

	resp, err = http.Get(ts.URL + "/assets/index-abc.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if got, want := string(body), "assets/index-abc.js"; got != want {
		t.Fatalf("assets path: got %q want %q", got, want)
	}
}

package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"repoforge/internal/config"
	"repoforge/internal/db"
	"repoforge/internal/repoindex"
	"repoforge/internal/storage"
)

func TestServeRepoRepodataDirectoryRedirectsToRepomd(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dbPath := filepath.Join(root, "test.db")
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	store := db.NewStore(sqlDB)
	if _, err := store.CreateRepository(context.Background(), "Prod", "production", db.RepoRpm, nil); err != nil {
		t.Fatal(err)
	}

	dataDir := filepath.Join(root, "data")
	repoRoot := filepath.Join(dataDir, "repos", "production", "rpms", "repodata")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	repomd := filepath.Join(repoRoot, "repomd.xml")
	if err := os.WriteFile(repomd, []byte("<repomd>\n</repomd>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{DataDir: dataDir, DBPath: dbPath}
	h := New(cfg, store, storage.New(dataDir), repoindex.NewSet(cfg))
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	noFollow := &http.Client{
		Transport: ts.Client().Transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := noFollow.Get(ts.URL + "/repo/production/rpms/repodata/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusFound)
	}
	loc := resp.Header.Get("Location")
	want := "/repo/production/rpms/repodata/repomd.xml"
	if loc != want {
		t.Fatalf("Location: got %q want %q", loc, want)
	}

	resp2, err := http.Get(ts.URL + want)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("repomd GET: got %d", resp2.StatusCode)
	}
	body, _ := io.ReadAll(resp2.Body)
	if string(body) != "<repomd>\n</repomd>\n" {
		t.Fatalf("body: got %q", body)
	}
}

package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FS struct {
	dataDir string
}

func New(dataDir string) *FS {
	return &FS{dataDir: dataDir}
}

func (f *FS) RepoRoot(slug string) string {
	return filepath.Join(f.dataDir, "repos", slug)
}

func (f *FS) EnsureRepoDir(slug string) error {
	return os.MkdirAll(f.RepoRoot(slug), 0o755)
}

func (f *FS) WriteTempAndHash(r io.Reader, maxBytes int64) (tmpPath string, sum string, size int64, err error) {
	dir := filepath.Join(f.dataDir, "_tmp")
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", "", 0, err
	}
	tmp, err := os.CreateTemp(dir, "upload-*")
	if err != nil {
		return "", "", 0, err
	}
	tmpPath = tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()
	h := sha256.New()
	lr := &io.LimitedReader{R: r, N: maxBytes + 1}
	n, cerr := io.Copy(io.MultiWriter(tmp, h), lr)
	if cerr != nil {
		_ = tmp.Close()
		return "", "", 0, cerr
	}
	if n > maxBytes {
		_ = tmp.Close()
		return "", "", 0, fmt.Errorf("upload exceeds max size")
	}
	if cerr := tmp.Close(); cerr != nil {
		return "", "", 0, cerr
	}
	return tmpPath, hex.EncodeToString(h.Sum(nil)), n, nil
}

func (f *FS) RenameInto(destAbs string, tmpPath string) error {
	if err := os.MkdirAll(filepath.Dir(destAbs), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, destAbs); err != nil {
		return err
	}
	return nil
}

func (f *FS) RemoveRepo(slug string) error {
	return os.RemoveAll(f.RepoRoot(slug))
}

func SafeLogicalPath(p string) (string, error) {
	p = filepath.ToSlash(strings.TrimSpace(p))
	if p == "" || p == "." {
		return "", fmt.Errorf("empty path")
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", fmt.Errorf("invalid path")
		}
	}
	if strings.Contains(p, "\x00") {
		return "", fmt.Errorf("invalid path")
	}
	clean := filepath.Clean("/" + p)
	if clean == "/" || clean == "." {
		return "", fmt.Errorf("invalid path")
	}
	rel := strings.TrimPrefix(clean, "/")
	if rel == ".." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
		return "", fmt.Errorf("path escapes root")
	}
	return rel, nil
}

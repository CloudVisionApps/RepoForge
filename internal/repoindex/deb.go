package repoindex

import (
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"repoforge/internal/db"
)

type debArtifact struct {
	path   string
	rel    string
	ctrl   debControl
	size   int64
	md5    string
	sha256 string
}

func poolRelativePath(packageName, baseName string) string {
	pkg := strings.TrimSpace(strings.ToLower(packageName))
	if pkg == "" {
		pkg = "unknown"
	}
	prefix := "x"
	for _, r := range pkg {
		prefix = strings.ToLower(string(r))
		break
	}
	return filepath.ToSlash(filepath.Join("pool", prefix, pkg, baseName))
}

func RebuildDebIndex(_ context.Context, repoRoot string, cfg db.DebRepoConfig) error {
	poolDir := filepath.Join(repoRoot, "pool")
	entries, err := collectDebFiles(poolDir)
	if err != nil {
		return err
	}
	var all []debArtifact
	for _, abs := range entries {
		ctrl, err := parseDebControl(abs)
		if err != nil {
			return fmt.Errorf("%s: %w", abs, err)
		}
		st, err := os.Stat(abs)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repoRoot, abs)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		md5s, sha256s, err := fileHashes(abs)
		if err != nil {
			return err
		}
		all = append(all, debArtifact{
			path:   abs,
			rel:    rel,
			ctrl:   ctrl,
			size:   st.Size(),
			md5:    md5s,
			sha256: sha256s,
		})
	}
	distsRoot := filepath.Join(repoRoot, "dists", cfg.Codename, cfg.Component)
	for _, arch := range cfg.Architectures {
		var list []debArtifact
		for _, d := range all {
			a := strings.TrimSpace(d.ctrl.get("Architecture"))
			if a == "" {
				a = "unknown"
			}
			if a == arch || strings.EqualFold(a, "all") {
				list = append(list, d)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].rel < list[j].rel })
		var buf strings.Builder
		for _, d := range list {
			buf.WriteString(formatPackagesEntry(d.ctrl, d.rel, d.size, d.md5, d.sha256))
			buf.WriteString("\n\n")
		}
		pkgDir := filepath.Join(distsRoot, "binary-"+arch)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			return err
		}
		packagesPath := filepath.Join(pkgDir, "Packages")
		body := strings.TrimSuffix(buf.String(), "\n")
		if err := os.WriteFile(packagesPath, []byte(body), 0o644); err != nil {
			return err
		}
		if err := writeGzip(packagesPath+".gz", []byte(body)); err != nil {
			return err
		}
	}
	return writeDebRelease(repoRoot, cfg)
}

func collectDebFiles(poolDir string) ([]string, error) {
	if st, err := os.Stat(poolDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	} else if !st.IsDir() {
		return nil, nil
	}
	var out []string
	if err := filepath.WalkDir(poolDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".deb") {
			out = append(out, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func formatPackagesEntry(c debControl, filename string, size int64, md5s, sha256s string) string {
	keys := []string{
		"Package", "Source", "Version", "Section", "Priority", "Architecture",
		"Essential", "Maintainer", "Installed-Size", "Depends", "Pre-Depends",
		"Recommends", "Suggests", "Conflicts", "Replaces", "Provides", "Description",
	}
	var b strings.Builder
	for _, k := range keys {
		if v := c.get(k); v != "" {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
	}
	fmt.Fprintf(&b, "Filename: %s\n", filename)
	fmt.Fprintf(&b, "Size: %d\n", size)
	fmt.Fprintf(&b, "MD5sum: %s\n", md5s)
	fmt.Fprintf(&b, "SHA256: %s\n", sha256s)
	return strings.TrimSuffix(b.String(), "\n")
}

func fileHashes(path string) (md5s string, sha256s string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	h1 := md5.New()
	h2 := sha256.New()
	if _, err := io.Copy(io.MultiWriter(h1, h2), f); err != nil {
		return "", "", err
	}
	return hex.EncodeToString(h1.Sum(nil)), hex.EncodeToString(h2.Sum(nil)), nil
}

func writeGzip(dst string, data []byte) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := gz.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeDebRelease(repoRoot string, cfg db.DebRepoConfig) error {
	distDir := filepath.Join(repoRoot, "dists", cfg.Codename)
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return err
	}
	type indexed struct {
		rel    string
		md5    string
		sha256 string
		size   int64
	}
	var files []indexed
	for _, arch := range cfg.Architectures {
		base := filepath.Join(distDir, cfg.Component, "binary-"+arch)
		for _, name := range []string{"Packages", "Packages.gz"} {
			p := filepath.Join(base, name)
			b, err := os.ReadFile(p)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return err
			}
			rel, err := filepath.Rel(distDir, p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			md5s, sha256s := hashBytes(b)
			files = append(files, indexed{rel: rel, md5: md5s, sha256: sha256s, size: int64(len(b))})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })

	var b strings.Builder
	origin := cfg.Origin
	if origin == "" {
		origin = "repoforge"
	}
	label := cfg.Label
	if label == "" {
		label = cfg.Codename
	}
	suite := cfg.Suite
	if suite == "" {
		suite = cfg.Codename
	}
	desc := cfg.Description
	if desc == "" {
		desc = "Repository managed by repoforge"
	}
	fmt.Fprintf(&b, "Origin: %s\n", origin)
	fmt.Fprintf(&b, "Label: %s\n", label)
	fmt.Fprintf(&b, "Suite: %s\n", suite)
	fmt.Fprintf(&b, "Codename: %s\n", cfg.Codename)
	fmt.Fprintf(&b, "Version: 1\n")
	fmt.Fprintf(&b, "Date: %s\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Architectures: %s\n", strings.Join(cfg.Architectures, " "))
	fmt.Fprintf(&b, "Components: %s\n", cfg.Component)
	fmt.Fprintf(&b, "Description: %s\n", desc)
	b.WriteString("MD5Sum:\n")
	for _, f := range files {
		fmt.Fprintf(&b, " %s %d %s\n", f.md5, f.size, f.rel)
	}
	b.WriteString("SHA256:\n")
	for _, f := range files {
		fmt.Fprintf(&b, " %s %d %s\n", f.sha256, f.size, f.rel)
	}
	releasePath := filepath.Join(distDir, "Release")
	return os.WriteFile(releasePath, []byte(b.String()), 0o644)
}

func hashBytes(b []byte) (md5s string, sha256s string) {
	h1 := md5.Sum(b)
	h2 := sha256.Sum256(b)
	return hex.EncodeToString(h1[:]), hex.EncodeToString(h2[:])
}

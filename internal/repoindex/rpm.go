package repoindex

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RebuildRpmIndex(ctx context.Context, createrepoPath, repoRoot string) error {
	rpmsDir := filepath.Join(repoRoot, "rpms")
	if err := os.MkdirAll(rpmsDir, 0o755); err != nil {
		return err
	}
	hasRPM, err := dirHasSuffixFiles(rpmsDir, ".rpm")
	if err != nil {
		return err
	}
	if !hasRPM {
		return nil
	}
	args := []string{"--verbose", "."}
	if _, err := os.Stat(filepath.Join(rpmsDir, "repodata", "repomd.xml")); err == nil {
		args = []string{"--verbose", "--update", "."}
	}
	cmd := exec.CommandContext(ctx, createrepoPath, args...)
	cmd.Dir = rpmsDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("createrepo_c: %w: %s", err, string(out))
	}
	return nil
}

func dirHasSuffixFiles(dir string, suffix string) (bool, error) {
	var found bool
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), strings.ToLower(suffix)) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found, err
}

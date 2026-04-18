package repoindex

import (
	"context"
	"fmt"

	"repoforge/internal/config"
	"repoforge/internal/db"
)

type Set struct {
	cfg config.Config
}

func NewSet(cfg config.Config) *Set {
	return &Set{cfg: cfg}
}

func (s *Set) Reindex(ctx context.Context, repo db.Repository, repoRoot string) error {
	switch repo.Type {
	case db.RepoFile:
		return nil
	case db.RepoRpm:
		return RebuildRpmIndex(ctx, s.cfg.CreaterepoPath, repoRoot)
	case db.RepoDeb:
		cfg, err := db.ParseDebConfig(repo.ConfigJSON)
		if err != nil {
			return fmt.Errorf("deb config: %w", err)
		}
		return RebuildDebIndex(ctx, repoRoot, cfg)
	default:
		return fmt.Errorf("unknown repository type %q", repo.Type)
	}
}

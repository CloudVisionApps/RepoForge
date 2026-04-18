package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) CreateRepository(ctx context.Context, name, slug string, typ RepoType, config any) (Repository, error) {
	var cfgJSON []byte
	var err error
	if config == nil {
		cfgJSON = []byte("{}")
	} else {
		cfgJSON, err = json.Marshal(config)
		if err != nil {
			return Repository{}, err
		}
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO repositories (name, slug, type, config_json) VALUES (?, ?, ?, ?)`,
		name, slug, string(typ), string(cfgJSON),
	)
	if err != nil {
		return Repository{}, err
	}
	id, _ := res.LastInsertId()
	return s.GetRepositoryByID(ctx, id)
}

func (s *Store) GetRepositoryByID(ctx context.Context, id int64) (Repository, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, slug, type, config_json, created_at FROM repositories WHERE id = ?`, id,
	)
	return scanRepo(row)
}

func (s *Store) GetRepositoryBySlug(ctx context.Context, slug string) (Repository, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, slug, type, config_json, created_at FROM repositories WHERE slug = ?`, slug,
	)
	return scanRepo(row)
}

func scanRepo(row *sql.Row) (Repository, error) {
	var r Repository
	var typ string
	if err := row.Scan(&r.ID, &r.Name, &r.Slug, &typ, &r.ConfigJSON, &r.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Repository{}, ErrNotFound
		}
		return Repository{}, err
	}
	r.Type = RepoType(typ)
	return r, nil
}

func (s *Store) ListRepositories(ctx context.Context) ([]Repository, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, slug, type, config_json, created_at FROM repositories ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Repository
	for rows.Next() {
		var r Repository
		var typ string
		if err := rows.Scan(&r.ID, &r.Name, &r.Slug, &typ, &r.ConfigJSON, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Type = RepoType(typ)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) ListArtifactsByRepositorySlug(ctx context.Context, slug string) ([]Artifact, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.repository_id, a.logical_path, a.sha256, a.size, a.content_type, a.created_at
		FROM artifacts a
		INNER JOIN repositories r ON r.id = a.repository_id
		WHERE r.slug = ?
		ORDER BY a.id DESC
	`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Artifact
	for rows.Next() {
		var a Artifact
		if err := rows.Scan(&a.ID, &a.RepositoryID, &a.LogicalPath, &a.SHA256, &a.Size, &a.ContentType, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) CreateArtifact(ctx context.Context, repoID int64, logicalPath, sha256, contentType string, size int64) (Artifact, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO artifacts (repository_id, logical_path, sha256, size, content_type) VALUES (?, ?, ?, ?, ?)`,
		repoID, logicalPath, sha256, size, contentType,
	)
	if err != nil {
		return Artifact{}, err
	}
	id, _ := res.LastInsertId()
	row := s.db.QueryRowContext(ctx,
		`SELECT id, repository_id, logical_path, sha256, size, content_type, created_at FROM artifacts WHERE id = ?`, id,
	)
	var a Artifact
	if err := row.Scan(&a.ID, &a.RepositoryID, &a.LogicalPath, &a.SHA256, &a.Size, &a.ContentType, &a.CreatedAt); err != nil {
		return Artifact{}, err
	}
	return a, nil
}

func (s *Store) StartIndexRun(ctx context.Context, repoID int64) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO index_runs (repository_id, started_at, status) VALUES (?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), 'running')`,
		repoID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) FinishIndexRun(ctx context.Context, runID int64, ok bool, errMsg string) error {
	status := "ok"
	var em any
	if !ok {
		status = "failed"
		em = errMsg
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE index_runs SET finished_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), status = ?, error_message = ? WHERE id = ?`,
		status, em, runID,
	)
	return err
}

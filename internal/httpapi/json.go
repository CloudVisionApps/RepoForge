package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"

	"repoforge/internal/db"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		http.Error(w, `{"error":"encode"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func repoJSON(r db.Repository) map[string]any {
	var raw any
	if len(r.ConfigJSON) > 0 {
		_ = json.Unmarshal(r.ConfigJSON, &raw)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	return map[string]any{
		"id":         r.ID,
		"name":       r.Name,
		"slug":       r.Slug,
		"type":       r.Type,
		"config":     raw,
		"created_at": r.CreatedAt,
	}
}

func artifactJSON(a db.Artifact) map[string]any {
	return map[string]any{
		"id":             a.ID,
		"repository_id":  a.RepositoryID,
		"logical_path":   a.LogicalPath,
		"sha256":         a.SHA256,
		"size":           a.Size,
		"content_type":   a.ContentType,
		"created_at":     a.CreatedAt,
	}
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Listen         string
	DataDir        string
	DBPath         string
	BearerToken    string
	CreaterepoPath string
	MaxUploadBytes int64
}

func Load() (Config, error) {
	cfg := Config{
		Listen:         getenv("LISTEN", ":8080"),
		DataDir:        getenv("DATA_DIR", "./data"),
		DBPath:         getenv("DB_PATH", "./repoforge.db"),
		CreaterepoPath: getenv("CREATEREPO_C_PATH", "createrepo_c"),
		MaxUploadBytes: 512 << 20, // 512 MiB
	}
	if s := strings.TrimSpace(os.Getenv("REPOFORGE_TOKEN")); s != "" {
		cfg.BearerToken = s
	}
	if s := os.Getenv("MAX_UPLOAD_BYTES"); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("MAX_UPLOAD_BYTES: %w", err)
		}
		if n <= 0 {
			return Config{}, fmt.Errorf("MAX_UPLOAD_BYTES must be positive")
		}
		cfg.MaxUploadBytes = n
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

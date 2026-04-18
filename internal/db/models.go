package db

import "encoding/json"

type RepoType string

const (
	RepoDeb  RepoType = "deb"
	RepoRpm  RepoType = "rpm"
	RepoFile RepoType = "file"
)

type Repository struct {
	ID         int64
	Name       string
	Slug       string
	Type       RepoType
	ConfigJSON []byte
	CreatedAt  string
}

type Artifact struct {
	ID            int64
	RepositoryID  int64
	LogicalPath   string
	SHA256        string
	Size          int64
	ContentType   string
	CreatedAt     string
}

type IndexRun struct {
	ID            int64
	RepositoryID  int64
	StartedAt     string
	FinishedAt    *string
	Status        string
	ErrorMessage  *string
}

type DebRepoConfig struct {
	Codename       string   `json:"codename"`
	Component      string   `json:"component"`
	Architectures  []string `json:"architectures"`
	Origin         string   `json:"origin,omitempty"`
	Label          string   `json:"label,omitempty"`
	Suite          string   `json:"suite,omitempty"`
	Description    string   `json:"description,omitempty"`
}

func ParseDebConfig(raw []byte) (DebRepoConfig, error) {
	var c DebRepoConfig
	if len(raw) == 0 || string(raw) == "{}" {
		return DebRepoConfig{
			Codename:      "stable",
			Component:     "main",
			Architectures: []string{"amd64"},
		}, nil
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return DebRepoConfig{}, err
	}
	if c.Codename == "" {
		c.Codename = "stable"
	}
	if c.Component == "" {
		c.Component = "main"
	}
	if len(c.Architectures) == 0 {
		c.Architectures = []string{"amd64"}
	}
	return c, nil
}

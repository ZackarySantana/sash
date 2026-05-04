package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Name     string    `json:"name"`
	Frontend Frontend  `json:"frontend"`
	Bindings *Bindings `json:"bindings,omitempty"`
	GoMain   string    `json:"goMain"`
}

type Bindings struct {
	APIImportPath string         `json:"apiImportPath"`
	Type          string         `json:"type"`
	GoBindingsDir string         `json:"goBindingsDir"`
	TSBindingsDir string         `json:"tsBindingsDir"`
	MountPath     string         `json:"mountPath"`
	DevListenAddr string         `json:"devListenAddr"`
	DevAPIBaseURL string         `json:"devAPIBaseURL"`
	SSEEvents     []SSEEventDecl `json:"sseEvents,omitempty"`
}

type SSEEventDecl struct {
	Event  string            `json:"event"`
	Fields map[string]string `json:"fields,omitempty"`
}

type Frontend struct {
	Dir          string `json:"dir"`
	Install      string `json:"install"`
	Build        string `json:"build"`
	Dev          string `json:"dev"`
	DevServerURL string `json:"devServerURL"`
	DevPort      int    `json:"devPort"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

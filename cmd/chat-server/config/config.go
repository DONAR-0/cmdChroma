// Package config provides configuration types for the chat server.
package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config represents the full server configuration.
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Chroma      ChromaConfig      `yaml:"chroma"`
	Embedder    EmbedderConfig    `yaml:"embedder"`
	LLM         LLMConfig         `yaml:"llm"`
	Collections []CollectionEntry `yaml:"collections"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host               string   `yaml:"host"`
	Port               int      `yaml:"port"`
	APIKey             string   `yaml:"api_key"`
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins"`
}

// ChromaConfig holds ChromaDB connection settings.
type ChromaConfig struct {
	URL      string `yaml:"url"`
	Tenant   string `yaml:"tenant"`
	Database string `yaml:"database"`
}

// EmbedderConfig holds ONNX embedder settings.
type EmbedderConfig struct {
	ModelPath   string `yaml:"model_path"`
	LibraryPath string `yaml:"library_path"`
}

// LLMConfig holds LLM provider settings.
type LLMConfig struct {
	DefaultModel string `yaml:"default_model"`
	OllamaURL    string `yaml:"ollama_url"`
	NIMURL       string `yaml:"nim_url"`
}

// CollectionEntry describes a single collection to manage.
type CollectionEntry struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Load reads the config file and applies env var overrides.
// If the file doesn't exist, returns an error and the caller can fall back to Default.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Start with defaults so any field missing from the YAML file retains its default value.
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("yaml parse error: %w", err)
	}

	applyEnvOverrides(cfg)

	return cfg, nil
}

// Default returns a config with all defaults set.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:               "0.0.0.0",
			Port:               6700,
			APIKey:             "changeme",
			CORSAllowedOrigins: []string{"http://localhost:3000"},
		},
		Chroma: ChromaConfig{
			URL:      "http://localhost:8000",
			Tenant:   "default_tenant",
			Database: "default_database",
		},
		Embedder: EmbedderConfig{
			ModelPath:   "./models/all-MiniLM-L6-v2/model.onnx",
			LibraryPath: "./models/onnx_runtime/lib/libonnxruntime.so",
		},
		LLM: LLMConfig{
			DefaultModel: "google/gemma-2-2b-it",
			OllamaURL:    "http://localhost:11434",
			NIMURL:       "https://integrate.api.nvidia.com/v1",
		},
		Collections: nil,
	}
}

// applyEnvOverrides reads prefixed env vars and overrides config fields.
// Prefix: CHAT_
// Examples: CHAT_SERVER_API_KEY, CHAT_SERVER_PORT, CHAT_CHROMA_URL, CHAT_LLM_DEFAULT_MODEL.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("CHAT_SERVER_API_KEY"); v != "" {
		cfg.Server.APIKey = v
	}

	if v := os.Getenv("CHAT_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}

	if v := os.Getenv("CHAT_CHROMA_URL"); v != "" {
		cfg.Chroma.URL = v
	}

	if v := os.Getenv("CHAT_OLLAMA_URL"); v != "" {
		cfg.LLM.OllamaURL = v
	}

	if v := os.Getenv("CHAT_NIM_URL"); v != "" {
		cfg.LLM.NIMURL = v
	}

	if v := os.Getenv("CHAT_LLM_DEFAULT_MODEL"); v != "" {
		cfg.LLM.DefaultModel = v
	}

	if v := os.Getenv("CHAT_EMBEDDER_MODEL_PATH"); v != "" {
		cfg.Embedder.ModelPath = v
	}

	if v := os.Getenv("CHAT_EMBEDDER_LIBRARY_PATH"); v != "" {
		cfg.Embedder.LibraryPath = v
	}
}

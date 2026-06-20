package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Env prefix shared by every env-var override. Flat (CHROMA_MCP_<FIELD>) per
// testing.md §1.5 and to keep Claude Code .mcp.json injections short.
const envPrefix = "CHROMA_MCP_"

// Exposed so T-13 main.go can quote a single source of truth in --help /
// error messages without re-importing constants.
const (
	DefaultMode        = ""
	DefaultTransport   = "stdio"
	DefaultPort        = 9090
	DefaultCollection  = "mcp_memory"
	DefaultLogLevel    = "info"
	DefaultChromaURL   = "http://localhost:8000"
	DefaultTenant      = "default_tenant"
	DefaultDatabase    = "default_database"
	DefaultModelPath   = "./models/all-MiniLM-L6-v2/model.onnx"
	DefaultLibraryPath = "./models/onnx_runtime/lib/libonnxruntime.so"
)

// Config is the resolved mcp-server configuration. Lifecycle (cascade):
//
//	cfg := Default()        // pure defaults
//	cfg.ApplyEnvOverrides() // env wins over defaults
//	yaml.Unmarshal(...)     // file wins over env (applied inside Load)
//	cfg.Validate()          // last gate, called by Load() and LoadAuto().
//
// The CLI flag overlay lives on a separate struct so T-13 (stdlib flag) does
// not need to be imported from this package.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Chroma   ChromaConfig   `yaml:"chroma"`
	Embedder EmbedderConfig `yaml:"embedder"`
}

type ServerConfig struct {
	Mode       string `yaml:"mode"`
	Transport  string `yaml:"transport"`
	Port       int    `yaml:"port"`
	Collection string `yaml:"collection"`
	LogLevel   string `yaml:"log_level"`
}

type ChromaConfig struct {
	URL      string `yaml:"url"`
	Tenant   string `yaml:"tenant"`
	Database string `yaml:"database"`
}

type EmbedderConfig struct {
	ModelPath   string `yaml:"model_path"`
	LibraryPath string `yaml:"library_path"`
}

// CLIOverrides holds values parsed by T-13 from stdlib flag. Zero-value
// fields are ignored so the underlying cascade (env / file / default) is
// preserved when a flag was not provided.
type CLIOverrides struct {
	Mode           string
	Transport      string
	Port           int
	Collection     string
	LogLevel       string
	ChromaURL      string
	ChromaTenant   string
	ChromaDatabase string
	ModelPath      string
	LibraryPath    string
	ConfigFile     string // reserved for future --config flag support
}

// Default returns the canonical defaults. Pure function.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Mode:       DefaultMode,
			Transport:  DefaultTransport,
			Port:       DefaultPort,
			Collection: DefaultCollection,
			LogLevel:   DefaultLogLevel,
		},
		Chroma: ChromaConfig{
			URL:      DefaultChromaURL,
			Tenant:   DefaultTenant,
			Database: DefaultDatabase,
		},
		Embedder: EmbedderConfig{
			ModelPath:   DefaultModelPath,
			LibraryPath: DefaultLibraryPath,
		},
	}
}

// Load merges defaults <- YAML file at path (if non-empty) <- env overrides,
// and finally calls Validate. Empty path is a legal "no file" call.
//
// Errors:
//   - os.ReadFile → wrapped error (os.ErrNotExist or read failure)
//   - yaml.Unmarshal → wrapped yaml parse error
//   - Validate      → wrapped validation error
func Load(path string) (*Config, error) {
	cfg := Default()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("config file %q: %w", path, err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("yaml parse error in %q: %w", path, err)
		}
	}

	cfg.ApplyEnvOverrides()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config invalid: %w", err)
	}

	return cfg, nil
}

// LoadAuto searches the canonical mcp-server config-file locations and loads
// the first match. Falls through to defaults + env if no file is found.
//
// Search order (first wins):
//  1. ./.cmdChroma-mcp.yaml
//  2. ./cmdChroma-mcp.yaml
//  3. ./config/cmdChroma-mcp.yaml
//  4. ~/.config/cmdChroma/mcp.yaml
//  5. $XDG_CONFIG_HOME/cmdChroma/mcp.yaml
func LoadAuto() (*Config, error) {
	if path, err := findConfigFile(); err == nil {
		return Load(path)
	}

	cfg := Default()
	cfg.ApplyEnvOverrides()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config invalid: %w", err)
	}

	return cfg, nil
}

// findConfigFile is package-private — direct callers pass an explicit path
// to Load. Tested directly because the package is `main`.
func findConfigFile() (string, error) {
	projectLocations := []string{
		"./.cmdChroma-mcp.yaml",
		"./cmdChroma-mcp.yaml",
		"./config/cmdChroma-mcp.yaml",
	}
	for _, p := range projectLocations {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Home-dir lookup is best-effort: a failure here still falls through to
	// the XDG fallback rather than aborting the search.
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".config", "cmdChroma", "mcp.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidate := filepath.Join(xdg, "cmdChroma", "mcp.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", errors.New("no config file found")
}

// ApplyEnvOverrides reads flat-prefix CHROMA_MCP_<FIELD> variables and
// overlays them onto the receiver. Empty / unset variables are ignored
// (intentional: caller might come from defaulted Load). Atoi failures on
// CHROMA_MCP_PORT are silently dropped to mirror chat-server config's
// permissiveness; tracked as future enhancement.
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv(envPrefix + "MODE"); v != "" {
		c.Server.Mode = v
	}

	if v := os.Getenv(envPrefix + "TRANSPORT"); v != "" {
		c.Server.Transport = v
	}

	if v := os.Getenv(envPrefix + "PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}

	if v := os.Getenv(envPrefix + "COLLECTION"); v != "" {
		c.Server.Collection = v
	}

	if v := os.Getenv(envPrefix + "LOG_LEVEL"); v != "" {
		c.Server.LogLevel = v
	}

	if v := os.Getenv(envPrefix + "URL"); v != "" {
		c.Chroma.URL = v
	}

	if v := os.Getenv(envPrefix + "TENANT"); v != "" {
		c.Chroma.Tenant = v
	}

	if v := os.Getenv(envPrefix + "DATABASE"); v != "" {
		c.Chroma.Database = v
	}

	if v := os.Getenv(envPrefix + "MODEL_PATH"); v != "" {
		c.Embedder.ModelPath = v
	}

	if v := os.Getenv(envPrefix + "LIBRARY_PATH"); v != "" {
		c.Embedder.LibraryPath = v
	}
}

// ApplyOverrides overlays parsed CLI flags onto the receiver. Zero-value
// fields of o are left in place — the cascade already populated them.
// Called by main.go (T-13) after Load / LoadAuto so flags win over everything.
func (c *Config) ApplyOverrides(o CLIOverrides) {
	if o.Mode != "" {
		c.Server.Mode = o.Mode
	}

	if o.Transport != "" {
		c.Server.Transport = o.Transport
	}

	if o.Port != 0 {
		c.Server.Port = o.Port
	}

	if o.Collection != "" {
		c.Server.Collection = o.Collection
	}

	if o.LogLevel != "" {
		c.Server.LogLevel = o.LogLevel
	}

	if o.ChromaURL != "" {
		c.Chroma.URL = o.ChromaURL
	}

	if o.ChromaTenant != "" {
		c.Chroma.Tenant = o.ChromaTenant
	}

	if o.ChromaDatabase != "" {
		c.Chroma.Database = o.ChromaDatabase
	}

	if o.ModelPath != "" {
		c.Embedder.ModelPath = o.ModelPath
	}

	if o.LibraryPath != "" {
		c.Embedder.LibraryPath = o.LibraryPath
	}
}

// Validate enforces invariants callers must satisfy before starting the
// server. Exercised by both Load and LoadAuto so an env-overridden invalid
// value still surfaces.
func (c *Config) Validate() error {
	switch c.Server.Mode {
	case "", "memory":
	default:
		return fmt.Errorf("invalid mode %q (must be \"\" or \"memory\")", c.Server.Mode)
	}

	switch c.Server.Transport {
	case "stdio", "http":
	default:
		return fmt.Errorf("invalid transport %q (must be stdio or http)", c.Server.Transport)
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port %d (must be 1-65535)", c.Server.Port)
	}

	if c.Server.Collection == "" {
		return errors.New("collection name cannot be empty")
	}

	if c.Chroma.URL == "" {
		return errors.New("chroma url cannot be empty")
	}

	return nil
}

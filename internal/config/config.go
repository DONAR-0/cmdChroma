package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for cmdChroma.
type Config struct {
	Chroma  ChromaConfig
	Model   ModelConfig
	Logging LoggingConfig
}

// ChromaConfig holds ChromaDB connection settings.
type ChromaConfig struct {
	URL      string
	Host     string
	Port     string
	Tenant   string
	Database string
	Timeout  time.Duration
}

// ModelConfig holds paths to embedding model and tokenizer.
type ModelConfig struct {
	ONNXModel string
	Tokenizer string
	ONNXLib   string
}

// LoggingConfig controls logging behavior.
type LoggingConfig struct {
	Verbose bool
	Level   string
	Format  string
}

// ConfigFile represents the structure of a YAML configuration file on disk.
type ConfigFile struct {
	Version  string             `yaml:"version"`
	Chroma   ConfigFileChroma   `yaml:"chroma"`
	Model    ConfigFileModel    `yaml:"model"`
	Logging  ConfigFileLogging  `yaml:"logging"`
	Features ConfigFileFeatures `yaml:"features"`
}

// ConfigFileChroma represents the chroma section in the YAML config file.
type ConfigFileChroma struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Tenant   string `yaml:"tenant"`
	Database string `yaml:"database"`
	Timeout  int    `yaml:"timeout"`
}

// ConfigFileModel represents the model section in the YAML config file.
type ConfigFileModel struct {
	ONNXModel string `yaml:"onnx_model"`
	Tokenizer string `yaml:"tokenizer"`
	ONNXLib   string `yaml:"onnx_lib"`
}

// ConfigFileLogging represents the logging section in the YAML config file.
type ConfigFileLogging struct {
	Level   string `yaml:"level"`
	Format  string `yaml:"format"`
	Verbose bool   `yaml:"verbose"`
}

// ConfigFileFeatures represents the features section in the YAML config file.
type ConfigFileFeatures struct {
	CreateCollection ConfigFileCreateCollection `yaml:"create_collection"`
}

// ConfigFileCreateCollection represents feature flags for create_collection.
type ConfigFileCreateCollection struct {
	AutoCreateDatabase bool `yaml:"auto_create_database"`
}

// ErrNoConfigFile is returned when no config file is found in any of the search locations.
var ErrNoConfigFile = errors.New("no config file found")

// findConfigFile searches for a configuration file in the standard locations.
// Returns the path to the first found file, or ErrNoConfigFile if none found.
// Search order:
// 1. ./.cmdChroma.yaml (project-local)
// 2. ./cmdChroma.yaml (alternative project-local)
// 3. ./config/cmdChroma.yaml (optional subdirectory)
// 4. ~/.config/cmdChroma/config.yaml (global user config)
// 5. $XDG_CONFIG_HOME/cmdChroma/config.yaml (respects XDG env var)
func findConfigFile() (string, error) {
	// Check project-local locations first
	projectLocations := []string{
		"./.cmdChroma.yaml",
		"./cmdChroma.yaml",
		"./config/cmdChroma.yaml",
	}

	for _, path := range projectLocations {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Check global locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	globalLocations := []string{
		filepath.Join(homeDir, ".config", "cmdChroma", "config.yaml"),
	}

	// Check XDG config home if set
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		globalLocations = append(globalLocations, filepath.Join(xdgConfig, "cmdChroma", "config.yaml"))
	}

	for _, path := range globalLocations {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", ErrNoConfigFile
}

// applyFileConfig copies values from config file, only if field is non-zero
func (cfg *Config) applyFileConfig(f *ConfigFile) {
	if f.Chroma.Host != "" {
		cfg.Chroma.Host = f.Chroma.Host
	}

	if f.Chroma.Port != "" {
		cfg.Chroma.Port = f.Chroma.Port
	}

	if f.Chroma.Tenant != "" {
		cfg.Chroma.Tenant = f.Chroma.Tenant
	}

	if f.Chroma.Database != "" {
		cfg.Chroma.Database = f.Chroma.Database
	}

	if f.Chroma.Timeout != 0 {
		cfg.Chroma.Timeout = time.Duration(f.Chroma.Timeout) * time.Second
	}

	if f.Model.ONNXModel != "" {
		cfg.Model.ONNXModel = f.Model.ONNXModel
	}

	if f.Model.Tokenizer != "" {
		cfg.Model.Tokenizer = f.Model.Tokenizer
	}

	if f.Model.ONNXLib != "" {
		cfg.Model.ONNXLib = f.Model.ONNXLib
	}

	if f.Logging.Level != "" {
		cfg.Logging.Level = f.Logging.Level
	}

	if f.Logging.Format != "" {
		cfg.Logging.Format = f.Logging.Format
	}

	if f.Logging.Verbose {
		cfg.Logging.Verbose = f.Logging.Verbose
	}
}

// applyEnvVars overlays environment variables onto the config
func (cfg *Config) applyEnvVars() {
	if v := os.Getenv("CHROMA_HOST"); v != "" {
		cfg.Chroma.Host = v
	}

	if v := os.Getenv("CHROMA_PORT"); v != "" {
		cfg.Chroma.Port = v
	}

	if v := os.Getenv("CHROMA_TENANT"); v != "" {
		cfg.Chroma.Tenant = v
	}

	if v := os.Getenv("CHROMA_DATABASE"); v != "" {
		cfg.Chroma.Database = v
	}

	if v := os.Getenv("CHROMA_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}

	if v := os.Getenv("CHROMA_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}

	if v := os.Getenv("CHROMA_MODEL_PATH"); v != "" {
		cfg.Model.ONNXModel = v
	}

	if v := os.Getenv("CHROMA_TOKENIZER_PATH"); v != "" {
		cfg.Model.Tokenizer = v
	}

	if v := os.Getenv("CHROMA_ONNX_LIB"); v != "" {
		cfg.Model.ONNXLib = v
	}
}

// loadFromFile reads and parses a single YAML config file
func (cfg *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var fileCfg ConfigFile
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	cfg.applyFileConfig(&fileCfg)

	return nil
}

// applyCLIFlags applies CLI flag values to the config (highest priority)
// This was formerly LoadFromCLI
func applyCLIFlags(c *cli.Command, cfg *Config) {
	// Host: if explicitly set via CLI, override; else, if still empty, use flag default
	if c.IsSet("host") {
		cfg.Chroma.Host = c.String("host")
	} else if cfg.Chroma.Host == "" {
		cfg.Chroma.Host = c.String("host")
	}

	// Port
	if c.IsSet("port") {
		cfg.Chroma.Port = c.String("port")
	} else if cfg.Chroma.Port == "" {
		cfg.Chroma.Port = c.String("port")
	}

	// Tenant
	if c.IsSet("tenant") {
		cfg.Chroma.Tenant = c.String("tenant")
	} else if cfg.Chroma.Tenant == "" {
		cfg.Chroma.Tenant = c.String("tenant")
	}

	// Database
	if c.IsSet("database") {
		cfg.Chroma.Database = c.String("database")
	} else if cfg.Chroma.Database == "" {
		cfg.Chroma.Database = c.String("database")
	}

	// Timeout
	if c.IsSet("timeout") {
		cfg.Chroma.Timeout = time.Duration(c.Int("timeout")) * time.Second
	} else if cfg.Chroma.Timeout == 0 {
		cfg.Chroma.Timeout = time.Duration(c.Int("timeout")) * time.Second
	}

	// Model paths
	if c.IsSet("model-path") {
		cfg.Model.ONNXModel = c.String("model-path")
	} else if cfg.Model.ONNXModel == "" {
		cfg.Model.ONNXModel = c.String("model-path")
	}

	if c.IsSet("tokenizer-path") {
		cfg.Model.Tokenizer = c.String("tokenizer-path")
	} else if cfg.Model.Tokenizer == "" {
		cfg.Model.Tokenizer = c.String("tokenizer-path")
	}

	if c.IsSet("onnx-lib") {
		cfg.Model.ONNXLib = c.String("onnx-lib")
	} else if cfg.Model.ONNXLib == "" {
		cfg.Model.ONNXLib = c.String("onnx-lib")
	}

	// Verbose: only override if explicitly set
	if c.IsSet("verbose") {
		cfg.Logging.Verbose = c.Bool("verbose")
	}
	// Note: not setting default false since zero value is fine
}

// LoadConfig loads configuration from all sources with proper precedence:
// CLI flags > Environment variables > Local config > Global config > Defaults
func LoadConfig(c *cli.Command) (*Config, error) {
	cfg := &Config{}

	// 1. Load config file(s) – respects --config flag if set
	if cfgPath := c.String("config"); cfgPath != "" {
		// Explicit path provided via --config
		if cfgPath == "" {
			// --config "" disables config files
			// (nothing to load)
		} else {
			if err := cfg.loadFromFile(cfgPath); err != nil {
				return nil, fmt.Errorf("config file error: %w", err)
			}
		}
	} else {
		// Search default locations (local then global)
		if filePath, err := findConfigFile(); err == nil {
			if err := cfg.loadFromFile(filePath); err != nil {
				return nil, fmt.Errorf("config file error: %w", err)
			}
		} else if !errors.Is(err, ErrNoConfigFile) {
			return nil, fmt.Errorf("config error: %w", err)
		}
		// ErrNoConfigFile is silent – just use defaults
	}

	// 2. Overlay environment variables
	cfg.applyEnvVars()

	// 3. Overlay CLI flags (highest priority)
	applyCLIFlags(c, cfg)

	// 4. Resolve paths and final defaults
	if err := cfg.resolvePaths(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// resolvePaths computes defaults relative to the executable if needed.
func (cfg *Config) resolvePaths() error {
	// Build Chroma URL
	if cfg.Chroma.Host != "" && cfg.Chroma.Port != "" {
		cfg.Chroma.URL = fmt.Sprintf("http://%s:%s", cfg.Chroma.Host, cfg.Chroma.Port)
	}

	// Resolve model paths relative to executable if not provided
	ex, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	binDir := filepath.Dir(ex)
	projectRoot := filepath.Join(binDir, "..")

	if cfg.Model.ONNXModel == "" {
		cfg.Model.ONNXModel = filepath.Join(projectRoot, "models/all-MiniLM-L6-v2/model.onnx")
	}

	if cfg.Model.Tokenizer == "" {
		cfg.Model.Tokenizer = filepath.Join(projectRoot, "models/all-MiniLM-L6-v2/tokenizer.json")
	}

	if cfg.Model.ONNXLib == "" {
		cfg.Model.ONNXLib = filepath.Join(projectRoot, "models/onnx_runtime/lib/libonnxruntime.so")
	}

	return nil
}

// GetChromaURL returns the full ChromaDB URL.
func (cfg *Config) GetChromaURL() string {
	return cfg.Chroma.URL
}

// GetTenant returns the tenant name.
func (cfg *Config) GetTenant() string {
	return cfg.Chroma.Tenant
}

// GetDatabase returns the database name.
func (cfg *Config) GetDatabase() string {
	return cfg.Chroma.Database
}

// GetEmbedderModel returns the path to the ONNX model.
func (cfg *Config) GetEmbedderModel() string {
	return cfg.Model.ONNXModel
}

// GetEmbedderTokenizer returns the path to the tokenizer JSON.
func (cfg *Config) GetEmbedderTokenizer() string {
	return cfg.Model.Tokenizer
}

// GetEmbedderLib returns the path to the ONNX runtime library.
func (cfg *Config) GetEmbedderLib() string {
	return cfg.Model.ONNXLib
}

// IsVerbose returns whether verbose logging is enabled.
func (cfg *Config) IsVerbose() bool {
	return cfg.Logging.Verbose
}

// ConfigForEmbedder returns paths needed by the embedder.
func (cfg *Config) ConfigForEmbedder() (modelPath, tokenizerPath, libPath string) {
	return cfg.Model.ONNXModel, cfg.Model.Tokenizer, cfg.Model.ONNXLib
}

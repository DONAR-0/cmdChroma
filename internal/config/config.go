package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"
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
}

// ConfigFile represents the structure of a YAML configuration file on disk.
type ConfigFile struct {
	Version  string `yaml:"version"`
	Chroma   ConfigFileChroma `yaml:"chroma"`
	Model    ConfigFileModel `yaml:"model"`
	Logging  ConfigFileLogging `yaml:"logging"`
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

// LoadFromCLI builds a Config from a cli.Command.
// It resolves paths, validates settings, and applies defaults.
func LoadFromCLI(c *cli.Command) (*Config, error) {
	cfg := &Config{
		Chroma: ChromaConfig{
			Host:     c.String("host"),
			Port:     c.String("port"),
			Tenant:   c.String("tenant"),
			Database: c.String("database"),
			Timeout:  time.Duration(c.Int("timeout")) * time.Second,
		},
		Model: ModelConfig{
			ONNXModel: c.String("model-path"),
			Tokenizer: c.String("tokenizer-path"),
			ONNXLib:   c.String("onnx-lib"),
		},
		Logging: LoggingConfig{
			Verbose: c.Bool("verbose"),
		},
	}

	// Resolve URLs and paths
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

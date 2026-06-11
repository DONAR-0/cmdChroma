package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/urfave/cli/v3"
)

func TestConfig_Getters(t *testing.T) {
	cfg := &Config{
		Chroma: ChromaConfig{
			Host:     "testhost",
			Port:     "1234",
			Tenant:   "testtenant",
			Database: "testdb",
		},
		Model: ModelConfig{
			ONNXModel: "model.onnx",
			Tokenizer: "tokenizer.json",
			ONNXLib:   "libonnxruntime.so",
		},
		Logging: LoggingConfig{
			Verbose: true,
		},
	}

	// Chroma getters
	if cfg.GetTenant() != "testtenant" {
		t.Errorf("GetTenant() = %q, want %q", cfg.GetTenant(), "testtenant")
	}

	if cfg.GetDatabase() != "testdb" {
		t.Errorf("GetDatabase() = %q, want %q", cfg.GetDatabase(), "testdb")
	}

	// Model getters
	if cfg.GetEmbedderModel() != "model.onnx" {
		t.Errorf("GetEmbedderModel() = %q, want %q", cfg.GetEmbedderModel(), "model.onnx")
	}

	if cfg.GetEmbedderTokenizer() != "tokenizer.json" {
		t.Errorf("GetEmbedderTokenizer() = %q, want %q", cfg.GetEmbedderTokenizer(), "tokenizer.json")
	}

	if cfg.GetEmbedderLib() != "libonnxruntime.so" {
		t.Errorf("GetEmbedderLib() = %q, want %q", cfg.GetEmbedderLib(), "libonnxruntime.so")
	}

	// Logging getter
	if !cfg.IsVerbose() {
		t.Error("IsVerbose() = false, want true")
	}

	// ConfigForEmbedder
	m, tkn, lib := cfg.ConfigForEmbedder()
	if m != "model.onnx" || tkn != "tokenizer.json" || lib != "libonnxruntime.so" {
		t.Errorf("ConfigForEmbedder() = (%s,%s,%s), want (model.onnx, tokenizer.json, libonnxruntime.so)", m, tkn, lib)
	}
}

func TestConfig_GetChromaURL(t *testing.T) {
	cfg := &Config{
		Chroma: ChromaConfig{
			URL: "http://custom:8000",
		},
	}
	if cfg.GetChromaURL() != "http://custom:8000" {
		t.Errorf("GetChromaURL() = %q, want %q", cfg.GetChromaURL(), "http://custom:8000")
	}
}

// --- Coverage for previously uncovered functions ---

func TestConfig_ApplyFileConfig(t *testing.T) {
	cfg := &Config{}
	file := &ConfigFile{
		Version: "1.0",
		Chroma: ConfigFileChroma{
			Host:     "filehost",
			Port:     "9090",
			Tenant:   "filetenant",
			Database: "filedb",
		},
		Model: ConfigFileModel{
			ONNXModel: "file-model.onnx",
			Tokenizer: "file-tokenizer.json",
			ONNXLib:   "file-lib.so",
		},
		Logging: ConfigFileLogging{
			Level:   "debug",
			Format:  "json",
			Verbose: true,
		},
	}
	cfg.applyFileConfig(file)

	if cfg.Chroma.Host != "filehost" {
		t.Errorf("Host = %q, want %q", cfg.Chroma.Host, "filehost")
	}

	if cfg.Chroma.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Chroma.Port, "9090")
	}

	if cfg.Chroma.Tenant != "filetenant" {
		t.Errorf("Tenant = %q, want %q", cfg.Chroma.Tenant, "filetenant")
	}

	if cfg.Chroma.Database != "filedb" {
		t.Errorf("Database = %q, want %q", cfg.Chroma.Database, "filedb")
	}

	if cfg.Model.ONNXModel != "file-model.onnx" {
		t.Errorf("ONNXModel = %q, want %q", cfg.Model.ONNXModel, "file-model.onnx")
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Level = %q, want %q", cfg.Logging.Level, "debug")
	}

	if cfg.Logging.Format != "json" {
		t.Errorf("Format = %q, want %q", cfg.Logging.Format, "json")
	}

	if !cfg.Logging.Verbose {
		t.Error("Verbose = false, want true")
	}
}

func TestConfig_ApplyEnvVars(t *testing.T) {
	if err := os.Setenv("CHROMA_HOST", "envhost"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("CHROMA_PORT", "7070"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("CHROMA_TENANT", "envtenant"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("CHROMA_DATABASE", "envdb"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("CHROMA_LOG_LEVEL", "warn"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("CHROMA_LOG_FORMAT", "json"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("CHROMA_MODEL_PATH", "/env/model.onnx"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("CHROMA_TOKENIZER_PATH", "/env/tokenizer.json"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("CHROMA_ONNX_LIB", "/env/lib.so"); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Unsetenv("CHROMA_HOST"); err != nil {
			t.Error(err)
		}

		if err := os.Unsetenv("CHROMA_PORT"); err != nil {
			t.Error(err)
		}

		if err := os.Unsetenv("CHROMA_TENANT"); err != nil {
			t.Error(err)
		}

		if err := os.Unsetenv("CHROMA_DATABASE"); err != nil {
			t.Error(err)
		}

		if err := os.Unsetenv("CHROMA_LOG_LEVEL"); err != nil {
			t.Error(err)
		}

		if err := os.Unsetenv("CHROMA_LOG_FORMAT"); err != nil {
			t.Error(err)
		}

		if err := os.Unsetenv("CHROMA_MODEL_PATH"); err != nil {
			t.Error(err)
		}

		if err := os.Unsetenv("CHROMA_TOKENIZER_PATH"); err != nil {
			t.Error(err)
		}

		if err := os.Unsetenv("CHROMA_ONNX_LIB"); err != nil {
			t.Error(err)
		}
	}()

	cfg := &Config{}
	cfg.applyEnvVars()

	if cfg.Chroma.Host != "envhost" {
		t.Errorf("Host = %q, want %q", cfg.Chroma.Host, "envhost")
	}

	if cfg.Chroma.Port != "7070" {
		t.Errorf("Port = %q, want %q", cfg.Chroma.Port, "7070")
	}

	if cfg.Chroma.Tenant != "envtenant" {
		t.Errorf("Tenant = %q, want %q", cfg.Chroma.Tenant, "envtenant")
	}

	if cfg.Chroma.Database != "envdb" {
		t.Errorf("Database = %q, want %q", cfg.Chroma.Database, "envdb")
	}

	if cfg.Logging.Level != "warn" {
		t.Errorf("Level = %q, want %q", cfg.Logging.Level, "warn")
	}

	if cfg.Model.ONNXModel != "/env/model.onnx" {
		t.Errorf("ONNXModel = %q, want %q", cfg.Model.ONNXModel, "/env/model.onnx")
	}
}

func TestConfig_ApplyCLIFlags(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "host", Value: "clihost"},
			&cli.StringFlag{Name: "port", Value: "8080"},
			&cli.StringFlag{Name: "tenant", Value: "clitenant"},
			&cli.StringFlag{Name: "database", Value: "clidb"},
			&cli.IntFlag{Name: "timeout", Value: 30},
			&cli.StringFlag{Name: "model-path", Value: "/cli/model.onnx"},
			&cli.StringFlag{Name: "tokenizer-path", Value: "/cli/tokenizer.json"},
			&cli.StringFlag{Name: "onnx-lib", Value: "/cli/lib.so"},
			&cli.BoolFlag{Name: "verbose"},
		},
	}
	if err := cmd.Set("host", "clihost"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("port", "8080"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("tenant", "clitenant"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("database", "clidb"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("timeout", "30"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("model-path", "/cli/model.onnx"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("tokenizer-path", "/cli/tokenizer.json"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("onnx-lib", "/cli/lib.so"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("verbose", "true"); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{}
	applyCLIFlags(cmd, cfg)

	if cfg.Chroma.Host != "clihost" {
		t.Errorf("Host = %q, want %q", cfg.Chroma.Host, "clihost")
	}

	if cfg.Chroma.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Chroma.Port, "8080")
	}

	if cfg.Chroma.Tenant != "clitenant" {
		t.Errorf("Tenant = %q, want %q", cfg.Chroma.Tenant, "clitenant")
	}

	if cfg.Chroma.Database != "clidb" {
		t.Errorf("Database = %q, want %q", cfg.Chroma.Database, "clidb")
	}

	if cfg.Chroma.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Chroma.Timeout, 30*time.Second)
	}

	if cfg.Model.ONNXModel != "/cli/model.onnx" {
		t.Errorf("ONNXModel = %q, want %q", cfg.Model.ONNXModel, "/cli/model.onnx")
	}

	if !cfg.Logging.Verbose {
		t.Error("Verbose = false, want true")
	}
}

func TestConfig_LoadConfig_NoConfigFile(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "host", Value: "localhost"},
			&cli.StringFlag{Name: "port", Value: "8000"},
			&cli.StringFlag{Name: "tenant"},
			&cli.StringFlag{Name: "database"},
			&cli.StringFlag{Name: "model-path"},
			&cli.StringFlag{Name: "tokenizer-path"},
			&cli.StringFlag{Name: "onnx-lib"},
			&cli.IntFlag{Name: "timeout"},
			&cli.BoolFlag{Name: "verbose"},
		},
	}

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Error(err)
		}
	}()

	cfg, err := LoadConfig(cmd)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Chroma.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Chroma.Host, "localhost")
	}

	if cfg.Chroma.Port != "8000" {
		t.Errorf("Port = %q, want %q", cfg.Chroma.Port, "8000")
	}

	if cfg.Chroma.Timeout != 0 {
		t.Errorf("Timeout = %v, want 0", cfg.Chroma.Timeout)
	}
}

func TestConfig_LoadConfig_FromFile(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Error(err)
		}
	}()

	yamlData := "chroma:\n  host: filehost\n  port: \"1234\"\n  tenant: filetenant\n  database: filedb\n"
	if err := os.WriteFile(".cmdChroma.yaml", []byte(yamlData), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "host"},
			&cli.StringFlag{Name: "port"},
			&cli.StringFlag{Name: "tenant"},
			&cli.StringFlag{Name: "database"},
			&cli.StringFlag{Name: "model-path"},
			&cli.StringFlag{Name: "tokenizer-path"},
			&cli.StringFlag{Name: "onnx-lib"},
			&cli.IntFlag{Name: "timeout"},
			&cli.BoolFlag{Name: "verbose"},
			&cli.StringFlag{Name: "config"},
		},
	}

	// Explicitly set config flag to avoid searching default locations
	if err := cmd.Set("config", ".cmdChroma.yaml"); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cmd)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Chroma.Host != "filehost" {
		t.Errorf("Host = %q, want %q", cfg.Chroma.Host, "filehost")
	}

	if cfg.Chroma.Port != "1234" {
		t.Errorf("Port = %q, want %q", cfg.Chroma.Port, "1234")
	}

	if cfg.Chroma.Tenant != "filetenant" {
		t.Errorf("Tenant = %q, want %q", cfg.Chroma.Tenant, "filetenant")
	}

	if cfg.Chroma.Database != "filedb" {
		t.Errorf("Database = %q, want %q", cfg.Chroma.Database, "filedb")
	}
}

func TestConfig_ResolvePaths(t *testing.T) {
	cfg := &Config{
		Model: ModelConfig{
			ONNXModel: "/custom/model.onnx",
			Tokenizer: "/custom/tokenizer.json",
			ONNXLib:   "/custom/lib.so",
		},
	}

	err := cfg.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths failed: %v", err)
	}

	if cfg.Model.ONNXModel != "/custom/model.onnx" {
		t.Errorf("ONNXModel = %q, want %q", cfg.Model.ONNXModel, "/custom/model.onnx")
	}

	if !filepath.IsAbs(cfg.Model.ONNXModel) {
		t.Errorf("ONNXModel should be absolute, got %q", cfg.Model.ONNXModel)
	}
}

func TestConfigFindConfigFile_NoFile(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Error(err)
		}
	}()

	_, err := findConfigFile()
	if !errors.Is(err, ErrNoConfigFile) {
		t.Errorf("Expected ErrNoConfigFile, got %v", err)
	}
}

func TestConfigFindConfigFile_WithFile(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Error(err)
		}
	}()

	if err := os.WriteFile("cmdChroma.yaml", []byte("chroma:\n  host: testhost\n"), 0644); err != nil {
		t.Fatal(err)
	}

	path, err := findConfigFile()
	if err != nil {
		t.Fatalf("findConfigFile failed: %v", err)
	}

	if path != "./cmdChroma.yaml" {
		t.Errorf("Path = %q, want %q", path, "./cmdChroma.yaml")
	}
}

func TestConfig_LoadFromFile_InvalidPath(t *testing.T) {
	cfg := &Config{}

	err := cfg.loadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

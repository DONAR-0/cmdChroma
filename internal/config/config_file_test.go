package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigFile_YAMLUnmarshal(t *testing.T) {
	const yamlData = `
version: "1.0"
chroma:
  host: testhost
  port: "1234"
  timeout: 60
logging:
  level: debug
  format: json
features:
  create_collection:
    auto_create_database: true
`

	var cfg File
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Validate each field
	if cfg.Version != "1.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0")
	}

	if cfg.Chroma.Host != "testhost" {
		t.Errorf("Chroma.Host = %q, want %q", cfg.Chroma.Host, "testhost")
	}

	if cfg.Chroma.Port != "1234" {
		t.Errorf("Chroma.Port = %q, want %q", cfg.Chroma.Port, "1234")
	}

	if cfg.Chroma.Timeout != 60 {
		t.Errorf("Chroma.Timeout = %d, want %d", cfg.Chroma.Timeout, 60)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}

	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q, want %q", cfg.Logging.Format, "json")
	}

	if !cfg.Features.CreateCollection.AutoCreateDatabase {
		t.Errorf("Features.CreateCollection.AutoCreateDatabase = false, want true")
	}
}

func TestConfigFile_PartialYAML(t *testing.T) {
	const yamlData = `
chroma:
  host: only-host-set
`

	var cfg File
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Host should be set
	if cfg.Chroma.Host != "only-host-set" {
		t.Errorf("Host not set correctly, got %q", cfg.Chroma.Host)
	}

	// All other fields should be zero (empty string, 0, false)
	if cfg.Chroma.Port != "" {
		t.Errorf("Port should be empty, got %q", cfg.Chroma.Port)
	}

	if cfg.Chroma.Timeout != 0 {
		t.Errorf("Timeout should be 0, got %d", cfg.Chroma.Timeout)
	}

	if cfg.Logging.Level != "" {
		t.Errorf("Logging.Level should be empty, got %q", cfg.Logging.Level)
	}

	if cfg.Features.CreateCollection.AutoCreateDatabase {
		t.Errorf("AutoCreateDatabase should be false, got true")
	}
}

func TestConfigFile_EmptyYAML(t *testing.T) {
	var cfg File
	if err := yaml.Unmarshal([]byte(""), &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should not panic, all fields zero
	if cfg.Version != "" {
		t.Errorf("Version should be empty, got %q", cfg.Version)
	}

	if cfg.Chroma.Host != "" {
		t.Errorf("Chroma.Host should be empty, got %q", cfg.Chroma.Host)
	}

	if cfg.Logging.Level != "" {
		t.Errorf("Logging.Level should be empty, got %q", cfg.Logging.Level)
	}
}

func TestConfigFile_AllSections(t *testing.T) {
	// Verify all sections and fields are present and can be set
	yamlData := `
version: "1.0"
chroma:
  host: myhost
  port: "8080"
  tenant: mytenant
  database: mydb
  timeout: 45
model:
  onnx_model: /path/to/model.onnx
  tokenizer: /path/to/tokenizer.json
  onnx_lib: /path/to/libonnxruntime.so
logging:
  level: warn
  format: text
  verbose: true
features:
  create_collection:
    auto_create_database: true
`

	var cfg File
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Check all fields
	if cfg.Version != "1.0" {
		t.Errorf("Version mismatch")
	}

	if cfg.Chroma.Host != "myhost" {
		t.Errorf("Chroma.Host = %q", cfg.Chroma.Host)
	}

	if cfg.Chroma.Port != "8080" {
		t.Errorf("Chroma.Port = %q", cfg.Chroma.Port)
	}

	if cfg.Chroma.Tenant != "mytenant" {
		t.Errorf("Chroma.Tenant = %q", cfg.Chroma.Tenant)
	}

	if cfg.Chroma.Database != "mydb" {
		t.Errorf("Chroma.Database = %q", cfg.Chroma.Database)
	}

	if cfg.Chroma.Timeout != 45 {
		t.Errorf("Chroma.Timeout = %d", cfg.Chroma.Timeout)
	}

	if cfg.Model.ONNXModel != "/path/to/model.onnx" {
		t.Errorf("Model.ONNXModel = %q", cfg.Model.ONNXModel)
	}

	if cfg.Model.Tokenizer != "/path/to/tokenizer.json" {
		t.Errorf("Model.Tokenizer = %q", cfg.Model.Tokenizer)
	}

	if cfg.Model.ONNXLib != "/path/to/libonnxruntime.so" {
		t.Errorf("Model.ONNXLib = %q", cfg.Model.ONNXLib)
	}

	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level = %q", cfg.Logging.Level)
	}

	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %q", cfg.Logging.Format)
	}

	if !cfg.Logging.Verbose {
		t.Errorf("Logging.Verbose = false, want true")
	}

	if !cfg.Features.CreateCollection.AutoCreateDatabase {
		t.Errorf("Features.CreateCollection.AutoCreateDatabase = false, want true")
	}
}

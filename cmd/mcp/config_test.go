package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir is a tiny helper that swaps CWD for the lifetime of a test and
// restores it on cleanup. Identical pattern is used for HOME below.
func chdir(t *testing.T, dir string) {
	t.Helper()

	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd(): %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q): %v", dir, err)
	}

	t.Cleanup(func() { _ = os.Chdir(old) })
}

// setHome points $HOME (and blanks $XDG_CONFIG_HOME so the XDG branch doesn't
// silently swallow the test) at dir for the lifetime of the test.
func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", "")
}

// writeFile is shorthand so the test cases don't drown in os.WriteFile noise.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", path, err)
	}
}

// -----------------------------------------------------------------------------
// 1. Default values
// -----------------------------------------------------------------------------

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Server.Mode != DefaultMode {
		t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, DefaultMode)
	}

	if cfg.Server.Transport != DefaultTransport {
		t.Errorf("Server.Transport = %q, want %q", cfg.Server.Transport, DefaultTransport)
	}

	if cfg.Server.Port != DefaultPort {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, DefaultPort)
	}

	if cfg.Server.Collection != DefaultCollection {
		t.Errorf("Server.Collection = %q, want %q", cfg.Server.Collection, DefaultCollection)
	}

	if cfg.Server.LogLevel != DefaultLogLevel {
		t.Errorf("Server.LogLevel = %q, want %q", cfg.Server.LogLevel, DefaultLogLevel)
	}

	if cfg.Chroma.URL == "" || cfg.Chroma.Tenant == "" || cfg.Chroma.Database == "" {
		t.Errorf("Chroma defaults should not be empty: %+v", cfg.Chroma)
	}

	if cfg.Embedder.ModelPath == "" || cfg.Embedder.LibraryPath == "" {
		t.Errorf("Embedder defaults should not be empty: %+v", cfg.Embedder)
	}
}

// -----------------------------------------------------------------------------
// 2. Env overrides — table over the flat CHROMA_MCP_<FIELD> namespace
// -----------------------------------------------------------------------------

func TestApplyEnvOverrides(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
		want func(t *testing.T, c *Config)
	}{
		{
			name: "mode=memory", key: "CHROMA_MCP_MODE", val: "memory",
			want: func(t *testing.T, c *Config) {
				if c.Server.Mode != "memory" {
					t.Errorf("Mode = %q, want memory", c.Server.Mode)
				}
			},
		},
		{
			name: "transport=http", key: "CHROMA_MCP_TRANSPORT", val: "http",
			want: func(t *testing.T, c *Config) {
				if c.Server.Transport != "http" {
					t.Errorf("Transport = %q, want http", c.Server.Transport)
				}
			},
		},
		{
			name: "port=9091", key: "CHROMA_MCP_PORT", val: "9091",
			want: func(t *testing.T, c *Config) {
				if c.Server.Port != 9091 {
					t.Errorf("Port = %d, want 9091", c.Server.Port)
				}
			},
		},
		{
			name: "collection=foo", key: "CHROMA_MCP_COLLECTION", val: "foo",
			want: func(t *testing.T, c *Config) {
				if c.Server.Collection != "foo" {
					t.Errorf("Collection = %q, want foo", c.Server.Collection)
				}
			},
		},
		{
			name: "log_level=debug", key: "CHROMA_MCP_LOG_LEVEL", val: "debug",
			want: func(t *testing.T, c *Config) {
				if c.Server.LogLevel != "debug" {
					t.Errorf("LogLevel = %q, want debug", c.Server.LogLevel)
				}
			},
		},
		{
			name: "chroma_url", key: "CHROMA_MCP_URL", val: "http://env-host:9999",
			want: func(t *testing.T, c *Config) {
				if c.Chroma.URL != "http://env-host:9999" {
					t.Errorf("Chroma.URL = %q, want http://env-host:9999", c.Chroma.URL)
				}
			},
		},
		{
			name: "tenant", key: "CHROMA_MCP_TENANT", val: "env-tenant",
			want: func(t *testing.T, c *Config) {
				if c.Chroma.Tenant != "env-tenant" {
					t.Errorf("Chroma.Tenant = %q, want env-tenant", c.Chroma.Tenant)
				}
			},
		},
		{
			name: "database", key: "CHROMA_MCP_DATABASE", val: "env-db",
			want: func(t *testing.T, c *Config) {
				if c.Chroma.Database != "env-db" {
					t.Errorf("Chroma.Database = %q, want env-db", c.Chroma.Database)
				}
			},
		},
		{
			name: "model_path", key: "CHROMA_MCP_MODEL_PATH", val: "/env/model.onnx",
			want: func(t *testing.T, c *Config) {
				if c.Embedder.ModelPath != "/env/model.onnx" {
					t.Errorf("Embedder.ModelPath = %q, want /env/model.onnx", c.Embedder.ModelPath)
				}
			},
		},
		{
			name: "library_path", key: "CHROMA_MCP_LIBRARY_PATH", val: "/env/libonnxruntime.so",
			want: func(t *testing.T, c *Config) {
				if c.Embedder.LibraryPath != "/env/libonnxruntime.so" {
					t.Errorf("Embedder.LibraryPath = %q, want /env/libonnxruntime.so", c.Embedder.LibraryPath)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.val)

			cfg := Default()
			cfg.ApplyEnvOverrides()
			tc.want(t, cfg)
		})
	}
}

// Unset env vars must not clobber a non-default value already on the config
// (guards against the silent-call-after-Default pattern).
func TestApplyEnvOverrides_EmptyUnset(t *testing.T) {
	cfg := Default()
	cfg.Server.Transport = "http" // pre-populated

	t.Setenv("CHROMA_MCP_TRANSPORT", "")
	cfg.ApplyEnvOverrides()

	if cfg.Server.Transport != "http" {
		t.Errorf("empty env var overwrote non-default: got %q", cfg.Server.Transport)
	}
}

// Invalid CHROMA_MCP_PORT silently leaves the default in place (mirrors
// chat-server). Track as future enhancement.
func TestApplyEnvOverrides_PortParseErrorSwallowed(t *testing.T) {
	t.Setenv("CHROMA_MCP_PORT", "not-a-number")

	cfg := Default()
	cfg.ApplyEnvOverrides()

	if cfg.Server.Port != DefaultPort {
		t.Errorf("Port = %d, want default %d", cfg.Server.Port, DefaultPort)
	}
}

// -----------------------------------------------------------------------------
// 3. Load() — file merge + env overlay + validate
// -----------------------------------------------------------------------------

func TestLoad_EmptyPath(t *testing.T) {
	t.Setenv("CHROMA_MCP_PORT", "7070")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\"): %v", err)
	}

	if cfg.Server.Port != 7070 {
		t.Errorf("Port = %d, want 7070 (from env)", cfg.Server.Port)
	}

	if cfg.Server.Transport != DefaultTransport {
		t.Errorf("Transport = %q, want %q (default)", cfg.Server.Transport, DefaultTransport)
	}
}

func TestLoad_FileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	yaml := `
server:
  transport: http
  port: 8081
  collection: file-col
  log_level: debug
chroma:
  url: http://file-host:1234
  tenant: file-tenant
  database: file-db
embedder:
  model_path: /file/model.onnx
  library_path: /file/libonnxruntime.so
`
	writeFile(t, path, yaml)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q): %v", path, err)
	}

	if cfg.Server.Transport != "http" {
		t.Errorf("Transport = %q, want http", cfg.Server.Transport)
	}

	if cfg.Server.Port != 8081 {
		t.Errorf("Port = %d, want 8081", cfg.Server.Port)
	}

	if cfg.Server.Collection != "file-col" {
		t.Errorf("Collection = %q, want file-col", cfg.Server.Collection)
	}

	if cfg.Server.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.Server.LogLevel)
	}

	if cfg.Chroma.URL != "http://file-host:1234" {
		t.Errorf("Chroma.URL = %q", cfg.Chroma.URL)
	}

	if cfg.Chroma.Tenant != "file-tenant" {
		t.Errorf("Chroma.Tenant = %q", cfg.Chroma.Tenant)
	}

	if cfg.Chroma.Database != "file-db" {
		t.Errorf("Chroma.Database = %q", cfg.Chroma.Database)
	}

	if cfg.Embedder.ModelPath != "/file/model.onnx" {
		t.Errorf("Embedder.ModelPath = %q", cfg.Embedder.ModelPath)
	}

	if cfg.Embedder.LibraryPath != "/file/libonnxruntime.so" {
		t.Errorf("Embedder.LibraryPath = %q", cfg.Embedder.LibraryPath)
	}
}

func TestLoad_FileMissing(t *testing.T) {
	_, err := Load("/nonexistent/path/to/yaml")
	if err == nil {
		t.Fatal("Load() expected error on missing file")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want errors.Is(_, os.ErrNotExist)", err)
	}
}

func TestLoad_InvalidTransport(t *testing.T) {
	t.Setenv("CHROMA_MCP_TRANSPORT", "tcp")

	_, err := Load("")
	if err == nil {
		t.Fatal("Load() with invalid transport returned nil err")
	}

	if !strings.Contains(err.Error(), "invalid transport") {
		t.Errorf("err = %v, want contains 'invalid transport'", err)
	}
}

func TestLoad_PartialFile_FallsBackToDefaults(t *testing.T) {
	// Only one field set — the others should stay at defaults.
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.yaml")
	writeFile(t, path, "server:\n  collection: only-this\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}

	if cfg.Server.Collection != "only-this" {
		t.Errorf("Collection = %q, want only-this", cfg.Server.Collection)
	}

	if cfg.Server.Transport != DefaultTransport {
		t.Errorf("Transport = %q, want %q (default)", cfg.Server.Transport, DefaultTransport)
	}

	if cfg.Server.Port != DefaultPort {
		t.Errorf("Port = %d, want %d (default)", cfg.Server.Port, DefaultPort)
	}
}

// -----------------------------------------------------------------------------
// 4. Cascade matrix — flag > env > file > default  (one composite test)
// -----------------------------------------------------------------------------

func TestCascadeMatrix(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "cascade.yaml")
	writeFile(t, filePath, `
server:
  transport: http       # file-set
  port: 7000            # file-set, should be overridden by env
  collection: file-col  # file-set, should be overridden by flag
  log_level: debug      # file-only
`)

	t.Setenv("CHROMA_MCP_PORT", "8000")  // env beats file on port
	t.Setenv("CHROMA_MCP_LOG_LEVEL", "") // blank env doesn't override file's "debug"

	cfg, err := Load(filePath)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}

	// 1. file-only values leave the file setting intact.
	if cfg.Server.Transport != "http" {
		t.Errorf("Transport = %q, want http (file)", cfg.Server.Transport)
	}

	if cfg.Server.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug (file)", cfg.Server.LogLevel)
	}

	if cfg.Server.Collection != "file-col" {
		t.Errorf("Collection = %q, want file-col (file)", cfg.Server.Collection)
	}

	// 2. env overrides file.
	if cfg.Server.Port != 8000 {
		t.Errorf("Port = %d, want 8000 (env > file 7000)", cfg.Server.Port)
	}

	// 3. flag overrides env / file / default.
	cfg.ApplyOverrides(CLIOverrides{Collection: "flag-col", Port: 9999})

	if cfg.Server.Collection != "flag-col" {
		t.Errorf("Collection = %q, want flag-col (flag > file)", cfg.Server.Collection)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("Port = %d, want 9999 (flag > env)", cfg.Server.Port)
	}
}

// -----------------------------------------------------------------------------
// 5. ApplyOverrides — zero-value semantics
// -----------------------------------------------------------------------------

func TestApplyOverrides_ZeroIgnored(t *testing.T) {
	cfg := Default()
	cfg.Server.Port = 1234
	cfg.Server.Collection = "stay"

	// All zero except a single string — the port=0 must NOT clobber 1234.
	cfg.ApplyOverrides(CLIOverrides{Transport: "http"})

	if cfg.Server.Transport != "http" {
		t.Errorf("Transport = %q, want http", cfg.Server.Transport)
	}

	if cfg.Server.Port != 1234 {
		t.Errorf("zero Port clobbered non-default: got %d, want 1234", cfg.Server.Port)
	}

	if cfg.Server.Collection != "stay" {
		t.Errorf("zero Collection clobbered non-default: got %q, want stay", cfg.Server.Collection)
	}
}

func TestApplyOverrides_AllSet(t *testing.T) {
	cfg := Default()
	cfg.ApplyOverrides(CLIOverrides{
		Transport:      "http",
		Port:           9099,
		Collection:     "x",
		LogLevel:       "warn",
		ChromaURL:      "http://override:8000",
		ChromaTenant:   "t-override",
		ChromaDatabase: "d-override",
		ModelPath:      "/override/model.onnx",
		LibraryPath:    "/override/lib.so",
	})

	if cfg.Server.Transport != "http" || cfg.Server.Port != 9099 || cfg.Server.Collection != "x" || cfg.Server.LogLevel != "warn" {
		t.Error("Server overrides did not stick")
	}

	if cfg.Chroma.URL != "http://override:8000" || cfg.Chroma.Tenant != "t-override" || cfg.Chroma.Database != "d-override" {
		t.Error("Chroma overrides did not stick")
	}

	if cfg.Embedder.ModelPath != "/override/model.onnx" || cfg.Embedder.LibraryPath != "/override/lib.so" {
		t.Error("Embedder overrides did not stick")
	}
}

// -----------------------------------------------------------------------------
// 6. Validate — table over bad cases + sanity for the good case
// -----------------------------------------------------------------------------

func TestValidate_GoodConfig(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() on Default = %v, want nil", err)
	}
}

func TestValidate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantSub string
	}{
		{
			name:    "bad mode",
			mutate:  func(c *Config) { c.Server.Mode = "invalid" },
			wantSub: "invalid mode",
		},
		{
			name:    "bad transport",
			mutate:  func(c *Config) { c.Server.Transport = "tcp" },
			wantSub: "invalid transport",
		},
		{
			name:    "port too low",
			mutate:  func(c *Config) { c.Server.Port = 0 },
			wantSub: "invalid port",
		},
		{
			name:    "port too high",
			mutate:  func(c *Config) { c.Server.Port = 70000 },
			wantSub: "invalid port",
		},
		{
			name:    "empty collection",
			mutate:  func(c *Config) { c.Server.Collection = "" },
			wantSub: "collection name cannot be empty",
		},
		{
			name:    "empty chroma url",
			mutate:  func(c *Config) { c.Chroma.URL = "" },
			wantSub: "chroma url cannot be empty",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			tc.mutate(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() = nil, want error")
			}

			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("err = %q, want contains %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// 7. findConfigFile + LoadAuto
// -----------------------------------------------------------------------------

// writeYAML helper just adds a YAML file at dir/<name> for search tests.
func writeYAML(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	writeFile(t, path, content)

	return path
}

func TestFindConfigFile_NoMatch(t *testing.T) {
	dir := t.TempDir()
	setHome(t, dir)
	chdir(t, dir)
	// Ensure candidate locations are absent (default state of temp dir).
	for _, name := range []string{".cmdChroma-mcp.yaml", "cmdChroma-mcp.yaml"} {
		_ = os.Remove(filepath.Join(dir, name))
	}

	if _, err := findConfigFile(); err == nil {
		t.Error("findConfigFile() expected error when no candidate files exist")
	}
}

func TestFindConfigFile_Priority(t *testing.T) {
	dir := t.TempDir()
	setHome(t, dir)
	chdir(t, dir)

	// First candidate wins.
	first := writeYAML(t, dir, ".cmdChroma-mcp.yaml", "server: {}\n")
	_ = writeYAML(t, dir, "cmdChroma-mcp.yaml", "server: {}\n") // also present

	got, err := findConfigFile()
	if err != nil {
		t.Fatalf("findConfigFile(): %v", err)
	}

	gotAbs, _ := filepath.Abs(got)

	firstAbs, _ := filepath.Abs(first)
	if gotAbs != firstAbs {
		t.Errorf("findConfigFile() = %q, want %q (first wins)", got, first)
	}
}

func TestLoadAuto_NoConfigFile(t *testing.T) {
	dir := t.TempDir()
	setHome(t, dir)
	chdir(t, dir)

	t.Setenv("CHROMA_MCP_PORT", "7777")

	cfg, err := LoadAuto()
	if err != nil {
		t.Fatalf("LoadAuto(): %v", err)
	}

	if cfg.Server.Transport != DefaultTransport {
		t.Errorf("Transport = %q, want %q", cfg.Server.Transport, DefaultTransport)
	}

	if cfg.Server.Port != 7777 {
		t.Errorf("Port = %d, want 7777 (env default path)", cfg.Server.Port)
	}
}

func TestLoadAuto_FindsFile(t *testing.T) {
	dir := t.TempDir()
	setHome(t, dir)
	chdir(t, dir)

	writeYAML(t, dir, "cmdChroma-mcp.yaml", "server:\n  collection: auto-found\n")

	cfg, err := LoadAuto()
	if err != nil {
		t.Fatalf("LoadAuto(): %v", err)
	}

	if cfg.Server.Collection != "auto-found" {
		t.Errorf("Collection = %q, want auto-found", cfg.Server.Collection)
	}
}

// -----------------------------------------------------------------------------
// 8. Quick smoke that confirms YAML error path returns a wrapped error.
// -----------------------------------------------------------------------------

func TestLoad_YAMLErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	// Tab indentation breaks YAML parsing in v3.
	writeFile(t, path, "server:\n\ttransport: http\n")

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() with malformed YAML returned nil err")
	}

	if !strings.Contains(err.Error(), "yaml parse error") {
		t.Errorf("err = %v, want contains 'yaml parse error'", err)
	}
}

// -----------------------------------------------------------------------------
// 9. End-to-end: Load then ApplyOverrides pulls the same Config together.
// -----------------------------------------------------------------------------

func TestLoadThenApplyOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "e2e.yaml")
	// File sets transport=http + port=7777; both are valid so Load() succeeds.
	// CLI then overrides collection (not in file) and confirms everything sticks.
	writeFile(t, path, "server:\n  port: 7777\n  transport: http\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}

	cfg.ApplyOverrides(CLIOverrides{Collection: "post-override"})

	if cfg.Server.Transport != "http" {
		t.Errorf("Transport = %q, want http", cfg.Server.Transport)
	}

	if cfg.Server.Port != 7777 {
		t.Errorf("Port = %d, want 7777", cfg.Server.Port)
	}

	if cfg.Server.Collection != "post-override" {
		t.Errorf("Collection = %q, want post-override", cfg.Server.Collection)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("post-override Validate() = %v, want nil", err)
	}
}

// Keep fmt referenced in case later edits remove fmt.Sprintf-style debug lines.
var _ = fmt.Sprintf

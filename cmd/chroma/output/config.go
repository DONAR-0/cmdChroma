package output

import (
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"
)

// OutputMode represents the output format mode.
type OutputMode int

const (
	// ModeHuman outputs human-readable formatted text.
	ModeHuman OutputMode = iota
	// ModeJSON outputs machine-readable JSON.
	ModeJSON
	// ModeMinimal outputs minimal text (errors only).
	ModeMinimal
)

func (m OutputMode) String() string {
	switch m {
	case ModeJSON:
		return "json"
	case ModeMinimal:
		return "minimal"
	default:
		return "human"
	}
}

// OutputConfig holds configuration for all output behavior.
type OutputConfig struct {
	// Mode determines the output format (human, json, minimal).
	Mode OutputMode

	// Verbose controls whether diagnostic logs are shown.
	// When false, only errors and warnings are logged.
	Verbose bool

	// NoColor disables colored output.
	NoColor bool

	// NoTTY indicates the output is not a terminal.
	// When true, TUI elements (spinners, progress) are disabled.
	NoTTY bool

	// Stdout is the output writer for user-facing messages.
	Stdout io.Writer

	// Stderr is the output writer for diagnostic logs.
	Stderr io.Writer
}

// DefaultOutputConfig returns the default configuration for interactive terminal use.
func DefaultOutputConfig() *OutputConfig {
	return &OutputConfig{
		Mode:    ModeHuman,
		Verbose: false,
		NoColor: false,
		NoTTY:   !isInteractive(),
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

// NewOutputConfig creates an OutputConfig from CLI flags.
func NewOutputConfig(cmd *cli.Command) *OutputConfig {
	cfg := DefaultOutputConfig()

	// JSON mode takes precedence over quiet/minimal
	if cmd.Bool("json") {
		cfg.Mode = ModeJSON
		cfg.Verbose = false // JSON mode suppresses verbose output
	} else if cmd.Bool("quiet") {
		cfg.Mode = ModeMinimal
		cfg.Verbose = false
	}

	if cmd.Bool("no-color") {
		cfg.NoColor = true
	}

	// Detect non-TTY environment
	if !isInteractive() || cmd.Bool("no-tui") {
		cfg.NoTTY = true
	}

	return cfg
}

// IsInteractive returns true if stdout is a terminal.
func IsInteractive() bool {
	return isInteractive()
}

// HasColorSupport returns true if the terminal supports colors.
func HasColorSupport() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	// Check for common color-capable terminals
	term := os.Getenv("TERM")
	supportedTerms := map[string]bool{
		"xterm":           true,
		"xterm-256color":  true,
		"screen":          true,
		"screen-256color": true,
		"tmux":            true,
		"vt100":           false, // Explicitly not supported
		"dumb":            false, // Explicitly not supported
	}

	if supported, ok := supportedTerms[term]; ok {
		return supported
	}

	// Default to true if TERM contains "color" or "256"
	return contains(term, "256") || contains(term, "color")
}

// isInteractive checks if stdout is a terminal.
func isInteractive() bool {
	file, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (TTY)
	return (file.Mode() & os.ModeCharDevice) != 0
}

// contains is a simple helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// String returns a human-readable description of the config.
func (c *OutputConfig) String() string {
	return fmt.Sprintf("OutputConfig{Mode:%s, Verbose:%v, NoColor:%v, NoTTY:%v}",
		c.Mode, c.Verbose, c.NoColor, c.NoTTY)
}

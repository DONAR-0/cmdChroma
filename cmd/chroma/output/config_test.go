package output

import (
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestMode_String(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeHuman, "human"},
		{ModeJSON, "json"},
		{ModeMinimal, "minimal"},
		{Mode(99), "human"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("Mode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Mode != ModeHuman {
		t.Errorf("DefaultConfig().Mode = %v, want ModeHuman", cfg.Mode)
	}

	if cfg.Verbose {
		t.Error("DefaultConfig().Verbose = true, want false")
	}

	if cfg.Stdout == nil {
		t.Error("DefaultConfig().Stdout = nil, want non-nil")
	}

	if cfg.Stderr == nil {
		t.Error("DefaultConfig().Stderr = nil, want non-nil")
	}
}

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name     string
		flags    func(t *testing.T) *cli.Command
		wantMode Mode
		wantTTY  bool
	}{
		{
			name: "json_mode",
			flags: func(t *testing.T) *cli.Command {
				cmd := &cli.Command{Flags: []cli.Flag{&cli.BoolFlag{Name: "json"}, &cli.BoolFlag{Name: "quiet"}, &cli.BoolFlag{Name: "no-color"}, &cli.BoolFlag{Name: "no-tui"}}}
				if err := cmd.Set("json", "true"); err != nil {
					t.Fatal(err)
				}

				return cmd
			},
			wantMode: ModeJSON,
			wantTTY:  false,
		},
		{
			name: "quiet_mode",
			flags: func(t *testing.T) *cli.Command {
				cmd := &cli.Command{Flags: []cli.Flag{&cli.BoolFlag{Name: "json"}, &cli.BoolFlag{Name: "quiet"}, &cli.BoolFlag{Name: "no-color"}, &cli.BoolFlag{Name: "no-tui"}}}
				if err := cmd.Set("quiet", "true"); err != nil {
					t.Fatal(err)
				}

				return cmd
			},
			wantMode: ModeMinimal,
			wantTTY:  false,
		},
		{
			name: "no_color",
			flags: func(t *testing.T) *cli.Command {
				cmd := &cli.Command{Flags: []cli.Flag{&cli.BoolFlag{Name: "json"}, &cli.BoolFlag{Name: "quiet"}, &cli.BoolFlag{Name: "no-color"}, &cli.BoolFlag{Name: "no-tui"}}}
				if err := cmd.Set("no-color", "true"); err != nil {
					t.Fatal(err)
				}

				return cmd
			},
			wantMode: ModeHuman,
			wantTTY:  false,
		},
		{
			name: "no_tui",
			flags: func(t *testing.T) *cli.Command {
				cmd := &cli.Command{Flags: []cli.Flag{&cli.BoolFlag{Name: "json"}, &cli.BoolFlag{Name: "quiet"}, &cli.BoolFlag{Name: "no-color"}, &cli.BoolFlag{Name: "no-tui"}}}
				if err := cmd.Set("no-tui", "true"); err != nil {
					t.Fatal(err)
				}

				return cmd
			},
			wantMode: ModeHuman,
			wantTTY:  false,
		},
		{
			name: "default_human",
			flags: func(t *testing.T) *cli.Command {
				return &cli.Command{Flags: []cli.Flag{&cli.BoolFlag{Name: "json"}, &cli.BoolFlag{Name: "quiet"}, &cli.BoolFlag{Name: "no-color"}, &cli.BoolFlag{Name: "no-tui"}}}
			},
			wantMode: ModeHuman,
			wantTTY:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig(tt.flags(t))
			if cfg.Mode != tt.wantMode {
				t.Errorf("NewConfig().Mode = %v, want %v", cfg.Mode, tt.wantMode)
			}
		})
	}
}

func TestConfig_String(t *testing.T) {
	cfg := &Config{Mode: ModeJSON, Verbose: true, NoColor: true, NoTTY: true}

	s := cfg.String()
	if !strings.Contains(s, "Mode:") {
		t.Errorf("Config.String() missing Mode, got: %s", s)
	}

	if !strings.Contains(s, "Verbose:true") && !strings.Contains(s, "Verbose: true") {
		t.Errorf("Config.String() missing Verbose, got: %s", s)
	}
}

func TestHasColorSupport(t *testing.T) {
	t.Setenv("TERM", "dumb")

	if HasColorSupport() {
		t.Error("HasColorSupport() = true, want false for TERM=dumb")
	}

	t.Setenv("TERM", "xterm-256color")

	if !HasColorSupport() {
		t.Error("HasColorSupport() = false, want true for TERM=xterm-256color")
	}

	t.Setenv("TERM", "vt100")

	if HasColorSupport() {
		t.Error("HasColorSupport() = true, want false for TERM=vt100")
	}

	t.Setenv("TERM", "something-with-256")

	if !HasColorSupport() {
		t.Error("HasColorSupport() = false, want true for TERM with 256")
	}
}

func TestIsInteractive(t *testing.T) {
	// In test environment, typically not interactive
	_ = IsInteractive() // just ensure it doesn't panic
}

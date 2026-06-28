package output

import (
	"strings"
	"testing"
)

func TestConsolePrinter_Stream(t *testing.T) {
	var buf strings.Builder

	cfg := &Config{
		Mode:   ModeHuman,
		Stdout: &buf,
		NoTTY:  true,
	}
	printer := NewConsolePrinter(cfg)

	printer.Stream("hello")

	if buf.String() != "hello" {
		t.Errorf("Expected 'hello', got %q", buf.String())
	}
}

func TestConsolePrinter_Stream_JSONMode(t *testing.T) {
	var buf strings.Builder

	cfg := &Config{
		Mode:   ModeJSON,
		Stdout: &buf,
		NoTTY:  true,
	}
	printer := NewConsolePrinter(cfg)

	printer.Stream("hello")

	if !strings.Contains(buf.String(), `"stream"`) {
		t.Errorf("Expected JSON with 'stream' key, got: %s", buf.String())
	}

	if !strings.Contains(buf.String(), `"hello"`) {
		t.Errorf("Expected JSON with 'hello' value, got: %s", buf.String())
	}
}

func TestConsolePrinter_Stdout(t *testing.T) {
	var buf strings.Builder

	cfg := &Config{
		Mode:   ModeHuman,
		Stdout: &buf,
		NoTTY:  true,
	}
	printer := NewConsolePrinter(cfg)

	w := printer.Stdout()
	if w == nil {
		t.Error("Expected non-nil Stdout writer")
	}

	// Writing to returned writer should affect the buffer
	_, _ = w.Write([]byte("direct write"))

	if buf.String() != "direct write" {
		t.Errorf("Expected 'direct write', got %q", buf.String())
	}
}

func TestConsolePrinter_Stream_Multiple(t *testing.T) {
	var buf strings.Builder

	cfg := &Config{
		Mode:   ModeHuman,
		Stdout: &buf,
		NoTTY:  true,
	}
	printer := NewConsolePrinter(cfg)

	printer.Stream("part1")
	printer.Stream("part2")

	if buf.String() != "part1part2" {
		t.Errorf("Expected 'part1part2', got %q", buf.String())
	}
}

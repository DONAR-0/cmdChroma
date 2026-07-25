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

func TestNewConsolePrinter(t *testing.T) {
	p := NewConsolePrinter(nil)
	if p == nil {
		t.Fatal("NewConsolePrinter(nil) = nil")
	}

	if p.config == nil {
		t.Error("NewConsolePrinter(nil).config = nil")
	}

	cfg := &Config{Mode: ModeJSON, Stdout: &strings.Builder{}, Stderr: &strings.Builder{}}

	p2 := NewConsolePrinter(cfg)
	if p2.config.Mode != ModeJSON {
		t.Errorf("config.Mode = %v, want ModeJSON", p2.config.Mode)
	}
}

func TestPrinter_Formatted_HumanMode(t *testing.T) {
	var buf strings.Builder

	p := &ConsolePrinter{config: &Config{Stdout: &buf, Stderr: &buf, Mode: ModeHuman}}

	p.Success("test %d", 1)
	p.Info("test %d", 2)
	p.Warn("test %d", 3)
	p.Error("test %d", 4)

	output := buf.String()
	if !strings.Contains(output, "✅") {
		t.Error("expected success emoji")
	}

	if !strings.Contains(output, "ℹ") {
		t.Error("expected info emoji")
	}

	if !strings.Contains(output, "⚠") {
		t.Error("expected warn emoji")
	}

	if !strings.Contains(output, "❌") {
		t.Error("expected error emoji")
	}

	if !strings.Contains(output, "test 1") {
		t.Error("expected 'test 1'")
	}

	if !strings.Contains(output, "test 4") {
		t.Error("expected 'test 4'")
	}
}

func TestPrinter_Formatted_JSONMode(t *testing.T) {
	var buf strings.Builder

	p := &ConsolePrinter{config: &Config{Stdout: &buf, Stderr: &buf, Mode: ModeJSON}}

	p.Success("test")
	p.Info("info msg")
	p.Error("err msg")

	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 JSON lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], `"success"`) {
		t.Errorf("line 0 missing level 'success': %s", lines[0])
	}

	if !strings.Contains(lines[1], `"info"`) {
		t.Errorf("line 1 missing level 'info': %s", lines[1])
	}

	if !strings.Contains(lines[2], `"err msg"`) {
		t.Errorf("line 2 missing message: %s", lines[2])
	}
}

func TestPrinter_Print(t *testing.T) {
	var buf strings.Builder

	p := &ConsolePrinter{config: &Config{Stdout: &buf, Stderr: &buf, Mode: ModeHuman}}
	p.Print("hello")

	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("Print(human) output = %q, want 'hello'", buf.String())
	}

	buf.Reset()

	p.config.Mode = ModeJSON
	p.Print("hello")

	if !strings.Contains(buf.String(), `"hello"`) {
		t.Errorf("Print(json) output = %q, want JSON with 'hello'", buf.String())
	}
}

func TestPrinter_Printf(t *testing.T) {
	var buf strings.Builder

	p := &ConsolePrinter{config: &Config{Stdout: &buf, Stderr: &buf, Mode: ModeHuman}}
	p.Printf("hello %s", "world")

	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("Printf output = %q, want 'hello world'", buf.String())
	}
}

func TestPrinter_Stdout(t *testing.T) {
	w := &strings.Builder{}

	p := &ConsolePrinter{config: &Config{Stdout: w, Stderr: &strings.Builder{}}}
	if p.Stdout() != w {
		t.Error("Stdout() returned wrong writer")
	}
}

func TestPrinter_PrintTable_Human(t *testing.T) {
	var buf strings.Builder

	cfg := &Config{Stdout: &buf, Stderr: &buf, Mode: ModeHuman, NoColor: true, NoTTY: true}
	p := &ConsolePrinter{config: cfg}

	headers := []string{"Name", "ID"}
	rows := [][]string{{"foo", "1"}, {"bar", "2"}}
	p.PrintTable(headers, rows)

	output := buf.String()
	if !strings.Contains(output, "Name") {
		t.Errorf("PrintTable missing header 'Name', got: %s", output)
	}

	if !strings.Contains(output, "foo") {
		t.Errorf("PrintTable missing row 'foo', got: %s", output)
	}
}

func TestPrinter_PrintTable_JSON(t *testing.T) {
	var buf strings.Builder

	cfg := &Config{Stdout: &buf, Stderr: &buf, Mode: ModeJSON}
	p := &ConsolePrinter{config: cfg}

	headers := []string{"Name", "ID"}
	rows := [][]string{{"foo", "1"}}
	p.PrintTable(headers, rows)

	output := buf.String()
	if !strings.Contains(output, "headers") || !strings.Contains(output, "Name") {
		t.Errorf("PrintTable(JSON) missing headers, got: %s", output)
	}
}

func TestFlattenRows(t *testing.T) {
	rows := [][]string{{"a", "b"}, {"c", "d"}}

	flat := flattenRows(rows)
	if len(flat) != 2 {
		t.Fatalf("flattenRows length = %d, want 2", len(flat))
	}

	if flat[0] != "a | b" {
		t.Errorf("flattenRows[0] = %q, want 'a | b'", flat[0])
	}

	if flat[1] != "c | d" {
		t.Errorf("flattenRows[1] = %q, want 'c | d'", flat[1])
	}
}

func TestGetColor(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"success", LightGreen},
		{"error", LightRed},
		{"warn", Yellow},
		{"info", Cyan},
		{"unknown", ""},
	}
	for _, tt := range tests {
		if got := getColor(tt.level); got != tt.want {
			t.Errorf("getColor(%q) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestPrinter_Formatted_NoColor(t *testing.T) {
	var buf strings.Builder

	p := &ConsolePrinter{config: &Config{Stdout: &buf, Stderr: &buf, Mode: ModeHuman, NoColor: true}}

	p.Success("test")

	output := buf.String()
	// NoColor still outputs Reset code (pre-existing behavior)
	// but shouldn't output color codes like LightGreen
	if strings.Contains(output, LightGreen) {
		t.Errorf("expected no LightGreen color code in output, got: %q", output)
	}

	if !strings.Contains(output, "✅") {
		t.Errorf("expected emoji in output, got: %q", output)
	}
}

func TestPrinter_PrintTable_Styled(t *testing.T) {
	var buf strings.Builder
	// Force styled mode (interactive TTY, no color disabled)
	cfg := &Config{Stdout: &buf, Stderr: &buf, Mode: ModeHuman, NoColor: false, NoTTY: false}
	p := &ConsolePrinter{config: cfg}

	headers := []string{"Name", "ID"}
	rows := [][]string{{"foo", "1"}, {"bar", "2"}}
	p.PrintTable(headers, rows)

	output := buf.String()
	// In styled mode, should use Bold+Cyan for header
	if !strings.Contains(output, "Name") {
		t.Errorf("PrintTable(styled) missing header, got: %s", output)
	}
	// Should contain color codes
	if !strings.Contains(output, "\033[1m") && !strings.Contains(output, "\033[36m") {
		t.Errorf("PrintTable(styled) missing color codes, got: %s", output)
	}
}

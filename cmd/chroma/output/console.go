package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ConsolePrinter handles user-facing output (stdout).
// All user-visible messages should go through this printer.
type ConsolePrinter struct {
	config *OutputConfig
}

// NewConsolePrinter creates a new ConsolePrinter with the given config.
func NewConsolePrinter(config *OutputConfig) *ConsolePrinter {
	if config == nil {
		config = DefaultOutputConfig()
	}

	return &ConsolePrinter{config: config}
}

// Success prints a success message with a checkmark.
func (p *ConsolePrinter) Success(msg string, args ...any) {
	p.printFormatted("success", "✅", msg, args...)
}

// Info prints an informational message.
func (p *ConsolePrinter) Info(msg string, args ...any) {
	p.printFormatted("info", "ℹ", msg, args...)
}

// Warn prints a warning message.
func (p *ConsolePrinter) Warn(msg string, args ...any) {
	p.printFormatted("warn", "⚠", msg, args...)
}

// Error prints an error message.
func (p *ConsolePrinter) Error(msg string, args ...any) {
	p.printFormatted("error", "❌", msg, args...)
}

// Print prints a plain message without formatting.
func (p *ConsolePrinter) Print(msg string) {
	if p.config.Mode == ModeJSON {
		p.printJSON(map[string]string{"message": msg})
		return
	}

	_, _ = fmt.Fprintln(p.config.Stdout, msg)
}

// Printf prints a formatted message.
func (p *ConsolePrinter) Printf(format string, args ...any) {
	if p.config.Mode == ModeJSON {
		p.printJSON(map[string]any{"message": fmt.Sprintf(format, args...)})
		return
	}

	_, _ = fmt.Fprintf(p.config.Stdout, format, args...)
}

// Stream prints content in streaming mode (no line breaks, direct write).
// In JSON mode, this formats as a streaming JSON event.
func (p *ConsolePrinter) Stream(content string) {
	if p.config.Mode == ModeJSON {
		p.printJSON(map[string]string{"stream": content})
		return
	}

	_, _ = fmt.Fprint(p.config.Stdout, content)
}

// Stdout returns the underlying stdout writer.
func (p *ConsolePrinter) Stdout() io.Writer {
	return p.config.Stdout
}

// PrintTable prints a formatted table.
// headers: column headers
// rows: table data rows
func (p *ConsolePrinter) PrintTable(headers []string, rows [][]string) {
	if p.config.Mode == ModeJSON {
		p.printJSON(map[string][]string{
			"headers": headers,
			"rows":    flattenRows(rows),
		})

		return
	}

	if p.config.NoColor || p.config.NoTTY {
		p.printTablePlain(headers, rows)
		return
	}

	p.printTableStyled(headers, rows)
}

// printFormatted prints with emoji prefix based on level.
func (p *ConsolePrinter) printFormatted(level, emoji, msg string, args ...any) {
	formatted := msg
	if len(args) > 0 {
		formatted = fmt.Sprintf(msg, args...)
	}

	if p.config.Mode == ModeJSON {
		p.printJSON(map[string]any{
			"level":   level,
			"message": formatted,
		})

		return
	}

	// Write to stdout; failures are non-recoverable
	var color string
	if p.config.NoColor || p.config.NoTTY {
		color = ""
	} else {
		color = getColor(level)
	}

	_, _ = fmt.Fprintf(p.config.Stdout, "%s %s%s%s\n", emoji, color, formatted, Reset)
}

// printJSON outputs data as JSON.
func (p *ConsolePrinter) printJSON(data any) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		// Fallback to stderr for marshalling errors
		_, _ = fmt.Fprintf(p.config.Stderr, "failed to marshal JSON: %v\n", err)
		return
	}

	_, _ = fmt.Fprintln(p.config.Stdout, string(jsonBytes))
}

// printTablePlain prints a simple ASCII table.
func (p *ConsolePrinter) printTablePlain(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	p.printTableRow(headers, widths)
	p.printTableSeparator(widths)

	// Print rows
	for _, row := range rows {
		p.printTableRow(row, widths)
	}
}

// printTableStyled prints a table with colors.
func (p *ConsolePrinter) printTableStyled(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header with styling
	_, _ = fmt.Fprint(p.config.Stdout, Bold+Cyan)

	for i, h := range headers {
		padding := widths[i] - len(h)
		_, _ = fmt.Fprintf(p.config.Stdout, "  %s%s%s", h, strings.Repeat(" ", padding), "  ")

		if i < len(headers)-1 {
			_, _ = fmt.Fprint(p.config.Stdout, "|")
		}
	}

	_, _ = fmt.Fprintln(p.config.Stdout, Reset)

	// Print separator
	p.printTableSeparator(widths)

	// Print rows
	for _, row := range rows {
		for i, v := range row {
			padding := widths[i] - len(v)
			_, _ = fmt.Fprintf(p.config.Stdout, "  %s%s%s", v, strings.Repeat(" ", padding), "  ")

			if i < len(row)-1 {
				_, _ = fmt.Fprint(p.config.Stdout, "|")
			}
		}

		_, _ = fmt.Fprintln(p.config.Stdout)
	}
}

// printTableRow prints a single table row.
func (p *ConsolePrinter) printTableRow(values []string, widths []int) {
	for i, v := range values {
		padded := v + strings.Repeat(" ", widths[i]-len(v))
		_, _ = fmt.Fprintf(p.config.Stdout, "  %s  ", padded)

		if i < len(values)-1 {
			_, _ = fmt.Fprint(p.config.Stdout, "|")
		}
	}

	_, _ = fmt.Fprintln(p.config.Stdout)
}

// printTableSeparator prints a line separating table sections.
func (p *ConsolePrinter) printTableSeparator(widths []int) {
	for _, w := range widths {
		_, _ = fmt.Fprint(p.config.Stdout, "+")
		_, _ = io.WriteString(p.config.Stdout, strings.Repeat("-", w+4))
	}

	_, _ = fmt.Fprintln(p.config.Stdout, "+")
}

// flattenRows converts 2D rows to 1D for JSON output.
func flattenRows(rows [][]string) []string {
	var flat []string
	for _, row := range rows {
		flat = append(flat, strings.Join(row, " | "))
	}

	return flat
}

// Color codes for terminal output.
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Dim        = "\033[2m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Magenta    = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	LightRed   = "\033[91m"
	LightGreen = "\033[92m"
)

// getColor returns the color code for a log level.
func getColor(level string) string {
	switch level {
	case "success":
		return LightGreen
	case "error":
		return LightRed
	case "warn":
		return Yellow
	case "info":
		return Cyan
	default:
		return ""
	}
}

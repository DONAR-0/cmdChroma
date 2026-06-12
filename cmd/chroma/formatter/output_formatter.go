package formatter

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// OutputFormat represents the supported output formats.
type OutputFormat string

const (
	FormatYAML OutputFormat = "yaml"
	FormatJSON OutputFormat = "json"
	FormatText OutputFormat = "text"
)

// OutputFormatter handles formatting data for display.
type OutputFormatter struct {
	format OutputFormat
}

// NewOutputFormatter creates a new OutputFormatter with the specified format.
// Defaults to YAML if an unrecognized format is provided.
func NewOutputFormatter(format string) *OutputFormatter {
	switch format {
	case "json":
		return &OutputFormatter{format: FormatJSON}
	case "text":
		return &OutputFormatter{format: FormatText}
	default:
		return &OutputFormatter{format: FormatYAML}
	}
}

// Format formats the data according to the configured output format.
func (f *OutputFormatter) Format(data any) (string, error) {
	switch f.format {
	case FormatJSON:
		return f.formatJSON(data)
	case FormatYAML:
		return f.formatYAML(data)
	default:
		return f.formatYAML(data)
	}
}

// formatJSON formats data as JSON.
func (f *OutputFormatter) formatJSON(data any) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return string(jsonData), nil
}

// formatYAML formats data as YAML.
func (f *OutputFormatter) formatYAML(data any) (string, error) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	return string(yamlData), nil
}

// FormatCollection formats a collection list for display.
func (f *OutputFormatter) FormatCollection(name, id string) string {
	switch f.format {
	case FormatJSON:
		return fmt.Sprintf(`{"name": %q, "id": %q}`, name, id)
	case FormatText:
		return fmt.Sprintf("• %s (ID: %s)", name, id)
	default:
		return fmt.Sprintf("%s (ID: %s)", name, id)
	}
}

// FormatDatabase formats a database entry for display.
func (f *OutputFormatter) FormatDatabase(name, id string) string {
	switch f.format {
	case FormatJSON:
		return fmt.Sprintf(`{"name": %q, "id": %q}`, name, id)
	case FormatText:
		return fmt.Sprintf("• %s (ID: %s)", name, id)
	default:
		return fmt.Sprintf("%s (ID: %s)", name, id)
	}
}

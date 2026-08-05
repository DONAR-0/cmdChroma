package main

import (
	"strings"
	"testing"
)

func TestValidateNIMModel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
		errMsg  string
	}{
		{"empty", "", "", true, "model is required"},
		{"nim_prefix", "nim://mistralai/mistral-7b", "mistralai/mistral-7b", false, ""},
		{"nim_empty_after_prefix", "nim://", "", true, "invalid NIM model format"},
		{"no_prefix", "llama2", "llama2", false, ""},
		{"nim_with_slash", "nim://org/model-name", "org/model-name", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := validateNIMModel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNIMModel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateNIMModel(%q) error = %v, want to contain %q", tt.input, err, tt.errMsg)
			}

			if !tt.wantErr && gotID != tt.wantID {
				t.Errorf("validateNIMModel(%q) = %q, want %q", tt.input, gotID, tt.wantID)
			}
		})
	}
}

func TestValidateCollectionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "my_collection", false},
		{"empty", "", true},
		{"whitespace", "   ", true},
		{"tabs", "\t\n", true},
		{"with_spaces", "my collection", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCollectionName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCollectionName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// Test handleTestConnection with mock
func TestHandleTestConnection(t *testing.T) {
	// This test is limited because handlers.go is in package main
	// and uses factory directly. We can only test validation functions
	// without major refactoring.
	t.Skip("Integration test - requires refactoring handlers to accept interfaces")
}

// Test handleCreateCollection with mock
func TestHandleCreateCollection(t *testing.T) {
	t.Skip("Integration test - requires refactoring handlers to accept interfaces")
}

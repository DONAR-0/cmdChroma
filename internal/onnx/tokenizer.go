package onnx

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Tokenizer struct {
	vocab     map[string]int
	maxLength int
}

type TokenizerConfig struct {
	Vocab map[string]int `json:"vocab"`
}

// NewTokenizer creates a new tokenizer from a JSON file
func NewTokenizer(path string) (*Tokenizer, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tokenizer file: %w", err)
	}

	var config TokenizerConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tokenizer config: %w", err)
	}

	return &Tokenizer{
		vocab:     config.Vocab,
		maxLength: 512,
	}, nil
}

func (t *Tokenizer) Encode(text string) ([]int, []int) {
	// Simple word-level tokenization (for demonstration)
	// In production, use a proper tokenizer like hunggingface tokenizer
	words := strings.Fields(strings.ToLower(text))

	//Add [CLS] token at start
	tokens := []int{101}
	attentionMask := []int{1}

	// Convert words to token IDs
	for _, word := range words {
		if tokenID, exists := t.vocab[word]; exists {
			tokens = append(tokens, tokenID)
			attentionMask = append(attentionMask, 1)
		} else {
			// Use [UNK] token for unknown words
			tokens = append(tokens, 100)
			attentionMask = append(attentionMask, 1)
		}
	}

	// Add [SEP] token at end
	tokens = append(tokens, 102)
	attentionMask = append(attentionMask, 1)

	//Pad to maxLength if needed
	for len(tokens) < t.maxLength {
		tokens = append(tokens, 0) //[PAD] token
		attentionMask = append(attentionMask, 0)
	}
	// Truncate if too long
	if len(tokens) > t.maxLength {
		tokens = tokens[:t.maxLength]
		attentionMask = attentionMask[:t.maxLength]
	}
	return tokens, attentionMask
}

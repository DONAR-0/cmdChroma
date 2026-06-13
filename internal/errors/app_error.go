package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents the category of an error.
type ErrorCode int

const (
	ErrUnknown ErrorCode = iota
	ErrConfig
	ErrConnection
	ErrCollection
	ErrDocument
	ErrIngestion
	ErrModel
	ErrValidation
)

// AppError represents a structured application error with context and hints.
type AppError struct {
	Err     error
	Code    ErrorCode
	Context string
	Hints   []string
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err == nil {
		return e.Context
	}

	return fmt.Sprintf("%s: %v", e.Context, e.Err)
}

// Unwrap allows errors.Is and errors.As to work with wrapped errors.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithHint appends a hint to the error.
func (e *AppError) WithHint(hint string) *AppError {
	e.Hints = append(e.Hints, hint)
	return e
}

// New creates a new AppError with the given code and context.
func New(code ErrorCode, context string) *AppError {
	return &AppError{Code: code, Context: context, Hints: []string{}}
}

// Wrap wraps an existing error with code and context.
func Wrap(err error, code ErrorCode, context string) *AppError {
	if err == nil {
		return nil
	}

	return &AppError{Err: err, Code: code, Context: context, Hints: []string{}}
}

// Is reports whether any error in err's chain matches target.
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
func (e *AppError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// ============ Helper Constructors ============

// CollectionNotFound creates an error for a missing collection.
func CollectionNotFound(name string) *AppError {
	return New(ErrCollection, fmt.Sprintf("collection '%s' not found", name)).
		WithHint("Use 'chroma collections' to list available collections").
		WithHint("Use 'chroma create <name>' to create a new collection")
}

// DatabaseNotFound creates an error for a missing database.
func DatabaseNotFound(name, tenant string) *AppError {
	return New(ErrCollection, fmt.Sprintf("database '%s' not found in tenant '%s'", name, tenant)).
		WithHint("Use 'chroma databases' to see available databases")
}

// InvalidFileFormat creates an error for unsupported file formats.
func InvalidFileFormat(path, ext string) *AppError {
	return New(ErrIngestion, fmt.Sprintf("unsupported file format: %s", ext)).
		WithHint("Supported formats: .jsonl, .parquet").
		WithHint("File: " + path)
}

// ConnectionFailed creates an error for connection failures.
func ConnectionFailed(host, port string) *AppError {
	return New(ErrConnection, fmt.Sprintf("connection to %s:%s failed", host, port)).
		WithHint("Verify ChromaDB is running: docker ps").
		WithHint("Check host/port settings with --host and --port flags")
}

// EmbedderInitFailed creates an error for embedder initialization failures.
func EmbedderInitFailed(modelPath string, err error) *AppError {
	return Wrap(err, ErrModel, fmt.Sprintf("failed to initialize embedder with model: %s", modelPath)).
		WithHint("Verify model files exist and paths are correct").
		WithHint("Run setup script to download models")
}

// InvalidInput creates an error for validation failures.
func InvalidInput(field, reason string) *AppError {
	return New(ErrValidation, fmt.Sprintf("invalid %s: %s", field, reason)).
		WithHint("Check the --help output for valid values")
}

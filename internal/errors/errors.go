package errors

import "errors"

var (
	// Embedder errors
	ErrEmbedderNotInitialized = errors.New("embedder not initialized")
	ErrEmbedderInitFailed     = errors.New("failed to initialize embedder")

	// Collection errors
	ErrCollectionNotFound = errors.New("collection not found")
	ErrCollectionExists   = errors.New("collection already exists")

	// Client/connection errors
	ErrConnectionFailed = errors.New("connection failed")
	ErrRequestFailed    = errors.New("request failed")
	ErrInvalidResponse  = errors.New("invalid response from server")

	// Validation errors
	ErrInvalidInput   = errors.New("invalid input")
	ErrFileNotFound   = errors.New("file not found")
	ErrPathTraversal  = errors.New("path traversal detected")
	ErrPathOutsideCWD = errors.New("path outside working directory")
	ErrAbsolutePath   = errors.New("absolute paths not allowed")
	ErrEmptyDocuments = errors.New("no documents provided")
	ErrEmptyQueries   = errors.New("no queries provided")

	// Configuration errors
	ErrMissingConfig = errors.New("missing configuration")
	ErrInvalidConfig = errors.New("invalid configuration")
)

// Is checks if an error matches any of the expected errors.
func Is(err error, target ...error) bool {
	for _, t := range target {
		if errors.Is(err, t) {
			return true
		}
	}
	return false
}

// As checks if an error can be cast to a specific type.
// Kept for convenience but standard errors.As can be used.
func As[T error](err error) (T, bool) {
	var t T
	if errors.As(err, &t) {
		return t, true
	}
	var zero T
	return zero, false
}

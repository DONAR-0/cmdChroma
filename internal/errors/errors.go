package errors

// Package errors defines canonical error variables for cmdChroma.
// Using named errors allows callers to check for specific failure modes
// via errors.Is. The package also provides convenience wrappers for errors.Is/As.
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
// Delegates to errors.AsType (Go 1.26+) for type-safe error unwrapping.
func As[T error](err error) (T, bool) {
	return errors.AsType[T](err)
}

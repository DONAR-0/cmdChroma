package internal

import (
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
)

// CheckDefer logs errors from deferred Close calls.
func CheckDefer(closeFunc func() error) {
	if closeFunc == nil {
		return
	}

	if err := closeFunc(); err != nil {
		slog.Warn("deferred close failed", "error", err)
	}
}

// SafeJoin joins root with relative path and ensures result stays within root.
// It rejects absolute paths and path traversal attempts.
func SafeJoin(root, relPath string) (string, error) {
	// Reject absolute paths
	if filepath.IsAbs(relPath) {
		return "", errors.New("absolute paths not allowed")
	}

	// Clean to resolve any internal .. components
	clean := filepath.Clean(relPath)

	// Join and verify the result is within root
	joined := filepath.Join(root, clean)
	if !strings.HasPrefix(joined, root) {
		return "", errors.New("path outside working directory")
	}

	return joined, nil
}

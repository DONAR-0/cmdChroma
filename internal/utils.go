package internal

import (
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
)

// CheckDefer logs errors from deferred Close calls.
func CheckDefer(closeFunc func() error) {
	if err := closeFunc(); err != nil {
		slog.Debug("error received", "err", err)
	}
}

// SafeJoin joins root with relative path and ensures result stays within root.
// It rejects paths containing ".." and absolute paths.
func SafeJoin(root, relPath string) (string, error) {
	// Reject absolute paths
	if filepath.IsAbs(relPath) {
		return "", errors.New("absolute paths not allowed")
	}

	// Reject paths containing .. after cleaning
	clean := filepath.Clean(relPath)
	if strings.Contains(clean, "..") {
		return "", errors.New("path traversal not allowed")
	}

	// Join and verify the result is within root
	joined := filepath.Join(root, clean)
	if !strings.HasPrefix(joined, root) {
		return "", errors.New("path outside working directory")
	}
	return joined, nil
}

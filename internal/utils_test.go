package internal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckDefer(t *testing.T) {
	// Test with nil function - should not panic
	CheckDefer(nil)

	// Test with function that returns no error
	var called bool

	fn := func() error {
		called = true
		return nil
	}
	CheckDefer(fn)
	require.True(t, called, "function should be called")

	// Test with function that returns an error - should log but not panic
	fnErr := func() error {
		return errors.New("test error")
	}
	CheckDefer(fnErr) // Should not panic
}

func TestSafeJoin(t *testing.T) {
	tests := []struct {
		root     string
		relPath  string
		wantPath string
		wantErr  bool
	}{
		{
			root:     "/home/user",
			relPath:  "documents/file.txt",
			wantPath: "/home/user/documents/file.txt",
			wantErr:  false,
		},
		{
			root:     "/home/user",
			relPath:  "../secret.txt",
			wantPath: "",
			wantErr:  true, // path traversal not allowed
		},
		{
			root:     "/home/user",
			relPath:  "/etc/passwd",
			wantPath: "",
			wantErr:  true, // absolute paths not allowed
		},
		{
			root:     "/home/user",
			relPath:  "subdir/../../etc/passwd",
			wantPath: "",
			wantErr:  true, // path traversal after cleaning
		},
		{
			root:     "/home/user",
			relPath:  "subdir/../documents/./file.txt",
			wantPath: "/home/user/documents/file.txt",
			wantErr:  false, // normal path with . and ..
		},
		{
			root:     "/home/user",
			relPath:  "",
			wantPath: "/home/user",
			wantErr:  false, // empty relative path
		},
		{
			root:     "/home/user",
			relPath:  ".",
			wantPath: "/home/user",
			wantErr:  false, // current directory
		},
		{
			root:     "/home/user",
			relPath:  "..",
			wantPath: "",
			wantErr:  true, // parent directory outside root
		},
	}

	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			gotPath, err := SafeJoin(tt.root, tt.relPath)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, gotPath)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantPath, gotPath)
			}
		})
	}
}

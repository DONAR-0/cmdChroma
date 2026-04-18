package internal

import (
	"testing"
)

func TestCheckDefer(t *testing.T) {
	// Test that CheckDefer doesn't panic when func returns nil
	CheckDefer(func() error {
		return nil
	})
	// Test that CheckDefer logs when func returns error
	CheckDefer(func() error {
		return &testError{"test error"}
	})
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

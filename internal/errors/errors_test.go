package errors

import (
	"errors"
	"testing"
)

func TestIs(t *testing.T) {
	if !Is(ErrEmbedderNotInitialized, ErrEmbedderNotInitialized) {
		t.Error("Is should match same error")
	}

	if Is(ErrEmbedderNotInitialized, ErrCollectionNotFound) {
		t.Error("Is should not match different errors")
	}

	if Is(nil, ErrEmbedderNotInitialized) {
		t.Error("Is(nil, err) should be false")
	}

	wrapped := errors.New("wrapped: embedder not initialized")
	if Is(wrapped, ErrEmbedderNotInitialized) {
		t.Error("Is should not match wrapped sentinel without errors.Is")
	}
}

func TestAs(t *testing.T) {
	wrapped := Wrap(errors.New("root"), ErrConfig, "config error")

	got, ok := As[*AppError](wrapped)
	if !ok {
		t.Error("As should find *AppError type")
	}

	if got == nil {
		t.Error("got should not be nil")
	}

	_, ok = As[error](wrapped)
	if !ok {
		t.Error("As[error] should match any error")
	}
}

func TestAppErrorConstructors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"CollectionNotFound", CollectionNotFound("test")},
		{"DatabaseNotFound", DatabaseNotFound("test", "tenant")},
		{"InvalidFileFormat", InvalidFileFormat("file.txt", ".txt")},
		{"ConnectionFailed", ConnectionFailed("localhost", "8000")},
		{"EmbedderInitFailed", EmbedderInitFailed("/path/model.onnx", errors.New("load failed"))},
		{"InvalidInput", InvalidInput("field", "too long")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s returned nil", tt.name)
			}

			if tt.err.Error() == "" {
				t.Errorf("%s returned empty error string", tt.name)
			}
		})
	}
}

func TestAppError_WithHint(t *testing.T) {
	err := New(ErrConfig, "config error").WithHint("hint1").WithHint("hint2")
	if len(err.Hints) != 2 {
		t.Errorf("expected 2 hints, got %d", len(err.Hints))
	}

	if err.Hints[0] != "hint1" || err.Hints[1] != "hint2" {
		t.Errorf("unexpected hints: %v", err.Hints)
	}
}

func TestAppError_Wrap(t *testing.T) {
	t.Run("wraps non-nil error", func(t *testing.T) {
		cause := errors.New("root cause")

		wrapped := Wrap(cause, ErrConnection, "connection failed")
		if wrapped == nil {
			t.Fatal("Wrap returned nil")
		}

		if !errors.Is(wrapped, cause) {
			t.Error("wrapped error should unwrap to cause")
		}
	})

	t.Run("nil error returns nil", func(t *testing.T) {
		wrapped := Wrap(nil, ErrConnection, "connection failed")
		if wrapped != nil {
			t.Error("Wrap(nil, ...) should return nil")
		}
	})
}

func TestAppError_Error(t *testing.T) {
	err := New(ErrConfig, "config error")
	if err.Error() != "config error" {
		t.Errorf("unexpected error string: %s", err.Error())
	}

	wrapped := Wrap(errors.New("cause"), ErrConfig, "config error")
	if wrapped.Error() != "config error: cause" {
		t.Errorf("unexpected error string: %s", wrapped.Error())
	}
}

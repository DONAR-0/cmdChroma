package output

import "testing"

func TestNewUI(t *testing.T) {
	u := NewUI()
	if u == nil {
		t.Fatal("NewUI() = nil")
	}
}

func TestUI_IsInteractive(t *testing.T) {
	u := NewUI()
	_ = u.IsInteractive()
}

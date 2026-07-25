package output

import (
	"strings"
	"testing"
	"time"
)

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

func TestUI_NewSpinner(t *testing.T) {
	u := &UI{interactive: true}

	s := u.NewSpinner("test title")
	if s == nil {
		t.Fatal("NewSpinner() = nil")
	}

	if s.Title() != "test title" {
		t.Errorf("Spinner.Title() = %q, want %q", s.Title(), "test title")
	}

	if s.IsDone() {
		t.Error("NewSpinner().IsDone() = true, want false")
	}

	if s.Output() != "" {
		t.Errorf("NewSpinner().Output() = %q, want empty", s.Output())
	}

	view := s.View()
	if !strings.Contains(view, "test title") {
		t.Errorf("Spinner.View() = %q, missing title", view)
	}
}

func TestUI_NewSpinner_NonInteractive(t *testing.T) {
	u := &UI{interactive: false}

	s := u.NewSpinner("test title")
	if s == nil {
		t.Fatal("NewSpinner() = nil for non-interactive")
	}
}

func TestSpinner_SetDone(t *testing.T) {
	u := &UI{interactive: true}
	s := u.NewSpinner("test")
	s.SetDone("completed")

	if !s.IsDone() {
		t.Error("SetDone() did not mark as done")
	}

	if s.Output() != "completed" {
		t.Errorf("Spinner.Output() = %q, want %q", s.Output(), "completed")
	}

	view := s.View()
	if view != "completed" {
		t.Errorf("Spinner.View() after done = %q, want %q", view, "completed")
	}
}

func TestUI_NewProgress(t *testing.T) {
	u := &UI{interactive: true}

	p := u.NewProgress("processing", 10)
	if p == nil {
		t.Fatal("NewProgress() = nil")
	}

	if p.Total() != 10 {
		t.Errorf("Progress.Total() = %d, want 10", p.Total())
	}

	if p.Current() != 0 {
		t.Errorf("Progress.Current() = %d, want 0", p.Current())
	}

	if p.IsComplete() {
		t.Error("Progress.IsComplete() = true initially, want false")
	}

	if p.Fraction() != 0 {
		t.Errorf("Progress.Fraction() = %f, want 0", p.Fraction())
	}
}

func TestProgress_Increment(t *testing.T) {
	u := &UI{interactive: true}
	p := u.NewProgress("test", 5)
	p.Increment()

	if p.Current() != 1 {
		t.Errorf("After 1 Increment(), Current() = %d, want 1", p.Current())
	}

	p.Increment()
	p.Increment()

	if p.Current() != 3 {
		t.Errorf("After 3 Increment(), Current() = %d, want 3", p.Current())
	}
}

func TestProgress_SetProgress(t *testing.T) {
	u := &UI{interactive: true}
	p := u.NewProgress("test", 10)
	p.SetProgress(7)

	if p.Current() != 7 {
		t.Errorf("SetProgress(7) -> Current() = %d, want 7", p.Current())
	}

	p.SetProgress(15)

	if p.Current() != 10 {
		t.Errorf("SetProgress(15) capped at total -> Current() = %d, want 10", p.Current())
	}
}

func TestProgress_View(t *testing.T) {
	u := &UI{interactive: true}
	p := u.NewProgress("test", 10)

	view := p.View()
	if !strings.Contains(view, "test") {
		t.Errorf("Progress.View() = %q, missing title", view)
	}

	if !strings.Contains(view, "0/10") {
		t.Errorf("Progress.View() = %q, missing 0/10", view)
	}

	p.SetProgress(5)

	view = p.View()
	if !strings.Contains(view, "5/10") {
		t.Errorf("Progress.View() at 5 = %q, missing 5/10", view)
	}
}

func TestProgress_ZeroTotal(t *testing.T) {
	u := &UI{interactive: true}

	p := u.NewProgress("zero", 0)
	if p.View() != "" {
		t.Errorf("Progress.View() with total=0 = %q, want empty", p.View())
	}

	if p.Fraction() != 0 {
		t.Errorf("Progress.Fraction() with total=0 = %f, want 0", p.Fraction())
	}
}

func TestUI_NewSimpleSpinner(t *testing.T) {
	u := &UI{interactive: true}

	s := u.NewSimpleSpinner("simple test")
	if s == nil {
		t.Fatal("NewSimpleSpinner() = nil")
	}

	if s.IsDone() {
		t.Error("NewSimpleSpinner().IsDone() = true, want false")
	}

	start := s.Start()
	if !strings.Contains(start, "⟳") || !strings.Contains(start, "simple test") {
		t.Errorf("SimpleSpinner.Start() = %q, unexpected", start)
	}

	s.Stop("finished")

	if !s.IsDone() {
		t.Error("SimpleSpinner.Stop() did not mark done")
	}

	if s.Message() != "finished" {
		t.Errorf("SimpleSpinner.Message() = %q, want %q", s.Message(), "finished")
	}
}

func TestUI_NewProgressWriter(t *testing.T) {
	u := &UI{interactive: true}

	w := u.NewProgressWriter("writer test", 100)
	if w == nil {
		t.Fatal("NewProgressWriter() = nil")
	}
	// Just verify it can be created and View() works
	view := w.View()
	if !strings.Contains(view, "writer test") {
		t.Errorf("ProgressWriter.View() = %q, missing title", view)
	}
}

func TestProgressWriter_Increment(t *testing.T) {
	u := &UI{interactive: true}
	w := u.NewProgressWriter("test", 10)
	updated := w.Increment()
	// First call returns false because lastUpdate is initialized to now
	// (time.Since < updateDelay)
	if updated {
		t.Error("First Increment() should return false (throttled)")
	}

	view := w.View()
	if !strings.Contains(view, "test") {
		t.Errorf("ProgressWriter.View() = %q, missing title", view)
	}
}

func TestProgressWriter_ShouldUpdate(t *testing.T) {
	u := &UI{interactive: true}

	w := u.NewProgressWriter("test", 10)
	if w.ShouldUpdate() {
		t.Error("ShouldUpdate() immediately after creation = true, want false")
	}

	time.Sleep(150 * time.Millisecond)

	if !w.ShouldUpdate() {
		t.Error("ShouldUpdate() after delay = false, want true")
	}
}

func TestProgressWriter_View(t *testing.T) {
	u := &UI{interactive: true}
	w := u.NewProgressWriter("writer", 5)

	view := w.View()
	if !strings.Contains(view, "writer") {
		t.Errorf("ProgressWriter.View() = %q, missing title", view)
	}
	// Increment a few times
	w.Increment()
	w.Increment()
	w.Increment()

	view = w.View()
	if !strings.Contains(view, "3/5") {
		t.Errorf("ProgressWriter.View() at 3 = %q, missing 3/5", view)
	}
}

func TestRepeat(t *testing.T) {
	result := repeat('x', 5)
	if result != "xxxxx" {
		t.Errorf("repeat('x', 5) = %q, want 'xxxxx'", result)
	}

	result = repeat('y', 0)
	if result != "" {
		t.Errorf("repeat('y', 0) = %q, want empty", result)
	}

	result = repeat('z', -1)
	if result != "" {
		t.Errorf("repeat('z', -1) = %q, want empty", result)
	}
}

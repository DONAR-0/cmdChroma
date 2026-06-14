package output

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// UI provides terminal UI components like spinners and progress bars.
// It automatically disables TUI elements when not in an interactive terminal.
type UI struct {
	interactive bool
}

// NewUI creates a new UI instance.
// TUI elements are disabled when output is not a terminal.
func NewUI() *UI {
	return &UI{
		interactive: IsInteractive(),
	}
}

// IsInteractive returns true if the terminal supports TUI elements.
func (u *UI) IsInteractive() bool {
	return u.interactive
}

// SpinnerModel wraps the bubbles spinner with title support.
type SpinnerModel struct {
	spinner spinner.Model
	title   string
	done    bool
	output  string
}

// NewSpinner creates a new spinner with the given title.
func (u *UI) NewSpinner(title string) *SpinnerModel {
	s := spinner.New(
		spinner.WithStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))),
	)

	return &SpinnerModel{
		spinner: s,
		title:   title,
	}
}

// Title returns the spinner's title.
func (s *SpinnerModel) Title() string {
	return s.title
}

// SetDone marks the spinner as done with a message.
func (s *SpinnerModel) SetDone(message string) {
	s.done = true
	s.output = message
}

// IsDone returns whether the spinner has finished.
func (s *SpinnerModel) IsDone() bool {
	return s.done
}

// Output returns the final output message.
func (s *SpinnerModel) Output() string {
	return s.output
}

// View returns the rendered spinner view.
func (s *SpinnerModel) View() string {
	if s.done {
		return s.output
	}

	return fmt.Sprintf("%s %s", s.spinner.View(), s.title)
}

// ProgressModel provides a simple progress display.
type ProgressModel struct {
	title   string
	total   int
	current int
}

// NewProgress creates a new progress tracker.
func (u *UI) NewProgress(title string, total int) *ProgressModel {
	return &ProgressModel{
		title: title,
		total: total,
	}
}

// Increment increases the progress by one.
func (p *ProgressModel) Increment() {
	if p.current < p.total {
		p.current++
	}
}

// SetProgress sets the current progress to a specific value.
func (p *ProgressModel) SetProgress(current int) {
	p.current = current
	if p.current > p.total {
		p.current = p.total
	}
}

// View returns the current progress as a string.
func (p *ProgressModel) View() string {
	if p.total <= 0 {
		return ""
	}

	filled := 20
	percentage := float64(p.current) / float64(p.total)

	barFilled := int(float64(filled) * percentage)
	barEmpty := filled - barFilled

	return fmt.Sprintf("%s [%s%s] %d/%d (%.0f%%)",
		p.title,
		repeat(rune('█'), barFilled),
		repeat(rune('░'), barEmpty),
		p.current, p.total,
		percentage*100)
}

// Fraction returns the progress as a float between 0 and 1.
func (p *ProgressModel) Fraction() float64 {
	if p.total <= 0 {
		return 0
	}

	return float64(p.current) / float64(p.total)
}

// Total returns the total items.
func (p *ProgressModel) Total() int {
	return p.total
}

// Current returns the current progress count.
func (p *ProgressModel) Current() int {
	return p.current
}

// IsComplete returns true if progress has reached 100%.
func (p *ProgressModel) IsComplete() bool {
	return p.current >= p.total
}

// SimpleSpinner provides a simple text-based spinner for non-interactive mode.
type SimpleSpinner struct {
	title   string
	done    bool
	message string
}

// NewSimpleSpinner creates a simple text-based spinner.
func (u *UI) NewSimpleSpinner(title string) *SimpleSpinner {
	return &SimpleSpinner{
		title: title,
	}
}

// Start returns a non-blocking start indicator.
func (s *SimpleSpinner) Start() string {
	return fmt.Sprintf("⟳ %s...", s.title)
}

// Stop sets the final message.
func (s *SimpleSpinner) Stop(message string) {
	s.done = true
	s.message = message
}

// IsDone returns whether the spinner has stopped.
func (s *SimpleSpinner) IsDone() bool {
	return s.done
}

// Message returns the final message.
func (s *SimpleSpinner) Message() string {
	return s.message
}

// ProgressWriter provides a simple progress display with throttled updates.
type ProgressWriter struct {
	title       string
	total       int
	current     int
	width       int
	updateDelay time.Duration
	lastUpdate  time.Time
}

// NewProgressWriter creates a new progress writer.
func (u *UI) NewProgressWriter(title string, total int) *ProgressWriter {
	return &ProgressWriter{
		title:       title,
		total:       total,
		current:     0,
		width:       40,
		updateDelay: 100 * time.Millisecond,
		lastUpdate:  time.Now(),
	}
}

// Increment increases progress and returns true if enough time has passed for display.
func (w *ProgressWriter) Increment() bool {
	w.current++

	if time.Since(w.lastUpdate) >= w.updateDelay {
		w.lastUpdate = time.Now()
		return true
	}

	return false
}

// ShouldUpdate returns true if enough time has passed to update display.
func (w *ProgressWriter) ShouldUpdate() bool {
	return time.Since(w.lastUpdate) >= w.updateDelay
}

// View returns the current progress as a string.
func (w *ProgressWriter) View() string {
	if w.total <= 0 {
		return ""
	}

	filled := int(float64(w.width) * float64(w.current) / float64(w.total))
	empty := w.width - filled

	bar := fmt.Sprintf("[%s%s]",
		repeat(rune('█'), filled),
		repeat(rune('░'), empty))

	return fmt.Sprintf("%s %s %d/%d (%.0f%%)",
		w.title, bar, w.current, w.total,
		float64(w.current)/float64(w.total)*100)
}

// repeat creates a string with n repetitions of the given rune.
func repeat(r rune, n int) string {
	if n <= 0 {
		return ""
	}

	result := make([]rune, n)
	for i := range result {
		result[i] = r
	}

	return string(result)
}

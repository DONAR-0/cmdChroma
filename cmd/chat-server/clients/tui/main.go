package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// styles

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")). // blue-ish
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("84")). // green-ish
			Bold(false)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	separator = dimStyle.Render(strings.Repeat("─", 50))
)

// Message is a single turn in the TUI message list.
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// State holds the TUI's application state.
type State struct {
	collection  string
	model       string
	serverURL   string
	apiKey      string
	sessionID   string
	messages    []Message
	input       string
	streaming   bool
	waiting     bool
	err         error
	width       int
	height      int
	showHelp    bool
	collections []string
	statusMsg   string
}

func NewState(collection, model, serverURL, apiKey string) *State {
	return &State{
		collection:  collection,
		model:       model,
		serverURL:   serverURL,
		apiKey:      apiKey,
		collections: []string{},
		statusMsg:   "connected",
	}
}

// Init implements tea.Model.
func (s *State) Init() tea.Cmd {
	return func() tea.Msg {
		if s.sessionID == "" {
			s.sessionID = newSessionID()
			s.statusMsg = "new session: " + s.sessionID[:8]
		}

		return nil
	}
}

// Update implements tea.Model.
func (s *State) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "ctrl+c", "q":
			return s, tea.Quit
		case "ctrl+l":
			s.messages = nil
			s.input = ""

			return s, nil
		case "ctrl+s":
			s.showHelp = !s.showHelp
			return s, nil
		case "enter":
			if s.waiting || s.streaming || strings.TrimSpace(s.input) == "" {
				return s, nil
			}

			userMsg := s.input
			s.input = ""
			s.waiting = true
			s.err = nil

			return s, func() tea.Msg {
				return s.sendChat(userMsg)
			}
		case "backspace":
			if len(s.input) > 0 {
				s.input = s.input[:len(s.input)-1]
			}

			return s, nil
		default:
			if !s.waiting && !s.streaming {
				s.input += m.String()
			}

			return s, nil
		}

	case Message:
		s.messages = append(s.messages, m)
		s.waiting = false
		s.streaming = false

		return s, nil

	case streamingMessage:
		s.streaming = true

		lastIdx := len(s.messages) - 1
		if lastIdx >= 0 && s.messages[lastIdx].Role == "assistant" {
			s.messages[lastIdx].Content += string(m)
		} else {
			s.messages = append(s.messages, Message{Role: "assistant", Content: string(m)})
		}

		return s, nil

	case error:
		s.waiting = false
		s.streaming = false
		s.err = m

		return s, nil

	case tea.WindowSizeMsg:
		s.width = m.Width
		s.height = m.Height

		return s, nil
	}

	return s, nil
}

// View implements tea.Model — renders the full TUI.
func (s *State) View() string {
	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render("┌ chatting_server "))
	b.WriteString(dimStyle.Render("│ session: " + s.sessionID[:8] + ".. │ col: " + s.collection + " │ model: " + s.model))
	b.WriteString(headerStyle.Render(" ┐\n"))
	b.WriteString(dimStyle.Render("├" + strings.Repeat("─", s.width-2) + "┤\n"))

	// Message history
	if len(s.messages) == 0 {
		b.WriteString(dimStyle.Render("  (start typing to chat — ctrl+s for help)\n"))
	} else {
		for _, msg := range s.messages {
			if msg.Role == "user" {
				b.WriteString(userStyle.Render("  [You] ") + msg.Content + "\n")
			} else {
				b.WriteString(assistantStyle.Render("  [RAG]  ") + msg.Content + "\n")
			}
		}
	}

	// Status bar
	b.WriteString(separator + "\n")

	if s.waiting {
		b.WriteString(dimStyle.Render("  💭 Retrieving documents & generating response...\n"))
	}

	if s.streaming {
		b.WriteString(dimStyle.Render("  📡 streaming...\n"))
	}

	if s.err != nil {
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("204")). // red-ish
			Render(fmt.Sprintf("  ✗ error: %s\n", s.err.Error())))
	}

	b.WriteString(dimStyle.Render("  " + s.statusMsg + "\n"))

	// Input
	b.WriteString(separator + "\n")

	if !s.waiting {
		b.WriteString(fmt.Sprintf("  > %s", s.input))
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  > %s", s.input)))
	}

	// Help overlay
	if s.showHelp {
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("  ── Help ──────────────────────────────────\n"))
		b.WriteString(dimStyle.Render("  Enter        send message\n"))
		b.WriteString(dimStyle.Render("  Ctrl+L       clear messages\n"))
		b.WriteString(dimStyle.Render("  Ctrl+S       toggle this help\n"))
		b.WriteString(dimStyle.Render("  q / Ctrl+C   quit\n"))
		b.WriteString(dimStyle.Render("  ──────────────────────────────────────────\n"))
	}

	return b.String()
}

// sendChat sends a message to the chat server and streams back tokens.
func (s *State) sendChat(message string) tea.Msg {
	// Add user message immediately
	return Message{Role: "user", Content: message}
}

type streamingMessage string

// newSessionID creates a unique-ish session ID.
// Uses a simple approach; for production this could be persisted to disk.
func newSessionID() string {
	return fmt.Sprintf("sess-%d", os.Getpid())
}

func main() {
	serverURL := getEnv("CHAT_SERVER_URL", "http://localhost:8080")
	apiKey := getEnv("CHAT_API_KEY", "changeme")
	collection := getEnv("CHAT_COLLECTION", "tech_faq")
	model := getEnv("CHAT_MODEL", "google/gemma-2-2b-it") // uses google/ prefix → routes to NIM

	initialState := NewState(collection, model, serverURL, apiKey)
	p := tea.NewProgram(initialState,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

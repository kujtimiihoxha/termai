package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/session"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

// SessionSelectedMsg is sent when a session is selected
type SessionSelectedMsg struct {
	Session session.Session
}

// CloseSessionDialogMsg is sent when the session dialog is closed
type CloseSessionDialogMsg struct{}

// SessionDialog interface for the session switching dialog
type SessionDialog interface {
	tea.Model
	layout.Bindings
	SetSessions(sessions []session.Session)
	SetSelectedSession(sessionID string)
}

type sessionDialogCmp struct {
	sessions     []session.Session
	selectedIdx  int
	width        int
	height       int
	selectedSessionID string
}

type sessionKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Escape     key.Binding
	J          key.Binding
	K          key.Binding
}

var sessionKeys = sessionKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "previous session"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next session"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select session"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
	J: key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "next session"),
	),
	K: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "previous session"),
	),
}

func (s *sessionDialogCmp) Init() tea.Cmd {
	return nil
}

func (s *sessionDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, sessionKeys.Up) || key.Matches(msg, sessionKeys.K):
			if s.selectedIdx > 0 {
				s.selectedIdx--
			}
			return s, nil
		case key.Matches(msg, sessionKeys.Down) || key.Matches(msg, sessionKeys.J):
			if s.selectedIdx < len(s.sessions)-1 {
				s.selectedIdx++
			}
			return s, nil
		case key.Matches(msg, sessionKeys.Enter):
			if len(s.sessions) > 0 {
				return s, util.CmdHandler(SessionSelectedMsg{
					Session: s.sessions[s.selectedIdx],
				})
			}
		case key.Matches(msg, sessionKeys.Escape):
			return s, util.CmdHandler(CloseSessionDialogMsg{})
		}
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
	}
	return s, nil
}

func (s *sessionDialogCmp) View() string {
	if len(s.sessions) == 0 {
		return styles.BaseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(styles.Background).
			BorderForeground(styles.ForgroundDim).
			Width(40).
			Render("No sessions available")
	}

	// Calculate max width needed for session titles
	maxWidth := 40 // Minimum width
	for _, sess := range s.sessions {
		if len(sess.Title) > maxWidth-4 { // Account for padding
			maxWidth = len(sess.Title) + 4
		}
	}

	// Limit height to avoid taking up too much screen space
	maxVisibleSessions := min(10, len(s.sessions))

	// Build the session list
	sessionItems := make([]string, 0, maxVisibleSessions)
	startIdx := 0
	
	// If we have more sessions than can be displayed, adjust the start index
	if len(s.sessions) > maxVisibleSessions {
		// Center the selected item when possible
		halfVisible := maxVisibleSessions / 2
		if s.selectedIdx >= halfVisible && s.selectedIdx < len(s.sessions)-halfVisible {
			startIdx = s.selectedIdx - halfVisible
		} else if s.selectedIdx >= len(s.sessions)-halfVisible {
			startIdx = len(s.sessions) - maxVisibleSessions
		}
	}

	endIdx := min(startIdx+maxVisibleSessions, len(s.sessions))

	for i := startIdx; i < endIdx; i++ {
		sess := s.sessions[i]
		itemStyle := styles.BaseStyle.Width(maxWidth)
		
		if i == s.selectedIdx {
			itemStyle = itemStyle.
				Background(styles.PrimaryColor).
				Foreground(styles.Background).
				Bold(true)
		}
		
		sessionItems = append(sessionItems, itemStyle.Padding(0, 1).Render(sess.Title))
	}

	title := styles.BaseStyle.
		Foreground(styles.PrimaryColor).
		Bold(true).
		Padding(0, 1).
		Render("Switch Session")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		styles.BaseStyle.Render(""),
		lipgloss.JoinVertical(lipgloss.Left, sessionItems...),
		styles.BaseStyle.Render(""),
		styles.BaseStyle.Foreground(styles.ForgroundDim).Render("↑/k: up  ↓/j: down  enter: select  esc: cancel"),
	)

	return styles.BaseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(styles.Background).
		BorderForeground(styles.ForgroundDim).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (s *sessionDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(sessionKeys)
}

func (s *sessionDialogCmp) SetSessions(sessions []session.Session) {
	s.sessions = sessions
	
	// If we have a selected session ID, find its index
	if s.selectedSessionID != "" {
		for i, sess := range sessions {
			if sess.ID == s.selectedSessionID {
				s.selectedIdx = i
				return
			}
		}
	}
	
	// Default to first session if selected not found
	s.selectedIdx = 0
}

func (s *sessionDialogCmp) SetSelectedSession(sessionID string) {
	s.selectedSessionID = sessionID
	
	// Update the selected index if sessions are already loaded
	if len(s.sessions) > 0 {
		for i, sess := range s.sessions {
			if sess.ID == sessionID {
				s.selectedIdx = i
				return
			}
		}
	}
}

// NewSessionDialogCmp creates a new session switching dialog
func NewSessionDialogCmp() SessionDialog {
	return &sessionDialogCmp{
		sessions:         []session.Session{},
		selectedIdx:      0,
		selectedSessionID: "",
	}
}
package chat

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/session"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
)

type sidebarCmp struct {
	width, height int
	session       session.Session
}

func (m *sidebarCmp) Init() tea.Cmd {
	return nil
}

func (m *sidebarCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *sidebarCmp) View() string {
	return styles.BaseStyle.
		Width(m.width).
		Height(m.height - 1).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				header(m.width),
				" ",
				m.sessionSection(),
				" ",
				m.modifiedFiles(),
				" ",
				lspsConfigured(m.width),
			),
		)
}

func (m *sidebarCmp) sessionSection() string {
	sessionKey := styles.BaseStyle.Foreground(styles.PrimaryColor).Bold(true).Render("Session")
	sessionValue := styles.BaseStyle.
		Foreground(styles.Forground).
		Width(m.width - lipgloss.Width(sessionKey)).
		Render(": New Session")
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		sessionKey,
		sessionValue,
	)
}

func (m *sidebarCmp) modifiedFile(filePath string, additions, removals int) string {
	stats := ""
	if additions > 0 && removals > 0 {
		stats = styles.BaseStyle.Foreground(styles.ForgroundDim).Render(fmt.Sprintf(" %d additions and  %d removals", additions, removals))
	} else if additions > 0 {
		stats = styles.BaseStyle.Foreground(styles.ForgroundDim).Render(fmt.Sprintf(" %d additions", additions))
	} else if removals > 0 {
		stats = styles.BaseStyle.Foreground(styles.ForgroundDim).Render(fmt.Sprintf(" %d removals", removals))
	}
	filePathStr := styles.BaseStyle.Foreground(styles.Forground).Render(filePath)

	return styles.BaseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				filePathStr,
				stats,
			),
		)
}

func (m *sidebarCmp) modifiedFiles() string {
	modifiedFiles := styles.BaseStyle.Width(m.width).Foreground(styles.PrimaryColor).Bold(true).Render("Modified Files:")
	files := []struct {
		path      string
		additions int
		removals  int
	}{
		{"file1.txt", 10, 5},
		{"file2.txt", 20, 0},
		{"file3.txt", 0, 15},
	}
	var fileViews []string
	for _, file := range files {
		fileViews = append(fileViews, m.modifiedFile(file.path, file.additions, file.removals))
	}

	return styles.BaseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				modifiedFiles,
				lipgloss.JoinVertical(
					lipgloss.Left,
					fileViews...,
				),
			),
		)
}

func (m *sidebarCmp) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *sidebarCmp) GetSize() (int, int) {
	return m.width, m.height
}

func NewSidebarCmp(session session.Session) tea.Model {
	return &sidebarCmp{
		session: session,
	}
}

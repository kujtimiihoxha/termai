package chat

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/diff"
	"github.com/kujtimiihoxha/termai/internal/history"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
	"github.com/kujtimiihoxha/termai/internal/session"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
)

type sidebarCmp struct {
	width, height int
	session       session.Session
	history       history.Service
	modFiles      map[string]struct {
		additions int
		removals  int
	}
}

func (m *sidebarCmp) Init() tea.Cmd {
	if m.history != nil {
		ctx := context.Background()
		// Subscribe to file events
		filesCh := m.history.Subscribe(ctx)

		// Initialize the modified files map
		m.modFiles = make(map[string]struct {
			additions int
			removals  int
		})

		// Load initial files and calculate diffs
		m.loadModifiedFiles(ctx)

		// Return a command that will send file events to the Update method
		return func() tea.Msg {
			return <-filesCh
		}
	}
	return nil
}

func (m *sidebarCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent {
			if m.session.ID == msg.Payload.ID {
				m.session = msg.Payload
			}
		}
	case pubsub.Event[history.File]:
		if msg.Payload.SessionID == m.session.ID {
			// When a file changes, reload all modified files
			// This ensures we have the complete and accurate list
			ctx := context.Background()
			m.loadModifiedFiles(ctx)
		}
	}
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
		Render(fmt.Sprintf(": %s", m.session.Title))
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

	// If no modified files, show a placeholder message
	if m.modFiles == nil || len(m.modFiles) == 0 {
		message := "No modified files"
		remainingWidth := m.width - lipgloss.Width(modifiedFiles)
		if remainingWidth > 0 {
			message += strings.Repeat(" ", remainingWidth)
		}
		return styles.BaseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					modifiedFiles,
					styles.BaseStyle.Foreground(styles.ForgroundDim).Render(message),
				),
			)
	}

	var fileViews []string
	for path, stats := range m.modFiles {
		fileViews = append(fileViews, m.modifiedFile(path, stats.additions, stats.removals))
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

func NewSidebarCmp(session session.Session, history history.Service) tea.Model {
	return &sidebarCmp{
		session: session,
		history: history,
	}
}

func (m *sidebarCmp) loadModifiedFiles(ctx context.Context) {
	if m.history == nil || m.session.ID == "" {
		return
	}

	// Get all latest files for this session
	latestFiles, err := m.history.ListLatestSessionFiles(ctx, m.session.ID)
	if err != nil {
		return
	}

	// Get all files for this session (to find initial versions)
	allFiles, err := m.history.ListBySession(ctx, m.session.ID)
	if err != nil {
		return
	}

	// Process each latest file
	for _, file := range latestFiles {
		// Skip if this is the initial version (no changes to show)
		if file.Version == history.InitialVersion {
			continue
		}

		// Find the initial version for this specific file
		var initialVersion history.File
		for _, v := range allFiles {
			if v.Path == file.Path && v.Version == history.InitialVersion {
				initialVersion = v
				break
			}
		}

		// Skip if we can't find the initial version
		if initialVersion.ID == "" {
			continue
		}

		// Calculate diff between initial and latest version
		_, additions, removals := diff.GenerateDiff(initialVersion.Content, file.Content, file.Path)

		// Only add to modified files if there are changes
		if additions > 0 || removals > 0 {
			// Remove working directory prefix from file path
			displayPath := file.Path
			workingDir := config.WorkingDirectory()
			displayPath = strings.TrimPrefix(displayPath, workingDir)
			displayPath = strings.TrimPrefix(displayPath, "/")

			m.modFiles[displayPath] = struct {
				additions int
				removals  int
			}{
				additions: additions,
				removals:  removals,
			}
		}
	}
}

func (m *sidebarCmp) processFileChanges(ctx context.Context, file history.File) {
	// Skip if not the latest version
	if file.Version == history.InitialVersion {
		return
	}

	// Get all versions of this file
	fileVersions, err := m.history.ListBySession(ctx, m.session.ID)
	if err != nil {
		return
	}

	// Find the initial version
	var initialVersion history.File
	for _, v := range fileVersions {
		if v.Path == file.Path && v.Version == history.InitialVersion {
			initialVersion = v
			break
		}
	}

	// Skip if we can't find the initial version
	if initialVersion.ID == "" {
		return
	}

	// Calculate diff between initial and latest version
	_, additions, removals := diff.GenerateDiff(initialVersion.Content, file.Content, file.Path)

	// Only add to modified files if there are changes
	if additions > 0 || removals > 0 {
		// Remove working directory prefix from file path
		displayPath := file.Path
		workingDir := config.WorkingDirectory()
		displayPath = strings.TrimPrefix(displayPath, workingDir)
		displayPath = strings.TrimPrefix(displayPath, "/")

		m.modFiles[displayPath] = struct {
			additions int
			removals  int
		}{
			additions: additions,
			removals:  removals,
		}
	}
}

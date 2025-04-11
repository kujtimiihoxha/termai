package chat

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/version"
)

type sidebarCmp struct {
	width, height int
}

func (m *sidebarCmp) Init() tea.Cmd {
	return nil
}

func (m *sidebarCmp) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *sidebarCmp) View() string {
	return styles.BaseStyle.Width(m.width).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			m.header(),
			" ",
			m.session(),
			" ",
			m.modifiedFiles(),
			" ",
			m.lspsConfigured(),
		),
	)
}

func (m *sidebarCmp) session() string {
	sessionKey := styles.BaseStyle.Foreground(styles.PrimaryColor).Render("Session")
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
		stats = styles.BaseStyle.Foreground(styles.ForgroundDim).Render(fmt.Sprintf("%d additions and  %d removals", additions, removals))
	} else if additions > 0 {
		stats = styles.BaseStyle.Foreground(styles.ForgroundDim).Render(fmt.Sprintf("%d additions", additions))
	} else if removals > 0 {
		stats = styles.BaseStyle.Foreground(styles.ForgroundDim).Render(fmt.Sprintf("%d removals", removals))
	}
	filePathStr := styles.BaseStyle.Foreground(styles.Forground).Render(filePath)

	return styles.BaseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				filePathStr,
				" ",
				stats,
			),
		)
}

func (m *sidebarCmp) lspsConfigured() string {
	lsps := styles.BaseStyle.Width(m.width).Foreground(styles.PrimaryColor).Render("LSP Configuration:")
	lspsConfigured := []struct {
		name string
		path string
	}{
		{"golsp", "path/to/lsp1"},
		{"vtsls", "path/to/lsp2"},
	}

	var lspViews []string
	for _, lsp := range lspsConfigured {
		lspName := styles.BaseStyle.Foreground(styles.Forground).Render(
			fmt.Sprintf("• %s", lsp.name),
		)
		lspPath := styles.BaseStyle.Foreground(styles.ForgroundDim).Render(
			fmt.Sprintf("(%s)", lsp.path),
		)
		lspViews = append(lspViews,
			styles.BaseStyle.
				Width(m.width).
				Render(
					lipgloss.JoinHorizontal(
						lipgloss.Left,
						lspName,
						" ",
						lspPath,
					),
				),
		)

	}
	return styles.BaseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lsps,
				lipgloss.JoinVertical(
					lipgloss.Left,
					lspViews...,
				),
			),
		)
}

func (m *sidebarCmp) modifiedFiles() string {
	modifiedFiles := styles.BaseStyle.Width(m.width).Foreground(styles.PrimaryColor).Render("Modified Files:")
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

func (m *sidebarCmp) logo() string {
	logo := fmt.Sprintf("%s %s", styles.OpenCodeIcon, "OpenCode")

	version := styles.BaseStyle.Foreground(styles.ForgroundDim).Render(version.Version)

	return styles.BaseStyle.
		Bold(true).
		Width(m.width).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				logo,
				" ",
				version,
			),
		)
}

func (m *sidebarCmp) header() string {
	header := lipgloss.JoinVertical(
		lipgloss.Top,
		m.logo(),
		m.cwd(),
	)
	return header
}

func (m *sidebarCmp) cwd() string {
	cwd := fmt.Sprintf("cwd: %s", config.WorkingDirectory())
	return styles.BaseStyle.
		Foreground(styles.ForgroundDim).
		Width(m.width).
		Render(cwd)
}

func (m *sidebarCmp) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *sidebarCmp) GetSize() (int, int) {
	return m.width, m.height
}

func NewSidebarCmp() tea.Model {
	return &sidebarCmp{}
}

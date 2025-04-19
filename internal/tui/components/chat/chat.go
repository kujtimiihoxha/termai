package chat

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kujtimiihoxha/opencode/internal/config"
	"github.com/kujtimiihoxha/opencode/internal/session"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/version"
)

type SendMsg struct {
	Text string
}

type SessionSelectedMsg = session.Session

type SessionClearedMsg struct{}

type EditorFocusMsg bool

func lspsConfigured(width int) string {
	cfg := config.Get()
	title := "LSP Configuration"
	title = ansi.Truncate(title, width, "…")

	lsps := styles.BaseStyle.Width(width).Foreground(styles.PrimaryColor).Bold(true).Render(title)

	// Get LSP names and sort them for consistent ordering
	var lspNames []string
	for name := range cfg.LSP {
		lspNames = append(lspNames, name)
	}
	sort.Strings(lspNames)

	var lspViews []string
	for _, name := range lspNames {
		lsp := cfg.LSP[name]
		lspName := styles.BaseStyle.Foreground(styles.Forground).Render(
			fmt.Sprintf("• %s", name),
		)
		cmd := lsp.Command
		cmd = ansi.Truncate(cmd, width-lipgloss.Width(lspName)-3, "…")
		lspPath := styles.BaseStyle.Foreground(styles.ForgroundDim).Render(
			fmt.Sprintf(" (%s)", cmd),
		)
		lspViews = append(lspViews,
			styles.BaseStyle.
				Width(width).
				Render(
					lipgloss.JoinHorizontal(
						lipgloss.Left,
						lspName,
						lspPath,
					),
				),
		)
	}
	return styles.BaseStyle.
		Width(width).
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

func logo(width int) string {
	logo := fmt.Sprintf("%s %s", styles.OpenCodeIcon, "OpenCode")

	version := styles.BaseStyle.Foreground(styles.ForgroundDim).Render(version.Version)

	return styles.BaseStyle.
		Bold(true).
		Width(width).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				logo,
				" ",
				version,
			),
		)
}

func repo(width int) string {
	repo := "https://github.com/kujtimiihoxha/opencode"
	return styles.BaseStyle.
		Foreground(styles.ForgroundDim).
		Width(width).
		Render(repo)
}

func cwd(width int) string {
	cwd := fmt.Sprintf("cwd: %s", config.WorkingDirectory())
	return styles.BaseStyle.
		Foreground(styles.ForgroundDim).
		Width(width).
		Render(cwd)
}

func header(width int) string {
	header := lipgloss.JoinVertical(
		lipgloss.Top,
		logo(width),
		repo(width),
		"",
		cwd(width),
	)
	return header
}

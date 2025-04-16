package chat

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/session"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/version"
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

	var lspViews []string
	for name, lsp := range cfg.LSP {
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

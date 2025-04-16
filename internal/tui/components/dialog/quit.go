package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

const question = "Are you sure you want to quit?"

type CloseQuitMsg struct{}

type QuitDialog interface {
	tea.Model
	layout.Bindings
}

type quitDialogCmp struct {
	selectedNo bool
}

type helpMapping struct {
	LeftRight  key.Binding
	EnterSpace key.Binding
	Yes        key.Binding
	No         key.Binding
	Tab        key.Binding
}

var helpKeys = helpMapping{
	LeftRight: key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←/→", "switch options"),
	),
	EnterSpace: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "confirm"),
	),
	Yes: key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y/Y", "yes"),
	),
	No: key.NewBinding(
		key.WithKeys("n", "N"),
		key.WithHelp("n/N", "no"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch options"),
	),
}

func (q *quitDialogCmp) Init() tea.Cmd {
	return nil
}

func (q *quitDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, helpKeys.LeftRight) || key.Matches(msg, helpKeys.Tab):
			q.selectedNo = !q.selectedNo
			return q, nil
		case key.Matches(msg, helpKeys.EnterSpace):
			if !q.selectedNo {
				return q, tea.Quit
			}
			return q, util.CmdHandler(CloseQuitMsg{})
		case key.Matches(msg, helpKeys.Yes):
			return q, tea.Quit
		case key.Matches(msg, helpKeys.No):
			return q, util.CmdHandler(CloseQuitMsg{})
		}
	}
	return q, nil
}

func (q *quitDialogCmp) View() string {
	yesStyle := styles.BaseStyle
	noStyle := styles.BaseStyle
	spacerStyle := styles.BaseStyle.Background(styles.Background)

	if q.selectedNo {
		noStyle = noStyle.Background(styles.PrimaryColor).Foreground(styles.Background)
		yesStyle = yesStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
	} else {
		yesStyle = yesStyle.Background(styles.PrimaryColor).Foreground(styles.Background)
		noStyle = noStyle.Background(styles.Background).Foreground(styles.PrimaryColor)
	}

	yesButton := yesStyle.Padding(0, 1).Render("Yes")
	noButton := noStyle.Padding(0, 1).Render("No")

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, yesButton, spacerStyle.Render("  "), noButton)

	width := lipgloss.Width(question)
	remainingWidth := width - lipgloss.Width(buttons)
	if remainingWidth > 0 {
		buttons = spacerStyle.Render(strings.Repeat(" ", remainingWidth)) + buttons
	}

	content := styles.BaseStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			question,
			"",
			buttons,
		),
	)

	return styles.BaseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(styles.Background).
		BorderForeground(styles.ForgroundDim).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (q *quitDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(helpKeys)
}

func NewQuitCmp() QuitDialog {
	return &quitDialogCmp{
		selectedNo: true,
	}
}

package logs

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/logging"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
)

type DetailComponent interface {
	tea.Model
	layout.Sizeable
	layout.Bindings
}

type detailCmp struct {
	width, height int
	currentLog    logging.LogMessage
	viewport      viewport.Model
}

func (i *detailCmp) Init() tea.Cmd {
	messages := logging.List()
	if len(messages) == 0 {
		return nil
	}
	i.currentLog = messages[0]
	return nil
}

func (i *detailCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case selectedLogMsg:
		if msg.ID != i.currentLog.ID {
			i.currentLog = logging.LogMessage(msg)
			i.updateContent()
		}
	}

	return i, nil
}

func (i *detailCmp) updateContent() {
	var content strings.Builder

	// Format the header with timestamp and level
	timeStyle := lipgloss.NewStyle().Foreground(styles.SubText0)
	levelStyle := getLevelStyle(i.currentLog.Level)

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		timeStyle.Render(i.currentLog.Time.Format(time.RFC3339)),
		"  ",
		levelStyle.Render(i.currentLog.Level),
	)

	content.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	content.WriteString("\n\n")

	// Message with styling
	messageStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Text)
	content.WriteString(messageStyle.Render("Message:"))
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Padding(0, 2).Render(i.currentLog.Message))
	content.WriteString("\n\n")

	// Attributes section
	if len(i.currentLog.Attributes) > 0 {
		attrHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Text)
		content.WriteString(attrHeaderStyle.Render("Attributes:"))
		content.WriteString("\n")

		// Create a table-like display for attributes
		keyStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
		valueStyle := lipgloss.NewStyle().Foreground(styles.Text)

		for _, attr := range i.currentLog.Attributes {
			attrLine := fmt.Sprintf("%s: %s",
				keyStyle.Render(attr.Key),
				valueStyle.Render(attr.Value),
			)
			content.WriteString(lipgloss.NewStyle().Padding(0, 2).Render(attrLine))
			content.WriteString("\n")
		}
	}

	i.viewport.SetContent(content.String())
}

func getLevelStyle(level string) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true)

	switch strings.ToLower(level) {
	case "info":
		return style.Foreground(styles.Blue)
	case "warn", "warning":
		return style.Foreground(styles.Warning)
	case "error", "err":
		return style.Foreground(styles.Error)
	case "debug":
		return style.Foreground(styles.Green)
	default:
		return style.Foreground(styles.Text)
	}
}

func (i *detailCmp) View() string {
	return styles.ForceReplaceBackgroundWithLipgloss(i.viewport.View(), styles.Background)
}

func (i *detailCmp) GetSize() (int, int) {
	return i.width, i.height
}

func (i *detailCmp) SetSize(width int, height int) tea.Cmd {
	i.width = width
	i.height = height
	i.viewport.Width = i.width
	i.viewport.Height = i.height
	i.updateContent()
	return nil
}

func (i *detailCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(i.viewport.KeyMap)
}

func NewLogsDetails() DetailComponent {
	return &detailCmp{
		viewport: viewport.New(0, 0),
	}
}

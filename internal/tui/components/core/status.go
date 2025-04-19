package core

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/config"
	"github.com/kujtimiihoxha/opencode/internal/llm/models"
	"github.com/kujtimiihoxha/opencode/internal/lsp"
	"github.com/kujtimiihoxha/opencode/internal/lsp/protocol"
	"github.com/kujtimiihoxha/opencode/internal/pubsub"
	"github.com/kujtimiihoxha/opencode/internal/session"
	"github.com/kujtimiihoxha/opencode/internal/tui/components/chat"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

type statusCmp struct {
	info       util.InfoMsg
	width      int
	messageTTL time.Duration
	lspClients map[string]*lsp.Client
	session    session.Session
}

// clearMessageCmd is a command that clears status messages after a timeout
func (m statusCmp) clearMessageCmd(ttl time.Duration) tea.Cmd {
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return util.ClearStatusMsg{}
	})
}

func (m statusCmp) Init() tea.Cmd {
	return nil
}

func (m statusCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case chat.SessionSelectedMsg:
		m.session = msg
	case chat.SessionClearedMsg:
		m.session = session.Session{}
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent {
			if m.session.ID == msg.Payload.ID {
				m.session = msg.Payload
			}
		}
	case util.InfoMsg:
		m.info = msg
		ttl := msg.TTL
		if ttl == 0 {
			ttl = m.messageTTL
		}
		return m, m.clearMessageCmd(ttl)
	case util.ClearStatusMsg:
		m.info = util.InfoMsg{}
	}
	return m, nil
}

var helpWidget = styles.Padded.Background(styles.ForgroundMid).Foreground(styles.BackgroundDarker).Bold(true).Render("ctrl+? help")

func formatTokensAndCost(tokens int64, cost float64) string {
	// Format tokens in human-readable format (e.g., 110K, 1.2M)
	var formattedTokens string
	switch {
	case tokens >= 1_000_000:
		formattedTokens = fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	case tokens >= 1_000:
		formattedTokens = fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	default:
		formattedTokens = fmt.Sprintf("%d", tokens)
	}

	// Remove .0 suffix if present
	if strings.HasSuffix(formattedTokens, ".0K") {
		formattedTokens = strings.Replace(formattedTokens, ".0K", "K", 1)
	}
	if strings.HasSuffix(formattedTokens, ".0M") {
		formattedTokens = strings.Replace(formattedTokens, ".0M", "M", 1)
	}

	// Format cost with $ symbol and 2 decimal places
	formattedCost := fmt.Sprintf("$%.2f", cost)

	return fmt.Sprintf("Tokens: %s, Cost: %s", formattedTokens, formattedCost)
}

func (m statusCmp) View() string {
	status := helpWidget
	if m.session.ID != "" {
		tokens := formatTokensAndCost(m.session.PromptTokens+m.session.CompletionTokens, m.session.Cost)
		tokensStyle := styles.Padded.
			Background(styles.Forground).
			Foreground(styles.BackgroundDim).
			Render(tokens)
		status += tokensStyle
	}

	diagnostics := styles.Padded.Background(styles.BackgroundDarker).Render(m.projectDiagnostics())
	if m.info.Msg != "" {
		infoStyle := styles.Padded.
			Foreground(styles.Base).
			Width(m.availableFooterMsgWidth(diagnostics))
		switch m.info.Type {
		case util.InfoTypeInfo:
			infoStyle = infoStyle.Background(styles.BorderColor)
		case util.InfoTypeWarn:
			infoStyle = infoStyle.Background(styles.Peach)
		case util.InfoTypeError:
			infoStyle = infoStyle.Background(styles.Red)
		}
		// Truncate message if it's longer than available width
		msg := m.info.Msg
		availWidth := m.availableFooterMsgWidth(diagnostics) - 10
		if len(msg) > availWidth && availWidth > 0 {
			msg = msg[:availWidth] + "..."
		}
		status += infoStyle.Render(msg)
	} else {
		status += styles.Padded.
			Foreground(styles.Base).
			Background(styles.BackgroundDim).
			Width(m.availableFooterMsgWidth(diagnostics)).
			Render("")
	}

	status += diagnostics
	status += m.model()
	return status
}

func (m *statusCmp) projectDiagnostics() string {
	// Check if any LSP server is still initializing
	initializing := false
	for _, client := range m.lspClients {
		if client.GetServerState() == lsp.StateStarting {
			initializing = true
			break
		}
	}
	
	// If any server is initializing, show that status
	if initializing {
		return lipgloss.NewStyle().
			Background(styles.BackgroundDarker).
			Foreground(styles.Peach).
			Render(fmt.Sprintf("%s Initializing LSP...", styles.SpinnerIcon))
	}
	
	errorDiagnostics := []protocol.Diagnostic{}
	warnDiagnostics := []protocol.Diagnostic{}
	hintDiagnostics := []protocol.Diagnostic{}
	infoDiagnostics := []protocol.Diagnostic{}
	for _, client := range m.lspClients {
		for _, d := range client.GetDiagnostics() {
			for _, diag := range d {
				switch diag.Severity {
				case protocol.SeverityError:
					errorDiagnostics = append(errorDiagnostics, diag)
				case protocol.SeverityWarning:
					warnDiagnostics = append(warnDiagnostics, diag)
				case protocol.SeverityHint:
					hintDiagnostics = append(hintDiagnostics, diag)
				case protocol.SeverityInformation:
					infoDiagnostics = append(infoDiagnostics, diag)
				}
			}
		}
	}

	if len(errorDiagnostics) == 0 && len(warnDiagnostics) == 0 && len(hintDiagnostics) == 0 && len(infoDiagnostics) == 0 {
		return "No diagnostics"
	}

	diagnostics := []string{}

	if len(errorDiagnostics) > 0 {
		errStr := lipgloss.NewStyle().
			Background(styles.BackgroundDarker).
			Foreground(styles.Error).
			Render(fmt.Sprintf("%s %d", styles.ErrorIcon, len(errorDiagnostics)))
		diagnostics = append(diagnostics, errStr)
	}
	if len(warnDiagnostics) > 0 {
		warnStr := lipgloss.NewStyle().
			Background(styles.BackgroundDarker).
			Foreground(styles.Warning).
			Render(fmt.Sprintf("%s %d", styles.WarningIcon, len(warnDiagnostics)))
		diagnostics = append(diagnostics, warnStr)
	}
	if len(hintDiagnostics) > 0 {
		hintStr := lipgloss.NewStyle().
			Background(styles.BackgroundDarker).
			Foreground(styles.Text).
			Render(fmt.Sprintf("%s %d", styles.HintIcon, len(hintDiagnostics)))
		diagnostics = append(diagnostics, hintStr)
	}
	if len(infoDiagnostics) > 0 {
		infoStr := lipgloss.NewStyle().
			Background(styles.BackgroundDarker).
			Foreground(styles.Peach).
			Render(fmt.Sprintf("%s %d", styles.InfoIcon, len(infoDiagnostics)))
		diagnostics = append(diagnostics, infoStr)
	}

	return strings.Join(diagnostics, " ")
}

func (m statusCmp) availableFooterMsgWidth(diagnostics string) int {
	tokens := ""
	tokensWidth := 0
	if m.session.ID != "" {
		tokens = formatTokensAndCost(m.session.PromptTokens+m.session.CompletionTokens, m.session.Cost)
		tokensWidth = lipgloss.Width(tokens) + 2
	}
	return max(0, m.width-lipgloss.Width(helpWidget)-lipgloss.Width(m.model())-lipgloss.Width(diagnostics)-tokensWidth)
}

func (m statusCmp) model() string {
	cfg := config.Get()

	coder, ok := cfg.Agents[config.AgentCoder]
	if !ok {
		return "Unknown"
	}
	model := models.SupportedModels[coder.Model]
	return styles.Padded.Background(styles.Grey).Foreground(styles.Text).Render(model.Name)
}

func NewStatusCmp(lspClients map[string]*lsp.Client) tea.Model {
	return &statusCmp{
		messageTTL: 10 * time.Second,
		lspClients: lspClients,
	}
}

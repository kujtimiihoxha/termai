package repl

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/app"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
	"github.com/kujtimiihoxha/termai/internal/session"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
)

type SessionsCmp interface {
	tea.Model
	layout.Sizeable
	layout.Focusable
	layout.Bordered
	layout.Bindings
}
type sessionsCmp struct {
	app     *app.App
	list    list.Model
	focused bool
}

type listItem struct {
	id, title, desc string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.title }

type InsertSessionsMsg struct {
	sessions []session.Session
}

type SelectedSessionMsg struct {
	SessionID string
}

type sessionsKeyMap struct {
	Select key.Binding
}

var sessionKeyMapValue = sessionsKeyMap{
	Select: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "select session"),
	),
}

func (i *sessionsCmp) Init() tea.Cmd {
	existing, err := i.app.Sessions.List(context.Background())
	if err != nil {
		return util.ReportError(err)
	}
	if len(existing) == 0 || existing[0].MessageCount > 0 {
		newSession, err := i.app.Sessions.Create(
			context.Background(),
			"New Session",
		)
		if err != nil {
			return util.ReportError(err)
		}
		existing = append([]session.Session{newSession}, existing...)
	}
	return tea.Batch(
		util.CmdHandler(InsertSessionsMsg{existing}),
		util.CmdHandler(SelectedSessionMsg{existing[0].ID}),
	)
}

func (i *sessionsCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case InsertSessionsMsg:
		items := make([]list.Item, len(msg.sessions))
		for i, s := range msg.sessions {
			items[i] = listItem{
				id:    s.ID,
				title: s.Title,
				desc:  formatTokensAndCost(s.PromptTokens+s.CompletionTokens, s.Cost),
			}
		}
		return i, i.list.SetItems(items)
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.CreatedEvent && msg.Payload.ParentSessionID == "" {
			// Check if the session is already in the list
			items := i.list.Items()
			for _, item := range items {
				s := item.(listItem)
				if s.id == msg.Payload.ID {
					return i, nil
				}
			}
			// insert the new session at the top of the list
			items = append([]list.Item{listItem{
				id:    msg.Payload.ID,
				title: msg.Payload.Title,
				desc:  formatTokensAndCost(msg.Payload.PromptTokens+msg.Payload.CompletionTokens, msg.Payload.Cost),
			}}, items...)
			return i, i.list.SetItems(items)
		} else if msg.Type == pubsub.UpdatedEvent {
			// update the session in the list
			items := i.list.Items()
			for idx, item := range items {
				s := item.(listItem)
				if s.id == msg.Payload.ID {
					s.title = msg.Payload.Title
					s.desc = formatTokensAndCost(msg.Payload.PromptTokens+msg.Payload.CompletionTokens, msg.Payload.Cost)
					items[idx] = s
					break
				}
			}
			return i, i.list.SetItems(items)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, sessionKeyMapValue.Select):
			selected := i.list.SelectedItem()
			if selected == nil {
				return i, nil
			}
			return i, util.CmdHandler(SelectedSessionMsg{selected.(listItem).id})
		}
	}
	if i.focused {
		u, cmd := i.list.Update(msg)
		i.list = u
		return i, cmd
	}
	return i, nil
}

func (i *sessionsCmp) View() string {
	return i.list.View()
}

func (i *sessionsCmp) Blur() tea.Cmd {
	i.focused = false
	return nil
}

func (i *sessionsCmp) Focus() tea.Cmd {
	i.focused = true
	return nil
}

func (i *sessionsCmp) GetSize() (int, int) {
	return i.list.Width(), i.list.Height()
}

func (i *sessionsCmp) IsFocused() bool {
	return i.focused
}

func (i *sessionsCmp) SetSize(width int, height int) {
	i.list.SetSize(width, height)
}

func (i *sessionsCmp) BorderText() map[layout.BorderPosition]string {
	totalCount := len(i.list.Items())
	itemsPerPage := i.list.Paginator.PerPage
	currentPage := i.list.Paginator.Page

	current := min(currentPage*itemsPerPage+itemsPerPage, totalCount)

	pageInfo := fmt.Sprintf(
		"%d-%d of %d",
		currentPage*itemsPerPage+1,
		current,
		totalCount,
	)

	title := "Sessions"
	if i.focused {
		title = lipgloss.NewStyle().Foreground(styles.Primary).Render(title)
	}
	return map[layout.BorderPosition]string{
		layout.TopMiddleBorder:    title,
		layout.BottomMiddleBorder: pageInfo,
	}
}

func (i *sessionsCmp) BindingKeys() []key.Binding {
	return append(layout.KeyMapToSlice(i.list.KeyMap), sessionKeyMapValue.Select)
}

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

func NewSessionsCmp(app *app.App) SessionsCmp {
	listDelegate := list.NewDefaultDelegate()
	defaultItemStyle := list.NewDefaultItemStyles()
	defaultItemStyle.SelectedTitle = defaultItemStyle.SelectedTitle.BorderForeground(styles.Secondary).Foreground(styles.Primary)
	defaultItemStyle.SelectedDesc = defaultItemStyle.SelectedDesc.BorderForeground(styles.Secondary).Foreground(styles.Primary)

	defaultStyle := list.DefaultStyles()
	defaultStyle.FilterPrompt = defaultStyle.FilterPrompt.Foreground(styles.Secondary)
	defaultStyle.FilterCursor = defaultStyle.FilterCursor.Foreground(styles.Flamingo)

	listDelegate.Styles = defaultItemStyle

	listComponent := list.New([]list.Item{}, listDelegate, 0, 0)
	listComponent.FilterInput.PromptStyle = defaultStyle.FilterPrompt
	listComponent.FilterInput.Cursor.Style = defaultStyle.FilterCursor
	listComponent.SetShowTitle(false)
	listComponent.SetShowPagination(false)
	listComponent.SetShowHelp(false)
	listComponent.SetShowStatusBar(false)
	listComponent.DisableQuitKeybindings()

	return &sessionsCmp{
		app:     app,
		list:    listComponent,
		focused: false,
	}
}

package core

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/tui/layout"
	"github.com/kujtimiihoxha/termai/internal/tui/util"
)

type SizeableModel interface {
	tea.Model
	layout.Sizeable
}

type DialogMsg struct {
	Content     SizeableModel
	WidthRatio  float64
	HeightRatio float64

	MinWidth  int
	MinHeight int
}

type DialogCloseMsg struct{}

type KeyBindings struct {
	Return key.Binding
}

var keys = KeyBindings{
	Return: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
}

type DialogCmp interface {
	tea.Model
	layout.Bindings
}

type dialogCmp struct {
	content      SizeableModel
	screenWidth  int
	screenHeight int

	widthRatio  float64
	heightRatio float64

	minWidth  int
	minHeight int

	width  int
	height int
}

func (d *dialogCmp) Init() tea.Cmd {
	return nil
}

func (d *dialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.screenWidth = msg.Width
		d.screenHeight = msg.Height
		d.width = max(int(float64(d.screenWidth)*d.widthRatio), d.minWidth)
		d.height = max(int(float64(d.screenHeight)*d.heightRatio), d.minHeight)
		if d.content != nil {
			d.content.SetSize(d.width, d.height)
		}
		return d, nil
	case DialogMsg:
		d.content = msg.Content
		d.widthRatio = msg.WidthRatio
		d.heightRatio = msg.HeightRatio
		d.minWidth = msg.MinWidth
		d.minHeight = msg.MinHeight
		d.width = max(int(float64(d.screenWidth)*d.widthRatio), d.minWidth)
		d.height = max(int(float64(d.screenHeight)*d.heightRatio), d.minHeight)
		if d.content != nil {
			d.content.SetSize(d.width, d.height)
		}
	case DialogCloseMsg:
		d.content = nil
		return d, nil
	case tea.KeyMsg:
		if key.Matches(msg, keys.Return) {
			return d, util.CmdHandler(DialogCloseMsg{})
		}
	}
	if d.content != nil {
		u, cmd := d.content.Update(msg)
		d.content = u.(SizeableModel)
		return d, cmd
	}
	return d, nil
}

func (d *dialogCmp) BindingKeys() []key.Binding {
	bindings := []key.Binding{keys.Return}
	if d.content == nil {
		return bindings
	}
	if c, ok := d.content.(layout.Bindings); ok {
		return append(bindings, c.BindingKeys()...)
	}
	return bindings
}

func (d *dialogCmp) View() string {
	return lipgloss.NewStyle().Width(d.width).Height(d.height).Render(d.content.View())
}

func NewDialogCmp() DialogCmp {
	return &dialogCmp{}
}

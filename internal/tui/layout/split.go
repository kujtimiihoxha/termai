package layout

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
)

type SplitPaneLayout interface {
	tea.Model
	Sizeable
	SetLeftPanel(panel Container)
	SetRightPanel(panel Container)
	SetBottomPanel(panel Container)
}

type splitPaneLayout struct {
	width         int
	height        int
	ratio         float64
	verticalRatio float64

	rightPanel  Container
	leftPanel   Container
	bottomPanel Container

	backgroundColor lipgloss.TerminalColor
}

type SplitPaneOption func(*splitPaneLayout)

func (s *splitPaneLayout) Init() tea.Cmd {
	var cmds []tea.Cmd

	if s.leftPanel != nil {
		cmds = append(cmds, s.leftPanel.Init())
	}

	if s.rightPanel != nil {
		cmds = append(cmds, s.rightPanel.Init())
	}

	if s.bottomPanel != nil {
		cmds = append(cmds, s.bottomPanel.Init())
	}

	return tea.Batch(cmds...)
}

func (s *splitPaneLayout) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
		return s, nil
	}

	if s.rightPanel != nil {
		u, cmd := s.rightPanel.Update(msg)
		s.rightPanel = u.(Container)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if s.leftPanel != nil {
		u, cmd := s.leftPanel.Update(msg)
		s.leftPanel = u.(Container)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if s.bottomPanel != nil {
		u, cmd := s.bottomPanel.Update(msg)
		s.bottomPanel = u.(Container)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return s, tea.Batch(cmds...)
}

func (s *splitPaneLayout) View() string {
	var topSection string

	if s.leftPanel != nil && s.rightPanel != nil {
		leftView := s.leftPanel.View()
		rightView := s.rightPanel.View()
		topSection = lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
	} else if s.leftPanel != nil {
		topSection = s.leftPanel.View()
	} else if s.rightPanel != nil {
		topSection = s.rightPanel.View()
	} else {
		topSection = ""
	}

	var finalView string

	if s.bottomPanel != nil && topSection != "" {
		bottomView := s.bottomPanel.View()
		finalView = lipgloss.JoinVertical(lipgloss.Left, topSection, bottomView)
	} else if s.bottomPanel != nil {
		finalView = s.bottomPanel.View()
	} else {
		finalView = topSection
	}

	if s.backgroundColor != nil && finalView != "" {
		style := lipgloss.NewStyle().
			Width(s.width).
			Height(s.height).
			Background(s.backgroundColor)

		return style.Render(finalView)
	}

	return finalView
}

func (s *splitPaneLayout) SetSize(width, height int) {
	s.width = width
	s.height = height

	var topHeight, bottomHeight int
	if s.bottomPanel != nil {
		topHeight = int(float64(height) * s.verticalRatio)
		bottomHeight = height - topHeight
	} else {
		topHeight = height
		bottomHeight = 0
	}

	var leftWidth, rightWidth int
	if s.leftPanel != nil && s.rightPanel != nil {
		leftWidth = int(float64(width) * s.ratio)
		rightWidth = width - leftWidth
	} else if s.leftPanel != nil {
		leftWidth = width
		rightWidth = 0
	} else if s.rightPanel != nil {
		leftWidth = 0
		rightWidth = width
	}

	if s.leftPanel != nil {
		s.leftPanel.SetSize(leftWidth, topHeight)
	}

	if s.rightPanel != nil {
		s.rightPanel.SetSize(rightWidth, topHeight)
	}

	if s.bottomPanel != nil {
		s.bottomPanel.SetSize(width, bottomHeight)
	}
}

func (s *splitPaneLayout) GetSize() (int, int) {
	return s.width, s.height
}

func (s *splitPaneLayout) SetLeftPanel(panel Container) {
	s.leftPanel = panel
	if s.width > 0 && s.height > 0 {
		s.SetSize(s.width, s.height)
	}
}

func (s *splitPaneLayout) SetRightPanel(panel Container) {
	s.rightPanel = panel
	if s.width > 0 && s.height > 0 {
		s.SetSize(s.width, s.height)
	}
}

func (s *splitPaneLayout) SetBottomPanel(panel Container) {
	s.bottomPanel = panel
	if s.width > 0 && s.height > 0 {
		s.SetSize(s.width, s.height)
	}
}

func (s *splitPaneLayout) BindingKeys() []key.Binding {
	keys := []key.Binding{}
	if s.leftPanel != nil {
		if b, ok := s.leftPanel.(Bindings); ok {
			keys = append(keys, b.BindingKeys()...)
		}
	}
	if s.rightPanel != nil {
		if b, ok := s.rightPanel.(Bindings); ok {
			keys = append(keys, b.BindingKeys()...)
		}
	}
	if s.bottomPanel != nil {
		if b, ok := s.bottomPanel.(Bindings); ok {
			keys = append(keys, b.BindingKeys()...)
		}
	}
	return keys
}

func NewSplitPane(options ...SplitPaneOption) SplitPaneLayout {
	layout := &splitPaneLayout{
		ratio:           0.7,
		verticalRatio:   0.9, // Default 80% for top section, 20% for bottom
		backgroundColor: styles.Background,
	}
	for _, option := range options {
		option(layout)
	}
	return layout
}

func WithLeftPanel(panel Container) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.leftPanel = panel
	}
}

func WithRightPanel(panel Container) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.rightPanel = panel
	}
}

func WithRatio(ratio float64) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.ratio = ratio
	}
}

func WithSplitBackgroundColor(color lipgloss.TerminalColor) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.backgroundColor = color
	}
}

func WithBottomPanel(panel Container) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.bottomPanel = panel
	}
}

func WithVerticalRatio(ratio float64) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.verticalRatio = ratio
	}
}

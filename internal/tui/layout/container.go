package layout

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/termai/internal/tui/styles"
)

type Container interface {
	tea.Model
	Sizeable
}
type container struct {
	width  int
	height int

	content tea.Model

	// Style options
	paddingTop    int
	paddingRight  int
	paddingBottom int
	paddingLeft   int

	borderTop    bool
	borderRight  bool
	borderBottom bool
	borderLeft   bool
	borderStyle  lipgloss.Border
	borderColor  lipgloss.TerminalColor

	backgroundColor lipgloss.TerminalColor
}

func (c *container) Init() tea.Cmd {
	return c.content.Init()
}

func (c *container) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	u, cmd := c.content.Update(msg)
	c.content = u
	return c, cmd
}

func (c *container) View() string {
	style := lipgloss.NewStyle()
	width := c.width
	height := c.height
	// Apply background color if specified
	if c.backgroundColor != nil {
		style = style.Background(c.backgroundColor)
	}

	// Apply border if any side is enabled
	if c.borderTop || c.borderRight || c.borderBottom || c.borderLeft {
		// Adjust width and height for borders
		if c.borderTop {
			height--
		}
		if c.borderBottom {
			height--
		}
		if c.borderLeft {
			width--
		}
		if c.borderRight {
			width--
		}
		style = style.Border(c.borderStyle, c.borderTop, c.borderRight, c.borderBottom, c.borderLeft)

		// Apply border color if specified
		if c.borderColor != nil {
			style = style.BorderBackground(c.backgroundColor).BorderForeground(c.borderColor)
		}
	}
	style = style.
		Width(width).
		Height(height).
		PaddingTop(c.paddingTop).
		PaddingRight(c.paddingRight).
		PaddingBottom(c.paddingBottom).
		PaddingLeft(c.paddingLeft)

	return style.Render(c.content.View())
}

func (c *container) SetSize(width, height int) {
	c.width = width
	c.height = height

	// If the content implements Sizeable, adjust its size to account for padding and borders
	if sizeable, ok := c.content.(Sizeable); ok {
		// Calculate horizontal space taken by padding and borders
		horizontalSpace := c.paddingLeft + c.paddingRight
		if c.borderLeft {
			horizontalSpace++
		}
		if c.borderRight {
			horizontalSpace++
		}

		// Calculate vertical space taken by padding and borders
		verticalSpace := c.paddingTop + c.paddingBottom
		if c.borderTop {
			verticalSpace++
		}
		if c.borderBottom {
			verticalSpace++
		}

		// Set content size with adjusted dimensions
		contentWidth := max(0, width-horizontalSpace)
		contentHeight := max(0, height-verticalSpace)
		sizeable.SetSize(contentWidth, contentHeight)
	}
}

func (c *container) GetSize() (int, int) {
	return c.width, c.height
}

func (c *container) BindingKeys() []key.Binding {
	if b, ok := c.content.(Bindings); ok {
		return b.BindingKeys()
	}
	return []key.Binding{}
}

type ContainerOption func(*container)

func NewContainer(content tea.Model, options ...ContainerOption) Container {
	c := &container{
		content:         content,
		borderColor:     styles.BorderColor,
		borderStyle:     lipgloss.NormalBorder(),
		backgroundColor: styles.Background,
	}

	for _, option := range options {
		option(c)
	}

	return c
}

// Padding options
func WithPadding(top, right, bottom, left int) ContainerOption {
	return func(c *container) {
		c.paddingTop = top
		c.paddingRight = right
		c.paddingBottom = bottom
		c.paddingLeft = left
	}
}

func WithPaddingAll(padding int) ContainerOption {
	return WithPadding(padding, padding, padding, padding)
}

func WithPaddingHorizontal(padding int) ContainerOption {
	return func(c *container) {
		c.paddingLeft = padding
		c.paddingRight = padding
	}
}

func WithPaddingVertical(padding int) ContainerOption {
	return func(c *container) {
		c.paddingTop = padding
		c.paddingBottom = padding
	}
}

func WithBorder(top, right, bottom, left bool) ContainerOption {
	return func(c *container) {
		c.borderTop = top
		c.borderRight = right
		c.borderBottom = bottom
		c.borderLeft = left
	}
}

func WithBorderAll() ContainerOption {
	return WithBorder(true, true, true, true)
}

func WithBorderHorizontal() ContainerOption {
	return WithBorder(true, false, true, false)
}

func WithBorderVertical() ContainerOption {
	return WithBorder(false, true, false, true)
}

func WithBorderStyle(style lipgloss.Border) ContainerOption {
	return func(c *container) {
		c.borderStyle = style
	}
}

func WithBorderColor(color lipgloss.TerminalColor) ContainerOption {
	return func(c *container) {
		c.borderColor = color
	}
}

func WithRoundedBorder() ContainerOption {
	return WithBorderStyle(lipgloss.RoundedBorder())
}

func WithThickBorder() ContainerOption {
	return WithBorderStyle(lipgloss.ThickBorder())
}

func WithDoubleBorder() ContainerOption {
	return WithBorderStyle(lipgloss.DoubleBorder())
}

func WithBackgroundColor(color lipgloss.TerminalColor) ContainerOption {
	return func(c *container) {
		c.backgroundColor = color
	}
}

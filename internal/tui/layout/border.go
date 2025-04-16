package layout

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
)

type BorderPosition int

const (
	TopLeftBorder BorderPosition = iota
	TopMiddleBorder
	TopRightBorder
	BottomLeftBorder
	BottomMiddleBorder
	BottomRightBorder
)

var (
	ActiveBorder          = styles.Blue
	InactivePreviewBorder = styles.Grey
)

type BorderOptions struct {
	Active         bool
	EmbeddedText   map[BorderPosition]string
	ActiveColor    lipgloss.TerminalColor
	InactiveColor  lipgloss.TerminalColor
	ActiveBorder   lipgloss.Border
	InactiveBorder lipgloss.Border
}

func Borderize(content string, opts BorderOptions) string {
	if opts.EmbeddedText == nil {
		opts.EmbeddedText = make(map[BorderPosition]string)
	}
	if opts.ActiveColor == nil {
		opts.ActiveColor = ActiveBorder
	}
	if opts.InactiveColor == nil {
		opts.InactiveColor = InactivePreviewBorder
	}
	if opts.ActiveBorder == (lipgloss.Border{}) {
		opts.ActiveBorder = lipgloss.ThickBorder()
	}
	if opts.InactiveBorder == (lipgloss.Border{}) {
		opts.InactiveBorder = lipgloss.NormalBorder()
	}

	var (
		thickness = map[bool]lipgloss.Border{
			true:  opts.ActiveBorder,
			false: opts.InactiveBorder,
		}
		color = map[bool]lipgloss.TerminalColor{
			true:  opts.ActiveColor,
			false: opts.InactiveColor,
		}
		border = thickness[opts.Active]
		style  = lipgloss.NewStyle().Foreground(color[opts.Active])
		width  = lipgloss.Width(content)
	)

	encloseInSquareBrackets := func(text string) string {
		if text != "" {
			return fmt.Sprintf("%s%s%s",
				style.Render(border.TopRight),
				text,
				style.Render(border.TopLeft),
			)
		}
		return text
	}
	buildHorizontalBorder := func(leftText, middleText, rightText, leftCorner, inbetween, rightCorner string) string {
		leftText = encloseInSquareBrackets(leftText)
		middleText = encloseInSquareBrackets(middleText)
		rightText = encloseInSquareBrackets(rightText)
		// Calculate length of border between embedded texts
		remaining := max(0, width-lipgloss.Width(leftText)-lipgloss.Width(middleText)-lipgloss.Width(rightText))
		leftBorderLen := max(0, (width/2)-lipgloss.Width(leftText)-(lipgloss.Width(middleText)/2))
		rightBorderLen := max(0, remaining-leftBorderLen)
		// Then construct border string
		s := leftText +
			style.Render(strings.Repeat(inbetween, leftBorderLen)) +
			middleText +
			style.Render(strings.Repeat(inbetween, rightBorderLen)) +
			rightText
		// Make it fit in the space available between the two corners.
		s = lipgloss.NewStyle().
			Inline(true).
			MaxWidth(width).
			Render(s)
		// Add the corners
		return style.Render(leftCorner) + s + style.Render(rightCorner)
	}
	// Stack top border, content and horizontal borders, and bottom border.
	return strings.Join([]string{
		buildHorizontalBorder(
			opts.EmbeddedText[TopLeftBorder],
			opts.EmbeddedText[TopMiddleBorder],
			opts.EmbeddedText[TopRightBorder],
			border.TopLeft,
			border.Top,
			border.TopRight,
		),
		lipgloss.NewStyle().
			BorderForeground(color[opts.Active]).
			Border(border, false, true, false, true).Render(content),
		buildHorizontalBorder(
			opts.EmbeddedText[BottomLeftBorder],
			opts.EmbeddedText[BottomMiddleBorder],
			opts.EmbeddedText[BottomRightBorder],
			border.BottomLeft,
			border.Bottom,
			border.BottomRight,
		),
	}, "\n")
}

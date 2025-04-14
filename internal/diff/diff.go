package diff

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// LineType represents the kind of line in a diff.
type LineType int

const (
	// LineContext represents a line that exists in both the old and new file.
	LineContext LineType = iota
	// LineAdded represents a line added in the new file.
	LineAdded
	// LineRemoved represents a line removed from the old file.
	LineRemoved
)

// DiffLine represents a single line in a diff, either from the old file,
// the new file, or a context line.
type DiffLine struct {
	OldLineNo int      // Line number in the old file (0 for added lines)
	NewLineNo int      // Line number in the new file (0 for removed lines)
	Kind      LineType // Type of line (added, removed, context)
	Content   string   // Content of the line
}

// Hunk represents a section of changes in a diff.
type Hunk struct {
	Header string
	Lines  []DiffLine
}

// DiffResult contains the parsed result of a diff.
type DiffResult struct {
	OldFile string
	NewFile string
	Hunks   []Hunk
}

// HunkDelta represents the change statistics for a hunk.
type HunkDelta struct {
	StartLine1 int
	LineCount1 int
	StartLine2 int
	LineCount2 int
}

// linePair represents a pair of lines to be displayed side by side.
type linePair struct {
	left  *DiffLine
	right *DiffLine
}

// -------------------------------------------------------------------------
// Style Configuration with Option Pattern
// -------------------------------------------------------------------------

// StyleConfig defines styling for diff rendering.
type StyleConfig struct {
	RemovedLineBg       lipgloss.Color
	AddedLineBg         lipgloss.Color
	ContextLineBg       lipgloss.Color
	HunkLineBg          lipgloss.Color
	HunkLineFg          lipgloss.Color
	RemovedFg           lipgloss.Color
	AddedFg             lipgloss.Color
	LineNumberFg        lipgloss.Color
	HighlightStyle      string
	RemovedHighlightBg  lipgloss.Color
	AddedHighlightBg    lipgloss.Color
	RemovedLineNumberBg lipgloss.Color
	AddedLineNamerBg    lipgloss.Color
	RemovedHighlightFg  lipgloss.Color
	AddedHighlightFg    lipgloss.Color
}

// StyleOption defines a function that modifies a StyleConfig.
type StyleOption func(*StyleConfig)

// NewStyleConfig creates a StyleConfig with default values and applies any provided options.
func NewStyleConfig(opts ...StyleOption) StyleConfig {
	// Set default values
	config := StyleConfig{
		RemovedLineBg:       lipgloss.Color("#3A3030"),
		AddedLineBg:         lipgloss.Color("#303A30"),
		ContextLineBg:       lipgloss.Color("#212121"),
		HunkLineBg:          lipgloss.Color("#2A2822"),
		HunkLineFg:          lipgloss.Color("#D4AF37"),
		RemovedFg:           lipgloss.Color("#7C4444"),
		AddedFg:             lipgloss.Color("#478247"),
		LineNumberFg:        lipgloss.Color("#888888"),
		HighlightStyle:      "dracula",
		RemovedHighlightBg:  lipgloss.Color("#612726"),
		AddedHighlightBg:    lipgloss.Color("#256125"),
		RemovedLineNumberBg: lipgloss.Color("#332929"),
		AddedLineNamerBg:    lipgloss.Color("#293229"),
		RemovedHighlightFg:  lipgloss.Color("#FADADD"),
		AddedHighlightFg:    lipgloss.Color("#DAFADA"),
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(&config)
	}

	return config
}

// WithRemovedLineBg sets the background color for removed lines.
func WithRemovedLineBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.RemovedLineBg = color
	}
}

// WithAddedLineBg sets the background color for added lines.
func WithAddedLineBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.AddedLineBg = color
	}
}

// WithContextLineBg sets the background color for context lines.
func WithContextLineBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.ContextLineBg = color
	}
}

// WithRemovedFg sets the foreground color for removed line markers.
func WithRemovedFg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.RemovedFg = color
	}
}

// WithAddedFg sets the foreground color for added line markers.
func WithAddedFg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.AddedFg = color
	}
}

// WithLineNumberFg sets the foreground color for line numbers.
func WithLineNumberFg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.LineNumberFg = color
	}
}

// WithHighlightStyle sets the syntax highlighting style.
func WithHighlightStyle(style string) StyleOption {
	return func(s *StyleConfig) {
		s.HighlightStyle = style
	}
}

// WithRemovedHighlightColors sets the colors for highlighted parts in removed text.
func WithRemovedHighlightColors(bg, fg lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.RemovedHighlightBg = bg
		s.RemovedHighlightFg = fg
	}
}

// WithAddedHighlightColors sets the colors for highlighted parts in added text.
func WithAddedHighlightColors(bg, fg lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.AddedHighlightBg = bg
		s.AddedHighlightFg = fg
	}
}

// WithRemovedLineNumberBg sets the background color for removed line numbers.
func WithRemovedLineNumberBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.RemovedLineNumberBg = color
	}
}

// WithAddedLineNumberBg sets the background color for added line numbers.
func WithAddedLineNumberBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.AddedLineNamerBg = color
	}
}

func WithHunkLineBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.HunkLineBg = color
	}
}

func WithHunkLineFg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.HunkLineFg = color
	}
}

// -------------------------------------------------------------------------
// Parse Options with Option Pattern
// -------------------------------------------------------------------------

// ParseConfig configures the behavior of diff parsing.
type ParseConfig struct {
	ContextSize int // Number of context lines to include
}

// ParseOption defines a function that modifies a ParseConfig.
type ParseOption func(*ParseConfig)

// WithContextSize sets the number of context lines to include.
func WithContextSize(size int) ParseOption {
	return func(p *ParseConfig) {
		if size >= 0 {
			p.ContextSize = size
		}
	}
}

// -------------------------------------------------------------------------
// Side-by-Side Options with Option Pattern
// -------------------------------------------------------------------------

// SideBySideConfig configures the rendering of side-by-side diffs.
type SideBySideConfig struct {
	TotalWidth int
	Style      StyleConfig
}

// SideBySideOption defines a function that modifies a SideBySideConfig.
type SideBySideOption func(*SideBySideConfig)

// NewSideBySideConfig creates a SideBySideConfig with default values and applies any provided options.
func NewSideBySideConfig(opts ...SideBySideOption) SideBySideConfig {
	// Set default values
	config := SideBySideConfig{
		TotalWidth: 160, // Default width for side-by-side view
		Style:      NewStyleConfig(),
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(&config)
	}

	return config
}

// WithTotalWidth sets the total width for side-by-side view.
func WithTotalWidth(width int) SideBySideOption {
	return func(s *SideBySideConfig) {
		if width > 0 {
			s.TotalWidth = width
		}
	}
}

// WithStyle sets the styling configuration.
func WithStyle(style StyleConfig) SideBySideOption {
	return func(s *SideBySideConfig) {
		s.Style = style
	}
}

// WithStyleOptions applies the specified style options.
func WithStyleOptions(opts ...StyleOption) SideBySideOption {
	return func(s *SideBySideConfig) {
		s.Style = NewStyleConfig(opts...)
	}
}

// -------------------------------------------------------------------------
// Diff Parsing and Generation
// -------------------------------------------------------------------------

// ParseUnifiedDiff parses a unified diff format string into structured data.
func ParseUnifiedDiff(diff string) (DiffResult, error) {
	var result DiffResult
	var currentHunk *Hunk

	hunkHeaderRe := regexp.MustCompile(`^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@`)
	lines := strings.Split(diff, "\n")

	var oldLine, newLine int
	inFileHeader := true

	for _, line := range lines {
		// Parse the file headers
		if inFileHeader {
			if strings.HasPrefix(line, "--- a/") {
				result.OldFile = strings.TrimPrefix(line, "--- a/")
				continue
			}
			if strings.HasPrefix(line, "+++ b/") {
				result.NewFile = strings.TrimPrefix(line, "+++ b/")
				inFileHeader = false
				continue
			}
		}

		// Parse hunk headers
		if matches := hunkHeaderRe.FindStringSubmatch(line); matches != nil {
			if currentHunk != nil {
				result.Hunks = append(result.Hunks, *currentHunk)
			}
			currentHunk = &Hunk{
				Header: line,
				Lines:  []DiffLine{},
			}

			oldStart, _ := strconv.Atoi(matches[1])
			newStart, _ := strconv.Atoi(matches[3])
			oldLine = oldStart
			newLine = newStart

			continue
		}

		// ignore the \\ No newline at end of file
		if strings.HasPrefix(line, "\\ No newline at end of file") {
			continue
		}
		if currentHunk == nil {
			continue
		}

		if len(line) > 0 {
			// Process the line based on its prefix
			switch line[0] {
			case '+':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					OldLineNo: 0,
					NewLineNo: newLine,
					Kind:      LineAdded,
					Content:   line[1:], // skip '+'
				})
				newLine++
			case '-':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					OldLineNo: oldLine,
					NewLineNo: 0,
					Kind:      LineRemoved,
					Content:   line[1:], // skip '-'
				})
				oldLine++
			default:
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					OldLineNo: oldLine,
					NewLineNo: newLine,
					Kind:      LineContext,
					Content:   line,
				})
				oldLine++
				newLine++
			}
		} else {
			// Handle empty lines
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				OldLineNo: oldLine,
				NewLineNo: newLine,
				Kind:      LineContext,
				Content:   "",
			})
			oldLine++
			newLine++
		}
	}

	// Add the last hunk if there is one
	if currentHunk != nil {
		result.Hunks = append(result.Hunks, *currentHunk)
	}

	return result, nil
}

// HighlightIntralineChanges updates the content of lines in a hunk to show
// character-level differences within lines.
func HighlightIntralineChanges(h *Hunk, style StyleConfig) {
	var updated []DiffLine
	dmp := diffmatchpatch.New()

	for i := 0; i < len(h.Lines); i++ {
		// Look for removed line followed by added line, which might have similar content
		if i+1 < len(h.Lines) &&
			h.Lines[i].Kind == LineRemoved &&
			h.Lines[i+1].Kind == LineAdded {

			oldLine := h.Lines[i]
			newLine := h.Lines[i+1]

			// Find character-level differences
			patches := dmp.DiffMain(oldLine.Content, newLine.Content, false)
			patches = dmp.DiffCleanupEfficiency(patches)
			patches = dmp.DiffCleanupSemantic(patches)

			// Apply highlighting to the differences
			oldLine.Content = colorizeSegments(patches, true, style)
			newLine.Content = colorizeSegments(patches, false, style)

			updated = append(updated, oldLine, newLine)
			i++ // Skip the next line as we've already processed it
		} else {
			updated = append(updated, h.Lines[i])
		}
	}

	h.Lines = updated
}

// colorizeSegments applies styles to the character-level diff segments.
func colorizeSegments(diffs []diffmatchpatch.Diff, isOld bool, style StyleConfig) string {
	var buf strings.Builder

	removeBg := lipgloss.NewStyle().
		Background(style.RemovedHighlightBg).
		Foreground(style.RemovedHighlightFg)

	addBg := lipgloss.NewStyle().
		Background(style.AddedHighlightBg).
		Foreground(style.AddedHighlightFg)

	removedLineStyle := lipgloss.NewStyle().Background(style.RemovedLineBg)
	addedLineStyle := lipgloss.NewStyle().Background(style.AddedLineBg)

	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			// Handle text that's the same in both versions
			buf.WriteString(d.Text)
		case diffmatchpatch.DiffDelete:
			// Handle deleted text (only show in old version)
			if isOld {
				buf.WriteString(removeBg.Render(d.Text))
				buf.WriteString(removedLineStyle.Render(""))
			}
		case diffmatchpatch.DiffInsert:
			// Handle inserted text (only show in new version)
			if !isOld {
				buf.WriteString(addBg.Render(d.Text))
				buf.WriteString(addedLineStyle.Render(""))
			}
		}
	}

	return buf.String()
}

// pairLines converts a flat list of diff lines to pairs for side-by-side display.
func pairLines(lines []DiffLine) []linePair {
	var pairs []linePair
	i := 0

	for i < len(lines) {
		switch lines[i].Kind {
		case LineRemoved:
			// Check if the next line is an addition, if so pair them
			if i+1 < len(lines) && lines[i+1].Kind == LineAdded {
				pairs = append(pairs, linePair{left: &lines[i], right: &lines[i+1]})
				i += 2
			} else {
				pairs = append(pairs, linePair{left: &lines[i], right: nil})
				i++
			}
		case LineAdded:
			pairs = append(pairs, linePair{left: nil, right: &lines[i]})
			i++
		case LineContext:
			pairs = append(pairs, linePair{left: &lines[i], right: &lines[i]})
			i++
		}
	}

	return pairs
}

// -------------------------------------------------------------------------
// Syntax Highlighting
// -------------------------------------------------------------------------

// SyntaxHighlight applies syntax highlighting to a string based on the file extension.
func SyntaxHighlight(w io.Writer, source, fileName, formatter string, bg lipgloss.TerminalColor) error {
	// Determine the language lexer to use
	l := lexers.Match(fileName)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	// Get the formatter
	f := formatters.Get(formatter)
	if f == nil {
		f = formatters.Fallback
	}

	// Get the style
	s := styles.Get("dracula")
	if s == nil {
		s = styles.Fallback
	}

	// Modify the style to use the provided background
	s, err := s.Builder().Transform(
		func(t chroma.StyleEntry) chroma.StyleEntry {
			r, g, b, _ := bg.RGBA()
			ru8 := uint8(r >> 8)
			gu8 := uint8(g >> 8)
			bu8 := uint8(b >> 8)
			t.Background = chroma.NewColour(ru8, gu8, bu8)
			return t
		},
	).Build()
	if err != nil {
		s = styles.Fallback
	}

	// Tokenize and format
	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}

	return f.Format(w, s, it)
}

// highlightLine applies syntax highlighting to a single line.
func highlightLine(fileName string, line string, bg lipgloss.TerminalColor) string {
	var buf bytes.Buffer
	err := SyntaxHighlight(&buf, line, fileName, "terminal16m", bg)
	if err != nil {
		return line
	}
	return buf.String()
}

// createStyles generates the lipgloss styles needed for rendering diffs.
func createStyles(config StyleConfig) (removedLineStyle, addedLineStyle, contextLineStyle, lineNumberStyle lipgloss.Style) {
	removedLineStyle = lipgloss.NewStyle().Background(config.RemovedLineBg)
	addedLineStyle = lipgloss.NewStyle().Background(config.AddedLineBg)
	contextLineStyle = lipgloss.NewStyle().Background(config.ContextLineBg)
	lineNumberStyle = lipgloss.NewStyle().Foreground(config.LineNumberFg)

	return
}

// renderLeftColumn formats the left side of a side-by-side diff.
func renderLeftColumn(fileName string, dl *DiffLine, colWidth int, styles StyleConfig) string {
	if dl == nil {
		contextLineStyle := lipgloss.NewStyle().Background(styles.ContextLineBg)
		return contextLineStyle.Width(colWidth).Render("")
	}

	removedLineStyle, _, contextLineStyle, lineNumberStyle := createStyles(styles)

	var marker string
	var bgStyle lipgloss.Style

	switch dl.Kind {
	case LineRemoved:
		marker = removedLineStyle.Foreground(styles.RemovedFg).Render("-")
		bgStyle = removedLineStyle
		lineNumberStyle = lineNumberStyle.Foreground(styles.RemovedFg).Background(styles.RemovedLineNumberBg)
	case LineAdded:
		marker = "?"
		bgStyle = contextLineStyle
	case LineContext:
		marker = contextLineStyle.Render(" ")
		bgStyle = contextLineStyle
	}

	lineNum := ""
	if dl.OldLineNo > 0 {
		lineNum = fmt.Sprintf("%6d", dl.OldLineNo)
	}

	prefix := lineNumberStyle.Render(lineNum + " " + marker)
	content := highlightLine(fileName, dl.Content, bgStyle.GetBackground())

	if dl.Kind == LineRemoved {
		content = bgStyle.Render(" ") + content
	}

	lineText := prefix + content
	return bgStyle.MaxHeight(1).Width(colWidth).Render(
		ansi.Truncate(
			lineText,
			colWidth,
			lipgloss.NewStyle().Background(styles.HunkLineBg).Foreground(styles.HunkLineFg).Render("..."),
		),
	)
}

// renderRightColumn formats the right side of a side-by-side diff.
func renderRightColumn(fileName string, dl *DiffLine, colWidth int, styles StyleConfig) string {
	if dl == nil {
		contextLineStyle := lipgloss.NewStyle().Background(styles.ContextLineBg)
		return contextLineStyle.Width(colWidth).Render("")
	}

	_, addedLineStyle, contextLineStyle, lineNumberStyle := createStyles(styles)

	var marker string
	var bgStyle lipgloss.Style

	switch dl.Kind {
	case LineAdded:
		marker = addedLineStyle.Foreground(styles.AddedFg).Render("+")
		bgStyle = addedLineStyle
		lineNumberStyle = lineNumberStyle.Foreground(styles.AddedFg).Background(styles.AddedLineNamerBg)
	case LineRemoved:
		marker = "?"
		bgStyle = contextLineStyle
	case LineContext:
		marker = contextLineStyle.Render(" ")
		bgStyle = contextLineStyle
	}

	lineNum := ""
	if dl.NewLineNo > 0 {
		lineNum = fmt.Sprintf("%6d", dl.NewLineNo)
	}

	prefix := lineNumberStyle.Render(lineNum + " " + marker)
	content := highlightLine(fileName, dl.Content, bgStyle.GetBackground())

	if dl.Kind == LineAdded {
		content = bgStyle.Render(" ") + content
	}

	lineText := prefix + content
	return bgStyle.MaxHeight(1).Width(colWidth).Render(
		ansi.Truncate(
			lineText,
			colWidth,
			lipgloss.NewStyle().Background(styles.HunkLineBg).Foreground(styles.HunkLineFg).Render("..."),
		),
	)
}

// -------------------------------------------------------------------------
// Public API Methods
// -------------------------------------------------------------------------

// RenderSideBySideHunk formats a hunk for side-by-side display.
func RenderSideBySideHunk(fileName string, h Hunk, opts ...SideBySideOption) string {
	// Apply options to create the configuration
	config := NewSideBySideConfig(opts...)

	// Make a copy of the hunk so we don't modify the original
	hunkCopy := Hunk{Lines: make([]DiffLine, len(h.Lines))}
	copy(hunkCopy.Lines, h.Lines)

	// Highlight changes within lines
	HighlightIntralineChanges(&hunkCopy, config.Style)

	// Pair lines for side-by-side display
	pairs := pairLines(hunkCopy.Lines)

	// Calculate column width
	colWidth := config.TotalWidth / 2

	var sb strings.Builder
	for _, p := range pairs {
		leftStr := renderLeftColumn(fileName, p.left, colWidth, config.Style)
		rightStr := renderRightColumn(fileName, p.right, colWidth, config.Style)
		sb.WriteString(leftStr + rightStr + "\n")
	}

	return sb.String()
}

// FormatDiff creates a side-by-side formatted view of a diff.
func FormatDiff(diffText string, opts ...SideBySideOption) (string, error) {
	diffResult, err := ParseUnifiedDiff(diffText)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	config := NewSideBySideConfig(opts...)
	for i, h := range diffResult.Hunks {
		if i > 0 {
			sb.WriteString(lipgloss.NewStyle().Background(config.Style.HunkLineBg).Foreground(config.Style.HunkLineFg).Width(config.TotalWidth).Render(h.Header) + "\n")
		}
		sb.WriteString(RenderSideBySideHunk(diffResult.OldFile, h, opts...))
	}

	return sb.String(), nil
}

// GenerateDiff creates a unified diff from two file contents.
func GenerateDiff(beforeContent, afterContent, fileName string) (string, int, int) {
	tempDir, err := os.MkdirTemp("", "git-diff-temp")
	if err != nil {
		return "", 0, 0
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		return "", 0, 0
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", 0, 0
	}

	fullPath := filepath.Join(tempDir, fileName)
	if err = os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", 0, 0
	}
	if err = os.WriteFile(fullPath, []byte(beforeContent), 0o644); err != nil {
		return "", 0, 0
	}

	_, err = wt.Add(fileName)
	if err != nil {
		return "", 0, 0
	}

	beforeCommit, err := wt.Commit("Before", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "OpenCode",
			Email: "coder@opencode.ai",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", 0, 0
	}

	if err = os.WriteFile(fullPath, []byte(afterContent), 0o644); err != nil {
	}

	_, err = wt.Add(fileName)
	if err != nil {
		return "", 0, 0
	}

	afterCommit, err := wt.Commit("After", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "OpenCode",
			Email: "coder@opencode.ai",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", 0, 0
	}

	beforeCommitObj, err := repo.CommitObject(beforeCommit)
	if err != nil {
		return "", 0, 0
	}

	afterCommitObj, err := repo.CommitObject(afterCommit)
	if err != nil {
		return "", 0, 0
	}

	patch, err := beforeCommitObj.Patch(afterCommitObj)
	if err != nil {
		return "", 0, 0
	}

	additions := 0
	removals := 0
	for _, fileStat := range patch.Stats() {
		additions += fileStat.Addition
		removals += fileStat.Deletion
	}

	return patch.String(), additions, removals
}

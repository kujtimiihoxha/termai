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

// -------------------------------------------------------------------------
// Core Types
// -------------------------------------------------------------------------

// LineType represents the kind of line in a diff.
type LineType int

const (
	LineContext LineType = iota // Line exists in both files
	LineAdded                   // Line added in the new file
	LineRemoved                 // Line removed from the old file
)

// Segment represents a portion of a line for intra-line highlighting
type Segment struct {
	Start int
	End   int
	Type  LineType
	Text  string
}

// DiffLine represents a single line in a diff
type DiffLine struct {
	OldLineNo int       // Line number in old file (0 for added lines)
	NewLineNo int       // Line number in new file (0 for removed lines)
	Kind      LineType  // Type of line (added, removed, context)
	Content   string    // Content of the line
	Segments  []Segment // Segments for intraline highlighting
}

// Hunk represents a section of changes in a diff
type Hunk struct {
	Header string
	Lines  []DiffLine
}

// DiffResult contains the parsed result of a diff
type DiffResult struct {
	OldFile string
	NewFile string
	Hunks   []Hunk
}

// linePair represents a pair of lines for side-by-side display
type linePair struct {
	left  *DiffLine
	right *DiffLine
}

// -------------------------------------------------------------------------
// Style Configuration
// -------------------------------------------------------------------------

// StyleConfig defines styling for diff rendering
type StyleConfig struct {
	// Background colors
	RemovedLineBg       lipgloss.Color
	AddedLineBg         lipgloss.Color
	ContextLineBg       lipgloss.Color
	HunkLineBg          lipgloss.Color
	RemovedLineNumberBg lipgloss.Color
	AddedLineNamerBg    lipgloss.Color

	// Foreground colors
	HunkLineFg         lipgloss.Color
	RemovedFg          lipgloss.Color
	AddedFg            lipgloss.Color
	LineNumberFg       lipgloss.Color
	RemovedHighlightFg lipgloss.Color
	AddedHighlightFg   lipgloss.Color

	// Highlight settings
	HighlightStyle     string
	RemovedHighlightBg lipgloss.Color
	AddedHighlightBg   lipgloss.Color
}

// StyleOption is a function that modifies a StyleConfig
type StyleOption func(*StyleConfig)

// NewStyleConfig creates a StyleConfig with default values
func NewStyleConfig(opts ...StyleOption) StyleConfig {
	// Default color scheme
	config := StyleConfig{
		RemovedLineBg:       lipgloss.Color("#3A3030"),
		AddedLineBg:         lipgloss.Color("#303A30"),
		ContextLineBg:       lipgloss.Color("#212121"),
		HunkLineBg:          lipgloss.Color("#23252D"),
		HunkLineFg:          lipgloss.Color("#8CA3B4"),
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

// Style option functions
func WithRemovedLineBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.RemovedLineBg = color }
}

func WithAddedLineBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.AddedLineBg = color }
}

func WithContextLineBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.ContextLineBg = color }
}

func WithRemovedFg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.RemovedFg = color }
}

func WithAddedFg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.AddedFg = color }
}

func WithLineNumberFg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.LineNumberFg = color }
}

func WithHighlightStyle(style string) StyleOption {
	return func(s *StyleConfig) { s.HighlightStyle = style }
}

func WithRemovedHighlightColors(bg, fg lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.RemovedHighlightBg = bg
		s.RemovedHighlightFg = fg
	}
}

func WithAddedHighlightColors(bg, fg lipgloss.Color) StyleOption {
	return func(s *StyleConfig) {
		s.AddedHighlightBg = bg
		s.AddedHighlightFg = fg
	}
}

func WithRemovedLineNumberBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.RemovedLineNumberBg = color }
}

func WithAddedLineNumberBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.AddedLineNamerBg = color }
}

func WithHunkLineBg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.HunkLineBg = color }
}

func WithHunkLineFg(color lipgloss.Color) StyleOption {
	return func(s *StyleConfig) { s.HunkLineFg = color }
}

// -------------------------------------------------------------------------
// Parse Configuration
// -------------------------------------------------------------------------

// ParseConfig configures the behavior of diff parsing
type ParseConfig struct {
	ContextSize int // Number of context lines to include
}

// ParseOption modifies a ParseConfig
type ParseOption func(*ParseConfig)

// WithContextSize sets the number of context lines to include
func WithContextSize(size int) ParseOption {
	return func(p *ParseConfig) {
		if size >= 0 {
			p.ContextSize = size
		}
	}
}

// -------------------------------------------------------------------------
// Side-by-Side Configuration
// -------------------------------------------------------------------------

// SideBySideConfig configures the rendering of side-by-side diffs
type SideBySideConfig struct {
	TotalWidth int
	Style      StyleConfig
}

// SideBySideOption modifies a SideBySideConfig
type SideBySideOption func(*SideBySideConfig)

// NewSideBySideConfig creates a SideBySideConfig with default values
func NewSideBySideConfig(opts ...SideBySideOption) SideBySideConfig {
	config := SideBySideConfig{
		TotalWidth: 160, // Default width for side-by-side view
		Style:      NewStyleConfig(),
	}

	for _, opt := range opts {
		opt(&config)
	}

	return config
}

// WithTotalWidth sets the total width for side-by-side view
func WithTotalWidth(width int) SideBySideOption {
	return func(s *SideBySideConfig) {
		if width > 0 {
			s.TotalWidth = width
		}
	}
}

// WithStyle sets the styling configuration
func WithStyle(style StyleConfig) SideBySideOption {
	return func(s *SideBySideConfig) {
		s.Style = style
	}
}

// WithStyleOptions applies the specified style options
func WithStyleOptions(opts ...StyleOption) SideBySideOption {
	return func(s *SideBySideConfig) {
		s.Style = NewStyleConfig(opts...)
	}
}

// -------------------------------------------------------------------------
// Diff Parsing
// -------------------------------------------------------------------------

// ParseUnifiedDiff parses a unified diff format string into structured data
func ParseUnifiedDiff(diff string) (DiffResult, error) {
	var result DiffResult
	var currentHunk *Hunk

	hunkHeaderRe := regexp.MustCompile(`^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@`)
	lines := strings.Split(diff, "\n")

	var oldLine, newLine int
	inFileHeader := true

	for _, line := range lines {
		// Parse file headers
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

		// Ignore "No newline at end of file" markers
		if strings.HasPrefix(line, "\\ No newline at end of file") {
			continue
		}

		if currentHunk == nil {
			continue
		}

		// Process the line based on its prefix
		if len(line) > 0 {
			switch line[0] {
			case '+':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					OldLineNo: 0,
					NewLineNo: newLine,
					Kind:      LineAdded,
					Content:   line[1:],
				})
				newLine++
			case '-':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					OldLineNo: oldLine,
					NewLineNo: 0,
					Kind:      LineRemoved,
					Content:   line[1:],
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

// HighlightIntralineChanges updates lines in a hunk to show character-level differences
func HighlightIntralineChanges(h *Hunk, style StyleConfig) {
	var updated []DiffLine
	dmp := diffmatchpatch.New()

	for i := 0; i < len(h.Lines); i++ {
		// Look for removed line followed by added line
		if i+1 < len(h.Lines) &&
			h.Lines[i].Kind == LineRemoved &&
			h.Lines[i+1].Kind == LineAdded {

			oldLine := h.Lines[i]
			newLine := h.Lines[i+1]

			// Find character-level differences
			patches := dmp.DiffMain(oldLine.Content, newLine.Content, false)
			patches = dmp.DiffCleanupSemantic(patches)
			patches = dmp.DiffCleanupMerge(patches)
			patches = dmp.DiffCleanupEfficiency(patches)

			segments := make([]Segment, 0)

			removeStart := 0
			addStart := 0
			for _, patch := range patches {
				switch patch.Type {
				case diffmatchpatch.DiffDelete:
					segments = append(segments, Segment{
						Start: removeStart,
						End:   removeStart + len(patch.Text),
						Type:  LineRemoved,
						Text:  patch.Text,
					})
					removeStart += len(patch.Text)
				case diffmatchpatch.DiffInsert:
					segments = append(segments, Segment{
						Start: addStart,
						End:   addStart + len(patch.Text),
						Type:  LineAdded,
						Text:  patch.Text,
					})
					addStart += len(patch.Text)
				default:
					// Context text, no highlighting needed
					removeStart += len(patch.Text)
					addStart += len(patch.Text)
				}
			}
			oldLine.Segments = segments
			newLine.Segments = segments

			updated = append(updated, oldLine, newLine)
			i++ // Skip the next line as we've already processed it
		} else {
			updated = append(updated, h.Lines[i])
		}
	}

	h.Lines = updated
}

// pairLines converts a flat list of diff lines to pairs for side-by-side display
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

// SyntaxHighlight applies syntax highlighting to text based on file extension
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
	theme := `
	<style name="vscode-dark-plus">
	<!-- Base colors -->
	<entry type="Background" style="bg:#1E1E1E"/>
	<entry type="Text" style="#D4D4D4"/>
	<entry type="Other" style="#D4D4D4"/>
	<entry type="Error" style="#F44747"/>
	<!-- Keywords - using the Control flow / Special keywords color -->
	<entry type="Keyword" style="#C586C0"/>
	<entry type="KeywordConstant" style="#4FC1FF"/>
	<entry type="KeywordDeclaration" style="#C586C0"/>
	<entry type="KeywordNamespace" style="#C586C0"/>
	<entry type="KeywordPseudo" style="#C586C0"/>
	<entry type="KeywordReserved" style="#C586C0"/>
	<entry type="KeywordType" style="#4EC9B0"/>
	<!-- Names -->
	<entry type="Name" style="#D4D4D4"/>
	<entry type="NameAttribute" style="#9CDCFE"/>
	<entry type="NameBuiltin" style="#4EC9B0"/>
	<entry type="NameBuiltinPseudo" style="#9CDCFE"/>
	<entry type="NameClass" style="#4EC9B0"/>
	<entry type="NameConstant" style="#4FC1FF"/>
	<entry type="NameDecorator" style="#DCDCAA"/>
	<entry type="NameEntity" style="#9CDCFE"/>
	<entry type="NameException" style="#4EC9B0"/>
	<entry type="NameFunction" style="#DCDCAA"/>
	<entry type="NameLabel" style="#C8C8C8"/>
	<entry type="NameNamespace" style="#4EC9B0"/>
	<entry type="NameOther" style="#9CDCFE"/>
	<entry type="NameTag" style="#569CD6"/>
	<entry type="NameVariable" style="#9CDCFE"/>
	<entry type="NameVariableClass" style="#9CDCFE"/>
	<entry type="NameVariableGlobal" style="#9CDCFE"/>
	<entry type="NameVariableInstance" style="#9CDCFE"/>
	<!-- Literals -->
	<entry type="Literal" style="#CE9178"/>
	<entry type="LiteralDate" style="#CE9178"/>
	<entry type="LiteralString" style="#CE9178"/>
	<entry type="LiteralStringBacktick" style="#CE9178"/>
	<entry type="LiteralStringChar" style="#CE9178"/>
	<entry type="LiteralStringDoc" style="#CE9178"/>
	<entry type="LiteralStringDouble" style="#CE9178"/>
	<entry type="LiteralStringEscape" style="#d7ba7d"/>
	<entry type="LiteralStringHeredoc" style="#CE9178"/>
	<entry type="LiteralStringInterpol" style="#CE9178"/>
	<entry type="LiteralStringOther" style="#CE9178"/>
	<entry type="LiteralStringRegex" style="#d16969"/>
	<entry type="LiteralStringSingle" style="#CE9178"/>
	<entry type="LiteralStringSymbol" style="#CE9178"/>
	<!-- Numbers - using the numberLiteral color -->
	<entry type="LiteralNumber" style="#b5cea8"/>
	<entry type="LiteralNumberBin" style="#b5cea8"/>
	<entry type="LiteralNumberFloat" style="#b5cea8"/>
	<entry type="LiteralNumberHex" style="#b5cea8"/>
	<entry type="LiteralNumberInteger" style="#b5cea8"/>
	<entry type="LiteralNumberIntegerLong" style="#b5cea8"/>
	<entry type="LiteralNumberOct" style="#b5cea8"/>
	<!-- Operators -->
	<entry type="Operator" style="#D4D4D4"/>
	<entry type="OperatorWord" style="#C586C0"/>
	<entry type="Punctuation" style="#D4D4D4"/>
	<!-- Comments - standard VSCode Dark+ comment color -->
	<entry type="Comment" style="#6A9955"/>
	<entry type="CommentHashbang" style="#6A9955"/>
	<entry type="CommentMultiline" style="#6A9955"/>
	<entry type="CommentSingle" style="#6A9955"/>
	<entry type="CommentSpecial" style="#6A9955"/>
	<entry type="CommentPreproc" style="#C586C0"/>
	<!-- Generic styles -->
	<entry type="Generic" style="#D4D4D4"/>
	<entry type="GenericDeleted" style="#F44747"/>
	<entry type="GenericEmph" style="italic #D4D4D4"/>
	<entry type="GenericError" style="#F44747"/>
	<entry type="GenericHeading" style="bold #D4D4D4"/>
	<entry type="GenericInserted" style="#b5cea8"/>
	<entry type="GenericOutput" style="#808080"/>
	<entry type="GenericPrompt" style="#D4D4D4"/>
	<entry type="GenericStrong" style="bold #D4D4D4"/>
	<entry type="GenericSubheading" style="bold #D4D4D4"/>
	<entry type="GenericTraceback" style="#F44747"/>
	<entry type="GenericUnderline" style="underline"/>
	<entry type="TextWhitespace" style="#D4D4D4"/>
</style>
`

	r := strings.NewReader(theme)
	style := chroma.MustNewXMLStyle(r)
	// Modify the style to use the provided background
	s, err := style.Builder().Transform(
		func(t chroma.StyleEntry) chroma.StyleEntry {
			r, g, b, _ := bg.RGBA()
			t.Background = chroma.NewColour(uint8(r>>8), uint8(g>>8), uint8(b>>8))
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

// highlightLine applies syntax highlighting to a single line
func highlightLine(fileName string, line string, bg lipgloss.TerminalColor) string {
	var buf bytes.Buffer
	err := SyntaxHighlight(&buf, line, fileName, "terminal16m", bg)
	if err != nil {
		return line
	}
	return buf.String()
}

// createStyles generates the lipgloss styles needed for rendering diffs
func createStyles(config StyleConfig) (removedLineStyle, addedLineStyle, contextLineStyle, lineNumberStyle lipgloss.Style) {
	removedLineStyle = lipgloss.NewStyle().Background(config.RemovedLineBg)
	addedLineStyle = lipgloss.NewStyle().Background(config.AddedLineBg)
	contextLineStyle = lipgloss.NewStyle().Background(config.ContextLineBg)
	lineNumberStyle = lipgloss.NewStyle().Foreground(config.LineNumberFg)

	return
}

// -------------------------------------------------------------------------
// Rendering Functions
// -------------------------------------------------------------------------

// applyHighlighting applies intra-line highlighting to a piece of text
func applyHighlighting(content string, segments []Segment, segmentType LineType, highlightBg lipgloss.Color,
) string {
	// Find all ANSI sequences in the content
	ansiRegex := regexp.MustCompile(`\x1b(?:[@-Z\\-_]|\[[0-9?]*(?:;[0-9?]*)*[@-~])`)
	ansiMatches := ansiRegex.FindAllStringIndex(content, -1)

	// Build a mapping of visible character positions to their actual indices
	visibleIdx := 0
	ansiSequences := make(map[int]string)
	lastAnsiSeq := "\x1b[0m" // Default reset sequence

	for i := 0; i < len(content); {
		isAnsi := false
		for _, match := range ansiMatches {
			if match[0] == i {
				ansiSequences[visibleIdx] = content[match[0]:match[1]]
				lastAnsiSeq = content[match[0]:match[1]]
				i = match[1]
				isAnsi = true
				break
			}
		}
		if isAnsi {
			continue
		}

		// For non-ANSI positions, store the last ANSI sequence
		if _, exists := ansiSequences[visibleIdx]; !exists {
			ansiSequences[visibleIdx] = lastAnsiSeq
		}
		visibleIdx++
		i++
	}

	// Apply highlighting
	var sb strings.Builder
	inSelection := false
	currentPos := 0

	for i := 0; i < len(content); {
		// Check if we're at an ANSI sequence
		isAnsi := false
		for _, match := range ansiMatches {
			if match[0] == i {
				sb.WriteString(content[match[0]:match[1]]) // Preserve ANSI sequence
				i = match[1]
				isAnsi = true
				break
			}
		}
		if isAnsi {
			continue
		}

		// Check for segment boundaries
		for _, seg := range segments {
			if seg.Type == segmentType {
				if currentPos == seg.Start {
					inSelection = true
				}
				if currentPos == seg.End {
					inSelection = false
				}
			}
		}

		// Get current character
		char := string(content[i])

		if inSelection {
			// Get the current styling
			currentStyle := ansiSequences[currentPos]

			// Apply background highlight
			sb.WriteString("\x1b[48;2;")
			r, g, b, _ := highlightBg.RGBA()
			sb.WriteString(fmt.Sprintf("%d;%d;%dm", r>>8, g>>8, b>>8))
			sb.WriteString(char)
			sb.WriteString("\x1b[49m") // Reset only background

			// Reapply the original ANSI sequence
			sb.WriteString(currentStyle)
		} else {
			// Not in selection, just copy the character
			sb.WriteString(char)
		}

		currentPos++
		i++
	}

	return sb.String()
}

// renderLeftColumn formats the left side of a side-by-side diff
func renderLeftColumn(fileName string, dl *DiffLine, colWidth int, styles StyleConfig) string {
	if dl == nil {
		contextLineStyle := lipgloss.NewStyle().Background(styles.ContextLineBg)
		return contextLineStyle.Width(colWidth).Render("")
	}

	removedLineStyle, _, contextLineStyle, lineNumberStyle := createStyles(styles)

	// Determine line style based on line type
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

	// Format line number
	lineNum := ""
	if dl.OldLineNo > 0 {
		lineNum = fmt.Sprintf("%6d", dl.OldLineNo)
	}

	// Create the line prefix
	prefix := lineNumberStyle.Render(lineNum + " " + marker)

	// Apply syntax highlighting
	content := highlightLine(fileName, dl.Content, bgStyle.GetBackground())

	// Apply intra-line highlighting for removed lines
	if dl.Kind == LineRemoved && len(dl.Segments) > 0 {
		content = applyHighlighting(content, dl.Segments, LineRemoved, styles.RemovedHighlightBg)
	}

	// Add a padding space for removed lines
	if dl.Kind == LineRemoved {
		content = bgStyle.Render(" ") + content
	}

	// Create the final line and truncate if needed
	lineText := prefix + content
	return bgStyle.MaxHeight(1).Width(colWidth).Render(
		ansi.Truncate(
			lineText,
			colWidth,
			lipgloss.NewStyle().Background(styles.HunkLineBg).Foreground(styles.HunkLineFg).Render("..."),
		),
	)
}

// renderRightColumn formats the right side of a side-by-side diff
func renderRightColumn(fileName string, dl *DiffLine, colWidth int, styles StyleConfig) string {
	if dl == nil {
		contextLineStyle := lipgloss.NewStyle().Background(styles.ContextLineBg)
		return contextLineStyle.Width(colWidth).Render("")
	}

	_, addedLineStyle, contextLineStyle, lineNumberStyle := createStyles(styles)

	// Determine line style based on line type
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

	// Format line number
	lineNum := ""
	if dl.NewLineNo > 0 {
		lineNum = fmt.Sprintf("%6d", dl.NewLineNo)
	}

	// Create the line prefix
	prefix := lineNumberStyle.Render(lineNum + " " + marker)

	// Apply syntax highlighting
	content := highlightLine(fileName, dl.Content, bgStyle.GetBackground())

	// Apply intra-line highlighting for added lines
	if dl.Kind == LineAdded && len(dl.Segments) > 0 {
		content = applyHighlighting(content, dl.Segments, LineAdded, styles.AddedHighlightBg)
	}

	// Add a padding space for added lines
	if dl.Kind == LineAdded {
		content = bgStyle.Render(" ") + content
	}

	// Create the final line and truncate if needed
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
// Public API
// -------------------------------------------------------------------------

// RenderSideBySideHunk formats a hunk for side-by-side display
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

// FormatDiff creates a side-by-side formatted view of a diff
func FormatDiff(diffText string, opts ...SideBySideOption) (string, error) {
	diffResult, err := ParseUnifiedDiff(diffText)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	config := NewSideBySideConfig(opts...)

	for i, h := range diffResult.Hunks {
		if i > 0 {
			// Render hunk header
			sb.WriteString(
				lipgloss.NewStyle().
					Background(config.Style.HunkLineBg).
					Foreground(config.Style.HunkLineFg).
					Width(config.TotalWidth).
					Render(h.Header) + "\n",
			)
		}
		sb.WriteString(RenderSideBySideHunk(diffResult.OldFile, h, opts...))
	}

	return sb.String(), nil
}

// GenerateDiff creates a unified diff from two file contents
func GenerateDiff(beforeContent, afterContent, fileName string) (string, int, int) {
	// Create temporary directory for git operations
	tempDir, err := os.MkdirTemp("", "git-diff-temp")
	if err != nil {
		return "", 0, 0
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		return "", 0, 0
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", 0, 0
	}

	// Write the "before" content and commit it
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

	// Write the "after" content and commit it
	if err = os.WriteFile(fullPath, []byte(afterContent), 0o644); err != nil {
		return "", 0, 0
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

	// Get the diff between the two commits
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

	// Count additions and removals
	additions := 0
	removals := 0
	for _, fileStat := range patch.Stats() {
		additions += fileStat.Addition
		removals += fileStat.Deletion
	}

	return patch.String(), additions, removals
}

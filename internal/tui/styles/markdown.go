package styles

import (
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

const defaultMargin = 1

// Helper functions for style pointers
func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }

// CatppuccinMarkdownStyle is the Catppuccin Mocha style for Glamour markdown rendering.
func CatppuccinMarkdownStyle() ansi.StyleConfig {
	isDark := lipgloss.HasDarkBackground()
	if isDark {
		return catppuccinDark
	}
	return catppuccinLight
}

var catppuccinDark = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "\n",
			BlockSuffix: "",
			Color:       stringPtr(dark.Text().Hex),
		},
		Margin: uintPtr(defaultMargin),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr(dark.Yellow().Hex),
			Italic: boolPtr(true),
			Prefix: "┃ ",
		},
		Indent:      uintPtr(1),
		IndentToken: stringPtr(BaseStyle.Render(" ")),
	},
	List: ansi.StyleList{
		LevelIndent: defaultMargin,
		StyleBlock: ansi.StyleBlock{
			IndentToken: stringPtr(BaseStyle.Render(" ")),
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(dark.Text().Hex),
			},
		},
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       stringPtr(dark.Mauve().Hex),
			Bold:        boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:      "# ",
			Color:       stringPtr(dark.Lavender().Hex),
			Bold:        boolPtr(true),
			BlockPrefix: "\n",
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "## ",
			Color:  stringPtr(dark.Mauve().Hex),
			Bold:   boolPtr(true),
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "### ",
			Color:  stringPtr(dark.Pink().Hex),
			Bold:   boolPtr(true),
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "#### ",
			Color:  stringPtr(dark.Flamingo().Hex),
			Bold:   boolPtr(true),
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "##### ",
			Color:  stringPtr(dark.Rosewater().Hex),
			Bold:   boolPtr(true),
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "###### ",
			Color:  stringPtr(dark.Rosewater().Hex),
			Bold:   boolPtr(true),
		},
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: boolPtr(true),
		Color:      stringPtr(dark.Overlay1().Hex),
	},
	Emph: ansi.StylePrimitive{
		Color:  stringPtr(dark.Yellow().Hex),
		Italic: boolPtr(true),
	},
	Strong: ansi.StylePrimitive{
		Bold:  boolPtr(true),
		Color: stringPtr(dark.Peach().Hex),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  stringPtr(dark.Overlay0().Hex),
		Format: "\n─────────────────────────────────────────\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
		Color:       stringPtr(dark.Blue().Hex),
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
		Color:       stringPtr(dark.Sky().Hex),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{},
		Ticked:         "[✓] ",
		Unticked:       "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:     stringPtr(dark.Sky().Hex),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: stringPtr(dark.Pink().Hex),
		Bold:  boolPtr(true),
	},
	Image: ansi.StylePrimitive{
		Color:     stringPtr(dark.Sapphire().Hex),
		Underline: boolPtr(true),
		Format:    "🖼 {{.text}}",
	},
	ImageText: ansi.StylePrimitive{
		Color:  stringPtr(dark.Pink().Hex),
		Format: "{{.text}}",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr(dark.Green().Hex),
			Prefix: "",
			Suffix: "",
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: " ",
				Color:  stringPtr(dark.Text().Hex),
			},

			Margin: uintPtr(defaultMargin),
		},
		Chroma: &ansi.Chroma{
			Text: ansi.StylePrimitive{
				Color: stringPtr(dark.Text().Hex),
			},
			Error: ansi.StylePrimitive{
				Color: stringPtr(dark.Text().Hex),
			},
			Comment: ansi.StylePrimitive{
				Color: stringPtr(dark.Overlay1().Hex),
			},
			CommentPreproc: ansi.StylePrimitive{
				Color: stringPtr(dark.Pink().Hex),
			},
			Keyword: ansi.StylePrimitive{
				Color: stringPtr(dark.Pink().Hex),
			},
			KeywordReserved: ansi.StylePrimitive{
				Color: stringPtr(dark.Pink().Hex),
			},
			KeywordNamespace: ansi.StylePrimitive{
				Color: stringPtr(dark.Pink().Hex),
			},
			KeywordType: ansi.StylePrimitive{
				Color: stringPtr(dark.Sky().Hex),
			},
			Operator: ansi.StylePrimitive{
				Color: stringPtr(dark.Pink().Hex),
			},
			Punctuation: ansi.StylePrimitive{
				Color: stringPtr(dark.Text().Hex),
			},
			Name: ansi.StylePrimitive{
				Color: stringPtr(dark.Sky().Hex),
			},
			NameBuiltin: ansi.StylePrimitive{
				Color: stringPtr(dark.Sky().Hex),
			},
			NameTag: ansi.StylePrimitive{
				Color: stringPtr(dark.Pink().Hex),
			},
			NameAttribute: ansi.StylePrimitive{
				Color: stringPtr(dark.Green().Hex),
			},
			NameClass: ansi.StylePrimitive{
				Color: stringPtr(dark.Sky().Hex),
			},
			NameConstant: ansi.StylePrimitive{
				Color: stringPtr(dark.Mauve().Hex),
			},
			NameDecorator: ansi.StylePrimitive{
				Color: stringPtr(dark.Green().Hex),
			},
			NameFunction: ansi.StylePrimitive{
				Color: stringPtr(dark.Green().Hex),
			},
			LiteralNumber: ansi.StylePrimitive{
				Color: stringPtr(dark.Teal().Hex),
			},
			LiteralString: ansi.StylePrimitive{
				Color: stringPtr(dark.Yellow().Hex),
			},
			LiteralStringEscape: ansi.StylePrimitive{
				Color: stringPtr(dark.Pink().Hex),
			},
			GenericDeleted: ansi.StylePrimitive{
				Color: stringPtr(dark.Red().Hex),
			},
			GenericEmph: ansi.StylePrimitive{
				Color:  stringPtr(dark.Yellow().Hex),
				Italic: boolPtr(true),
			},
			GenericInserted: ansi.StylePrimitive{
				Color: stringPtr(dark.Green().Hex),
			},
			GenericStrong: ansi.StylePrimitive{
				Color: stringPtr(dark.Peach().Hex),
				Bold:  boolPtr(true),
			},
			GenericSubheading: ansi.StylePrimitive{
				Color: stringPtr(dark.Mauve().Hex),
			},
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
			},
		},
		CenterSeparator: stringPtr("┼"),
		ColumnSeparator: stringPtr("│"),
		RowSeparator:    stringPtr("─"),
	},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix: "\n ❯ ",
		Color:       stringPtr(dark.Sapphire().Hex),
	},
}

var catppuccinLight = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "\n",
			BlockSuffix: "\n",
			Color:       stringPtr(light.Text().Hex),
		},
		Margin: uintPtr(defaultMargin),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr(light.Yellow().Hex),
			Italic: boolPtr(true),
			Prefix: "┃ ",
		},
		Indent: uintPtr(1),
		Margin: uintPtr(defaultMargin),
	},
	List: ansi.StyleList{
		LevelIndent: defaultMargin,
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(light.Text().Hex),
			},
		},
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       stringPtr(light.Mauve().Hex),
			Bold:        boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:      "# ",
			Color:       stringPtr(light.Lavender().Hex),
			Bold:        boolPtr(true),
			BlockPrefix: "\n",
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "## ",
			Color:  stringPtr(light.Mauve().Hex),
			Bold:   boolPtr(true),
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "### ",
			Color:  stringPtr(light.Pink().Hex),
			Bold:   boolPtr(true),
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "#### ",
			Color:  stringPtr(light.Flamingo().Hex),
			Bold:   boolPtr(true),
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "##### ",
			Color:  stringPtr(light.Rosewater().Hex),
			Bold:   boolPtr(true),
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "###### ",
			Color:  stringPtr(light.Rosewater().Hex),
			Bold:   boolPtr(true),
		},
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: boolPtr(true),
		Color:      stringPtr(light.Overlay1().Hex),
	},
	Emph: ansi.StylePrimitive{
		Color:  stringPtr(light.Yellow().Hex),
		Italic: boolPtr(true),
	},
	Strong: ansi.StylePrimitive{
		Bold:  boolPtr(true),
		Color: stringPtr(light.Peach().Hex),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  stringPtr(light.Overlay0().Hex),
		Format: "\n─────────────────────────────────────────\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
		Color:       stringPtr(light.Blue().Hex),
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
		Color:       stringPtr(light.Sky().Hex),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{},
		Ticked:         "[✓] ",
		Unticked:       "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:     stringPtr(light.Sky().Hex),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: stringPtr(light.Pink().Hex),
		Bold:  boolPtr(true),
	},
	Image: ansi.StylePrimitive{
		Color:     stringPtr(light.Sapphire().Hex),
		Underline: boolPtr(true),
		Format:    "🖼 {{.text}}",
	},
	ImageText: ansi.StylePrimitive{
		Color:  stringPtr(light.Pink().Hex),
		Format: "{{.text}}",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  stringPtr(light.Green().Hex),
			Prefix: " ",
			Suffix: " ",
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "   ",
				Color:  stringPtr(light.Text().Hex),
			},

			Margin: uintPtr(defaultMargin),
		},
		Chroma: &ansi.Chroma{
			Text: ansi.StylePrimitive{
				Color: stringPtr(light.Text().Hex),
			},
			Error: ansi.StylePrimitive{
				Color: stringPtr(light.Text().Hex),
			},
			Comment: ansi.StylePrimitive{
				Color: stringPtr(light.Overlay1().Hex),
			},
			CommentPreproc: ansi.StylePrimitive{
				Color: stringPtr(light.Pink().Hex),
			},
			Keyword: ansi.StylePrimitive{
				Color: stringPtr(light.Pink().Hex),
			},
			KeywordReserved: ansi.StylePrimitive{
				Color: stringPtr(light.Pink().Hex),
			},
			KeywordNamespace: ansi.StylePrimitive{
				Color: stringPtr(light.Pink().Hex),
			},
			KeywordType: ansi.StylePrimitive{
				Color: stringPtr(light.Sky().Hex),
			},
			Operator: ansi.StylePrimitive{
				Color: stringPtr(light.Pink().Hex),
			},
			Punctuation: ansi.StylePrimitive{
				Color: stringPtr(light.Text().Hex),
			},
			Name: ansi.StylePrimitive{
				Color: stringPtr(light.Sky().Hex),
			},
			NameBuiltin: ansi.StylePrimitive{
				Color: stringPtr(light.Sky().Hex),
			},
			NameTag: ansi.StylePrimitive{
				Color: stringPtr(light.Pink().Hex),
			},
			NameAttribute: ansi.StylePrimitive{
				Color: stringPtr(light.Green().Hex),
			},
			NameClass: ansi.StylePrimitive{
				Color: stringPtr(light.Sky().Hex),
			},
			NameConstant: ansi.StylePrimitive{
				Color: stringPtr(light.Mauve().Hex),
			},
			NameDecorator: ansi.StylePrimitive{
				Color: stringPtr(light.Green().Hex),
			},
			NameFunction: ansi.StylePrimitive{
				Color: stringPtr(light.Green().Hex),
			},
			LiteralNumber: ansi.StylePrimitive{
				Color: stringPtr(light.Teal().Hex),
			},
			LiteralString: ansi.StylePrimitive{
				Color: stringPtr(light.Yellow().Hex),
			},
			LiteralStringEscape: ansi.StylePrimitive{
				Color: stringPtr(light.Pink().Hex),
			},
			GenericDeleted: ansi.StylePrimitive{
				Color: stringPtr(light.Red().Hex),
			},
			GenericEmph: ansi.StylePrimitive{
				Color:  stringPtr(light.Yellow().Hex),
				Italic: boolPtr(true),
			},
			GenericInserted: ansi.StylePrimitive{
				Color: stringPtr(light.Green().Hex),
			},
			GenericStrong: ansi.StylePrimitive{
				Color: stringPtr(light.Peach().Hex),
				Bold:  boolPtr(true),
			},
			GenericSubheading: ansi.StylePrimitive{
				Color: stringPtr(light.Mauve().Hex),
			},
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
			},
		},
		CenterSeparator: stringPtr("┼"),
		ColumnSeparator: stringPtr("│"),
		RowSeparator:    stringPtr("─"),
	},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix: "\n ❯ ",
		Color:       stringPtr(light.Sapphire().Hex),
	},
}

func MarkdownTheme(focused bool) ansi.StyleConfig {
	if !focused {
		return ASCIIStyleConfig
	} else {
		return DraculaStyleConfig
	}
}

const (
	defaultListIndent      = 2
	defaultListLevelIndent = 4
)

var ASCIIStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
		},
		Indent:      uintPtr(1),
		IndentToken: stringPtr(BaseStyle.Render(" ")),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
		},
		Indent:      uintPtr(1),
		IndentToken: stringPtr("| "),
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{
			IndentToken: stringPtr(BaseStyle.Render(" ")),
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
		},
		LevelIndent: defaultListLevelIndent,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
			BlockSuffix:     "\n",
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
			Prefix:          "# ",
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
			Prefix:          "## ",
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
			Prefix:          "### ",
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
			Prefix:          "#### ",
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
			Prefix:          "##### ",
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
			Prefix:          "###### ",
		},
	},
	Strikethrough: ansi.StylePrimitive{
		BackgroundColor: stringPtr(Background.Dark),
		BlockPrefix:     "~~",
		BlockSuffix:     "~~",
	},
	Emph: ansi.StylePrimitive{
		BackgroundColor: stringPtr(Background.Dark),
		BlockPrefix:     "*",
		BlockSuffix:     "*",
	},
	Strong: ansi.StylePrimitive{
		BackgroundColor: stringPtr(Background.Dark),
		BlockPrefix:     "**",
		BlockSuffix:     "**",
	},
	HorizontalRule: ansi.StylePrimitive{
		BackgroundColor: stringPtr(Background.Dark),
		Format:          "\n--------\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix:     "• ",
		BackgroundColor: stringPtr(Background.Dark),
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix:     ". ",
		BackgroundColor: stringPtr(Background.Dark),
	},
	Task: ansi.StyleTask{
		Ticked:   "[x] ",
		Unticked: "[ ] ",
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	ImageText: ansi.StylePrimitive{
		BackgroundColor: stringPtr(Background.Dark),
		Format:          "Image: {{.text}} →",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix:     "`",
			BlockSuffix:     "`",
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
			Margin: uintPtr(defaultMargin),
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
			IndentToken: stringPtr(BaseStyle.Render(" ")),
		},
		CenterSeparator: stringPtr("|"),
		ColumnSeparator: stringPtr("|"),
		RowSeparator:    stringPtr("-"),
	},
	DefinitionDescription: ansi.StylePrimitive{
		BackgroundColor: stringPtr(Background.Dark),
		BlockPrefix:     "\n* ",
	},
}

var DraculaStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr(Forground.Dark),
			BackgroundColor: stringPtr(Background.Dark),
		},
		Indent:      uintPtr(defaultMargin),
		IndentToken: stringPtr(BaseStyle.Render(" ")),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr("#f1fa8c"),
			Italic:          boolPtr(true),
			BackgroundColor: stringPtr(Background.Dark),
		},
		Indent:      uintPtr(defaultMargin),
		IndentToken: stringPtr(BaseStyle.Render(" ")),
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	List: ansi.StyleList{
		LevelIndent: defaultMargin,
		StyleBlock: ansi.StyleBlock{
			IndentToken: stringPtr(BaseStyle.Render(" ")),
			StylePrimitive: ansi.StylePrimitive{
				Color:           stringPtr(Forground.Dark),
				BackgroundColor: stringPtr(Background.Dark),
			},
		},
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix:     "\n",
			Color:           stringPtr("#bd93f9"),
			Bold:            boolPtr(true),
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "# ",
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "## ",
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "### ",
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "#### ",
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "##### ",
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "###### ",
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut:      boolPtr(true),
		BackgroundColor: stringPtr(Background.Dark),
	},
	Emph: ansi.StylePrimitive{
		Color:           stringPtr("#f1fa8c"),
		Italic:          boolPtr(true),
		BackgroundColor: stringPtr(Background.Dark),
	},
	Strong: ansi.StylePrimitive{
		Bold:            boolPtr(true),
		Color:           stringPtr("#ffb86c"),
		BackgroundColor: stringPtr(Background.Dark),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:           stringPtr("#6272A4"),
		Format:          "\n--------\n",
		BackgroundColor: stringPtr(Background.Dark),
	},
	Item: ansi.StylePrimitive{
		BlockPrefix:     "• ",
		BackgroundColor: stringPtr(Background.Dark),
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix:     ". ",
		Color:           stringPtr("#8be9fd"),
		BackgroundColor: stringPtr(Background.Dark),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(Background.Dark),
		},
		Ticked:   "[✓] ",
		Unticked: "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:           stringPtr("#8be9fd"),
		Underline:       boolPtr(true),
		BackgroundColor: stringPtr(Background.Dark),
	},
	LinkText: ansi.StylePrimitive{
		Color:           stringPtr("#ff79c6"),
		BackgroundColor: stringPtr(Background.Dark),
	},
	Image: ansi.StylePrimitive{
		Color:           stringPtr("#8be9fd"),
		Underline:       boolPtr(true),
		BackgroundColor: stringPtr(Background.Dark),
	},
	ImageText: ansi.StylePrimitive{
		Color:           stringPtr("#ff79c6"),
		Format:          "Image: {{.text}} →",
		BackgroundColor: stringPtr(Background.Dark),
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr("#50fa7b"),
			BackgroundColor: stringPtr(Background.Dark),
		},
	},
	Text: ansi.StylePrimitive{
		BackgroundColor: stringPtr(Background.Dark),
	},
	DefinitionList: ansi.StyleBlock{},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           stringPtr("#ffb86c"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			Margin: uintPtr(defaultMargin),
		},
		Chroma: &ansi.Chroma{
			NameOther: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
			Literal: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
			NameException: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
			LiteralDate: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
			Text: ansi.StylePrimitive{
				Color:           stringPtr(Forground.Dark),
				BackgroundColor: stringPtr(Background.Dark),
			},
			Error: ansi.StylePrimitive{
				Color:           stringPtr("#f8f8f2"),
				BackgroundColor: stringPtr("#ff5555"),
			},
			Comment: ansi.StylePrimitive{
				Color:           stringPtr("#6272A4"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			CommentPreproc: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			Keyword: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			KeywordReserved: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			KeywordNamespace: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			KeywordType: ansi.StylePrimitive{
				Color:           stringPtr("#8be9fd"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			Operator: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			Punctuation: ansi.StylePrimitive{
				Color:           stringPtr(Forground.Dark),
				BackgroundColor: stringPtr(Background.Dark),
			},
			Name: ansi.StylePrimitive{
				Color:           stringPtr("#8be9fd"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			NameBuiltin: ansi.StylePrimitive{
				Color:           stringPtr("#8be9fd"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			NameTag: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			NameAttribute: ansi.StylePrimitive{
				Color:           stringPtr("#50fa7b"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			NameClass: ansi.StylePrimitive{
				Color:           stringPtr("#8be9fd"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			NameConstant: ansi.StylePrimitive{
				Color:           stringPtr("#bd93f9"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			NameDecorator: ansi.StylePrimitive{
				Color:           stringPtr("#50fa7b"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			NameFunction: ansi.StylePrimitive{
				Color:           stringPtr("#50fa7b"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			LiteralNumber: ansi.StylePrimitive{
				Color:           stringPtr("#6EEFC0"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			LiteralString: ansi.StylePrimitive{
				Color:           stringPtr("#f1fa8c"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			LiteralStringEscape: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			GenericDeleted: ansi.StylePrimitive{
				Color:           stringPtr("#ff5555"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			GenericEmph: ansi.StylePrimitive{
				Color:           stringPtr("#f1fa8c"),
				Italic:          boolPtr(true),
				BackgroundColor: stringPtr(Background.Dark),
			},
			GenericInserted: ansi.StylePrimitive{
				Color:           stringPtr("#50fa7b"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			GenericStrong: ansi.StylePrimitive{
				Color:           stringPtr("#ffb86c"),
				Bold:            boolPtr(true),
				BackgroundColor: stringPtr(Background.Dark),
			},
			GenericSubheading: ansi.StylePrimitive{
				Color:           stringPtr("#bd93f9"),
				BackgroundColor: stringPtr(Background.Dark),
			},
			Background: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: stringPtr(Background.Dark),
			},
			IndentToken: stringPtr(BaseStyle.Render(" ")),
		},
	},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix:     "\n* ",
		BackgroundColor: stringPtr(Background.Dark),
	},
}

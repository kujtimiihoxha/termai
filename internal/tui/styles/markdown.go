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
		Indent: uintPtr(1),
		Margin: uintPtr(defaultMargin),
	},
	List: ansi.StyleList{
		LevelIndent: defaultMargin,
		StyleBlock: ansi.StyleBlock{
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

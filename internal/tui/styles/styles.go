package styles

import (
	catppuccin "github.com/catppuccin/go"
	"github.com/charmbracelet/lipgloss"
)

var (
	light = catppuccin.Latte
	dark  = catppuccin.Mocha
)

// NEW STYLES
var (
	Background = lipgloss.AdaptiveColor{
		Dark:  "#212121",
		Light: "#212121",
	}
	BackgroundDim = lipgloss.AdaptiveColor{
		Dark:  "#2c2c2c",
		Light: "#2c2c2c",
	}
	BackgroundDarker = lipgloss.AdaptiveColor{
		Dark:  "#181818",
		Light: "#181818",
	}
	BorderColor = lipgloss.AdaptiveColor{
		Dark:  "#4b4c5c",
		Light: "#4b4c5c",
	}

	Forground = lipgloss.AdaptiveColor{
		Dark:  "#d3d3d3",
		Light: "#d3d3d3",
	}

	ForgroundMid = lipgloss.AdaptiveColor{
		Dark:  "#a0a0a0",
		Light: "#a0a0a0",
	}

	ForgroundDim = lipgloss.AdaptiveColor{
		Dark:  "#737373",
		Light: "#737373",
	}

	BaseStyle = lipgloss.NewStyle().
			Background(Background).
			Foreground(Forground)

	PrimaryColor = lipgloss.AdaptiveColor{
		Dark:  "#fab283",
		Light: "#fab283",
	}
)

var (
	Regular = lipgloss.NewStyle()
	Bold    = Regular.Bold(true)
	Padded  = Regular.Padding(0, 1)

	Border       = Regular.Border(lipgloss.NormalBorder())
	ThickBorder  = Regular.Border(lipgloss.ThickBorder())
	DoubleBorder = Regular.Border(lipgloss.DoubleBorder())

	// Colors
	White    = lipgloss.Color("#ffffff")
	Surface0 = lipgloss.AdaptiveColor{
		Dark:  dark.Surface0().Hex,
		Light: light.Surface0().Hex,
	}

	Overlay0 = lipgloss.AdaptiveColor{
		Dark:  dark.Overlay0().Hex,
		Light: light.Overlay0().Hex,
	}

	Ovelay1 = lipgloss.AdaptiveColor{
		Dark:  dark.Overlay1().Hex,
		Light: light.Overlay1().Hex,
	}

	Text = lipgloss.AdaptiveColor{
		Dark:  dark.Text().Hex,
		Light: light.Text().Hex,
	}

	SubText0 = lipgloss.AdaptiveColor{
		Dark:  dark.Subtext0().Hex,
		Light: light.Subtext0().Hex,
	}

	SubText1 = lipgloss.AdaptiveColor{
		Dark:  dark.Subtext1().Hex,
		Light: light.Subtext1().Hex,
	}

	LightGrey = lipgloss.AdaptiveColor{
		Dark:  dark.Surface0().Hex,
		Light: light.Surface0().Hex,
	}
	Grey = lipgloss.AdaptiveColor{
		Dark:  dark.Surface1().Hex,
		Light: light.Surface1().Hex,
	}

	DarkGrey = lipgloss.AdaptiveColor{
		Dark:  dark.Surface2().Hex,
		Light: light.Surface2().Hex,
	}

	Base = lipgloss.AdaptiveColor{
		Dark:  dark.Base().Hex,
		Light: light.Base().Hex,
	}

	Crust = lipgloss.AdaptiveColor{
		Dark:  dark.Crust().Hex,
		Light: light.Crust().Hex,
	}

	Blue = lipgloss.AdaptiveColor{
		Dark:  dark.Blue().Hex,
		Light: light.Blue().Hex,
	}

	Red = lipgloss.AdaptiveColor{
		Dark:  dark.Red().Hex,
		Light: light.Red().Hex,
	}

	Green = lipgloss.AdaptiveColor{
		Dark:  dark.Green().Hex,
		Light: light.Green().Hex,
	}

	Mauve = lipgloss.AdaptiveColor{
		Dark:  dark.Mauve().Hex,
		Light: light.Mauve().Hex,
	}

	Teal = lipgloss.AdaptiveColor{
		Dark:  dark.Teal().Hex,
		Light: light.Teal().Hex,
	}

	Rosewater = lipgloss.AdaptiveColor{
		Dark:  dark.Rosewater().Hex,
		Light: light.Rosewater().Hex,
	}

	Flamingo = lipgloss.AdaptiveColor{
		Dark:  dark.Flamingo().Hex,
		Light: light.Flamingo().Hex,
	}

	Lavender = lipgloss.AdaptiveColor{
		Dark:  dark.Lavender().Hex,
		Light: light.Lavender().Hex,
	}

	Peach = lipgloss.AdaptiveColor{
		Dark:  dark.Peach().Hex,
		Light: light.Peach().Hex,
	}

	Yellow = lipgloss.AdaptiveColor{
		Dark:  dark.Yellow().Hex,
		Light: light.Yellow().Hex,
	}

	Primary   = Blue
	Secondary = Mauve

	Warning = Peach
	Error   = Red
)

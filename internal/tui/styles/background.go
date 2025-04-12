package styles

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

func getColorRGB(c lipgloss.TerminalColor) (uint8, uint8, uint8) {
	r, g, b, a := c.RGBA()

	// Un-premultiply alpha if needed
	if a > 0 && a < 0xffff {
		r = (r * 0xffff) / a
		g = (g * 0xffff) / a
		b = (b * 0xffff) / a
	}

	// Convert from 16-bit to 8-bit color
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

func ForceReplaceBackgroundWithLipgloss(input string, newBgColor lipgloss.TerminalColor) string {
	r, g, b := getColorRGB(newBgColor)

	newBg := fmt.Sprintf("48;2;%d;%d;%d", r, g, b)

	return ansiEscape.ReplaceAllStringFunc(input, func(seq string) string {
		// Extract content between "\x1b[" and "m"
		content := seq[2 : len(seq)-1]
		tokens := strings.Split(content, ";")
		var newTokens []string

		// Skip background color tokens
		for i := 0; i < len(tokens); i++ {
			if tokens[i] == "" {
				continue
			}

			val, err := strconv.Atoi(tokens[i])
			if err != nil {
				newTokens = append(newTokens, tokens[i])
				continue
			}

			// Skip background color tokens
			if val == 48 {
				// Skip "48;5;N" or "48;2;R;G;B" sequences
				if i+1 < len(tokens) {
					if nextVal, err := strconv.Atoi(tokens[i+1]); err == nil {
						switch nextVal {
						case 5:
							i += 2 // Skip "5" and color index
						case 2:
							i += 4 // Skip "2" and RGB components
						}
					}
				}
			} else if (val < 40 || val > 47) && (val < 100 || val > 107) && val != 49 {
				// Keep non-background tokens
				newTokens = append(newTokens, tokens[i])
			}
		}

		// Add new background if provided
		if newBg != "" {
			newTokens = append(newTokens, strings.Split(newBg, ";")...)
		}

		if len(newTokens) == 0 {
			return ""
		}

		return "\x1b[" + strings.Join(newTokens, ";") + "m"
	})
}

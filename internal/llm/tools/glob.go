package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

type globTool struct {
	workingDir string
}

const (
	GlobToolName = "glob"
)

type fileInfo struct {
	path    string
	modTime time.Time
}

type GlobParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

func (g *globTool) Info() ToolInfo {
	return ToolInfo{
		Name:        GlobToolName,
		Description: globDescription(),
		Parameters: map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The glob pattern to match files against",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The directory to search in. Defaults to the current working directory.",
			},
		},
		Required: []string{"pattern"},
	}
}

// Run implements Tool.
func (g *globTool) Run(ctx context.Context, args string) (ToolResponse, error) {
	var params GlobParams
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	if params.Pattern == "" {
		return NewTextErrorResponse("pattern is required"), nil
	}

	// If path is empty, use current working directory
	searchPath := params.Path
	if searchPath == "" {
		searchPath = g.workingDir
	}

	files, truncated, err := globFiles(params.Pattern, searchPath, 100)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error performing glob search: %s", err)), nil
	}

	// Format the output for the assistant
	var output string
	if len(files) == 0 {
		output = "No files found"
	} else {
		output = strings.Join(files, "\n")
		if truncated {
			output += "\n\n(Results are truncated. Consider using a more specific path or pattern.)"
		}
	}

	return NewTextResponse(output), nil
}

func globFiles(pattern, searchPath string, limit int) ([]string, bool, error) {
	// Make sure pattern starts with the search path if not absolute
	if !strings.HasPrefix(pattern, "/") && !strings.HasPrefix(pattern, searchPath) {
		// If searchPath doesn't end with a slash, add one before appending the pattern
		if !strings.HasSuffix(searchPath, "/") {
			searchPath += "/"
		}
		pattern = searchPath + pattern
	}

	// Open the filesystem for walking
	fsys := os.DirFS("/")

	// Convert the absolute pattern to a relative one for the DirFS
	// DirFS uses the root directory ("/") so we should strip leading "/"
	relPattern := strings.TrimPrefix(pattern, "/")

	// Collect matching files
	var matches []fileInfo

	// Use doublestar to walk the filesystem and find matches
	err := doublestar.GlobWalk(fsys, relPattern, func(path string, d fs.DirEntry) error {
		// Skip directories from results
		if d.IsDir() {
			return nil
		}
		if skipHidden(path) {
			return nil
		}

		// Get file info for modification time
		info, err := d.Info()
		if err != nil {
			return nil // Skip files we can't access
		}

		// Add to matches
		absPath := "/" + path // Restore absolute path
		matches = append(matches, fileInfo{
			path:    absPath,
			modTime: info.ModTime(),
		})

		// Check limit
		if len(matches) >= limit*2 { // Collect more than needed for sorting
			return fs.SkipAll
		}

		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("glob walk error: %w", err)
	}

	// Sort files by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	// Check if we need to truncate the results
	truncated := len(matches) > limit
	if truncated {
		matches = matches[:limit]
	}

	// Extract just the paths
	results := make([]string, len(matches))
	for i, m := range matches {
		results[i] = m.path
	}

	return results, truncated, nil
}

func skipHidden(path string) bool {
	base := filepath.Base(path)
	return base != "." && strings.HasPrefix(base, ".")
}

func globDescription() string {
	return `Fast file pattern matching tool that finds files by name and pattern, returning matching paths sorted by modification time (newest first).

WHEN TO USE THIS TOOL:
- Use when you need to find files by name patterns or extensions
- Great for finding specific file types across a directory structure
- Useful for discovering files that match certain naming conventions

HOW TO USE:
- Provide a glob pattern to match against file paths
- Optionally specify a starting directory (defaults to current working directory)
- Results are sorted with most recently modified files first

GLOB PATTERN SYNTAX:
- '*' matches any sequence of non-separator characters
- '**' matches any sequence of characters, including separators
- '?' matches any single non-separator character
- '[...]' matches any character in the brackets
- '[!...]' matches any character not in the brackets

COMMON PATTERN EXAMPLES:
- '*.js' - Find all JavaScript files in the current directory
- '**/*.js' - Find all JavaScript files in any subdirectory
- 'src/**/*.{ts,tsx}' - Find all TypeScript files in the src directory
- '*.{html,css,js}' - Find all HTML, CSS, and JS files

LIMITATIONS:
- Results are limited to 100 files (newest first)
- Does not search file contents (use Grep tool for that)
- Hidden files (starting with '.') are skipped

TIPS:
- For the most useful results, combine with the Grep tool: first find files with Glob, then search their contents with Grep
- When doing iterative exploration that may require multiple rounds of searching, consider using the Agent tool instead
- Always check if results are truncated and refine your search pattern if needed`
}

func NewGlobTool(workingDir string) BaseTool {
	return &globTool{
		workingDir: workingDir,
	}
}

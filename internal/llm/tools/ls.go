package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type lsTool struct {
	workingDir string
}

const (
	LSToolName = "ls"
	MaxLSFiles = 1000
)

type LSParams struct {
	Path   string   `json:"path"`
	Ignore []string `json:"ignore"`
}

type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"` // "file" or "directory"
	Children []TreeNode `json:"children,omitempty"`
}

func (l *lsTool) Info() ToolInfo {
	return ToolInfo{
		Name:        LSToolName,
		Description: lsDescription(),
		Parameters: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the directory to list (defaults to current working directory)",
			},
			"ignore": map[string]any{
				"type":        "array",
				"description": "List of glob patterns to ignore",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		Required: []string{"path"},
	}
}

// Run implements Tool.
func (l *lsTool) Run(ctx context.Context, args string) (ToolResponse, error) {
	var params LSParams
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	// If path is empty, use current working directory
	searchPath := params.Path
	if searchPath == "" {
		searchPath = l.workingDir
	}

	// Ensure the path is absolute
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(l.workingDir, searchPath)
	}

	// Check if the path exists
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return NewTextErrorResponse(fmt.Sprintf("path does not exist: %s", searchPath)), nil
	}

	files, truncated, err := listDirectory(searchPath, l.workingDir, params.Ignore, MaxLSFiles)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error listing directory: %s", err)), nil
	}

	tree := createFileTree(files)
	output := printTree(tree, searchPath)

	if truncated {
		output = fmt.Sprintf("There are more than %d files in the directory. Use a more specific path or use the Glob tool to find specific files. The first %d files and directories are included below:\n\n%s", MaxLSFiles, MaxLSFiles, output)
	}

	return NewTextResponse(output), nil
}

func listDirectory(initialPath, workingDir string, ignorePatterns []string, limit int) ([]string, bool, error) {
	var results []string
	truncated := false

	err := filepath.Walk(initialPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we don't have permission to access
		}

		if shouldSkip(path, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if path != initialPath {
			if info.IsDir() {
				path = path + string(filepath.Separator)
			}

			relPath, err := filepath.Rel(workingDir, path)
			if err == nil {
				results = append(results, relPath)
			} else {
				results = append(results, path)
			}
		}

		if len(results) >= limit {
			truncated = true
			return filepath.SkipAll
		}

		return nil
	})
	if err != nil {
		return nil, truncated, err
	}

	return results, truncated, nil
}

func shouldSkip(path string, ignorePatterns []string) bool {
	base := filepath.Base(path)

	// Skip hidden files and directories
	if base != "." && strings.HasPrefix(base, ".") {
		return true
	}

	// Skip __pycache__ directories
	if strings.Contains(path, filepath.Join("__pycache__", "")) {
		return true
	}

	// Check against ignore patterns
	for _, pattern := range ignorePatterns {
		matched, err := filepath.Match(pattern, base)
		if err == nil && matched {
			return true
		}
	}

	return false
}

func createFileTree(sortedPaths []string) []TreeNode {
	root := []TreeNode{}

	for _, path := range sortedPaths {
		parts := strings.Split(path, string(filepath.Separator))
		currentLevel := &root
		currentPath := ""

		for i, part := range parts {
			if part == "" {
				continue
			}

			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			isLastPart := i == len(parts)-1
			isDir := !isLastPart || strings.HasSuffix(path, string(filepath.Separator))

			found := false
			for i := range *currentLevel {
				if (*currentLevel)[i].Name == part {
					found = true
					if (*currentLevel)[i].Children != nil {
						currentLevel = &(*currentLevel)[i].Children
					}
					break
				}
			}

			if !found {
				nodeType := "file"
				if isDir {
					nodeType = "directory"
				}

				newNode := TreeNode{
					Name: part,
					Path: currentPath,
					Type: nodeType,
				}

				if isDir {
					newNode.Children = []TreeNode{}
					*currentLevel = append(*currentLevel, newNode)
					currentLevel = &(*currentLevel)[len(*currentLevel)-1].Children
				} else {
					*currentLevel = append(*currentLevel, newNode)
				}
			}
		}
	}

	return root
}

func printTree(tree []TreeNode, rootPath string) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("- %s%s\n", rootPath, string(filepath.Separator)))

	printTreeRecursive(&result, tree, 0, "  ")

	return result.String()
}

func printTreeRecursive(builder *strings.Builder, tree []TreeNode, level int, prefix string) {
	for _, node := range tree {
		linePrefix := prefix + "- "

		nodeName := node.Name
		if node.Type == "directory" {
			nodeName += string(filepath.Separator)
		}
		fmt.Fprintf(builder, "%s%s\n", linePrefix, nodeName)

		if node.Type == "directory" && len(node.Children) > 0 {
			printTreeRecursive(builder, node.Children, level+1, prefix+"  ")
		}
	}
}

func lsDescription() string {
	return `Directory listing tool that shows files and subdirectories in a tree structure, helping you explore and understand the project organization.

WHEN TO USE THIS TOOL:
- Use when you need to explore the structure of a directory
- Helpful for understanding the organization of a project
- Good first step when getting familiar with a new codebase

HOW TO USE:
- Provide a path to list (defaults to current working directory)
- Optionally specify glob patterns to ignore
- Results are displayed in a tree structure

FEATURES:
- Displays a hierarchical view of files and directories
- Automatically skips hidden files/directories (starting with '.')
- Skips common system directories like __pycache__
- Can filter out files matching specific patterns

LIMITATIONS:
- Results are limited to 1000 files
- Very large directories will be truncated
- Does not show file sizes or permissions
- Cannot recursively list all directories in a large project

TIPS:
- Use Glob tool for finding files by name patterns instead of browsing
- Use Grep tool for searching file contents
- Combine with other tools for more effective exploration`
}

func NewLsTool(workingDir string) BaseTool {
	return &lsTool{
		workingDir: workingDir,
	}
}

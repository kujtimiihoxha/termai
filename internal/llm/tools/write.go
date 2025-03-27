package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kujtimiihoxha/termai/internal/permission"
)

type writeTool struct {
	workingDir string
}

const (
	WriteToolName = "write"
)

type WriteParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type WritePermissionsParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (w *writeTool) Info() ToolInfo {
	return ToolInfo{
		Name:        WriteToolName,
		Description: writeDescription(),
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		Required: []string{"file_path", "content"},
	}
}

// Run implements Tool.
func (w *writeTool) Run(ctx context.Context, args string) (ToolResponse, error) {
	var params WriteParams
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	if params.Content == "" {
		return NewTextErrorResponse("content is required"), nil
	}

	// Handle relative paths
	filePath := params.FilePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(w.workingDir, filePath)
	}

	// Check if file exists and is a directory
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if fileInfo.IsDir() {
			return NewTextErrorResponse(fmt.Sprintf("Path is a directory, not a file: %s", filePath)), nil
		}

		// Check if file was modified since last read
		modTime := fileInfo.ModTime()
		lastRead := getLastReadTime(filePath)
		if modTime.After(lastRead) {
			return NewTextErrorResponse(fmt.Sprintf("File %s has been modified since it was last read.\nLast modification: %s\nLast read: %s\n\nPlease read the file again before modifying it.",
				filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339))), nil
		}

		// Optional: Get old content for diff
		oldContent, readErr := os.ReadFile(filePath)
		if readErr == nil && string(oldContent) == params.Content {
			return NewTextErrorResponse(fmt.Sprintf("File %s already contains the exact content. No changes made.", filePath)), nil
		}
	} else if !os.IsNotExist(err) {
		return NewTextErrorResponse(fmt.Sprintf("Failed to access file: %s", err)), nil
	}

	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to create parent directories: %s", err)), nil
	}
	p := permission.Default.Request(
		permission.CreatePermissionRequest{
			Path:        filepath.Dir(filePath),
			ToolName:    WriteToolName,
			Action:      "create",
			Description: fmt.Sprintf("Create file %s", filePath),
			Params: WritePermissionsParams{
				FilePath: filePath,
				Content:  GenerateDiff("", params.Content),
			},
		},
	)
	if !p {
		return NewTextErrorResponse(fmt.Sprintf("Permission denied to create file: %s", filePath)), nil
	}

	// Write the file
	err = os.WriteFile(filePath, []byte(params.Content), 0o644)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to write file: %s", err)), nil
	}

	// Record the file write
	recordFileWrite(filePath)
	recordFileRead(filePath)

	return NewTextResponse(fmt.Sprintf("File successfully written: %s", filePath)), nil
}

func writeDescription() string {
	return `File writing tool that creates or updates files in the filesystem, allowing you to save or modify text content.

WHEN TO USE THIS TOOL:
- Use when you need to create a new file
- Helpful for updating existing files with modified content
- Perfect for saving generated code, configurations, or text data

HOW TO USE:
- Provide the path to the file you want to write
- Include the content to be written to the file
- The tool will create any necessary parent directories

FEATURES:
- Can create new files or overwrite existing ones
- Creates parent directories automatically if they don't exist
- Checks if the file has been modified since last read for safety
- Avoids unnecessary writes when content hasn't changed

LIMITATIONS:
- You should read a file before writing to it to avoid conflicts
- Cannot append to files (rewrites the entire file)


TIPS:
- Use the View tool first to examine existing files before modifying them
- Use the LS tool to verify the correct location when creating new files
- Combine with Glob and Grep tools to find and modify multiple files
- Always include descriptive comments when making changes to existing code`
}

func NewWriteTool(workingDir string) BaseTool {
	return &writeTool{
		workingDir: workingDir,
	}
}

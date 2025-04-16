package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kujtimiihoxha/opencode/internal/config"
	"github.com/kujtimiihoxha/opencode/internal/diff"
	"github.com/kujtimiihoxha/opencode/internal/history"
	"github.com/kujtimiihoxha/opencode/internal/lsp"
	"github.com/kujtimiihoxha/opencode/internal/permission"
)

type PatchParams struct {
	FilePath string `json:"file_path"`
	Patch    string `json:"patch"`
}

type PatchPermissionsParams struct {
	FilePath string `json:"file_path"`
	Diff     string `json:"diff"`
}

type PatchResponseMetadata struct {
	Diff      string `json:"diff"`
	Additions int    `json:"additions"`
	Removals  int    `json:"removals"`
}

type patchTool struct {
	lspClients  map[string]*lsp.Client
	permissions permission.Service
	files       history.Service
}

const (
	// TODO: test if this works as expected
	PatchToolName    = "patch"
	patchDescription = `Applies a patch to a file. This tool is similar to the edit tool but accepts a unified diff patch instead of old/new strings.

Before using this tool:

1. Use the FileRead tool to understand the file's contents and context

2. Verify the directory path is correct:
   - Use the LS tool to verify the parent directory exists and is the correct location

To apply a patch, provide the following:
1. file_path: The absolute path to the file to modify (must be absolute, not relative)
2. patch: A unified diff patch to apply to the file

The tool will apply the patch to the specified file. The patch must be in unified diff format.

CRITICAL REQUIREMENTS FOR USING THIS TOOL:

1. PATCH FORMAT: The patch must be in unified diff format, which includes:
   - File headers (--- a/file_path, +++ b/file_path)
   - Hunk headers (@@ -start,count +start,count @@)
   - Added lines (prefixed with +)
   - Removed lines (prefixed with -)

2. CONTEXT: The patch must include sufficient context around the changes to ensure it applies correctly.

3. VERIFICATION: Before using this tool:
   - Ensure the patch applies cleanly to the current state of the file
   - Check that the file exists and you have read it first

WARNING: If you do not follow these requirements:
   - The tool will fail if the patch doesn't apply cleanly
   - You may change the wrong parts of the file if the context is insufficient

When applying patches:
   - Ensure the patch results in idiomatic, correct code
   - Do not leave the code in a broken state
   - Always use absolute file paths (starting with /)

Remember: patches are a powerful way to make multiple related changes at once, but they require careful preparation.`
)

func NewPatchTool(lspClients map[string]*lsp.Client, permissions permission.Service, files history.Service) BaseTool {
	return &patchTool{
		lspClients:  lspClients,
		permissions: permissions,
		files:       files,
	}
}

func (p *patchTool) Info() ToolInfo {
	return ToolInfo{
		Name:        PatchToolName,
		Description: patchDescription,
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"patch": map[string]any{
				"type":        "string",
				"description": "The unified diff patch to apply",
			},
		},
		Required: []string{"file_path", "patch"},
	}
}

func (p *patchTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params PatchParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	if params.Patch == "" {
		return NewTextErrorResponse("patch is required"), nil
	}

	if !filepath.IsAbs(params.FilePath) {
		wd := config.WorkingDirectory()
		params.FilePath = filepath.Join(wd, params.FilePath)
	}

	// Check if file exists
	fileInfo, err := os.Stat(params.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTextErrorResponse(fmt.Sprintf("file not found: %s", params.FilePath)), nil
		}
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", params.FilePath)), nil
	}

	if getLastReadTime(params.FilePath).IsZero() {
		return NewTextErrorResponse("you must read the file before patching it. Use the View tool first"), nil
	}

	modTime := fileInfo.ModTime()
	lastRead := getLastReadTime(params.FilePath)
	if modTime.After(lastRead) {
		return NewTextErrorResponse(
			fmt.Sprintf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
				params.FilePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
			)), nil
	}

	// Read the current file content
	content, err := os.ReadFile(params.FilePath)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)

	// Parse and apply the patch
	diffResult, err := diff.ParseUnifiedDiff(params.Patch)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to parse patch: %v", err)), nil
	}

	// Apply the patch to get the new content
	newContent, err := applyPatch(oldContent, diffResult)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to apply patch: %v", err)), nil
	}

	if oldContent == newContent {
		return NewTextErrorResponse("patch did not result in any changes to the file"), nil
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for patching a file")
	}

	// Generate a diff for permission request and metadata
	diffText, additions, removals := diff.GenerateDiff(
		oldContent,
		newContent,
		params.FilePath,
	)

	// Request permission to apply the patch
	p.permissions.Request(
		permission.CreatePermissionRequest{
			Path:        filepath.Dir(params.FilePath),
			ToolName:    PatchToolName,
			Action:      "patch",
			Description: fmt.Sprintf("Apply patch to file %s", params.FilePath),
			Params: PatchPermissionsParams{
				FilePath: params.FilePath,
				Diff:     diffText,
			},
		},
	)

	// Write the new content to the file
	err = os.WriteFile(params.FilePath, []byte(newContent), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	// Update file history
	file, err := p.files.GetByPathAndSession(ctx, params.FilePath, sessionID)
	if err != nil {
		_, err = p.files.Create(ctx, sessionID, params.FilePath, oldContent)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("error creating file history: %w", err)
		}
	}
	if file.Content != oldContent {
		// User manually changed the content, store an intermediate version
		_, err = p.files.CreateVersion(ctx, sessionID, params.FilePath, oldContent)
		if err != nil {
			fmt.Printf("Error creating file history version: %v\n", err)
		}
	}
	// Store the new version
	_, err = p.files.CreateVersion(ctx, sessionID, params.FilePath, newContent)
	if err != nil {
		fmt.Printf("Error creating file history version: %v\n", err)
	}

	recordFileWrite(params.FilePath)
	recordFileRead(params.FilePath)

	// Wait for LSP diagnostics and include them in the response
	waitForLspDiagnostics(ctx, params.FilePath, p.lspClients)
	text := fmt.Sprintf("<r>\nPatch applied to file: %s\n</r>\n", params.FilePath)
	text += getDiagnostics(params.FilePath, p.lspClients)

	return WithResponseMetadata(
		NewTextResponse(text),
		PatchResponseMetadata{
			Diff:      diffText,
			Additions: additions,
			Removals:  removals,
		}), nil
}

// applyPatch applies a parsed diff to a string and returns the resulting content
func applyPatch(content string, diffResult diff.DiffResult) (string, error) {
	lines := strings.Split(content, "\n")

	// Process each hunk in the diff
	for _, hunk := range diffResult.Hunks {
		// Parse the hunk header to get line numbers
		var oldStart, oldCount, newStart, newCount int
		_, err := fmt.Sscanf(hunk.Header, "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount)
		if err != nil {
			// Try alternative format with single line counts
			_, err = fmt.Sscanf(hunk.Header, "@@ -%d +%d @@", &oldStart, &newStart)
			if err != nil {
				return "", fmt.Errorf("invalid hunk header format: %s", hunk.Header)
			}
			oldCount = 1
			newCount = 1
		}

		// Adjust for 0-based array indexing
		oldStart--
		newStart--

		// Apply the changes
		newLines := make([]string, 0)
		newLines = append(newLines, lines[:oldStart]...)

		// Process the hunk lines in order
		currentOldLine := oldStart
		for _, line := range hunk.Lines {
			switch line.Kind {
			case diff.LineContext:
				newLines = append(newLines, line.Content)
				currentOldLine++
			case diff.LineRemoved:
				// Skip this line in the output (it's being removed)
				currentOldLine++
			case diff.LineAdded:
				// Add the new line
				newLines = append(newLines, line.Content)
			}
		}

		// Append the rest of the file
		newLines = append(newLines, lines[currentOldLine:]...)
		lines = newLines
	}

	return strings.Join(lines, "\n"), nil
}


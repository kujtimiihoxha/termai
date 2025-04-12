package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/git"
	"github.com/kujtimiihoxha/termai/internal/lsp"
	"github.com/kujtimiihoxha/termai/internal/permission"
)

type EditParams struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

type EditPermissionsParams struct {
	FilePath string `json:"file_path"`
	Diff     string `json:"diff"`
}

type EditResponseMetadata struct {
	Additions int `json:"additions"`
	Removals  int `json:"removals"`
}

type editTool struct {
	lspClients  map[string]*lsp.Client
	permissions permission.Service
}

const (
	EditToolName    = "edit"
	editDescription = `Edits files by replacing text, creating new files, or deleting content. For moving or renaming files, use the Bash tool with the 'mv' command instead. For larger file edits, use the FileWrite tool to overwrite files.

Before using this tool:

1. Use the FileRead tool to understand the file's contents and context

2. Verify the directory path is correct (only applicable when creating new files):
   - Use the LS tool to verify the parent directory exists and is the correct location

To make a file edit, provide the following:
1. file_path: The absolute path to the file to modify (must be absolute, not relative)
2. old_string: The text to replace (must be unique within the file, and must match the file contents exactly, including all whitespace and indentation)
3. new_string: The edited text to replace the old_string

Special cases:
- To create a new file: provide file_path and new_string, leave old_string empty
- To delete content: provide file_path and old_string, leave new_string empty

The tool will replace ONE occurrence of old_string with new_string in the specified file.

CRITICAL REQUIREMENTS FOR USING THIS TOOL:

1. UNIQUENESS: The old_string MUST uniquely identify the specific instance you want to change. This means:
   - Include AT LEAST 3-5 lines of context BEFORE the change point
   - Include AT LEAST 3-5 lines of context AFTER the change point
   - Include all whitespace, indentation, and surrounding code exactly as it appears in the file

2. SINGLE INSTANCE: This tool can only change ONE instance at a time. If you need to change multiple instances:
   - Make separate calls to this tool for each instance
   - Each call must uniquely identify its specific instance using extensive context

3. VERIFICATION: Before using this tool:
   - Check how many instances of the target text exist in the file
   - If multiple instances exist, gather enough context to uniquely identify each one
   - Plan separate tool calls for each instance

WARNING: If you do not follow these requirements:
   - The tool will fail if old_string matches multiple locations
   - The tool will fail if old_string doesn't match exactly (including whitespace)
   - You may change the wrong instance if you don't include enough context

When making edits:
   - Ensure the edit results in idiomatic, correct code
   - Do not leave the code in a broken state
   - Always use absolute file paths (starting with /)

Remember: when making multiple file edits in a row to the same file, you should prefer to send all edits in a single message with multiple calls to this tool, rather than multiple messages with a single call each.`
)

func NewEditTool(lspClients map[string]*lsp.Client, permissions permission.Service) BaseTool {
	return &editTool{
		lspClients:  lspClients,
		permissions: permissions,
	}
}

func (e *editTool) Info() ToolInfo {
	return ToolInfo{
		Name:        EditToolName,
		Description: editDescription,
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The text to replace",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The text to replace it with",
			},
		},
		Required: []string{"file_path", "old_string", "new_string"},
	}
}

func (e *editTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params EditParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	if !filepath.IsAbs(params.FilePath) {
		wd := config.WorkingDirectory()
		params.FilePath = filepath.Join(wd, params.FilePath)
	}

	if params.OldString == "" {
		result, err := e.createNewFile(ctx, params.FilePath, params.NewString)
		if err != nil {
			return NewTextErrorResponse(fmt.Sprintf("error creating file: %s", err)), nil
		}
		return WithResponseMetadata(NewTextResponse(result.text), EditResponseMetadata{
			Additions: result.additions,
			Removals:  result.removals,
		}), nil
	}

	if params.NewString == "" {
		result, err := e.deleteContent(ctx, params.FilePath, params.OldString)
		if err != nil {
			return NewTextErrorResponse(fmt.Sprintf("error deleting content: %s", err)), nil
		}
		return WithResponseMetadata(NewTextResponse(result.text), EditResponseMetadata{
			Additions: result.additions,
			Removals:  result.removals,
		}), nil
	}

	result, err := e.replaceContent(ctx, params.FilePath, params.OldString, params.NewString)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error replacing content: %s", err)), nil
	}

	waitForLspDiagnostics(ctx, params.FilePath, e.lspClients)
	text := fmt.Sprintf("<result>\n%s\n</result>\n", result.text)
	text += appendDiagnostics(params.FilePath, e.lspClients)
	return WithResponseMetadata(NewTextResponse(text), EditResponseMetadata{
		Additions: result.additions,
		Removals:  result.removals,
	}), nil
}

type editResponse struct {
	text      string
	additions int
	removals  int
}

func (e *editTool) createNewFile(ctx context.Context, filePath, content string) (editResponse, error) {
	er := editResponse{}
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if fileInfo.IsDir() {
			return er, fmt.Errorf("path is a directory, not a file: %s", filePath)
		}
		return er, fmt.Errorf("file already exists: %s. Use the Replace tool to overwrite an existing file", filePath)
	} else if !os.IsNotExist(err) {
		return er, fmt.Errorf("failed to access file: %w", err)
	}

	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return er, fmt.Errorf("failed to create parent directories: %w", err)
	}

	sessionID, messageID := getContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return er, fmt.Errorf("session ID and message ID are required for creating a new file")
	}

	diff, stats, err := git.GenerateGitDiffWithStats(
		removeWorkingDirectoryPrefix(filePath),
		"",
		content,
	)
	if err != nil {
		return er, fmt.Errorf("failed to get file diff: %w", err)
	}
	p := e.permissions.Request(
		permission.CreatePermissionRequest{
			Path:        filepath.Dir(filePath),
			ToolName:    EditToolName,
			Action:      "create",
			Description: fmt.Sprintf("Create file %s", filePath),
			Params: EditPermissionsParams{
				FilePath: filePath,
				Diff:     diff,
			},
		},
	)
	if !p {
		return er, fmt.Errorf("permission denied")
	}

	err = os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		return er, fmt.Errorf("failed to write file: %w", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)

	er.text = "File created: " + filePath
	er.additions = stats.Additions
	er.removals = stats.Removals
	return er, nil
}

func (e *editTool) deleteContent(ctx context.Context, filePath, oldString string) (editResponse, error) {
	er := editResponse{}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return er, fmt.Errorf("file not found: %s", filePath)
		}
		return er, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return er, fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	if getLastReadTime(filePath).IsZero() {
		return er, fmt.Errorf("you must read the file before editing it. Use the View tool first")
	}

	modTime := fileInfo.ModTime()
	lastRead := getLastReadTime(filePath)
	if modTime.After(lastRead) {
		return er, fmt.Errorf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
			filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339))
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return er, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)

	index := strings.Index(oldContent, oldString)
	if index == -1 {
		return er, fmt.Errorf("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks")
	}

	lastIndex := strings.LastIndex(oldContent, oldString)
	if index != lastIndex {
		return er, fmt.Errorf("old_string appears multiple times in the file. Please provide more context to ensure a unique match")
	}

	newContent := oldContent[:index] + oldContent[index+len(oldString):]

	sessionID, messageID := getContextValues(ctx)

	if sessionID == "" || messageID == "" {
		return er, fmt.Errorf("session ID and message ID are required for creating a new file")
	}

	diff, stats, err := git.GenerateGitDiffWithStats(
		removeWorkingDirectoryPrefix(filePath),
		oldContent,
		newContent,
	)
	if err != nil {
		return er, fmt.Errorf("failed to get file diff: %w", err)
	}

	p := e.permissions.Request(
		permission.CreatePermissionRequest{
			Path:        filepath.Dir(filePath),
			ToolName:    EditToolName,
			Action:      "delete",
			Description: fmt.Sprintf("Delete content from file %s", filePath),
			Params: EditPermissionsParams{
				FilePath: filePath,
				Diff:     diff,
			},
		},
	)
	if !p {
		return er, fmt.Errorf("permission denied")
	}

	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return er, fmt.Errorf("failed to write file: %w", err)
	}
	recordFileWrite(filePath)
	recordFileRead(filePath)

	er.text = "Content deleted from file: " + filePath
	er.additions = stats.Additions
	er.removals = stats.Removals
	return er, nil
}

func (e *editTool) replaceContent(ctx context.Context, filePath, oldString, newString string) (editResponse, error) {
	er := editResponse{}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return er, fmt.Errorf("file not found: %s", filePath)
		}
		return er, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return er, fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	if getLastReadTime(filePath).IsZero() {
		return er, fmt.Errorf("you must read the file before editing it. Use the View tool first")
	}

	modTime := fileInfo.ModTime()
	lastRead := getLastReadTime(filePath)
	if modTime.After(lastRead) {
		return er, fmt.Errorf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
			filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339))
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return er, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)

	index := strings.Index(oldContent, oldString)
	if index == -1 {
		return er, fmt.Errorf("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks")
	}

	lastIndex := strings.LastIndex(oldContent, oldString)
	if index != lastIndex {
		return er, fmt.Errorf("old_string appears multiple times in the file. Please provide more context to ensure a unique match")
	}

	newContent := oldContent[:index] + newString + oldContent[index+len(oldString):]

	sessionID, messageID := getContextValues(ctx)

	if sessionID == "" || messageID == "" {
		return er, fmt.Errorf("session ID and message ID are required for creating a new file")
	}
	diff, stats, err := git.GenerateGitDiffWithStats(
		removeWorkingDirectoryPrefix(filePath),
		oldContent,
		newContent,
	)
	if err != nil {
		return er, fmt.Errorf("failed to get file diff: %w", err)
	}

	p := e.permissions.Request(
		permission.CreatePermissionRequest{
			Path:        filepath.Dir(filePath),
			ToolName:    EditToolName,
			Action:      "replace",
			Description: fmt.Sprintf("Replace content in file %s", filePath),
			Params: EditPermissionsParams{
				FilePath: filePath,

				Diff: diff,
			},
		},
	)
	if !p {
		return er, fmt.Errorf("permission denied")
	}

	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return er, fmt.Errorf("failed to write file: %w", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)
	er.text = "Content replaced in file: " + filePath
	er.additions = stats.Additions
	er.removals = stats.Removals

	return er, nil
}


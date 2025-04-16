package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kujtimiihoxha/opencode/internal/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditTool_Info(t *testing.T) {
	tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())
	info := tool.Info()

	assert.Equal(t, EditToolName, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Parameters, "file_path")
	assert.Contains(t, info.Parameters, "old_string")
	assert.Contains(t, info.Parameters, "new_string")
	assert.Contains(t, info.Required, "file_path")
	assert.Contains(t, info.Required, "old_string")
	assert.Contains(t, info.Required, "new_string")
}

func TestEditTool_Run(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "edit_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("creates a new file successfully", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		filePath := filepath.Join(tempDir, "new_file.txt")
		content := "This is a test content"

		params := EditParams{
			FilePath:  filePath,
			OldString: "",
			NewString: content,
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "File created")

		// Verify file was created with correct content
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, string(fileContent))
	})

	t.Run("creates file with nested directories", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		filePath := filepath.Join(tempDir, "nested/dirs/new_file.txt")
		content := "Content in nested directory"

		params := EditParams{
			FilePath:  filePath,
			OldString: "",
			NewString: content,
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "File created")

		// Verify file was created with correct content
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, string(fileContent))
	})

	t.Run("fails to create file that already exists", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file first
		filePath := filepath.Join(tempDir, "existing_file.txt")
		initialContent := "Initial content"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Try to create the same file
		params := EditParams{
			FilePath:  filePath,
			OldString: "",
			NewString: "New content",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "file already exists")
	})

	t.Run("fails to create file when path is a directory", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a directory
		dirPath := filepath.Join(tempDir, "test_dir")
		err := os.Mkdir(dirPath, 0o755)
		require.NoError(t, err)

		// Try to create a file with the same path as the directory
		params := EditParams{
			FilePath:  dirPath,
			OldString: "",
			NewString: "Some content",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "path is a directory")
	})

	t.Run("replaces content successfully", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file first
		filePath := filepath.Join(tempDir, "replace_content.txt")
		initialContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Record the file read to avoid modification time check failure
		recordFileRead(filePath)

		// Replace content
		oldString := "Line 2\nLine 3"
		newString := "Line 2 modified\nLine 3 modified"
		params := EditParams{
			FilePath:  filePath,
			OldString: oldString,
			NewString: newString,
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "Content replaced")

		// Verify file was updated with correct content
		expectedContent := "Line 1\nLine 2 modified\nLine 3 modified\nLine 4\nLine 5"
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(fileContent))
	})

	t.Run("deletes content successfully", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file first
		filePath := filepath.Join(tempDir, "delete_content.txt")
		initialContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Record the file read to avoid modification time check failure
		recordFileRead(filePath)

		// Delete content
		oldString := "Line 2\nLine 3\n"
		params := EditParams{
			FilePath:  filePath,
			OldString: oldString,
			NewString: "",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "Content deleted")

		// Verify file was updated with correct content
		expectedContent := "Line 1\nLine 4\nLine 5"
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(fileContent))
	})

	t.Run("handles invalid parameters", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		call := ToolCall{
			Name:  EditToolName,
			Input: "invalid json",
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "invalid parameters")
	})

	t.Run("handles missing file_path", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		params := EditParams{
			FilePath:  "",
			OldString: "old",
			NewString: "new",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "file_path is required")
	})

	t.Run("handles file not found", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		filePath := filepath.Join(tempDir, "non_existent_file.txt")
		params := EditParams{
			FilePath:  filePath,
			OldString: "old content",
			NewString: "new content",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "file not found")
	})

	t.Run("handles old_string not found in file", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file first
		filePath := filepath.Join(tempDir, "content_not_found.txt")
		initialContent := "Line 1\nLine 2\nLine 3"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Record the file read to avoid modification time check failure
		recordFileRead(filePath)

		// Try to replace content that doesn't exist
		params := EditParams{
			FilePath:  filePath,
			OldString: "This content does not exist",
			NewString: "new content",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "old_string not found in file")
	})

	t.Run("handles multiple occurrences of old_string", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file with duplicate content
		filePath := filepath.Join(tempDir, "duplicate_content.txt")
		initialContent := "Line 1\nDuplicate\nLine 3\nDuplicate\nLine 5"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Record the file read to avoid modification time check failure
		recordFileRead(filePath)

		// Try to replace content that appears multiple times
		params := EditParams{
			FilePath:  filePath,
			OldString: "Duplicate",
			NewString: "Replaced",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "appears multiple times")
	})

	t.Run("handles file modified since last read", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file
		filePath := filepath.Join(tempDir, "modified_file.txt")
		initialContent := "Initial content"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Record an old read time
		fileRecordMutex.Lock()
		fileRecords[filePath] = fileRecord{
			path:     filePath,
			readTime: time.Now().Add(-1 * time.Hour),
		}
		fileRecordMutex.Unlock()

		// Try to update the file
		params := EditParams{
			FilePath:  filePath,
			OldString: "Initial",
			NewString: "Updated",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "has been modified since it was last read")

		// Verify file was not modified
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, initialContent, string(fileContent))
	})

	t.Run("handles file not read before editing", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file
		filePath := filepath.Join(tempDir, "not_read_file.txt")
		initialContent := "Initial content"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Try to update the file without reading it first
		params := EditParams{
			FilePath:  filePath,
			OldString: "Initial",
			NewString: "Updated",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "you must read the file before editing it")
	})

	t.Run("handles permission denied", func(t *testing.T) {
		tool := NewEditTool(make(map[string]*lsp.Client), newMockPermissionService(false), newMockFileHistoryService())

		// Create a file
		filePath := filepath.Join(tempDir, "permission_denied.txt")
		initialContent := "Initial content"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Record the file read to avoid modification time check failure
		recordFileRead(filePath)

		// Try to update the file
		params := EditParams{
			FilePath:  filePath,
			OldString: "Initial",
			NewString: "Updated",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  EditToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "permission denied")

		// Verify file was not modified
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, initialContent, string(fileContent))
	})
}

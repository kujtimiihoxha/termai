package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kujtimiihoxha/termai/internal/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTool_Info(t *testing.T) {
	tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())
	info := tool.Info()

	assert.Equal(t, WriteToolName, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Parameters, "file_path")
	assert.Contains(t, info.Parameters, "content")
	assert.Contains(t, info.Required, "file_path")
	assert.Contains(t, info.Required, "content")
}

func TestWriteTool_Run(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "write_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("creates a new file successfully", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		filePath := filepath.Join(tempDir, "new_file.txt")
		content := "This is a test content"

		params := WriteParams{
			FilePath: filePath,
			Content:  content,
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "successfully written")

		// Verify file was created with correct content
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, string(fileContent))
	})

	t.Run("creates file with nested directories", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		filePath := filepath.Join(tempDir, "nested/dirs/new_file.txt")
		content := "Content in nested directory"

		params := WriteParams{
			FilePath: filePath,
			Content:  content,
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "successfully written")

		// Verify file was created with correct content
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, string(fileContent))
	})

	t.Run("updates existing file", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file first
		filePath := filepath.Join(tempDir, "existing_file.txt")
		initialContent := "Initial content"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		// Record the file read to avoid modification time check failure
		recordFileRead(filePath)

		// Update the file
		updatedContent := "Updated content"
		params := WriteParams{
			FilePath: filePath,
			Content:  updatedContent,
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "successfully written")

		// Verify file was updated with correct content
		fileContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, updatedContent, string(fileContent))
	})

	t.Run("handles invalid parameters", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		call := ToolCall{
			Name:  WriteToolName,
			Input: "invalid json",
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "error parsing parameters")
	})

	t.Run("handles missing file_path", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		params := WriteParams{
			FilePath: "",
			Content:  "Some content",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "file_path is required")
	})

	t.Run("handles missing content", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		params := WriteParams{
			FilePath: filepath.Join(tempDir, "file.txt"),
			Content:  "",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "content is required")
	})

	t.Run("handles writing to a directory path", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a directory
		dirPath := filepath.Join(tempDir, "test_dir")
		err := os.Mkdir(dirPath, 0o755)
		require.NoError(t, err)

		params := WriteParams{
			FilePath: dirPath,
			Content:  "Some content",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "Path is a directory")
	})

	t.Run("handles permission denied", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(false), newMockFileHistoryService())

		filePath := filepath.Join(tempDir, "permission_denied.txt")
		params := WriteParams{
			FilePath: filePath,
			Content:  "Content that should not be written",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "Permission denied")

		// Verify file was not created
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("detects file modified since last read", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

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
		params := WriteParams{
			FilePath: filePath,
			Content:  "Updated content",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
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

	t.Run("skips writing when content is identical", func(t *testing.T) {
		tool := NewWriteTool(make(map[string]*lsp.Client), newMockPermissionService(true), newMockFileHistoryService())

		// Create a file
		filePath := filepath.Join(tempDir, "identical_content.txt")
		content := "Content that won't change"
		err := os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)

		// Record a read time
		recordFileRead(filePath)

		// Try to write the same content
		params := WriteParams{
			FilePath: filePath,
			Content:  content,
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			Name:  WriteToolName,
			Input: string(paramsJSON),
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.Contains(t, response.Content, "already contains the exact content")
	})
}

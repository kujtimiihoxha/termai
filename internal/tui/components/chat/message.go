package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kujtimiihoxha/opencode/internal/config"
	"github.com/kujtimiihoxha/opencode/internal/diff"
	"github.com/kujtimiihoxha/opencode/internal/llm/agent"
	"github.com/kujtimiihoxha/opencode/internal/llm/models"
	"github.com/kujtimiihoxha/opencode/internal/llm/tools"
	"github.com/kujtimiihoxha/opencode/internal/message"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
)

type uiMessageType int

const (
	userMessageType uiMessageType = iota
	assistantMessageType
	toolMessageType

	maxResultHeight = 15
)

var diffStyle = diff.NewStyleConfig(diff.WithShowHeader(false), diff.WithShowHunkHeader(false))

type uiMessage struct {
	ID          string
	messageType uiMessageType
	position    int
	height      int
	content     string
}

type renderCache struct {
	mutex sync.Mutex
	cache map[string][]uiMessage
}

func toMarkdown(content string, focused bool, width int) string {
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.MarkdownTheme(false)),
		glamour.WithWordWrap(width),
	)
	if focused {
		r, _ = glamour.NewTermRenderer(
			glamour.WithStyles(styles.MarkdownTheme(true)),
			glamour.WithWordWrap(width),
		)
	}
	rendered, _ := r.Render(content)
	return rendered
}

func renderMessage(msg string, isUser bool, isFocused bool, width int, info ...string) string {
	style := styles.BaseStyle.
		Width(width - 1).
		BorderLeft(true).
		Foreground(styles.ForgroundDim).
		BorderForeground(styles.PrimaryColor).
		BorderStyle(lipgloss.ThickBorder())
	if isUser {
		style = style.
			BorderForeground(styles.Blue)
	}
	parts := []string{
		styles.ForceReplaceBackgroundWithLipgloss(toMarkdown(msg, isFocused, width), styles.Background),
	}

	// remove newline at the end
	parts[0] = strings.TrimSuffix(parts[0], "\n")
	if len(info) > 0 {
		parts = append(parts, info...)
	}
	rendered := style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		),
	)

	return rendered
}

func renderUserMessage(msg message.Message, isFocused bool, width int, position int) uiMessage {
	content := renderMessage(msg.Content().String(), true, isFocused, width)
	userMsg := uiMessage{
		ID:          msg.ID,
		messageType: userMessageType,
		position:    position,
		height:      lipgloss.Height(content),
		content:     content,
	}
	return userMsg
}

// Returns multiple uiMessages because of the tool calls
func renderAssistantMessage(
	msg message.Message,
	msgIndex int,
	allMessages []message.Message, // we need this to get tool results and the user message
	messagesService message.Service, // We need this to get the task tool messages
	focusedUIMessageId string,
	width int,
	position int,
) []uiMessage {
	// find the user message that is before this assistant message
	var userMsg message.Message
	for i := msgIndex - 1; i >= 0; i-- {
		msg := allMessages[i]
		if msg.Role == message.User {
			userMsg = allMessages[i]
			break
		}
	}

	messages := []uiMessage{}
	content := msg.Content().String()
	finished := msg.IsFinished()
	finishData := msg.FinishPart()
	info := []string{}

	// Add finish info if available
	if finished {
		switch finishData.Reason {
		case message.FinishReasonEndTurn:
			took := formatTimeDifference(userMsg.CreatedAt, finishData.Time)
			info = append(info, styles.BaseStyle.Width(width-1).Foreground(styles.ForgroundDim).Render(
				fmt.Sprintf(" %s (%s)", models.SupportedModels[msg.Model].Name, took),
			))
		case message.FinishReasonCanceled:
			info = append(info, styles.BaseStyle.Width(width-1).Foreground(styles.ForgroundDim).Render(
				fmt.Sprintf(" %s (%s)", models.SupportedModels[msg.Model].Name, "canceled"),
			))
		case message.FinishReasonError:
			info = append(info, styles.BaseStyle.Width(width-1).Foreground(styles.ForgroundDim).Render(
				fmt.Sprintf(" %s (%s)", models.SupportedModels[msg.Model].Name, "error"),
			))
		case message.FinishReasonPermissionDenied:
			info = append(info, styles.BaseStyle.Width(width-1).Foreground(styles.ForgroundDim).Render(
				fmt.Sprintf(" %s (%s)", models.SupportedModels[msg.Model].Name, "permission denied"),
			))
		}
	}
	if content != "" {
		content = renderMessage(content, false, msg.ID == focusedUIMessageId, width, info...)
		messages = append(messages, uiMessage{
			ID:          msg.ID,
			messageType: assistantMessageType,
			position:    position,
			height:      lipgloss.Height(content),
			content:     content,
		})
		position += messages[0].height
		position++ // for the space
	}

	for i, toolCall := range msg.ToolCalls() {
		toolCallContent := renderToolMessage(
			toolCall,
			allMessages,
			messagesService,
			focusedUIMessageId,
			false,
			width,
			i+1,
		)
		messages = append(messages, toolCallContent)
		position += toolCallContent.height
		position++ // for the space
	}
	return messages
}

func findToolResponse(toolCallID string, futureMessages []message.Message) *message.ToolResult {
	for _, msg := range futureMessages {
		for _, result := range msg.ToolResults() {
			if result.ToolCallID == toolCallID {
				return &result
			}
		}
	}
	return nil
}

func toolName(name string) string {
	switch name {
	case agent.AgentToolName:
		return "Task"
	case tools.BashToolName:
		return "Bash"
	case tools.EditToolName:
		return "Edit"
	case tools.FetchToolName:
		return "Fetch"
	case tools.GlobToolName:
		return "Glob"
	case tools.GrepToolName:
		return "Grep"
	case tools.LSToolName:
		return "List"
	case tools.SourcegraphToolName:
		return "Sourcegraph"
	case tools.ViewToolName:
		return "View"
	case tools.WriteToolName:
		return "Write"
	}
	return name
}

// renders params, params[0] (params[1]=params[2] ....)
func renderParams(paramsWidth int, params ...string) string {
	if len(params) == 0 {
		return ""
	}
	mainParam := params[0]
	if len(mainParam) > paramsWidth {
		mainParam = mainParam[:paramsWidth-3] + "..."
	}

	if len(params) == 1 {
		return mainParam
	}
	otherParams := params[1:]
	// create pairs of key/value
	// if odd number of params, the last one is a key without value
	if len(otherParams)%2 != 0 {
		otherParams = append(otherParams, "")
	}
	parts := make([]string, 0, len(otherParams)/2)
	for i := 0; i < len(otherParams); i += 2 {
		key := otherParams[i]
		value := otherParams[i+1]
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	partsRendered := strings.Join(parts, ", ")
	remainingWidth := paramsWidth - lipgloss.Width(partsRendered) - 5 // for the space
	if remainingWidth < 30 {
		// No space for the params, just show the main
		return mainParam
	}

	if len(parts) > 0 {
		mainParam = fmt.Sprintf("%s (%s)", mainParam, strings.Join(parts, ", "))
	}

	return ansi.Truncate(mainParam, paramsWidth, "...")
}

func removeWorkingDirPrefix(path string) string {
	wd := config.WorkingDirectory()
	if strings.HasPrefix(path, wd) {
		path = strings.TrimPrefix(path, wd)
	}
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	if strings.HasPrefix(path, "./") {
		path = strings.TrimPrefix(path, "./")
	}
	if strings.HasPrefix(path, "../") {
		path = strings.TrimPrefix(path, "../")
	}
	return path
}

func renderToolParams(paramWidth int, toolCall message.ToolCall) string {
	params := ""
	switch toolCall.Name {
	case agent.AgentToolName:
		var params agent.AgentParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		prompt := strings.ReplaceAll(params.Prompt, "\n", " ")
		return renderParams(paramWidth, prompt)
	case tools.BashToolName:
		var params tools.BashParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		command := strings.ReplaceAll(params.Command, "\n", " ")
		return renderParams(paramWidth, command)
	case tools.EditToolName:
		var params tools.EditParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		filePath := removeWorkingDirPrefix(params.FilePath)
		return renderParams(paramWidth, filePath)
	case tools.FetchToolName:
		var params tools.FetchParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		url := params.URL
		toolParams := []string{
			url,
		}
		if params.Format != "" {
			toolParams = append(toolParams, "format", params.Format)
		}
		if params.Timeout != 0 {
			toolParams = append(toolParams, "timeout", (time.Duration(params.Timeout) * time.Second).String())
		}
		return renderParams(paramWidth, toolParams...)
	case tools.GlobToolName:
		var params tools.GlobParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		pattern := params.Pattern
		toolParams := []string{
			pattern,
		}
		if params.Path != "" {
			toolParams = append(toolParams, "path", params.Path)
		}
		return renderParams(paramWidth, toolParams...)
	case tools.GrepToolName:
		var params tools.GrepParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		pattern := params.Pattern
		toolParams := []string{
			pattern,
		}
		if params.Path != "" {
			toolParams = append(toolParams, "path", params.Path)
		}
		if params.Include != "" {
			toolParams = append(toolParams, "include", params.Include)
		}
		if params.LiteralText {
			toolParams = append(toolParams, "literal", "true")
		}
		return renderParams(paramWidth, toolParams...)
	case tools.LSToolName:
		var params tools.LSParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		path := params.Path
		if path == "" {
			path = "."
		}
		return renderParams(paramWidth, path)
	case tools.SourcegraphToolName:
		var params tools.SourcegraphParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		return renderParams(paramWidth, params.Query)
	case tools.ViewToolName:
		var params tools.ViewParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		filePath := removeWorkingDirPrefix(params.FilePath)
		toolParams := []string{
			filePath,
		}
		if params.Limit != 0 {
			toolParams = append(toolParams, "limit", fmt.Sprintf("%d", params.Limit))
		}
		if params.Offset != 0 {
			toolParams = append(toolParams, "offset", fmt.Sprintf("%d", params.Offset))
		}
		return renderParams(paramWidth, toolParams...)
	case tools.WriteToolName:
		var params tools.WriteParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		filePath := removeWorkingDirPrefix(params.FilePath)
		return renderParams(paramWidth, filePath)
	default:
		input := strings.ReplaceAll(toolCall.Input, "\n", " ")
		params = renderParams(paramWidth, input)
	}
	return params
}

func truncateHeight(content string, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		return strings.Join(lines[:height], "\n")
	}
	return content
}

func renderToolResponse(toolCall message.ToolCall, response message.ToolResult, width int) string {
	if response.IsError {
		errContent := fmt.Sprintf("Error: %s", strings.ReplaceAll(response.Content, "\n", " "))
		errContent = ansi.Truncate(errContent, width-1, "...")
		return styles.BaseStyle.
			Foreground(styles.Error).
			Render(errContent)
	}
	resultContent := truncateHeight(response.Content, maxResultHeight)
	switch toolCall.Name {
	case agent.AgentToolName:
		return styles.ForceReplaceBackgroundWithLipgloss(
			toMarkdown(resultContent, false, width),
			styles.Background,
		)
	case tools.BashToolName:
		resultContent = fmt.Sprintf("```bash\n%s\n```", resultContent)
		return styles.ForceReplaceBackgroundWithLipgloss(
			toMarkdown(resultContent, true, width),
			styles.Background,
		)
	case tools.EditToolName:
		metadata := tools.EditResponseMetadata{}
		json.Unmarshal([]byte(response.Metadata), &metadata)
		truncDiff := truncateHeight(metadata.Diff, maxResultHeight)
		formattedDiff, _ := diff.FormatDiff(truncDiff, diff.WithTotalWidth(width), diff.WithStyle(diffStyle))
		return formattedDiff
	case tools.FetchToolName:
		var params tools.FetchParams
		json.Unmarshal([]byte(toolCall.Input), &params)
		mdFormat := "markdown"
		switch params.Format {
		case "text":
			mdFormat = "text"
		case "html":
			mdFormat = "html"
		}
		resultContent = fmt.Sprintf("```%s\n%s\n```", mdFormat, resultContent)
		return styles.ForceReplaceBackgroundWithLipgloss(
			toMarkdown(resultContent, true, width),
			styles.Background,
		)
	case tools.GlobToolName:
		return styles.BaseStyle.Width(width).Foreground(styles.ForgroundMid).Render(resultContent)
	case tools.GrepToolName:
		return styles.BaseStyle.Width(width).Foreground(styles.ForgroundMid).Render(resultContent)
	case tools.LSToolName:
		return styles.BaseStyle.Width(width).Foreground(styles.ForgroundMid).Render(resultContent)
	case tools.SourcegraphToolName:
		return styles.BaseStyle.Width(width).Foreground(styles.ForgroundMid).Render(resultContent)
	case tools.ViewToolName:
		metadata := tools.ViewResponseMetadata{}
		json.Unmarshal([]byte(response.Metadata), &metadata)
		ext := filepath.Ext(metadata.FilePath)
		if ext == "" {
			ext = ""
		} else {
			ext = strings.ToLower(ext[1:])
		}
		resultContent = fmt.Sprintf("```%s\n%s\n```", ext, truncateHeight(metadata.Content, maxResultHeight))
		return styles.ForceReplaceBackgroundWithLipgloss(
			toMarkdown(resultContent, true, width),
			styles.Background,
		)
	case tools.WriteToolName:
		params := tools.WriteParams{}
		json.Unmarshal([]byte(toolCall.Input), &params)
		metadata := tools.WriteResponseMetadata{}
		json.Unmarshal([]byte(response.Metadata), &metadata)
		ext := filepath.Ext(params.FilePath)
		if ext == "" {
			ext = ""
		} else {
			ext = strings.ToLower(ext[1:])
		}
		resultContent = fmt.Sprintf("```%s\n%s\n```", ext, truncateHeight(params.Content, maxResultHeight))
		return styles.ForceReplaceBackgroundWithLipgloss(
			toMarkdown(resultContent, true, width),
			styles.Background,
		)
	default:
		resultContent = fmt.Sprintf("```text\n%s\n```", resultContent)
		return styles.ForceReplaceBackgroundWithLipgloss(
			toMarkdown(resultContent, true, width),
			styles.Background,
		)
	}
}

func renderToolMessage(
	toolCall message.ToolCall,
	allMessages []message.Message,
	messagesService message.Service,
	focusedUIMessageId string,
	nested bool,
	width int,
	position int,
) uiMessage {
	if nested {
		width = width - 3
	}
	response := findToolResponse(toolCall.ID, allMessages)
	toolName := styles.BaseStyle.Foreground(styles.ForgroundDim).Render(fmt.Sprintf("%s: ", toolName(toolCall.Name)))
	params := renderToolParams(width-2-lipgloss.Width(toolName), toolCall)
	responseContent := ""
	if response != nil {
		responseContent = renderToolResponse(toolCall, *response, width-2)
		responseContent = strings.TrimSuffix(responseContent, "\n")
	} else {
		responseContent = styles.BaseStyle.
			Italic(true).
			Width(width - 2).
			Foreground(styles.ForgroundDim).
			Render("Waiting for response...")
	}
	style := styles.BaseStyle.
		Width(width - 1).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1).
		BorderForeground(styles.ForgroundDim)

	parts := []string{}
	if !nested {
		params := styles.BaseStyle.
			Width(width - 2 - lipgloss.Width(toolName)).
			Foreground(styles.ForgroundDim).
			Render(params)

		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Left, toolName, params))
	} else {
		prefix := styles.BaseStyle.
			Foreground(styles.ForgroundDim).
			Render(" └ ")
		params := styles.BaseStyle.
			Width(width - 2 - lipgloss.Width(toolName)).
			Foreground(styles.ForgroundMid).
			Render(params)
		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Left, prefix, toolName, params))
	}
	if toolCall.Name == agent.AgentToolName {
		taskMessages, _ := messagesService.List(context.Background(), toolCall.ID)
		toolCalls := []message.ToolCall{}
		for _, v := range taskMessages {
			toolCalls = append(toolCalls, v.ToolCalls()...)
		}
		for _, call := range toolCalls {
			rendered := renderToolMessage(call, []message.Message{}, messagesService, focusedUIMessageId, true, width, 0)
			parts = append(parts, rendered.content)
		}
	}
	if responseContent != "" && !nested {
		parts = append(parts, responseContent)
	}

	content := style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		),
	)
	if nested {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		)
	}
	toolMsg := uiMessage{
		messageType: toolMessageType,
		position:    position,
		height:      lipgloss.Height(content),
		content:     content,
	}
	return toolMsg
}

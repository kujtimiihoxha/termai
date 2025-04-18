# OpenCode

> **⚠️ Early Development Notice:** This project is in early development and is not yet ready for production use. Features may change, break, or be incomplete. Use at your own risk.

A powerful terminal-based AI assistant for developers, providing intelligent coding assistance directly in your terminal.

## Overview

OpenCode is a Go-based CLI application that brings AI assistance to your terminal. It provides a TUI (Terminal User Interface) for interacting with various AI models to help with coding tasks, debugging, and more.

## Features

- **Interactive TUI**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a smooth terminal experience
- **Multiple AI Providers**: Support for OpenAI, Anthropic Claude, Google Gemini, AWS Bedrock, and Groq
- **Session Management**: Save and manage multiple conversation sessions
- **Tool Integration**: AI can execute commands, search files, and modify code
- **Vim-like Editor**: Integrated editor with text input capabilities
- **Persistent Storage**: SQLite database for storing conversations and sessions
- **LSP Integration**: Language Server Protocol support for code intelligence
- **File Change Tracking**: Track and visualize file changes during sessions

## Installation

```bash
# Coming soon
go install github.com/kujtimiihoxha/opencode@latest
```

## Configuration

OpenCode looks for configuration in the following locations:

- `$HOME/.opencode.json`
- `$XDG_CONFIG_HOME/opencode/.opencode.json`
- `./.opencode.json` (local directory)

### Environment Variables

You can configure OpenCode using environment variables:

| Environment Variable    | Purpose                  |
| ----------------------- | ------------------------ |
| `ANTHROPIC_API_KEY`     | For Claude models        |
| `OPENAI_API_KEY`        | For OpenAI models        |
| `GEMINI_API_KEY`        | For Google Gemini models |
| `GROQ_API_KEY`          | For Groq models          |
| `AWS_ACCESS_KEY_ID`     | For AWS Bedrock (Claude) |
| `AWS_SECRET_ACCESS_KEY` | For AWS Bedrock (Claude) |
| `AWS_REGION`            | For AWS Bedrock (Claude) |

### Configuration File Structure

```json
{
  "data": {
    "directory": ".opencode"
  },
  "providers": {
    "openai": {
      "apiKey": "your-api-key",
      "disabled": false
    },
    "anthropic": {
      "apiKey": "your-api-key",
      "disabled": false
    }
  },
  "agents": {
    "coder": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 5000
    },
    "task": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 5000
    },
    "title": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 80
    }
  },
  "mcpServers": {
    "example": {
      "type": "stdio",
      "command": "path/to/mcp-server",
      "env": [],
      "args": []
    }
  },
  "lsp": {
    "go": {
      "disabled": false,
      "command": "gopls"
    }
  },
  "debug": false,
  "debugLSP": false
}
```

## Supported AI Models

### OpenAI Models

| Model ID          | Name            | Context Window   |
| ----------------- | --------------- | ---------------- |
| `gpt-4.1`         | GPT 4.1         | 1,047,576 tokens |
| `gpt-4.1-mini`    | GPT 4.1 Mini    | 200,000 tokens   |
| `gpt-4.1-nano`    | GPT 4.1 Nano    | 1,047,576 tokens |
| `gpt-4.5-preview` | GPT 4.5 Preview | 128,000 tokens   |
| `gpt-4o`          | GPT-4o          | 128,000 tokens   |
| `gpt-4o-mini`     | GPT-4o Mini     | 128,000 tokens   |
| `o1`              | O1              | 200,000 tokens   |
| `o1-pro`          | O1 Pro          | 200,000 tokens   |
| `o1-mini`         | O1 Mini         | 128,000 tokens   |
| `o3`              | O3              | 200,000 tokens   |
| `o3-mini`         | O3 Mini         | 200,000 tokens   |
| `o4-mini`         | O4 Mini         | 128,000 tokens   |

### Anthropic Models

| Model ID            | Name              | Context Window |
| ------------------- | ----------------- | -------------- |
| `claude-3.5-sonnet` | Claude 3.5 Sonnet | 200,000 tokens |
| `claude-3-haiku`    | Claude 3 Haiku    | 200,000 tokens |
| `claude-3.7-sonnet` | Claude 3.7 Sonnet | 200,000 tokens |
| `claude-3.5-haiku`  | Claude 3.5 Haiku  | 200,000 tokens |
| `claude-3-opus`     | Claude 3 Opus     | 200,000 tokens |

### Other Models

| Model ID                    | Provider    | Name              | Context Window |
| --------------------------- | ----------- | ----------------- | -------------- |
| `gemini-2.5`                | Google      | Gemini 2.5 Pro    | -              |
| `gemini-2.0-flash`          | Google      | Gemini 2.0 Flash  | -              |
| `qwen-qwq`                  | Groq        | Qwen Qwq          | -              |
| `bedrock.claude-3.7-sonnet` | AWS Bedrock | Claude 3.7 Sonnet | -              |

## Usage

```bash
# Start OpenCode
opencode

# Start with debug logging
opencode -d

# Start with a specific working directory
opencode -c /path/to/project
```

## Command-line Flags

| Flag      | Short | Description                   |
| --------- | ----- | ----------------------------- |
| `--help`  | `-h`  | Display help information      |
| `--debug` | `-d`  | Enable debug mode             |
| `--cwd`   | `-c`  | Set current working directory |

## Keyboard Shortcuts

### Global Shortcuts

| Shortcut | Action                                                  |
| -------- | ------------------------------------------------------- |
| `Ctrl+C` | Quit application                                        |
| `Ctrl+?` | Toggle help dialog                                      |
| `Ctrl+L` | View logs                                               |
| `Esc`    | Close current overlay/dialog or return to previous mode |

### Chat Page Shortcuts

| Shortcut | Action                                  |
| -------- | --------------------------------------- |
| `Ctrl+N` | Create new session                      |
| `Ctrl+X` | Cancel current operation/generation     |
| `i`      | Focus editor (when not in writing mode) |
| `Esc`    | Exit writing mode and focus messages    |

### Editor Shortcuts

| Shortcut            | Action                                    |
| ------------------- | ----------------------------------------- |
| `Ctrl+S`            | Send message (when editor is focused)     |
| `Enter` or `Ctrl+S` | Send message (when editor is not focused) |
| `Esc`               | Blur editor and focus messages            |

### Logs Page Shortcuts

| Shortcut    | Action              |
| ----------- | ------------------- |
| `Backspace` | Return to chat page |

## AI Assistant Tools

OpenCode's AI assistant has access to various tools to help with coding tasks:

### File and Code Tools

| Tool          | Description                 | Parameters                                                                               |
| ------------- | --------------------------- | ---------------------------------------------------------------------------------------- |
| `glob`        | Find files by pattern       | `pattern` (required), `path` (optional)                                                  |
| `grep`        | Search file contents        | `pattern` (required), `path` (optional), `include` (optional), `literal_text` (optional) |
| `ls`          | List directory contents     | `path` (optional), `ignore` (optional array of patterns)                                 |
| `view`        | View file contents          | `file_path` (required), `offset` (optional), `limit` (optional)                          |
| `write`       | Write to files              | `file_path` (required), `content` (required)                                             |
| `edit`        | Edit files                  | Various parameters for file editing                                                      |
| `patch`       | Apply patches to files      | `file_path` (required), `diff` (required)                                                |
| `diagnostics` | Get diagnostics information | `file_path` (optional)                                                                   |

### Other Tools

| Tool          | Description                            | Parameters                                                                                |
| ------------- | -------------------------------------- | ----------------------------------------------------------------------------------------- |
| `bash`        | Execute shell commands                 | `command` (required), `timeout` (optional)                                                |
| `fetch`       | Fetch data from URLs                   | `url` (required), `format` (required), `timeout` (optional)                               |
| `sourcegraph` | Search code across public repositories | `query` (required), `count` (optional), `context_window` (optional), `timeout` (optional) |
| `agent`       | Run sub-tasks with the AI agent        | `prompt` (required)                                                                       |

## Architecture

OpenCode is built with a modular architecture:

- **cmd**: Command-line interface using Cobra
- **internal/app**: Core application services
- **internal/config**: Configuration management
- **internal/db**: Database operations and migrations
- **internal/llm**: LLM providers and tools integration
- **internal/tui**: Terminal UI components and layouts
- **internal/logging**: Logging infrastructure
- **internal/message**: Message handling
- **internal/session**: Session management
- **internal/lsp**: Language Server Protocol integration

## MCP (Model Context Protocol)

OpenCode implements the Model Context Protocol (MCP) to extend its capabilities through external tools. MCP provides a standardized way for the AI assistant to interact with external services and tools.

### MCP Features

- **External Tool Integration**: Connect to external tools and services via a standardized protocol
- **Tool Discovery**: Automatically discover available tools from MCP servers
- **Multiple Connection Types**:
  - **Stdio**: Communicate with tools via standard input/output
  - **SSE**: Communicate with tools via Server-Sent Events
- **Security**: Permission system for controlling access to MCP tools

### Configuring MCP Servers

MCP servers are defined in the configuration file under the `mcpServers` section:

```json
{
  "mcpServers": {
    "example": {
      "type": "stdio",
      "command": "path/to/mcp-server",
      "env": [],
      "args": []
    },
    "web-example": {
      "type": "sse",
      "url": "https://example.com/mcp",
      "headers": {
        "Authorization": "Bearer token"
      }
    }
  }
}
```

### MCP Tool Usage

Once configured, MCP tools are automatically available to the AI assistant alongside built-in tools. They follow the same permission model as other tools, requiring user approval before execution.

## LSP (Language Server Protocol)

OpenCode integrates with Language Server Protocol to provide rich code intelligence features across multiple programming languages.

### LSP Features

- **Multi-language Support**: Connect to language servers for different programming languages
- **Code Intelligence**: Get diagnostics, completions, and navigation assistance
- **File Watching**: Automatically notify language servers of file changes
- **Diagnostics**: Display errors, warnings, and hints in your code

### Supported LSP Features

| Feature           | Description                         |
| ----------------- | ----------------------------------- |
| Diagnostics       | Error checking and linting          |
| Completions       | Code suggestions and autocompletion |
| Hover             | Documentation on hover              |
| Definition        | Go to definition                    |
| References        | Find all references                 |
| Document Symbols  | Navigate symbols in current file    |
| Workspace Symbols | Search symbols across workspace     |
| Formatting        | Code formatting                     |
| Code Actions      | Quick fixes and refactorings        |

### Configuring LSP

Language servers are configured in the configuration file under the `lsp` section:

```json
{
  "lsp": {
    "go": {
      "disabled": false,
      "command": "gopls"
    },
    "typescript": {
      "disabled": false,
      "command": "typescript-language-server",
      "args": ["--stdio"]
    }
  }
}
```

### LSP Integration with AI

The AI assistant can access LSP features through the `diagnostics` tool, allowing it to:

- Check for errors in your code
- Suggest fixes based on diagnostics
- Provide intelligent code assistance

## Development

### Prerequisites

- Go 1.23.5 or higher

### Building from Source

```bash
# Clone the repository
git clone https://github.com/kujtimiihoxha/opencode.git
cd opencode

# Build the diff script first
go run cmd/diff/main.go

# Build
go build -o opencode

# Run
./opencode
```

## Acknowledgments

OpenCode builds upon the work of several open source projects and developers:

- [@isaacphi](https://github.com/isaacphi) - LSP client implementation

## License

OpenCode is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Here's how you can contribute:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please make sure to update tests as appropriate and follow the existing code style.

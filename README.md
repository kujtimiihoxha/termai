# OpenCode

> **⚠️ Early Development Notice:** This project is in early development and is not yet ready for production use. Features may change, break, or be incomplete. Use at your own risk.

A powerful terminal-based AI assistant for developers, providing intelligent coding assistance directly in your terminal.

[![OpenCode Demo](https://asciinema.org/a/dtc4nJyGSZX79HRUmFLY3gmoy.svg)](https://asciinema.org/a/dtc4nJyGSZX79HRUmFLY3gmoy)

## Overview

OpenCode is a Go-based CLI application that brings AI assistance to your terminal. It provides a TUI (Terminal User Interface) for interacting with various AI models to help with coding tasks, debugging, and more.

## Features

- **Interactive TUI**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a smooth terminal experience
- **Multiple AI Providers**: Support for OpenAI, Anthropic Claude, and Google Gemini models
- **Session Management**: Save and manage multiple conversation sessions
- **Tool Integration**: AI can execute commands, search files, and modify code
- **Vim-like Editor**: Integrated editor with Vim keybindings for text input
- **Persistent Storage**: SQLite database for storing conversations and sessions

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

You can also use environment variables:

- `ANTHROPIC_API_KEY`: For Claude models
- `OPENAI_API_KEY`: For OpenAI models
- `GEMINI_API_KEY`: For Google Gemini models

## Usage

```bash
# Start OpenCode
opencode

# Start with debug logging
opencode -d
```

### Keyboard Shortcuts

#### Global Shortcuts

- `?`: Toggle help panel
- `Ctrl+C` or `q`: Quit application
- `L`: View logs
- `Backspace`: Go back to previous page
- `Esc`: Close current view/dialog or return to normal mode

#### Session Management

- `N`: Create new session
- `Enter` or `Space`: Select session (in sessions list)

#### Editor Shortcuts (Vim-like)

- `i`: Enter insert mode
- `Esc`: Enter normal mode
- `v`: Enter visual mode
- `V`: Enter visual line mode
- `Enter`: Send message (in normal mode)
- `Ctrl+S`: Send message (in insert mode)

#### Navigation

- Arrow keys: Navigate through lists and content
- Page Up/Down: Scroll through content

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

[License information coming soon]

## Contributing

[Contribution guidelines coming soon]

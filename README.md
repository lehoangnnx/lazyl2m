# LazyL2M TUI

A Go-based Terminal User Interface (TUI) application for managing CLIProxyAPI - a local proxy server for AI coding assistants.

## Overview

This project is a TUI clone of [Quotio](https://github.com/nguyenphutrong/quotio), built using Go and [tview](https://github.com/rivo/tview). It provides a comprehensive terminal interface for managing multiple AI provider accounts, monitoring usage quotas, and controlling a local proxy server.

## Features

- **Multi-Provider Management** - Support for 10 AI providers:
  - 💎 Gemini
  - 🤖 Claude
  - ⚡ Codex (OpenAI)
  - 🐉 Qwen
  - 🌊 iFlow
  - 🚀 Antigravity
  - 🔷 Vertex AI
  - 🎯 Kiro
  - 🐙 GitHub Copilot
  - ➡️ Cursor

- **Proxy Server Control** - Start/stop local proxy server with one keystroke
- **Quota Tracking** - Real-time monitoring of usage quotas per account
- **Agent Configuration** - Manage CLI agent installations and configurations
- **API Key Management** - Generate and manage API keys for proxy authentication
- **Real-time Dashboard** - Live statistics including:
  - Server status
  - Total/success/failed requests
  - Token usage
  - Success rate
  - Connected accounts overview
- **Colored Logs** - Scrollable log viewer with level-based coloring
- **Persistent Settings** - Configuration saved to `~/.config/lazyl2m-tui/config.json`

## Technology Stack

- **Language**: Go 1.24+
- **TUI Framework**: [github.com/rivo/tview](https://github.com/rivo/tview)
- **Terminal Rendering**: [github.com/gdamore/tcell/v2](https://github.com/gdamore/tcell)

## Installation

### Prerequisites

- Go 1.24 or higher
- Terminal with 256 color support

### Build from Source

```bash
# Clone the repository
git clone https://github.com/lehoangnnx/lazyl2m.git
cd lazyl2m

# Download dependencies
go mod tidy

# Build the application
go build -o lazyl2m .

# Run the application
./lazyl2m
```

### Quick Run (without building)

```bash
go run .
```

## Usage

### Navigation

The application has a sidebar navigation menu with 7 main screens:

- **Dashboard (d)** - Server status, usage stats, and connected accounts
- **Quota (q)** - Per-account quota usage table
- **Providers (p)** - List of supported AI providers
- **Agents (a)** - CLI agent installation and configuration status
- **API Keys (k)** - API key management
- **Logs (l)** - Application logs with color coding
- **Settings (s)** - Configuration form
- **Quit (x)** - Exit the application

### Keyboard Shortcuts

#### Global
- `Tab` - Toggle focus between sidebar and content area
- `d` - Go to Dashboard
- `q` - Go to Quota screen
- `p` - Go to Providers screen
- `a` - Go to Agents screen
- `k` - Go to API Keys screen
- `l` - Go to Logs screen
- `s` - Go to Settings screen
- `x` - Quit application

#### Dashboard Screen
- `s` - Start/stop the proxy server
- `r` - Refresh data

#### Quota Screen
- `r` - Refresh quota data

#### Logs Screen
- `c` - Clear all logs

#### API Keys Screen
- `g` - Generate new API key
- `d` - Delete selected key (when implemented)

### Configuration

The application stores its configuration in `~/.config/lazyl2m-tui/config.json`.

#### Default Settings

- **Port**: 8317
- **Routing Strategy**: round-robin
- **Auto-start Server**: false
- **Debug Mode**: false
- **Log to File**: false
- **Usage Statistics**: enabled
- **Request Retry Count**: 3

#### Configuration Options

All settings can be modified through the Settings screen:

- **Port** - Local proxy server port (1-65535)
- **Routing Strategy** - Load balancing method:
  - `Round Robin` - Distribute requests evenly across accounts
  - `Fill First` - Use first account until quota exhausted
- **Auto-start Server** - Start proxy automatically on launch
- **Debug Mode** - Enable verbose debug logging
- **Log to File** - Write logs to file system
- **Usage Statistics** - Track and display usage metrics
- **Request Retry Count** - Number of retry attempts for failed requests

## Project Structure

```
.
├── main.go           # Application entry point and UI setup
├── models.go         # Data models (providers, auth files, stats, etc.)
├── config.go         # Configuration management
├── proxy_manager.go  # CLIProxyAPI process management
├── screens.go        # TUI screen implementations
├── go.mod            # Go module definition
├── go.sum            # Dependency checksums
├── .gitignore        # Git ignore patterns
└── README.md         # This file
```

## Screens

### 1. Dashboard
- Shows server status (running/stopped) with color indicator
- Displays real-time usage statistics
- Lists connected accounts with status
- Start/stop server controls
- Refresh button

### 2. Quota
- Table view of per-account quota usage
- Columns: Provider, Account, Used, Limit, Usage %, Status, Reset Time
- Color-coded status indicators:
  - 🟢 Green (ok) - Usage < 70%
  - 🟡 Yellow (warning) - Usage 70-90%
  - 🔴 Red (exceeded) - Usage > 90%

### 3. Providers
- List of all 10 supported AI providers
- Shows provider icon and display name
- Account count per provider
- Option to manage accounts (press Enter)

### 4. Agents
- Table of CLI agents
- Shows installation status (✓/✗)
- Shows configuration status (✓/✗)
- Supported agents:
  - Claude Code
  - Codex CLI
  - Gemini CLI
  - Amp CLI
  - OpenCode
  - Factory Droid

### 5. API Keys
- Lists all generated API keys (masked display)
- Generate new key with 'g'
- Delete keys (when selected)

### 6. Logs
- Scrollable log viewer
- Auto-scrolls to newest entries
- Color-coded by level:
  - ⚪ White (INFO)
  - 🟡 Yellow (WARN)
  - 🔴 Red (ERROR)
  - ⚫ Gray (DEBUG)
- Timestamp + Level + Message format
- Maximum 1000 entries retained

### 7. Settings
- Form-based configuration editor
- Real-time field validation
- Save/Reset buttons
- Changes persist to config file

## Development

### Running in Development Mode

```bash
go run .
```

### Building

```bash
go build -o lazyl2m-tui .
```

### Dependencies

The application uses the following Go modules:

- `github.com/rivo/tview` - TUI framework
- `github.com/gdamore/tcell/v2` - Terminal cell rendering
- `github.com/rivo/uniseg` - Unicode text segmentation
- `github.com/lucasb-eyer/go-colorful` - Color manipulation
- `github.com/mattn/go-runewidth` - Unicode character width
- `golang.org/x/sys` - System calls
- `golang.org/x/term` - Terminal control
- `golang.org/x/text` - Text processing

## API Integration

The application expects CLIProxyAPI to expose the following management endpoints:

- `GET /management/auth-files` - Returns list of authenticated accounts
- `GET /management/usage-statistics` - Returns usage statistics

Response formats should match the `AuthFile` and `UsageStats` structs defined in `models.go`.

## Troubleshooting

### Application won't start
- Ensure Go 1.24+ is installed: `go version`
- Check dependencies: `go mod tidy`
- Verify terminal supports 256 colors

### Proxy server won't start
- Check if port 8317 (or configured port) is available
- Verify CLIProxyAPI binary is in PATH or configured location
- Check logs screen for error messages

### Configuration not saving
- Ensure write permissions to `~/.config/lazyl2m-tui/`
- Check disk space
- Review logs for error messages

### Display issues
- Ensure terminal window is at least 80x24
- Try resizing terminal window
- Check terminal emulator compatibility

## Limitations

- This is a TUI frontend for CLIProxyAPI
- Requires CLIProxyAPI binary to be installed separately for full functionality
- Mock data is displayed when CLIProxyAPI is not running
- Some features (like actual account management) require backend API implementation

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## License

MIT

## Acknowledgments

- Inspired by [Quotio](https://github.com/nguyenphutrong/quotio) by @nguyenphutrong
- Built with [tview](https://github.com/rivo/tview) by @rivo
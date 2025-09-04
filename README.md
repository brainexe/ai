# AI CLI

A command-line tool that generates shell commands from natural language descriptions using OpenAI's API.

## Features

- 🤖 Natural language to shell command conversion
- 🔄 Concurrent API calls for better performance
- 🛡️ Safety-first approach with read-only preferences
- 📊 Verbose mode with detailed API response information
- 🎯 Interactive command selection
- 🔧 Cross-platform support (Linux, macOS)

## Installation

### Prerequisites

- Go 1.24.6 or later
- OpenAI API token

### Option 1: Install with go install (Recommended)

```bash
go install github.com/brainexe/ai@latest
```

### Option 2: Download Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/brainexe/ai/releases).

Available platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/brainexe/ai.git
cd ai-cli

# Install dependencies
go mod tidy

# Build the project
make build
```

## Setup

1. Set your OpenAI API token:
   ```bash
   export OPENAI_TOKEN="your-openai-token-here"
   ```

2. Make the binary executable (if needed):
   ```bash
   chmod +x ai
   ```

## Usage

### Basic Usage

```bash
# Find the largest file in current directory
./ai "find biggest file here"

# List files in current directory
./ai "list files in current dir"

# Search for text in files
./ai "search for TODO in all files"
```

### Command Options

#### Verbose Mode

Use the `-v` flag to see detailed information about API calls and generated commands:

```bash
./ai -v "show disk usage of directories"
```

#### Number of Commands

Use the `-n` flag to specify how many commands to generate (default: 3):

```bash
# Generate 5 command options
./ai -n 5 "find large files"

# Combine with verbose mode
./ai -v -n 2 "show disk usage"
```

Verbose mode displays:
- Number of commands generated
- API request timing information
- All generated command options
- Raw API responses (pretty-printed JSON)

### Interactive Selection

When multiple commands are generated, you'll be prompted to select one:

```
Select a command:
  1) find . -type f -exec ls -la {} + | sort -k5 -nr | head -1
  2) du -ah . | sort -rh | head -1
  3) ls -lah | sort -k5 -nr | head -1
Enter number: 1
```

## How It Works

1. **Context Gathering**: Collects system information (OS, shell, architecture)
2. **Prompt Building**: Creates a safety-focused prompt with task description
3. **Concurrent API Calls**: Makes multiple concurrent requests to OpenAI API
4. **Command Generation**: Extracts and sanitizes shell commands from responses
5. **Deduplication**: Removes duplicate commands across API responses
6. **Interactive Selection**: Allows user to choose from available options
7. **Safe Execution**: Runs the selected command with inherited stdio

## Safety Features

- **Read-only preference**: Prioritizes non-destructive commands
- **Destructive action warnings**: Avoids `rm -rf`, `chmod -R`, `sudo` unless explicitly requested
- **Single command output**: Ensures only one safe command per response
- **Path safety**: Properly quotes paths containing spaces
- **Command sanitization**: Removes code blocks and extra formatting

## Development

### Build Commands

```bash
# Build the project
make build

# Clean build artifacts
make clean

# Run linter
make lint
```

### CI/CD Pipeline

The project uses GitHub Actions for continuous integration and deployment:

- **CI Pipeline**: Runs on every push and pull request
  - Builds and tests the code
  - Runs linter checks
  - Creates build artifacts for all supported platforms
  
- **Release Pipeline**: Triggered on version tags
  - Builds binaries for all platforms (Linux, macOS, Windows)
  - Creates GitHub releases with downloadable artifacts
  - Supports both amd64 and arm64 architectures

### Code Style

- Follow Go standard formatting (`gofmt`)
- Adhere to `golangci-lint` rules
- Use single responsibility functions
- Employ clear variable names

## Examples

```bash
# File operations
./ai "show files modified in last 24 hours"
./ai "count lines in all go files"

# System information
./ai "show memory usage"
./ai "list running processes"

# Text processing
./ai "find all TODO comments in source code"
./ai "replace tabs with spaces in all files"

# Network operations
./ai "check if port 8080 is open"
./ai "show network connections"
```

## Configuration

The tool uses the following environment variables:

- `OPENAI_TOKEN`: Your OpenAI API token (required)
- `SHELL`: Shell to use for command execution (defaults to system shell)

## API Details

- **Model**: gpt-5-mini
- **Endpoint**: OpenAI Responses API
- **Concurrent calls**: Configurable (default: 3)
- **Timeout**: 30 seconds per request
- **Max output tokens**: 500

## License

This project is licensed under the terms specified in the LICENSE file.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make lint` to ensure code quality
5. Submit a pull request

## Troubleshooting

### Common Issues

**"OPENAI_TOKEN not set" error**
- Ensure you've exported your OpenAI API token as an environment variable

**"No commands generated" error**
- Try rephrasing your request more clearly
- Check your internet connection
- Verify your OpenAI API token is valid

**Command not found**
- Make sure the binary is in your PATH or use the full path `./ai`
- Verify the binary has execute permissions

### Verbose Mode for Debugging

Use the `-v` flag to see detailed information about what's happening:

```bash
./ai -v "your command description"
```

This will show API response times, generated commands, and raw API responses to help diagnose issues.

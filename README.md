# AI CLI

A command-line tool that generates shell commands from natural language descriptions using OpenAI's API.

## Features

- 🤖 Natural language to shell command conversion
- 🔄 Concurrent API calls for better performance
- 🛡️ Safety-first approach with read-only preferences
- 📊 Verbose mode with detailed API response information
- 🎯 Interactive command selection
- 🔧 Cross-platform support (Linux, macOS, windows etc)

## Installation

### Prerequisites

- Go 1.24.6 or later
- OpenAI API token

### Option 1: Install with go install (Recommended)

```bash
go install github.com/brainexe/ai@latest
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/brainexe/ai.git
cd ai-cli

make build
```

## Setup

1. Set your OpenAI API token:
```bash
export OPENAI_TOKEN="your-openai-token-here"
```

## Usage

### Examples

```bash
ai "find biggest file here"
Select a command:
  1) find . -type f -exec ls -la {} + | sort -k5 -nr | head -1
  2) du -ah . | sort -rh | head -1
  3) ls -lah | sort -k5 -nr | head -1
Enter number: 1
```

```bash
ai "search for TODO in all files"
Select a command:
  1) grep -r "TODO" .
  2) find . -name "*.go" -exec grep -l "TODO" {} +
  3) grep -rn "TODO" .
Enter number: 1
```

```bash
ai "show files modified in last 24 hours"
Select a command:
  1) find . -type f -mtime -1
  2) ls -lt | head -10
  3) find . -type f -newermt "1 day ago"
Enter number: 1
```

```bash
ai bpftrace trace all page faults, show pid, application name
Select a command:
  1) sudo bpftrace -e 'tracepoint:exceptions:page_fault_user { printf("%d %s\n", pid, comm); }'
  2) sudo bpftrace -e 'tracepoint:exceptions:page_fault_user { printf("%d %s %s\n", pid, comm, str(args->address)); }'
  3) sudo bpftrace -e 'tracepoint:exceptions:page_fault_user { printf("%d %s %s\n", pid, comm, args->message ? args->message : ""); }'
```

```bash
ai "count lines in all go files"
Select a command:
  1) find . -name "*.go" -exec wc -l {} +
  2) cloc .
  3) find . -name "*.go" | xargs wc -l
Enter number: 1
```

```bash
ai "find all TODO comments in source code"
Select a command:
  1) grep -r "TODO" .
  2) find . -name "*.go" -exec grep -n "TODO" {} +
  3) grep -rn "TODO" .
Enter number: 1
```

```bash
ai "replace tabs with spaces in all files"
Select a command:
  1) find . -name "*.go" -exec sed -i 's/\t/    /g' {} +
  2) find . -type f -name "*.go" -exec expand -t 4 {} \; -exec mv {}.exp {} \;
  3) sed -i 's/\t/    /g' *.go
Enter number: 1
```

```bash
ai "check if port 8080 is open"
Select a command:
  1) netstat -tuln | grep 8080
  2) lsof -i :8080
  3) ss -tuln | grep 8080
Enter number: 1
```

### Command Options

#### Verbose Mode

Use the `-v` flag to see detailed information about API calls and generated commands:

#### Number of Commands

Use the `-n` flag to specify how many commands to generate (default: 3):

```bash
# Generate 5 command options
ai -n 5 "find large files"
Select a command:
  1) find . -type f -size +100M
  2) du -ah . | sort -rh | head -10
  3) find . -type f -exec ls -lh {} + | awk '$5 ~ /[0-9]+M/'
  4) ls -lah | sort -k5 -hr | head -10
  5) find . -type f -size +50M -exec ls -lh {} +
Enter number: 1
```

## Safety Features

- **Read-only preference**: Prioritizes non-destructive commands
- **Destructive action warnings**: Avoids `rm -rf`, `chmod -R`, `sudo` unless explicitly requested
- **Single command output**: Ensures only one safe command per response
- **Path safety**: Properly quotes paths containing spaces
- **Command sanitization**: Removes code blocks and extra formatting

## Development

### Build Commands

```bash
# build the "ai" binary
make build

# Run linter
make lint
```

## Configuration

The tool uses the following environment variables:

- `OPENAI_TOKEN`: Your OpenAI API token in env vars (required)

## License

This project is licensed under MIT License, see the LICENSE file.

### Common Issues

**"OPENAI_TOKEN not set" error**
- Ensure you've exported your OpenAI API token as an environment variable

**"No commands generated" error**
- Try rephrasing your request more clearly
- Check your internet connection
- Verify your OpenAI API token is valid

**Command not found**
- Make sure the binary is in your PATH or use the full path `ai`
- Verify the binary has execute permissions

### Verbose Mode for Debugging

Use the `-v` flag to see detailed information about what's happening:

```bash
ai -v "your command description"
```

This will show API response times, generated commands, and raw API responses to help diagnose issues.

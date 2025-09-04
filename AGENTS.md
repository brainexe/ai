# AGENTS.md

## Project Overview
AI CLI tool that generates shell commands from natural language descriptions using OpenAI API.

## Setup Instructions
```bash
# Install dependencies
go mod tidy

# Set environment variable
export OPENAI_TOKEN="your-openai-token"

# Build the project
make build
```

## Build and Test Commands
- Build: `make build`
- Clean: `make clean`
- Lint: `make lint`

**Validation**: After each change, run `make lint` to ensure code quality.

## Usage
```bash
# Basic usage
./ai "find biggest file here"

# Verbose mode
./ai -v "list files in current dir"
```

## Code Style
- Go standard formatting (gofmt)
- Follow golangci-lint rules
- Single responsibility functions
- Clear variable names

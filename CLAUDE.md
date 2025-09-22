# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based backend for a webshell application that provides shell access and file management capabilities through WebSocket connections. The backend supports both local shell execution and SSH connections to remote servers.

## Common Commands

### Build and Run
```bash
# Build the application
go build -o webshell .

# Run the application directly
go run main.go

# Run with race detection during development
go run -race main.go

# Build for production with optimizations
go build -ldflags="-s -w" -o webshell .
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run specific test file
go test ./websocket/service/shell/

# Run tests with race detection
go test -race ./...
```

### Development Tools
```bash
# Format code
go fmt ./...

# Vet code for potential issues
go vet ./...

# Download and tidy dependencies
go mod tidy

# Update dependencies
go get -u ./...
```

## Architecture

### Core Components

1. **Main Entry Point** (`main.go`): Initializes Gin web server on port 1234
2. **Controller Layer** (`controller/`): HTTP route handlers for shell endpoints
3. **WebSocket Layer** (`websocket/`): Real-time communication infrastructure
4. **Service Layer** (`websocket/service/`): Business logic for different functionalities

### WebSocket Architecture

The application uses a service-oriented WebSocket architecture:

- **Server** (`websocket/server.go`): Manages WebSocket connections and service registration
- **Connection** (`websocket/conn.go`): Handles WebSocket message dispatching
- **Services**: Modular components that handle specific functionality

### Available Services

1. **Shell Service** (`websocket/service/shell/`):
   - Local shell execution via PTY
   - SSH remote shell connections
   - Supports both local and remote environments

2. **File System Service** (`websocket/service/fs/`):
   - File operations (list, read, write, delete)
   - Both local filesystem and SFTP support

3. **Upload Service** (`websocket/service/upload/`):
   - File upload capabilities
   - Local and SFTP destinations

4. **Heartbeat Service** (`websocket/service/heartbeat/`):
   - Connection keep-alive mechanism

### Key Design Patterns

- **Service Registration**: Services register themselves with the WebSocket server
- **Message Routing**: Text and binary messages are routed to appropriate services
- **Provider Pattern**: Local and SSH implementations use provider interfaces
- **Environment Abstraction**: Separate environment handling for local vs SSH contexts

## Dependencies

Key external dependencies:
- `github.com/gin-gonic/gin`: HTTP web framework
- `github.com/gorilla/websocket`: WebSocket implementation
- `github.com/creack/pty`: PTY (pseudo-terminal) support for local shells
- `github.com/pkg/sftp`: SFTP client for remote file operations
- `golang.org/x/crypto`: SSH client implementation
- `github.com/stretchr/testify`: Testing framework

## Development Notes

- Server runs on port 1234 by default
- WebSocket connections have a 10-second timeout check
- The application supports both local shell execution and SSH remote connections
- File operations support both local filesystem and remote SFTP
- All services implement a common Service interface for consistent message handling
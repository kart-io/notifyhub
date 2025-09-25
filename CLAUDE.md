# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Build all packages
go build ./...

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -run TestHubCreation ./client

# Run tests in a specific package
go test ./client
go test ./notifiers
go test ./queue

# Run integration tests
go test -v ./integration_test.go

# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Run validation tests
go test ./_tests
```

## Architecture Overview

NotifyHub is a unified notification system built with a modular, plugin-based architecture:

### Core Components

- **Hub** (`client/hub.go`) - Central coordinator that orchestrates message sending through notifiers
- **Notifiers** (`notifiers/`) - Platform adapters implementing the `Notifier` interface (Feishu, Email)
- **Queue System** (`queue/`) - Async message processing with retry logic and multiple backend support (Memory, Redis)
- **Template Engine** (`template/`) - Message templating with variable substitution
- **Routing Engine** (`config/routing.go`) - Priority-based rule matching for automatic platform routing
- **Configuration** (`config/`) - Option-based configuration system with environment variable support

### Key Design Patterns

**Builder Pattern with Functional Options**: Configuration uses functional options pattern

```go
hub, err := client.New(
    config.WithFeishu("webhook", "secret"),
    config.WithQueue("memory", 1000, 4),
)
```

**Interface-Based Plugin System**: All major components implement interfaces for extensibility

- `Notifier` interface for platform adapters
- `Queue` interface for different queue backends
- `Logger` interface for different logging backends

**Message-Target Abstraction**: Unified message structure with platform-agnostic targets

```go
type Message struct {
    Title    string
    Body     string
    Targets  []Target
    Priority int
    // ...
}
```

### Message Flow Architecture

1. **Message Creation** - Using builder pattern (`NewMessage()`, `NewAlert()`)
2. **Routing Processing** - Rule-based platform selection based on priority/metadata
3. **Template Rendering** - Variable substitution and format conversion
4. **Queue Processing** - Async handling with retry policies and worker pools
5. **Platform Delivery** - Through notifier adapters with rate limiting

### Key Architectural Features

- **Graceful Shutdown**: Complete lifecycle management with context-based cancellation
- **Retry Mechanisms**: Exponential backoff with jitter to prevent thundering herd
- **Priority-Based Routing**: Rule engine automatically selects platforms based on message characteristics
- **Resource Management**: All notifiers implement `Shutdown()` for cleanup
- **Observability**: Integrated metrics, health checks, and telemetry support

## Project Structure

- `/client` - Main Hub implementation and client API
- `/notifiers` - Platform-specific adapters (Feishu, Email)
- `/queue` - Queue systems (Simple in-memory, Redis Streams, Message scheduler)
- `/config` - Configuration management with functional options
- `/template` - Template engine for message formatting
- `/monitoring` - Metrics and monitoring
- `/observability` - OpenTelemetry integration for tracing and metrics
- `/internal` - Internal utilities (rate limiting, ID generation)
- `/logger` - Logging interface and implementations
- `/_tests` - Validation and integration tests
- `/docs` - Technical documentation and design specs

## Technical Implementation Details

### Queue System Design

The queue system supports multiple backends with pluggable interfaces:

- **SimpleQueue**: In-memory with buffered channels
- **RedisQueue**: Redis Streams with consumer groups
- **MessageScheduler**: Min-heap for delayed message scheduling

### Rate Limiting Implementation

Token bucket algorithm implementation in `/internal/ratelimiter.go`:

- Configurable refill rates and capacity
- Thread-safe with mutex protection
- Timeout support for graceful handling

### Retry Policy Architecture

Sophisticated retry system with multiple strategies:

- Exponential backoff with configurable multiplier
- Jitter to prevent thundering herd problems
- Maximum retry limits and custom policies

### OpenTelemetry Integration

Full observability stack in `/observability/telemetry.go`:

- OTLP exporter configuration
- Distributed tracing spans for message operations
- Metrics collection (counters, histograms, gauges)
- Environment-based configuration

## Configuration Patterns

The system uses functional options pattern extensively:

```go
// Environment-based configuration
hub, err := client.New(config.WithDefaults())

// Explicit configuration
hub, err := client.New(
    config.WithFeishu(webhookURL, secret),
    config.WithEmail(host, port, username, password, from, useTLS, timeout),
    config.WithQueue("memory", 1000, 4),
    config.WithTelemetry("service", "version", "env", "endpoint"),
)

// Test configuration
hub, err := client.New(config.WithTestDefaults())
```

## Testing Approach

- **Unit Tests**: Individual component testing with interfaces
- **Integration Tests**: End-to-end workflow testing
- **Validation Tests**: Architecture and design validation
- **Mock Usage**: Interface-based mocking for external dependencies

Test files follow Go conventions (`*_test.go`) and are distributed across packages for component isolation.

## Version Information

Current implementation version: v1.2.0 with advanced features:

- Delay message scheduling (FR12)
- Rate limiting (NFR7)
- Redis queue adapter
- OpenTelemetry integration (NFR5)
- Enhanced error handling and graceful shutdown

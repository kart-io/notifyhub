# NotifyHub Project Context

## Project Overview

NotifyHub is a comprehensive, unified notification dispatch system written in Go. Its purpose is to provide a single, consistent API for sending notifications across various platforms like Email, Feishu, Slack, and SMS.

The architecture is modular and extensible. The main components are:
- **Client (`client.go`)**: The primary, user-facing entry point for the library. It handles the initialization and wiring of all other components.
- **Hub (`core/hub/hub.go`)**: The central engine that receives messages from the client. It applies middleware (like rate limiting and retries) and dispatches messages to the appropriate platform.
- **Platforms (`platforms/`)**: A collection of pluggable modules, each implementing the logic for a specific notification service (e.g., `platforms/feishu`, `platforms/email`). The system is designed to be easily extended with new platforms.
- **Queue (`queue/` and `core/queue.go`)**: An optional component for asynchronous message processing. It supports different backends, such as an in-memory queue or Redis, allowing the system to handle high throughput.
- **Configuration (`config/`)**: A robust configuration system that uses YAML files (`example.yaml`) and environment variables to manage settings for all components.
- **Middleware (`middleware/`)**: Provides cross-cutting functionality like rate limiting and automatic retries.

The project aims for a clean, type-safe, and fluent API for developers, abstracting away the complexities of each individual notification platform.

## Building and Running

The project uses a `Makefile` for common development tasks. The key commands are:

- **Build the project:**
  ```bash
  make build-all
  ```

- **Run all tests:** This includes unit tests, race detection, and coverage.
  ```bash
  make test-all
  # For coverage report:
  make test-coverage
  ```

- **Run linters:**
  ```bash
  make lint-all
  ```

- **Format code:**
  ```bash
  make fmt
  ```

- **Run example applications:**
  ```bash
  make run-examples
  ```

The Continuous Integration (CI) pipeline is defined in `.github/workflows/ci.yml` and automates these checks.

## Development Conventions

- **Structure**: The project is organized by feature into distinct packages (`core`, `platforms`, `middleware`, `config`, `logger`). The primary user-facing API is exposed in the root `notifyhub` package, following modern Go library conventions.
- **Configuration**: All configuration is managed via the `config` package, which loads settings from a YAML file and/or environment variables. The `config/example.yaml` file serves as a template.
- **Abstraction & Extensibility**: The system is built around interfaces to promote decoupling and extensibility. The `platforms` and `queue` packages are key examples, allowing new notification services and queue backends to be added with minimal changes to the core logic.
- **Testing**: The project maintains a high standard of testing. Unit tests are co-located with the source code (e.g., `hub_test.go`). A dedicated `tests/` directory likely contains integration and end-to-end tests. The `Makefile` and CI pipeline enforce testing, including race detection.
- **Code Style**: Code formatting is enforced using `gofmt`. Linting is performed with `golangci-lint`. These checks are integrated into the CI pipeline.

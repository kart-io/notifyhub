# Makefile for NotifyHub v3.0.0

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Project info
BINARY_NAME=notifyhub
BINARY_UNIX=$(BINARY_NAME)_unix

# Directories
PKG_DIR=./pkg/...
EXAMPLES_DIR=./examples/...
LINT_DIRS=./pkg/... ./examples/...
ROOT_DIR=.

.PHONY: all build clean test coverage deps fmt lint vet check help \
	git-prune git-fetch git-clean-branches git-sync git-show-merged git-cleanup

# Default target
all: check build

# Format code using go fmt
fmt:
	@echo "Running go fmt..."
	$(GOFMT) $(LINT_DIRS)
	@echo "Code formatting complete."

# Lint code using golangci-lint (more comprehensive than golint)
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run $(LINT_DIRS)
	@echo "Linting complete."

# Alternative lint using staticcheck (if golangci-lint is not preferred)
lint-static:
	@echo "Running staticcheck..."
	@which staticcheck > /dev/null || (echo "staticcheck not found. Installing..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck $(LINT_DIRS)
	@echo "Static analysis complete."

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) $(LINT_DIRS)
	@echo "Go vet complete."

# Comprehensive check: fmt + vet + lint
check: fmt vet lint
	@echo "All code quality checks passed!"

# Build the binary
build:
	@echo "Building..."
	$(GOBUILD) -o $(BINARY_NAME) -v $(PKG_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v $(PKG_DIR)

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic $(PKG_DIR)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u honnef.co/go/tools/cmd/staticcheck
	@echo "Development tools installed."

# Cross compilation
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(PKG_DIR)

# Development workflow
dev: deps fmt vet lint test
	@echo "Development workflow complete!"

# CI/CD workflow
ci: deps check test coverage
	@echo "CI/CD workflow complete!"

# Show help
help:
	@echo "📦 NotifyHub v3.0.0 - Makefile Commands"
	@echo "========================================"
	@echo ""
	@echo "🔨 Build & Development:"
	@echo "  make build         - Build the binary"
	@echo "  make build-linux   - Cross compile for Linux"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Download and tidy dependencies"
	@echo "  make install-tools - Install development tools"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test          - Run tests"
	@echo "  make coverage      - Run tests with coverage report"
	@echo ""
	@echo "✨ Code Quality:"
	@echo "  make fmt           - Format code using go fmt"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Lint code using golangci-lint"
	@echo "  make lint-static   - Static analysis using staticcheck"
	@echo "  make check         - Run fmt + vet + lint"
	@echo ""
	@echo "🔄 Git Branch Management:"
	@echo "  make git-fetch     - 🔄 Quick fetch with prune"
	@echo "  make git-sync      - 🌟 Sync with remote and show status"
	@echo "  make git-prune     - Remove remote-deleted branch references"
	@echo "  make git-clean-branches - 🗑️  Delete local stale branches (interactive)"
	@echo "  make git-show-merged    - Show merged branches"
	@echo "  make git-cleanup        - Comprehensive cleanup (interactive)"
	@echo ""
	@echo "🚀 Workflows:"
	@echo "  make dev           - Complete development workflow"
	@echo "  make ci            - Complete CI/CD workflow"
	@echo ""
	@echo "📖 Documentation:"
	@echo "  make help          - Show this help message"
	@echo ""
	@echo "💡 Tips:"
	@echo "  • Run 'make git-sync' daily to keep branches in sync"
	@echo "  • See docs/GIT_BRANCH_MANAGEMENT.md for detailed guide"

# ============================================================================
# Git Branch Management
# ============================================================================

# Quick fetch with prune (most common command)
git-fetch:
	@echo "🔄 Fetching from remote and pruning stale references..."
	@git fetch --prune --all
	@echo "✅ Fetch complete"
	@echo ""
	@STALE_COUNT=$$(git branch -vv 2>/dev/null | grep -c ': gone]' || echo "0"); \
	if [ "$$STALE_COUNT" != "0" ] && [ "$$STALE_COUNT" -gt 0 ] 2>/dev/null; then \
		echo "⚠️  Found $$STALE_COUNT local branch(es) tracking deleted remotes"; \
		echo "💡 Run 'make git-clean-branches' to remove them"; \
	else \
		echo "✅ No stale branches found"; \
	fi

# Remove references to remote branches that have been deleted
git-prune: git-fetch
	@echo ""
	@echo "📋 Current local branches:"
	@git branch -vv
	@echo ""
	@echo "🗑️  Branches tracking deleted remotes:"
	@git branch -vv | grep ': gone]' || echo "   No stale branches found"

# Clean up local branches that have been deleted on remote
git-clean-branches: git-fetch
	@echo ""
	@echo "🔍 Finding branches to clean up..."
	@echo ""
	@STALE_BRANCHES=$$(git branch -vv | grep ': gone]' | awk '{print $$1}'); \
	if [ -z "$$STALE_BRANCHES" ]; then \
		echo "✅ No stale branches found"; \
	else \
		echo "🗑️  Found stale branches (tracking deleted remotes):"; \
		echo "$$STALE_BRANCHES" | sed 's/^/   - /'; \
		echo ""; \
		read -p "Delete these branches? [y/N] " -n 1 -r; \
		echo ""; \
		if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
			echo "$$STALE_BRANCHES" | xargs -I {} git branch -D {}; \
			echo "✅ Stale branches deleted"; \
		else \
			echo "⏭️  Skipped branch deletion"; \
		fi \
	fi

# Full sync: fetch, prune, and show status
git-sync: git-fetch
	@echo ""
	@echo "📊 Repository Status:"
	@echo "===================="
	@echo ""
	@echo "📍 Current branch:"
	@git branch --show-current
	@echo ""
	@echo "📋 All local branches:"
	@git branch -vv
	@echo ""
	@STALE_COUNT=$$(git branch -vv 2>/dev/null | grep -c ': gone]' || echo "0"); \
	if [ "$$STALE_COUNT" != "0" ] && [ "$$STALE_COUNT" -gt 0 ] 2>/dev/null; then \
		echo "⚠️  Found $$STALE_COUNT stale branch(es) tracking deleted remotes:"; \
		git branch -vv | grep ': gone]' | awk '{print "   - " $$1}'; \
		echo ""; \
		echo "💡 Run 'make git-clean-branches' to clean them up"; \
	else \
		echo "✅ All branches are up to date"; \
	fi
	@echo ""
	@AHEAD_BEHIND=$$(git status -sb | head -1); \
	echo "🔍 Tracking status: $$AHEAD_BEHIND"

# Advanced: Show merged branches that can be safely deleted
git-show-merged:
	@echo "🔍 Branches merged into current branch:"
	@CURRENT_BRANCH=$$(git branch --show-current); \
	git branch --merged | grep -v "^\*" | grep -v "main" | grep -v "master" | grep -v "develop" || echo "   No merged branches found"; \
	echo ""; \
	echo "💡 These branches are fully merged and may be safe to delete"

# Interactive branch cleanup (merged + stale)
git-cleanup: git-show-merged
	@echo ""
	@read -p "Do you want to see stale branches too? [y/N] " -n 1 -r; \
	echo ""; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		make git-clean-branches; \
	fi
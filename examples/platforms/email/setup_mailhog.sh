#!/bin/bash

# Email Platform - MailHog Setup Script
# This script sets up MailHog for local email testing

set -e

echo "📧 Setting up MailHog for Email Testing"
echo "========================================"
echo ""

# Check if MailHog is already installed
if command -v mailhog &> /dev/null; then
    echo "✅ MailHog is already installed"
else
    echo "📦 Installing MailHog..."

    if command -v brew &> /dev/null; then
        brew install mailhog
        echo "✅ MailHog installed via Homebrew"
    else
        echo "⚠️  Homebrew not found. Installing manually..."

        # Detect OS
        OS="$(uname -s)"
        case "${OS}" in
            Linux*)     MACHINE=Linux;;
            Darwin*)    MACHINE=Mac;;
            *)          MACHINE="UNKNOWN:${OS}"
        esac

        if [ "$MACHINE" = "Mac" ]; then
            curl -L https://github.com/mailhog/MailHog/releases/download/v1.0.1/MailHog_darwin_amd64 -o /usr/local/bin/mailhog
            chmod +x /usr/local/bin/mailhog
        elif [ "$MACHINE" = "Linux" ]; then
            curl -L https://github.com/mailhog/MailHog/releases/download/v1.0.1/MailHog_linux_amd64 -o /usr/local/bin/mailhog
            chmod +x /usr/local/bin/mailhog
        else
            echo "❌ Unsupported OS: $MACHINE"
            exit 1
        fi

        echo "✅ MailHog installed manually"
    fi
fi

echo ""
echo "🚀 Starting MailHog..."
echo "   SMTP Server: localhost:1025"
echo "   Web UI: http://localhost:8025"
echo ""

# Start MailHog in background
mailhog > /dev/null 2>&1 &
MAILHOG_PID=$!

echo "✅ MailHog started (PID: $MAILHOG_PID)"
echo ""

# Wait for MailHog to be ready
sleep 2

# Run the local test
echo "🧪 Running local email tests..."
echo ""
go run test_local.go

echo ""
echo "📊 Test Results:"
echo "   • All emails sent to MailHog"
echo "   • View them at: http://localhost:8025"
echo ""
echo "🔧 To stop MailHog:"
echo "   kill $MAILHOG_PID"
echo ""
echo "💡 Tip: Open the web UI to see your emails!"
echo "   open http://localhost:8025"
#!/bin/bash

# Script to build Caddy with the immosquare DNS module


echo "🔨 Building Caddy with the immosquare DNS module"

# Check if xcaddy is installed
if ! command -v xcaddy &> /dev/null; then
    echo "📦 Installing xcaddy..."
    go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
    export PATH="$PATH:$(go env GOPATH)/bin"
fi

# Build Caddy with the immosquare module (chosen version)
xcaddy build \
    --with github.com/immosquare/caddy-dns-immosquare@latest

echo "✅ Caddy built successfully!"
echo "🚀 You can now use the immosquare provider in your Caddy configuration" 
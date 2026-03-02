#!/bin/bash

echo "Installing xcaddy..."
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

echo ""
echo "Downloading Go dependencies..."
go mod download

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Run ./build.sh to build Caddy with the plugin"
echo "  2. Copy example.Caddyfile to your Caddyfile location"
echo "  3. Run: ./caddy run --config /path/to/Caddyfile"
echo ""

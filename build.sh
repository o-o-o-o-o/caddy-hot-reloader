#!/bin/bash
set -e

echo "Building Caddy with hot-reloader plugin..."

# Resolve xcaddy (PATH first, then Go bin locations)
XCADDY_BIN=""
if command -v xcaddy &> /dev/null; then
    XCADDY_BIN="$(command -v xcaddy)"
elif [ -n "$(go env GOBIN)" ] && [ -x "$(go env GOBIN)/xcaddy" ]; then
    XCADDY_BIN="$(go env GOBIN)/xcaddy"
elif [ -x "$(go env GOPATH)/bin/xcaddy" ]; then
    XCADDY_BIN="$(go env GOPATH)/bin/xcaddy"
else
    echo "Error: xcaddy is not installed or not discoverable."
    echo "Install it with: go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest"
    echo "Then rerun this script."
    exit 1
fi

# Get the current directory (plugin source)
PLUGIN_PATH=$(pwd)

# Build Caddy with the plugin
"$XCADDY_BIN" build \
    --with github.com/yourusername/caddy-hot-reloader="$PLUGIN_PATH"

echo ""
echo "✅ Build complete!"
echo ""
echo "The custom Caddy binary is ready: ./caddy"
echo ""
echo "To run with your Caddyfile:"
echo "  ./caddy run --config /path/to/Caddyfile"
echo ""
echo "Or install system-wide (requires sudo):"
echo "  sudo mv ./caddy /usr/local/bin/caddy"
echo ""

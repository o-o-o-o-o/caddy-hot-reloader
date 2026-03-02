# Caddy Hot Reloader

A Caddy build that provides hot-reloading functionality for local development, with per-site isolation and CSS injection support.

// NOTE: This plugin is designed for LOCAL DEVELOPMENT only.
// Do not use in production on internet-facing servers.

## Features

- **Per-Site Hot Reload**: Only reloads the affected site, not all open browser tabs
- **CSS Injection**: Updates stylesheets without full page reload (preserves scroll position and form state)
- **Request-Time Discovery**: Automatically detects and watches sites as they're accessed
- **Smart Filtering**: Respects `.gitignore` and allows explicit exclude patterns
- **Efficient Watching**: Uses `fsnotify` for event-driven file watching (not polling)
- **Zero Configuration**: Works out of the box with sensible defaults for Kirby CMS
- **Debug Logging**: Comprehensive logging for troubleshooting

## Why This Approach (vs Alternatives)

This plugin approach is the best fit for your setup when all of these are true:

- You want to keep browsing real wildcard dev URLs like `https://test.laac-acc.why`
- You do not want to manually add script tags to templates
- You want per-site reload isolation across many local sites
- You want minimal runtime moving parts (single Caddy process)

Compared to common alternatives:

- **BrowserSync proxy mode**: Great for single-site flows, but awkward with wildcard multi-site routing
- **BrowserSync snippet/sidecar mode**: Works, but still needs script injection or template edits
- **External watcher + browser extension**: No template edits, but requires extension dependency and extra process
- **This plugin**: No manual template injection, works at real wildcard URLs, and keeps reload logic in Caddy

Tradeoff: this is a **compile-time Caddy module**, so you build a custom Caddy binary with `xcaddy`.

## Homebrew Caddy Coexistence (No Uninstall Needed)

You do **not** need to uninstall Homebrew Caddy.

- Homebrew Caddy is a prebuilt binary without this custom module
- This plugin requires a custom-built Caddy binary
- You can keep both installed and choose which one to run

Recommended local-dev workflow:

1. Keep Brew Caddy installed
2. Build plugin-enabled Caddy with `xcaddy`
3. Run the custom binary for your dev wildcard setup

Service management notes:

- If Brew service is running (`brew services start caddy`), stop it before running your custom binary on the same ports:

```bash
brew services stop caddy
```

- Run custom Caddy manually while developing:

```bash
./caddy run --config /path/to/Caddyfile
```

- If you want to go back to Brew-managed service later:

```bash
brew services start caddy
```

In short: keep Brew for package management/updates, use the custom binary when you need `hot_reloader`.

## Installation

### Option 1: Homebrew Tap (Recommended)

The easiest way to install and manage Caddy with hot-reloader:

```bash
# Add the tap
brew tap o-o-o-o-o/caddy-hot-reloader https://github.com/o-o-o-o-o/caddy-hot-reloader

# Install
brew install caddy-hot-reloader

# Copy and configure the Caddyfile
cp $(brew --prefix)/etc/caddy-hot-reloader/Caddyfile.example \
   $(brew --prefix)/etc/caddy-hot-reloader/Caddyfile

# Edit the Caddyfile to match your paths
nano $(brew --prefix)/etc/caddy-hot-reloader/Caddyfile

# Start as a service (auto-starts on boot)
brew services start caddy-hot-reloader

# Check logs
tail -f $(brew --prefix)/var/log/caddy-hot-reloader.log
```

**Service Management:**

```bash
brew services start caddy-hot-reloader   # Start service
brew services stop caddy-hot-reloader    # Stop service
brew services restart caddy-hot-reloader # Restart service
brew services info caddy-hot-reloader    # View status
```

**Updating:**

```bash
brew update
brew upgrade caddy-hot-reloader
brew services restart caddy-hot-reloader
```

### Option 2: Build from Source

For development or if you want to modify the plugin:

1. Clone this repository:

```bash
git clone https://github.com/o-o-o-o-o/caddy-hot-reloader
cd caddy-hot-reloader
```

2. Build Caddy with the plugin using `xcaddy`:

```bash
xcaddy build --with github.com/o-o-o-o-o/caddy-hot-reloader=.
```

3. Use the newly built local binary (`./caddy`) for dev, or replace your existing binary if you prefer.

## Configuration

### Zero-Config (Uses Smart Defaults)

```caddy
*.*.why {
    hot_reloader
    root "/Users/why/Test Sites/{labels.2}/{labels.1}/www"
    file_server
    php_fastcgi 127.0.0.1:9000
}
```

**Defaults:**

- Base directory: `/Users/why/Test Sites`
- Watch patterns: `site/**`, `assets/**`, `content/**`
- Exclude patterns: `**.cache`, `**/vendor/**`, `**/node_modules/**`, `**/.DS_Store`
- File extensions: `html`, `css`, `js`, `php`, `scss`, `sass`
- Respect `.gitignore`: `true`
- Idle watcher shutdown: `30m` (configurable)

### Custom Configuration

```caddy
*.*.why {
    route {
        root * "/Users/why/Test Sites/{labels.2}/{labels.1}/www"

        encode gzip

        hot_reloader {
            base_dir "/Users/why/Test Sites"
            watch "site/**" "assets/**" "content/**" "custom/**"
            exclude "site/cache/**" "**.tmp"
            extensions "html" "css" "js" "php" "vue"
            respect_gitignore true
            idle_timeout "30m"
        }

        php_fastcgi 127.0.0.1:9000
        file_server
    }
}
```

### Using Snippets (Global Config)

```caddy
(hot_reload_config) {
    hot_reloader {
        watch "site/**" "assets/**" "content/**"
        exclude "site/cache/**"
    }
}

*.*.why {
    import hot_reload_config
    root "/Users/why/Test Sites/{labels.2}/{labels.1}/www"
    file_server
    php_fastcgi 127.0.0.1:9000
}
```

## How It Works

1. **Request-Time Discovery**: When a request hits `test.laac-acc.why`, the plugin:
   - Extracts domain labels
   - Derives site path: `/Users/why/Test Sites/test/laac-acc/www`
   - Sets up file watcher for that site (if not already watching)

2. **File Watching**: Uses `fsnotify` to watch configured directories:
   - Respects `.gitignore` rules
   - Applies exclude patterns
   - Filters by file extensions

3. **WebSocket Connection**: Client script connects to `wss://domain/hot-reload`
   - Receives reload messages when files change
   - CSS files → inject without reload
   - HTML/PHP/JS → full page reload

4. **Broadcast**: File changes broadcast only to clients of the affected site

## Architecture

```
Plugin Components:
├── hot_reloader.go    - Main module, HTTP middleware, config parsing
├── manager.go         - Site manager, WebSocket handling, broadcast
├── watcher.go         - File watching with fsnotify, .gitignore support
└── README.md          - Documentation
```

## Development

### Prerequisites

- Go 1.21+
- Caddy v2.7+
- `xcaddy` for building

### Testing Locally

1. Build Caddy with the plugin:

```bash
xcaddy build --with github.com/o-o-o-o-o/caddy-hot-reloader=./
```

2. Run with your Caddyfile:

```bash
./caddy run --config /path/to/Caddyfile
```

3. Open your site in a browser and edit files to test hot reload.

### Debugging

Enable debug logging in Caddy:

```caddy
{
    log {
        level DEBUG
    }
}
```

Watch logs for hot reload activity:

```bash
tail -f /var/log/caddy/caddy.log | grep hot_reloader
```

## Implementation Notes from This Setup (Important)

These are the exact integration issues we hit and fixed in this project:

1. **Handler order matters for injection**
   - If `php_fastcgi` runs before `hot_reloader`, script injection won't happen.
   - Use a `route` block to force deterministic order.

2. **Gzip + injection requires correct chain order**
   - Keep `encode gzip` in the route, but before `hot_reloader` in request order.
   - That gives response flow: `php/file -> hot_reloader(inject) -> encode(gzip) -> client`.

3. **Wildcard host mapping was easy to invert**
   - Correct mapping for `test.laac-acc.why` is `{labels.2}/{labels.1}` in Caddy root and `test/laac-acc` in filesystem.

4. **Chunked/streamed HTML responses need buffering**
   - Injecting on a single write causes short-write/length mismatch risk.
   - Buffer full HTML response, inject once near `</body>`, then write.

5. **HEAD requests have no body**
   - Do not treat empty body for `HEAD` as an injection failure.

6. **Long-running usage needs watcher lifecycle controls**
   - Idle watcher shutdown is enabled (`idle_timeout`) to keep fd/memory use bounded.

### Recommended `*.*.why` Block (Wildcard + PHP + gzip + hot reload)

```caddy
*.*.why {
    import tls_local

    route {
        root * "/Users/why/Test Sites/{labels.2}/{labels.1}/www"

        encode gzip

        hot_reloader {
            idle_timeout "30m"
        }

        php_fastcgi 127.0.0.1:9000 {
            dial_timeout 5s
            read_timeout 30s
        }

        file_server
    }

    handle_errors {
        respond "{http.error.status_code} {http.error.status_text}"
    }
}
```

## Persistent macOS Service (Survives Restart)

### Option 1: Using Homebrew Services (Recommended)

If you installed via Homebrew tap, use `brew services`:

```bash
brew services start caddy-hot-reloader
```

This automatically creates and manages the launchd service for you.

### Option 2: Manual launchd Setup

For custom builds or non-Homebrew installations, use `launchd` with a per-user LaunchAgent.

1. Create plist at `~/Library/LaunchAgents/dev.local.caddy-hot-reloader.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>dev.local.caddy-hot-reloader</string>

    <key>ProgramArguments</key>
    <array>
      <string>/Users/why/🍇/Servers/Caddy/Plugins/caddy-hot-reloader/caddy</string>
      <string>run</string>
      <string>--config</string>
      <string>/Users/why/🍇/Servers/Caddy/Plugins/caddy-hot-reloader/Caddyfile</string>
    </array>

    <key>WorkingDirectory</key>
    <string>/Users/why/🍇/Servers/Caddy/Plugins/caddy-hot-reloader</string>

    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>/tmp/caddy-hot-reloader.out.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/caddy-hot-reloader.err.log</string>
  </dict>
</plist>
```

2. Load/start service:

```bash
launchctl unload ~/Library/LaunchAgents/dev.local.caddy-hot-reloader.plist 2>/dev/null || true
launchctl load -w ~/Library/LaunchAgents/dev.local.caddy-hot-reloader.plist
launchctl start dev.local.caddy-hot-reloader
```

3. Verify:

```bash
launchctl list | grep caddy-hot-reloader
tail -f /tmp/caddy-hot-reloader.err.log
```

4. Stop/remove:

```bash
launchctl stop dev.local.caddy-hot-reloader
launchctl unload -w ~/Library/LaunchAgents/dev.local.caddy-hot-reloader.plist
```

## Make It Available System-Wide

### With Homebrew (Already Done)

If you installed via `brew install caddy-hot-reloader`, the `caddy` binary is already in your PATH at:

```bash
$(brew --prefix)/bin/caddy
```

### Manual Installation

Option A: Keep dedicated binary name (recommended)

```bash
sudo cp /Users/why/🍇/Servers/Caddy/Plugins/caddy-hot-reloader/caddy /usr/local/bin/caddy-hot-reloader
caddy-hot-reloader version
```

Option B: Replace global `caddy` binary (only if intentional)

```bash
sudo cp /Users/why/🍇/Servers/Caddy/Plugins/caddy-hot-reloader/caddy /usr/local/bin/caddy
caddy version
```

## Long-Term Maintenance & Performance Tips

### Homebrew Tap Maintenance

If you're maintaining the Homebrew tap:

1. **Use this repository as the tap source** (it already contains `Formula/caddy-hot-reloader.rb`)
2. **Tag releases** in this repo (e.g., `v1.0.0`)
3. **GitHub Actions** can auto-update formula metadata for new tags
4. **Dependabot** monitors dependencies weekly and creates PRs for Go module updates
5. **Test locally** before publishing:
   ```bash
   brew install --build-from-source ./Formula/caddy-hot-reloader.rb
   brew test caddy-hot-reloader
   ```

See [HOMEBREW_SETUP.md](HOMEBREW_SETUP.md) for complete instructions.

### Performance Tuning

- Keep `idle_timeout` between `15m` and `60m` depending on your workflow.
- Keep exclude patterns aggressive for `vendor`, `node_modules`, caches, and generated media trees.
- Keep log level at `INFO` normally; use `DEBUG` only for troubleshooting.
- Prefer one long-running Caddy instance rather than repeated start/stop cycles.
- Periodically check fd usage if you open many sites in a day:

```bash
pid=$(pgrep -f "./caddy run --config ./Caddyfile" | head -1)
lsof -p "$pid" | wc -l
```

## Caveats

- **macOS File Descriptor Limits**: `fsnotify` uses `kqueue` on macOS, which consumes one file descriptor per watched directory. For hundreds of sites, ensure your system limits are adequate.
- **Performance**: Watching thousands of files can impact performance. Use exclude patterns liberally.
- **Production Use**: This plugin is designed for local development. Do not use in production.

## Troubleshooting

### Hot reload not triggering

1. Check logs for "file changed" messages
2. Verify file is not in exclude patterns or `.gitignore`
3. Confirm file extension is in the extensions list
4. Check WebSocket connection in browser DevTools

### WebSocket connection fails

1. Verify `/hot-reload` endpoint is not blocked by other handlers
2. Check for CSP (Content-Security-Policy) headers blocking WebSocket
3. Ensure HTTPS/WSS protocol match

### Too many open files error

Increase file descriptor limit on macOS:

```bash
ulimit -n 10000
```

Or reduce watched directories with more specific patterns.

## Disclaimer

🤖🧞🪄 Implemented

## License

MIT License - see LICENSE file for details.

## Credits

Inspired by:

- [Browser Sync](https://browsersync.io/)
- [caddy-hot-loader](https://github.com/JHKennedy4/caddy-hot-loader)
- [CodeKit](https://codekitapp.com/)

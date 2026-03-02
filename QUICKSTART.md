# Quick Start Guide

Get hot-reloading working in under 5 minutes.

## Installation

### Option 1: Homebrew (Easiest)

```bash
# Install
brew tap yourusername/caddy-hot-reloader
brew install caddy-hot-reloader

# Configure
cp $(brew --prefix)/etc/caddy-hot-reloader/example.Caddyfile \
   $(brew --prefix)/etc/caddy-hot-reloader/Caddyfile
nano $(brew --prefix)/etc/caddy-hot-reloader/Caddyfile  # Edit paths

# Start
brew services start caddy-hot-reloader
```

### Option 2: Build from Source

```bash
cd /path/to/caddy-hot-reloader
./setup.sh   # Install xcaddy
./build.sh   # Build Caddy with plugin
./caddy run --config ./Caddyfile
```

## Configuration

Add `hot_reloader` to your `*.*.why` block:

```caddy
*.*.why {
    import tls_local

    hot_reloader {
        idle_timeout "30m"
    }

    root "/Users/why/Test Sites/{labels.2}/{labels.1}/www"

    encode gzip
    php_fastcgi 127.0.0.1:9000
    file_server
}
```

**Important**: Use a `route` block if you have handler order issues (see [README.md](README.md) Implementation Notes).

## Testing

1. Open your site: `https://test.laac-acc.why`

2. Check browser console for:

   ```
   [Caddy 🔄] Connected
   ```

3. Edit a PHP file → watch page reload

4. Edit a CSS file → watch styles update without reload

## Troubleshooting

**"hot_reloader" directive not recognized**

- Homebrew: `brew reinstall caddy-hot-reloader`
- Source build: Verify `./caddy` binary (not system `caddy`)

**WebSocket fails**

```bash
# Homebrew
tail -f $(brew --prefix)/var/log/caddy-hot-reloader.log

# Manual
./caddy run --config ./Caddyfile
```

**Files not triggering reload**

- Check `.gitignore` (file might be excluded)
- Verify file extension is watched (default: html, css, js, php)
- Enable DEBUG logging in Caddyfile

## Next Steps

- Read [README.md](README.md) for detailed configuration options
- See Implementation Notes in README for handler order issues
- Configure `.gitignore` to exclude cache/vendor directories

---

That's it! You now have hot-reloading at your real URLs. 🎉

---

That's it! You now have hot-reloading at your real URLs. 🎉

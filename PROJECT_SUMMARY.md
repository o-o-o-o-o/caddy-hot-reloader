# Project Summary

## What This Is

A Caddy v2 module that provides hot-reloading for wildcard local development sites (e.g., `*.*.why` domains). Built specifically for multi-site Kirby CMS development with per-site isolation and CSS injection.

## Key Features

- ✅ Per-site reload isolation (editing site A doesn't reload site B)
- ✅ CSS injection without full page reload
- ✅ Request-time site discovery from wildcard domains
- ✅ Idle watcher shutdown after configurable timeout (30m default)
- ✅ .gitignore support
- ✅ WebSocket-based browser communication
- ✅ Console branding as `[Caddy 🔄]`

## Project Files

### Core Go Files

- **hot_reloader.go** (414 lines) - Main module, HTTP middleware, config parsing, client script injection
- **manager.go** (351 lines) - Site lifecycle, WebSocket handling, idle cleanup
- **watcher.go** (281 lines) - fsnotify integration, .gitignore support

### Configuration

- **Caddyfile** - Production configuration
- **example.Caddyfile** - Sample configuration

### Documentation

- **README.md** (13.6 KB) - Comprehensive documentation with:
  - Features and comparison to alternatives
  - Installation (Homebrew + manual build)
  - Configuration examples
  - Implementation lessons learned
  - macOS service setup
  - Troubleshooting
- **QUICKSTART.md** (2.1 KB) - Ultra-concise setup guide
- **HOMEBREW_SETUP.md** (6.1 KB) - Complete Homebrew tap creation guide
- **LICENSE** - Apache 2.0

### Build Scripts

- **build.sh** - xcaddy build script
- **setup.sh** - Dependency installer

### Homebrew Distribution

- **Formula/caddy-hot-reloader.rb** - Homebrew formula
- **.github/workflows/update-formula.yml** - Auto-update workflow
- **.github/dependabot.yml** - Automatic dependency updates (weekly)

## Installation Paths

### Path 1: Homebrew Tap (Recommended for End Users)

```bash
brew tap o-o-o-o-o/caddy-hot-reloader https://github.com/o-o-o-o-o/caddy-hot-reloader
brew install caddy-hot-reloader
brew services start caddy-hot-reloader
```

### Path 2: Build from Source (Development)

```bash
./setup.sh && ./build.sh
./caddy run --config ./Caddyfile
```

## Architecture

```
Request Flow:
Browser → Caddy (TLS) → hot_reloader middleware → php_fastcgi/file_server
                                 ↓
                          Inject client script
                                 ↓
                          WebSocket endpoint (/hot-reload)
                                 ↓
                          Site Manager (per-site isolation)
                                 ↓
                          File Watcher (fsnotify)
```

### Key Design Decisions

1. **Handler Order via Route Block**
   - Problem: Caddy's handler order is non-deterministic outside routes
   - Solution: Use `route` block with explicit order: `encode → hot_reloader → php_fastcgi`

2. **Response Buffering for Injection**
   - Problem: Chunked responses + Content-Length mismatches
   - Solution: Buffer full HTML, inject once before `</body>`, delete Content-Length header

3. **Idle Watcher Shutdown**
   - Problem: Long-running watchers consume file descriptors
   - Solution: Track LastActivity per site, cleanup after idle_timeout with 0 connected clients

4. **Request-Time Discovery**
   - Problem: Can't pre-configure hundreds of site paths
   - Solution: Extract site from domain labels on first request, setup watcher dynamically

## Critical Implementation Notes

### Gzip + Injection Chain

- **Correct**: `encode` before `hot_reloader` in route block (request order)
- **Response flow**: php → hot_reloader (inject) → encode (gzip) → client
- **Why**: Injection must happen on uncompressed HTML

### HEAD Request Handling

- HEAD requests have no body but trigger injection logic
- Solution: Check `r.Method == http.MethodHead` and return early in Flush()

### macOS File Descriptors

- fsnotify uses kqueue (1 FD per directory)
- For 100+ sites with deep trees, can hit system limits
- Solutions: Aggressive exclude patterns, idle timeout, ulimit increase

## Dependencies

```
Go 1.21+
├── github.com/caddyserver/caddy/v2 v2.7.6
├── github.com/fsnotify/fsnotify v1.7.0
├── github.com/gorilla/websocket v1.5.1
└── github.com/sabhiram/go-gitignore
```

Build dependencies:

- xcaddy (for building custom Caddy)

## Testing Checklist

- [x] Core functionality (reload on file change)
- [x] CSS injection without full page reload
- [x] Per-site isolation (multiple tabs, different sites)
- [x] Handler order fix (route block)
- [x] Gzip compatibility
- [x] Console branding ([Caddy 🔄])
- [x] Idle watcher shutdown
- [x] Build process (xcaddy)
- [x] Runtime verification

## Distribution Roadmap

1. **Use This Repo as Homebrew Tap Source**
   - Keep `Formula/caddy-hot-reloader.rb` in this repository
   - Keep `.github/workflows/update-formula.yml` in this repository
   - Follow HOMEBREW_SETUP.md

2. **Tag First Release**
   - Tag v1.0.0 in this repo
   - Calculate SHA256 of release tarball
   - Update formula with correct URL and SHA256

3. **Test Installation**
   - `brew tap o-o-o-o-o/caddy-hot-reloader https://github.com/o-o-o-o-o/caddy-hot-reloader`
   - `brew install caddy-hot-reloader`
   - Verify service starts and hot-reload works

4. **Announce**
   - Share on Caddy community forum
   - Link from Kirby CMS community
   - Document on your blog/site

## Maintenance

### For New Releases

1. Tag release in this repo: `git tag -a v1.0.1 -m "..."`
2. Push tag: `git push origin v1.0.1`
3. GitHub Actions auto-creates PR in tap repo
4. Review and merge PR
5. Users update: `brew upgrade caddy-hot-reloader`

### For Bug Reports

- Plugin bugs: Open issue in this repo
- Formula bugs: Open issue in tap repo
- Caddy bugs: Report upstream

## Performance Characteristics

- **Memory**: ~50MB per 100 sites (estimated)
- **CPU (idle)**: Negligible
- **CPU (file change)**: Brief spike for broadcast
- **File Descriptors**: Linear with watched directories (~1 per dir on macOS)
- **Network**: WebSocket per browser tab (minimal overhead)

## Known Limitations

1. **Path Template**: Hardcoded `*.*.why` → `/Users/why/Test Sites/{label2}/{label1}/www`
2. **Network Filesystems**: fsnotify doesn't support NFS/SMB
3. **SCSS Compilation**: Watches `.scss` but treats as CSS (no compilation)
4. **macOS Only Testing**: Developed on macOS, should work on Linux/Windows but untested

## Future Enhancements (Optional)

- [ ] Configurable path templates
- [ ] Support multiple domain patterns
- [ ] Global reload mode (broadcast to all sites)
- [ ] Sourcemap support for CSS injection
- [ ] JavaScript HMR
- [ ] Rate limiting/debouncing
- [ ] Metrics endpoint
- [ ] Web UI for monitoring
- [ ] Max active sites LRU cap

## License

Apache License 2.0

---

**Status**: Production-ready for local development ✅

**Last Updated**: March 2, 2026

# caddy-hot-reloader — Agent instructions

Caddy plugin for local development. `README.md` describes the directives. `backlog/docs/HOMEBREW_SETUP.md` describes the tap.

## Release

1. Commit on `main` with a clean tree.
2. Run `./release.sh vX.Y.Z "title"`. It pushes `main`, creates the GitHub release, hashes the tarball and pushes the formula bump.
3. Do not edit `Formula/caddy-hot-reloader.rb` by hand.

## Install on a machine

Install only through Homebrew. Do not copy a built `./caddy` over an installed binary. A loose file in `/opt/homebrew/bin` shadows the brew build, and `brew link` then fails.

| | `why` | `uuu` |
|---|---|---|
| Upgrade | `brew upgrade o-o-o-o-o/caddy-hot-reloader/caddy-hot-reloader` | the same |
| Caddyfile | `/opt/homebrew/etc/Caddyfile` (plain file) | symlink to `~/🍉/Dotfiles/uuu/Caddyfile` |
| Restart | `brew services restart o-o-o-o-o/caddy-hot-reloader/caddy-hot-reloader` | `sudo launchctl kickstart -k system/local.uuu.caddy` |
| Site suffix | `.why` | `.uuu` |

- Do not use `brew services` on `uuu`. Caddy runs there as a LaunchDaemon.
- The first start of a new binary on `uuu` is killed once. launchd starts it again after about 10 seconds.
- Before a restart, run `caddy validate --config <Caddyfile>` with the new binary.
- Machine facts: DEVONthink note "Notes on uuu MacBook Air". Sync contract: `~/🍇/Scripts/util/uuu_includes`.
- Name the target machine in each plan.

## Rules

- `head_script <url>` adds a blocking `<script src>` as the first child of `<head>`. A URL that accepts a connection and never answers freezes every page of the site (measured 2026-11-19). `head_script <file>` (an absolute path) inlines the file instead, with no host to hang. The plugin does not wrap or configure the file: the file holds the complete script.
- The two Caddyfiles differ. `uuu` (`~/🍇/Dotfiles/uuu/Caddyfile`, in git) writes `{$SITE_TLD:uuu}` and has one `hot_reloader` block. `why` (`/opt/homebrew/etc/Caddyfile`, not in git) writes `.why` in full and has four. Edit each for its machine.
- `hot_reloader` injects before the first `</body>`, even inside an inline script string.
- `caddy.prev` is a copy of the old v2.11.7 binary. It is not installed. The README rollback steps copy to `/opt/homebrew/bin/caddy`, which no longer exists on `why`. Roll back through brew.

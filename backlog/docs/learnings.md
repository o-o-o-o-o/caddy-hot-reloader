# Learnings

- 2026-07-05: Moved supplemental root-level docs into `backlog/docs/` and added a docs index so the main README remains the primary entry point.
## TASK-3 (2026-07-05): Live-reload hardening from brandur.org/live-reload comparison
- Empty `extensions` config matched nothing instead of everything — zero-config setups (including our own Caddyfile) never broadcast a reload. Docs and code disagreed; the docs were the intent.
- gorilla/websocket forbids concurrent writers on one conn: the goroutine-per-message broadcast pattern could panic and take down Caddy. Fixed with a single sequential writer per site + 5s write deadline.
- The client's visibilitychange handler closed the socket when a tab was hidden but never reconnected on visible — switching browser tabs permanently killed hot reload for that page.
- fsnotify Create events must be stat'ed for directories: new subtrees need walk+Add, and files already inside must be classified since their events can fire before the watch exists.
- Debounce in a select loop: one timer, Stop-and-drain before Reset; coalesce burst into one message (any non-CSS change wins as a full reload).

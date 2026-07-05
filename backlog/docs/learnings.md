# Learnings

- 2026-07-05: Moved supplemental root-level docs into `backlog/docs/` and added a docs index so the main README remains the primary entry point.
## TASK-3 (2026-07-05): Live-reload hardening from brandur.org/live-reload comparison
- Empty `extensions` config matched nothing instead of everything — zero-config setups (including our own Caddyfile) never broadcast a reload. Docs and code disagreed; the docs were the intent.
- gorilla/websocket forbids concurrent writers on one conn: the goroutine-per-message broadcast pattern could panic and take down Caddy. Fixed with a single sequential writer per site + 5s write deadline.
- The client's visibilitychange handler closed the socket when a tab was hidden but never reconnected on visible — switching browser tabs permanently killed hot reload for that page.
- fsnotify Create events must be stat'ed for directories: new subtrees need walk+Add, and files already inside must be classified since their events can fire before the watch exists.
- Debounce in a select loop: one timer, Stop-and-drain before Reset; coalesce burst into one message (any non-CSS change wins as a full reload).

## TASK-4 (2026-07-05): Formula automation without a PAT
- The v0.7.0 release exposed that FORMULA_PAT was never added as a repo secret; the workflow failed at checkout with "Input required and not supplied: token".
- PRs created with the built-in GITHUB_TOKEN cannot trigger other workflows (CI/automerge) — that was the whole reason the PAT design existed. Pushing the formula bump directly to main with GITHUB_TOKEN sidesteps it: no secret, no expiry, one less failure mode.
- `gh workflow run --ref <branch>` runs the branch's version of a workflow — lets you validate workflow changes before merging them to main.

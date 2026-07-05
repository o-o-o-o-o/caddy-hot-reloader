---
id: TASK-3
title: Evaluate brandur.org live-reload article for applicable improvements
status: Done
assignee: []
created_date: '2026-07-05 16:05'
updated_date: '2026-07-05 16:36'
labels: []
dependencies: []
priority: medium
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Compared brandur.org/live-reload (and a pasted feasibility analysis) against our implementation. Project already exceeds the article on per-site isolation, CSS injection, gitignore support, idle shutdown, script auto-injection, and reconnect backoff. Applicable findings, ranked:
1. BUG: empty 'extensions' (the documented default, used by our own Caddyfile) makes matchesExtension() return false for every file — no reloads ever broadcast. Fix: return true when list is empty (watcher.go matchesExtension).
2. No debounce/coalescing — the article's core lesson. Each fsnotify event broadcasts immediately; editor saves and build bursts cause multiple rapid reloads and can hit half-written files. Add ~100ms per-site quiet-period coalescing (any non-CSS change in window => one full reload). Also fixes 'broadcast channel full, dropping message'.
3. Editor-junk defaults: article skips Vim's '4913' probe file and '~' backups. Our defaults exclude .DS_Store but not 4913, *~, .swp/.swo, #*#. Add to default excludes.
4. New subdirectories created after watcher setup are never watcher.Add()ed — files inside them don't trigger reloads. On Create event, if dir and not excluded, add watches recursively.
5. Concurrent-write panic risk: broadcastLoop spawns a goroutine per message per client calling WriteJSON on the same gorilla conn; gorilla forbids concurrent writers. Use per-client send channel + single writer.
6. Reload-on-reconnect: client reconnects after Caddy config reload but doesn't refresh, missing changes made while disconnected. Reload once on successful re-connect (not initial connect).
<!-- SECTION:DESCRIPTION:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implementation complete on branch feat/live-reload-hardening (branched from feat/one-button-release HEAD because the backlog/ infra only exists there; rebase onto main after one-button-release merges).

All 6 findings implemented:
1. matchesExtension: empty list now matches all (was: matched nothing, silently disabling reloads for zero-config setups incl. our own Caddyfile).
2. Debounce/coalesce: 100ms quiet period in Watch loop; burst -> single message; any non-CSS change wins as one full reload; multiple CSS -> single css message with empty file (client reloads all stylesheets).
3. isEditorJunk(): always-on filter for Vim 4913/backups~/.sw?, Emacs #autosave#/.#lock, .DS_Store.
4. watchIfNewDir(): Create events for dirs now walk+add watches recursively and classify files already present (covers recursive copies).
5. WebSocket writes serialized in broadcastLoop with 5s write deadline (was: goroutine per message per client -> gorilla concurrent-write panic risk).
6. Client script: reload once on reconnect after unexpected disconnect; BONUS fix: tab-hide previously killed the connection permanently (no reconnect on visibilitychange back to visible) - now resumes.

Also: removed dead sync.RWMutex field from HotReloader (fixed pre-existing go vet failure), gofmt'd pre-existing drift, README updated. Tests: 7 new tests in watcher_test.go (unit + fsnotify integration), full suite passes incl. -race. E2E verification via xcaddy build + live instance in progress.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Comparing brandur.org/live-reload against the code surfaced 6 fixes: empty extensions list matched NOTHING (zero-config setups never broadcast reloads — docs said the opposite); added 100ms debounce/coalescing of event bursts; always-on editor-junk filter (Vim 4913/backups/swaps, Emacs autosaves, .DS_Store); directories created after watcher setup are now watched on Create (walk + classify pre-existing files); WebSocket broadcasts serialized to a single writer with deadline (gorilla panics on concurrent writes); client reloads once after unexpected reconnect AND survives tab hide/show (visibilitychange previously killed the connection permanently). Verified: unit + fsnotify integration tests, -race, and live e2e against an xcaddy build.
<!-- SECTION:FINAL_SUMMARY:END -->

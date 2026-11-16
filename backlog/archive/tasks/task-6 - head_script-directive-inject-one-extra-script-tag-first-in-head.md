---
id: TASK-6
title: 'head_script directive: inject one extra script tag first in <head>'
status: Done
assignee: []
created_date: '2026-11-16T23:25:00Z'
updated_date: '2026-11-16T23:40:00Z'
labels:
  - model/sonnet
dependencies: []
priority: medium
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Prefactor for dev/faults (Test Sites). The capture script must run before every page script, and the reload client goes before </body> (hot_reloader.go:267), which is too late. The plugin stays a reloader: the directive takes a URL and knows nothing about faults.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 hot_reloader accepts head_script <url>; each injected HTML response carries <script src=url> as the first child of <head>, before any page script
- [x] #2 Without the directive the response is byte-identical to the response of the current build
- [x] #3 A response with no <head> gets no head script and one warning in the plugin log; the reload client still goes in
- [x] #4 Go tests cover the three cases above; each new test failed once before the code
- [x] #5 The previous caddy binary on why is kept beside the new one, and the rollback command is in the README
- [x] #6 Seams under test: the HTTP response body of the handler (hot_reloader_test.go)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Budget: $2.98 ($1–$7), ~9.0M tokens

Context: (2026-11-16) entry points — hot_reloader.go:267 (body injection), hot_reloader.go:353 (clientScript), hot_reloader_test.go, build.sh; consumer — ~/Test Sites/dev/faults TASK-1; live config — /opt/homebrew/etc/Caddyfile (five hot_reloader blocks)
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
head_script <url> injects a script tag first in <head>; no <head> logs one warning and the reload client still goes in; no directive leaves responses byte-identical. Four behaviour tests plus an extra-argument test. Review run (medium): extra-arg fix applied, rest dismissed; empty-body bug split to TASK-7. Binary built as ./caddy (v2.11.9), old binary kept as ./caddy.prev, rollback in README. Not installed on why.
<!-- SECTION:FINAL_SUMMARY:END -->

---
id: TASK-5
title: "\U0001F41B hot_reloader. Extensions allowlist to stop sourcemap/scss-source reload storms"
status: Done
assignee: []
created_date: '2026-09-30 18:55'
updated_date: '2026-09-30 19:04'
labels:
  - model/sonnet
dependencies: []
priority: medium
ordinal: 5000
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Confirm which Caddyfile is actually live (launchctl list / ps on the running process), not assumed from the repo
- [x] #2 If the live config lacks an extensions allowlist, add one; if it already has one, record that and don't claim a fix
- [x] #3 Repo's own Caddyfile and example.Caddyfile reflect the same extensions allowlist the live config already uses, so a fresh install/reinstall starts from a correct reference
- [x] #4 go test ./... passes
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Investigated a user report of 3-4 browser refreshes per CSS edit. Traced it to the Scripts repo's asset-compile daemon (fixed there, TASK-184) — not a bug in this plugin. Corrected my own initial mistake: I first edited this repo's root Caddyfile assuming it was live config; it isn't — the Homebrew service runs /opt/homebrew/etc/Caddyfile, which already had the extensions allowlist I was about to add. Brought the repo's own Caddyfile + example.Caddyfile in line with that already-correct live config for consistency (no live change made or needed).
<!-- SECTION:FINAL_SUMMARY:END -->

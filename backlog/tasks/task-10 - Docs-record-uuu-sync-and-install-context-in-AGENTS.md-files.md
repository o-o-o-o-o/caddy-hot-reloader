---
id: TASK-10
title: 'Docs: record uuu sync and install context in AGENTS.md files'
status: In Progress
assignee: []
created_date: '2026-11-19T22:01:00Z'
updated_date: '2026-11-19T22:31:00Z'
labels:
  - model/sonnet
dependencies: []
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
A session needed five sources to learn how uuu works: SSH, the sync contract, the DEVONthink note, Test Sites AGENTS.md and the Scripts docs. Add the missing pointers: repo AGENTS.md for caddy-hot-reloader, a Sync section in Test Sites/AGENTS.md, data and host rules in dev/faults/AGENTS.md, and Machines and data rules in the global CLAUDE.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 caddy-hot-reloader has AGENTS.md and CLAUDE.md with release, install and head_script rules
- [x] #2 Test Sites/AGENTS.md has a Sync section
- [x] #3 dev/faults/AGENTS.md records the data layout decision and the host rule
- [x] #4 Global CLAUDE.md has the Machines rule and the data rule
- [x] #5 The llm-learnings entry and the preference tag are logged
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Docs now carry the uuu sync, install and data rules: repo AGENTS.md and CLAUDE.md, Test Sites Sync section, faults data and host rules, global Machines and data rules, learnings entry. Corrections found while writing: dev/faults/www is its own repo; why's Caddyfile is not in git and writes .why in full, uuu's uses {$SITE_TLD:uuu}. Review: global CLAUDE.md change offered for the human to read; not run.
<!-- SECTION:FINAL_SUMMARY:END -->

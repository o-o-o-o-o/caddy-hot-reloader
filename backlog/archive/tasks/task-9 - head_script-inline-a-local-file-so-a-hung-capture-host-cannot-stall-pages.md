---
id: TASK-9
title: 'head_script: inline a local file so a hung capture host cannot stall pages'
status: Done
assignee: []
created_date: '2026-11-19T21:54:00Z'
updated_date: '2026-11-23T22:03:27Z'
labels:
  - model/sonnet
dependencies: []
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Measured 2026-11-19 on why: a head_script URL that accepts the connection and never answers leaves every page in readyState loading with an empty body (3 s and still blocked). The tag is a blocking script by design, so it runs before every page script. Make head_script accept a local file path as well as a URL. The plugin reads the file, caches it, reloads it when the mtime changes, and emits an inline script as the first child of head. The host then needs no name, so why and uuu use the same config. The settings that capture.php adds today (deny, warn, cap, max_per_load, batch_ms) must reach the inline script. The capture POST target still differs per machine (dev.faults.why, dev.faults.uuu): keep it in config, not in the repo.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 head_script <path> emits an inline script as the first child of head; head_script <url> keeps its current output
- [x] #2 The file is cached and reloads after an mtime change, without a Caddy restart
- [x] #3 A body that contains a literal closing script tag cannot end the inline script early
- [x] #4 A test with a never-answering host shows that inline mode renders the page at once
- [x] #5 README documents both forms and the blocking behaviour of the URL form
- [x] #6 Release as a new version; formula bumped by release.sh
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Context: (2026-11-19) review of faults found that src/faults.js sets self from document.currentScript.src. Inline mode leaves it empty, so the capture script's own frames are not skipped. Pass a self marker (the capture URL) in the config. See ~/Test Sites/dev/faults/www/src/faults.js:20.
<!-- SECTION:NOTES:END -->

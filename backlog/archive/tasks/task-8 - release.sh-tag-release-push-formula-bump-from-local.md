---
id: TASK-8
title: 'release.sh: tag, release, push formula bump from local'
status: Done
assignee: []
created_date: '2026-11-17T04:20:00Z'
updated_date: '2026-11-17T04:50:00Z'
labels:
  - model/sonnet
dependencies: []
ordinal: 5000
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 release.sh publishes a release and pushes the formula bump
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Budget: small. Context: update-formula.yml cannot push to protected main (GITHUB_TOKEN); v0.7.x formula bumps were pushed by hand. User chose a local script (path 1).
<!-- SECTION:NOTES:END -->

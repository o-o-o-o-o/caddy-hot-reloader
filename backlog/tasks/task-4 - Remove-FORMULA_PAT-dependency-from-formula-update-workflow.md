---
id: TASK-4
title: Remove FORMULA_PAT dependency from formula-update workflow
status: Done
assignee: []
created_date: '2026-07-05 21:21'
updated_date: '2026-07-05 21:23'
labels: []
dependencies: []
priority: medium
ordinal: 4000
---

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
FORMULA_PAT was required by the fully-automatic release flow (53331a6) but the secret was never added, so every release-published run failed at checkout. Root cause of the PAT design: GITHUB_TOKEN-created PRs can't trigger CI/automerge workflows. Removed the PR ceremony entirely — the workflow now seds the formula and pushes straight to main with the built-in token (contents: write), with a no-op guard and curl -f so a bad tag fails loudly. Validated via workflow_dispatch from the branch before merging: checkout + hash + no-op path all green with zero secrets configured.
<!-- SECTION:FINAL_SUMMARY:END -->

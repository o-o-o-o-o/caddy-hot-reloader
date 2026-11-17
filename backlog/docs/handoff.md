For: dev/faults TASK-1 — after TASK-7, 2026-11-16

- TASK-7 is done. Install of `./caddy` on `why` is approved by the user but not done. Rollback binary: `./caddy.prev` (v2.11.7), command in README. `./caddy` v2.11.9 was built before the TASK-7 fix; rebuild first if TASK-7 must ship.
- User said "publish then install". "Publish" is unclear (release, push, formula update?). Ask before any outward step.
- Install steps, machine `why`: install the binary; add `head_script https://dev.faults.why/capture.php` to all four `hot_reloader` blocks in `/opt/homebrew/etc/Caddyfile` (php74, php81, nested fallback, `*.*.why`); reload Caddy; open a real site; confirm a file in `~/Test Sites/dev/faults/data/`.
- Then tick AC #7 and #9 of dev/faults TASK-1. Code is uncommitted in `~/Test Sites/dev/faults/www`. The subagent suggests `/code-review high` on that path.
- Gotcha, no task yet: `hot_reloader` injects before the first `</body>`, even inside an inline script string.
- Map: `~/Test Sites/dev/faults/.backlog/docs/doc-1 - Faults-initiative-map.md`. Next after TASK-1: `~/🍇/Scripts` TASK-349 on `sonnet`.

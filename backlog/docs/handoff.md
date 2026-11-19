For: TASK-9 — after TASK-10, 2026-11-19

- Done 2026-11-19: v0.8.0 runs on `why` and `uuu` through brew. Both Caddyfiles load `capture.php` through `head_script`. A page on each machine wrote a file in `~/Test Sites/dev/faults/data/` (`<project>.<tld>.jsonl`). Faults TASK-1 and Dotfiles TASK-95 are archived.
- TASK-9 (inline a local file): a never-answering host freezes every page. Test method: `nc` listener held open with `tail -f /dev/null | nc -l 9099`, throwaway Caddy on `:8089` with `admin localhost:2999`. A bare `nc -l` exits at once and gives a false pass.
- TASK-9 also needs a `self` marker for `~/Test Sites/dev/faults/www/src/faults.js:20` (empty when the script is inline). See the Context note on TASK-9.
- README rollback lines 212-218 copy to `/opt/homebrew/bin/caddy`. That file no longer exists on `why`. Fix them to the brew path. No task yet.
- `brew link --overwrite caddy-hot-reloader` was not run on `why`: a bare `caddy` is not on PATH. Ask the user first.
- Open follow-ups: faults TASK-5 (collect.sh uses the live data file), faults TASK-2 (triage must merge `*.why.jsonl` and `*.uuu.jsonl`).
- Gotchas: `hot_reloader` injects before the first `</body>`, even inside an inline script string. `https://dev.faults.why/tests/` answers "File not found": use `/tests/index.html`.
- Docs: `AGENTS.md` (this repo) holds the install table. Machine facts: DEVONthink "Notes on uuu MacBook Air".

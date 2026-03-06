---
mode: agent
tools:
  - run_in_terminal
  - get_terminal_output
  - runSubagent
  - get_changed_files
  - grep_search
  - read_file
  - apply_patch
  - create_file
  - vscode_askQuestions
description: Prepare and publish a release tag with generated notes and formula update
---

Run the repository release-tag workflow exactly in this order.

1. Pre-flight checks:

- Verify current branch is `main`.
- Verify working tree is clean (or clearly report non-release changes).
- Get latest version tag using: `git tag -l | sort -V | tail -1`.

2. Suggest next version:

- If branch is `main`, increment patch (e.g., `v0.6.1` -> `v0.6.2`).
- Show proposed version and ask for confirmation before creating tag.

3. Generate tag message:

- Collect commits since last tag: `git log <last_tag>..HEAD --oneline`.
- Build a concise markdown changelog list.
- Ask for confirmation or edits.

4. Create and push:

- Create annotated tag.
- Push `main` then push tag.
- If any command fails, stop and explain exactly what failed.

5. Build verification reminder:

- Provide the GitHub Actions link and note expected checks.

6. Homebrew formula update:

- Compute source tarball SHA256 for the new tag.
- Update `Formula/*.rb` URL and SHA256.
- Show diff.
- Commit with `chore: update Homebrew formula for <tag> release` and push.

7. Final confirmation output:

- Show latest tag and summarize pushed changes.
- Include any manual follow-ups needed.

Important behavior:

- Use non-interactive git commands.
- Never use destructive git commands.
- If tree is dirty, ask before proceeding.
- Keep prompts concise and require explicit user confirmation before tagging or pushing.

# CLAUDE.md - Git

## Git Workflow

Never run `git commit` or `git push` automatically after making changes — always wait for an explicit instruction from the user.

- "tee commit" — run `git commit` only, then stop.
- "tee commit ja push" — run `git commit` followed immediately by `git push`, both in the same turn.

## Branch naming

These prefixes follow standard Git conventions used in code projects. Adapt as needed for documentation-only repos.

Full conventions in `README-Git.md`. Prefix summary:

| Prefix | Use |
| --- | --- |
| `feature/` | New feature |
| `fix/` | Bug fix (normal) |
| `hotfix/` | Urgent production fix — branch directly from `main` |
| `docs/` | Documentation |
| `chore/` | Maintenance, build, CI, dependency updates |
| `refactor/` | Code cleanup without functional changes |
| `experiment/` | Experiment, may not reach production |

## Merge Strategy

Use merge commits, never squash. Every individual commit must stay visible in `main` so `git log -S`/`-G`, `git bisect`, and `git blame` stay precise; the merge commit records the feature/branch boundary.

- `git config merge.ff false` is set for this repo, so `git merge` always creates a merge commit (no fast-forward).
- On GitLab merge requests, choose the "Merge commit" method (not "Squash" or "Fast-forward").
- Keep all branches — do not delete branch refs after merging.
- Overview vs detail: `git log --first-parent main` shows per-feature summaries; plain `git log` shows every commit.

## Commit Style

Commit messages are freeform — typically a topic or filename plus a date.

Format:

```shell
README-Plesk.md
Muutokset opendesk.fi nimipalveluun - 14.4.2026
Code-AI/CLAUDE-OD-Git.md
```

### Best practices

- **Subject line only** — one line is enough for most changes; no period at the end.
- **Topic + date** — use `<topic or filename> - DD.M.YYYY` when the date adds context (e.g., changes tied to a specific event or meeting).
- **Multi-file changes** — use a descriptive topic instead of listing every file: `Palvelut/Claude päivitys` rather than a file list.
- **Body for context** — add a blank line after the subject and write a short body only when the reason for the change is not obvious from the subject.
- **Language** — Finnish for Finnish-content commits; English is fine for code or tool files.
- **Atomic commits** — one logical change per commit.
- **No auto-commit** — never run `git commit` without an explicit instruction from the user (see Git Workflow above).

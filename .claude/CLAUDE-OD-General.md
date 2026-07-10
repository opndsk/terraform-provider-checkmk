# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Purpose

This is a **Finnish-language knowledge base** for Open Desk (OpenDesk Oy) operations — not a software project. It contains documentation, client records, infrastructure notes, and service guides. No build system or test runner exists.

## Linting Commands

```shell
npm run lint:md          # Markdown lint (all rules)
npm run lint:md:fix      # Auto-fix markdown issues
npm run lint:spell       # Spell-check (EN_GB, EN_US, FI, SV)
npm run lint:md009:fix   # Fix trailing spaces
npm run lint:md010:fix   # Fix hard tabs
npm run lint:md032:fix   # Fix blank lines around lists
```

**Lint cadence:** Run Markdown lint (`npm run lint:md`) and spell-check (`npm run lint:spell`) **always before a `git push`**. Do NOT run them automatically after each Markdown edit. Outside of a push, run them only when explicitly asked (e.g. "lint md ja spell").

## Spell Check Word Lists

Before running any spell-check (`npm run lint:spell` or `npm run lint:spell:file`), move every word in the `words` array of `.vscode/cspell.json` into `.vscode/cspell-words.txt`, then empty that `words` array (`"words": []`). The text file is the single source of truth for project words; keep `cspell.json` free of inline words.

- Insert the moved words into `.vscode/cspell-words.txt` in the file's existing alphabetical (case-insensitive) order, leaving the other lines untouched.
- Skip any word that already exists in `.vscode/cspell-words.txt` (case-insensitive) to avoid duplicates.
- When a spell-check reports a correctly spelled but unknown word, add it to `.vscode/cspell-words.txt` (not to `cspell.json`), in the same alphabetical position.

## Searching Content

```shell
rg "<term>"    # Search across all documentation
rg --files     # List tracked files quickly
```

## Presenting Choices

When asking the user to choose between alternatives, always present the options as a numbered list (1. 2. 3. ...). This lets the user answer with just the number.

- Number every distinct option, including a recommendation if there is one.
- Keep each option on its own line with a short label and, when helpful, a one-line trade-off.

## Agent Notes

- Context7 is available in Claude through MCP and can be used for up-to-date library and API documentation lookups when needed.

## Coding Style & Naming Conventions

- Documentation files: `README-<Topic>.md`
- Date-stamped folders: `2025-04-04 - Tarjous .../`
- Update logs: `z-paivitys-YYYY-MM-DD-HHMM`
- Keep files in UTF-8, but use ASCII when possible for new content.
- Finnish text should use Scandinavian characters (å, ä, ö); the ASCII preference does not apply to Finnish.
- Naming pattern for update logs: `z-paivitys-YYYY-MM-DD-HHMM`.

## Terraform (GitHub IaC)

`terraform/github/` manages the GitHub organization structure (teams, repos, members). See `terraform/README-Terraform-GitHub.md` for setup instructions. Terraform state and `.tfvars` files are gitignored.

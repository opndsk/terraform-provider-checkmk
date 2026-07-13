# CLAUDE.md - Markdown

## Markdown Style

- Use Markdown with clear headings and short paragraphs.
- In Finnish headings, use sentence case. Capitalize only the first word and proper nouns, not ordinary Finnish words in the middle of the heading
- Prefer fenced code blocks with `shell` for commands.
- Always specify a language for fenced code blocks — use `shell`, `ts`, `json`, `txt`, or another appropriate identifier. Never leave the opening fence bare (` ``` ` with no language).
- In Markdown, leave a blank line after every heading
- In Markdown, leave a blank line between any normal paragraph or sentence/label and the following list or code block.
- In Markdown, wrap URLs in angle brackets, e.g. <https://example.com>.
- In Markdown, wrap email addresses in angle brackets, e.g. <name@example.com>, unless they are inside a code block.
- Use "-" for bullet lists in Markdown (avoid "*").
- In Finnish Markdown, format euro amounts with a decimal comma and thousand separator dot, e.g. `5.880,56 €`.
- In Markdown bullet lists, always start the first word of every bullet with a capital letter.
- Do not wrap long sentences across multiple lines; keep each paragraph on a single line.
- Keep headings consistent and prefer short, action-oriented titles.
- Indentation: 2 spaces in nested Markdown lists when needed.
- In Markdown tables, always use spaces around the dashes in the separator row: `| --- | --- |`, not `|---|---|`. This is required by MD060.
- Do not use `---` horizontal rules as section separators between headings. Use headings alone to structure the document.
- Mark every command, folder name, and file name in prose with inline code formatting (backticks), e.g. `installimage.conf`, `Asennus-2026-07-08/`, `apt-get update`.

## Checklist items (`- [ ]` / `- [x]`)

`- [x]` means the checklist item's task is **done/resolved** — it does not mean that item's specific option was chosen, or that it was rejected. A conditional item (e.g. "if option B is chosen, do X") gets checked off `[x]` once the overall decision is finalized, even if that specific action was never carried out because a different option was picked.

When a checklist item in a planning/notes document resolves (decided, completed, or made moot by a different choice), keep the original line — do not delete or rewrite it. Change it to `- [x]` and add one short indented sub-bullet explaining the outcome, e.g.:

```markdown
- [x] Jos vaihtoehto B: valitse VM 108:lle vapaa osoite DEV3:n `vmbr1`-verkosta.
  - Ei enää kesken: A valittiin, joten tätä ei tarvinnut tehdä - jätetty näkyviin varasuunnitelmaa varten.
```

This preserves the decision history in the document instead of erasing what needed to be considered.

## Bullet Punctuation (Finnish)

- If a bullet item contains multiple sentences, separate the sentences with a period inside the bullet, but do not add a final period at the end of the item

## Shell code blocks in Markdown

When a section contains multiple sequential shell commands with explanatory text between them, combine them into a single fenced code block. Use `# comment` lines inside the block instead of prose paragraphs between blocks.

Example — prefer this:

```shell
# Asenna riippuvuudet
composer install

# Käynnistä palvelin
php artisan serve
```

Not this:

Asenna riippuvuudet:

```shell
composer install
```

Käynnistä palvelin:

```shell
php artisan serve
```

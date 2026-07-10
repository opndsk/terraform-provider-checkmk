# CLAUDE.md - Docusaurus

## Docusaurus

### Docusaurus Docs Guidelines

- Every doc must define `title`, `id`, and `slug` in frontmatter.
- Use absolute slugs that start with `/` to avoid folder-prefixed URLs.
- Keep `id` stable once published to avoid breaking references.
- Link between docs using slug-based URLs (`/docs/<slug>`), not file paths.
- For doc folders, define `_category_.json` with `label` and `position` to control sidebar order.
- In MDX, avoid raw `<...>` tokens for URLs/emails; use markdown links with escaped angle brackets, e.g. `[\<https://example.com\>](https://example.com)` and `[\<name@example.com\>](mailto:name@example.com)`.

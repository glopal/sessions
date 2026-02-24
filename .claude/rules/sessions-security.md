---
description: Security review before writing to .sessions/
globs: .sessions/**
---

# Sessions Security Rule

Before writing any content to `.sessions/` files (session summaries, artifacts, or edits), scan the content for:

1. **Credentials** — API keys, tokens, passwords, secrets, connection strings
2. **Internal URLs** — Internal hostnames, VPN endpoints, intranet addresses
3. **PII** — Email addresses, phone numbers, employee names, IP addresses
4. **Proprietary business logic** — Algorithms, trade secrets, financial data
5. **Internal codenames** — Project codenames, internal team names, unreleased product names

If any of the above are detected:

- Flag the specific items found
- Suggest redacted alternatives (e.g., replace `api-key-abc123` with `<REDACTED_API_KEY>`, replace `internal.company.com` with `<INTERNAL_URL>`)
- Only proceed after the user explicitly confirms the content is safe to commit

This repository is public. All content in `.sessions/` will be visible to anyone.

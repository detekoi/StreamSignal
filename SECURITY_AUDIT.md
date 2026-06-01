# Security Posture Summary

## Scope

This document records the public-facing security summary for StreamSignal.

Latest review: June 1, 2026.

Review focus:

- secrets handling
- SQLite persistence
- logs and diagnostics
- outbound integration safety
- dependency review

## Executive Summary

The highest-severity issue found in this pass was that secret values could previously be persisted directly in SQLite through destination configuration, legacy test credential settings, and restart-safe Bluesky `Live Now` session tracking.

That issue has been remediated:

- runtime secrets are now stored through Windows Credential Manager
- SQLite stores only secret references instead of raw secret values
- legacy SQLite rows are migrated away from raw-secret storage when the app reads them
- automated tests verify that raw secrets are not left in SQLite

## Summary of Findings and Actions

### Fixed: secret values were previously persisted in SQLite

Severity:

- high

Affected areas at the time of review:

- destination `config_json`
- legacy test credential settings
- `bluesky_live_now_sessions.credential_key`

Why it mattered:

- local database access could expose live secrets directly
- it conflicted with the intended local security posture
- it increased the blast radius of local compromise, backup leakage, or accidental file sharing

Remediation:

- added a secret-store-backed persistence layer
- wired the runtime app to Windows Credential Manager
- kept only secret references in SQLite
- added automatic migration for older raw-secret rows when read through the app

Validation:

- repository tests confirm SQLite no longer stores raw destination, settings, or `Live Now` secrets
- legacy migration behavior is covered by tests

### Validated: diagnostics and logs avoid direct secret output

Severity:

- medium

What was checked:

- diagnostics output
- app log messages
- execution summary and recovery messaging

Result:

- current diagnostics and logs do not intentionally print webhook URLs, access tokens, app passwords, or other credential secrets
- diagnostics include operational context, but not raw stored secrets
- pending `Live Now` session values are redacted in exported diagnostics output

Residual caution:

- future logging changes should continue treating secret values as excluded data

### Validated: outbound integration clients use explicit request timeouts

Severity:

- low

What was checked:

- Discord publisher
- Bluesky publisher
- Mastodon publisher

Result:

- all default HTTP clients use explicit `10s` timeouts
- error responses are read with bounded body limits rather than unbounded reads

### Fixed: frontend build tooling had vulnerable dev dependencies

Severity:

- medium

What was checked:

- `npm audit`
- `npm outdated`

Finding:

- the previous frontend toolchain pulled vulnerable `esbuild` versions through old Vite and Vitest packages
- the issue affected local development/build tooling, not shipped runtime React dependencies

Remediation:

- upgraded Vite, the Vite React plugin, Vitest, V8 coverage, jsdom, and TypeScript
- updated TypeScript module-resolution settings for the newer toolchain
- changed the GitHub quality gate from production-only npm audit to full `npm audit`

Validation on June 1, 2026:

- `npm audit` reported `0` vulnerabilities
- remaining npm outdated entries are React 18 to React 19 migration items, not security fixes

### Fixed: Go vulnerability scan reported a non-reachable vulnerable module

Severity:

- low

What was checked:

- `govulncheck -show verbose ./...`

Finding:

- `golang.org/x/sys v0.34.0` was present with GO-2026-5024
- govulncheck reported no reachable vulnerable symbols in StreamSignal code

Remediation:

- upgraded `golang.org/x/sys` to `v0.44.0`, the fixed version reported by govulncheck

Validation on June 1, 2026:

- `govulncheck ./...` reported no vulnerabilities

### Improved: Go dependency vulnerability scanning is part of the quality gate

Status:

- `govulncheck` has been added to the GitHub quality gate workflow
- the live GitHub Actions quality gate has now been verified successfully against the public repository
- the workflow now builds the frontend before backend tests so `go test ./...` can validate the Wails embed path in CI
- the workflow uses a current Go toolchain declaration and Node 24-capable action versions

### Improved: stored secrets now use masked-edit UX in the desktop UI

Severity:

- low

What changed:

- destination secret inputs now render as a stable masked token instead of hydrating the live secret back into the visible form
- unchanged masked inputs preserve the stored secret on save
- replacing or clearing a secret still works through the existing save flow

Assessment:

- this is a safer default for a local-first desktop app
- it reduces casual secret exposure during ordinary configuration edits
- users still retain full control to replace or remove stored secrets when needed

### Reviewed: Bluesky rich cards and thumbnail uploads

Severity:

- low

What was checked:

- direct card-thumbnail URL handling
- local image upload handling
- frontend resizing before save
- backend thumbnail fetch, decode, and upload limits
- diagnostics and logging around destination configuration

Result:

- thumbnail URLs must be valid HTTP or HTTPS URLs before fetch
- fetched thumbnails use the existing HTTP timeout and bounded reads
- uploaded thumbnail data must be an image data URL and is rejected before decode if the encoded payload is too large
- decoded and fetched thumbnails are limited to Bluesky's current 1 MB blob size expectation
- thumbnail data is not a credential secret and is not exported through diagnostics, but it is stored locally in destination configuration when a local image is selected

Residual caution:

- local thumbnail images can still be personal content, so users should treat the local StreamSignal database as private application data
- if StreamSignal ever accepts remote or shared configuration from untrusted users, thumbnail URL fetching should be revisited as a stricter SSRF boundary

### Reviewed: Discord webhook posting and additional images

Severity:

- low

What was checked:

- webhook publishing
- optional image URL embeds
- optional uploaded image embeds
- selected-destination targeting
- destination-level End Stream enablement

Result:

- Discord webhook URLs are still treated as destination secrets and stored through the secret-store-backed persistence layer
- image URLs must be valid HTTP or HTTPS URLs before posting
- uploaded image data must be an image data URL and is rejected before decode if the encoded payload is too large
- decoded uploaded Discord images are limited to 8 MB before posting
- End Stream posts now respect destination-level enablement, reducing accidental cross-posting risk
- a local secret-pattern scan after this pass found only placeholders, test fixtures, generated model field names, and expected secret-handling code references

## OWASP Alignment Notes

This pass especially aligns with:

- OWASP Secrets Management Cheat Sheet
  - centralize and control secret storage instead of leaving secrets in configuration or plaintext persistence
  - https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html
- OWASP Logging Cheat Sheet
  - exclude or protect sensitive data such as access tokens, passwords, and other primary secrets from logs
  - https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
- OWASP ASVS project guidance
  - use structured verification standards to identify gaps and remediation work
  - https://owasp.org/www-project-application-security-verification-standard/

## Follow-Up Recommendations

1. consider whether diagnostics export needs multiple redaction tiers before broader release
2. continue reviewing destination targeting, recovery, and outbound adapters whenever new platforms or workflows are added
3. keep the Go toolchain and GitHub Action versions current as part of regular maintenance

# Security Posture Summary

## Scope

This document records the first public-facing Milestone 9 security summary for StreamSignal as of May 31, 2026.

Review focus:

- secrets handling
- SQLite persistence
- logs and diagnostics
- outbound integration safety
- dependency review

## Executive Summary

The highest-severity issue found in this pass was that secret values could previously be persisted directly in SQLite through destination configuration, Test Mode settings, and restart-safe Bluesky `Live Now` session tracking.

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
- Test Mode credential settings
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

### Validated: frontend production dependency audit is clean

Severity:

- low

What was checked:

- `npm audit --omit=dev --json`

Result on May 31, 2026:

- `0` production vulnerabilities reported

### Improved: Go dependency vulnerability scanning is now part of the quality gate

Status:

- `govulncheck` has been added to the GitHub quality gate workflow
- the live GitHub Actions quality gate has now been verified successfully against the public repository
- the workflow now builds the frontend before backend tests so `go test ./...` can validate the Wails embed path in CI
- the workflow uses a current Go toolchain declaration and Node 24-capable action versions

### Improved: stored secrets now use masked-edit UX in the desktop UI

Severity:

- low

What changed:

- destination and Test Mode secret inputs now render as a stable masked token instead of hydrating the live secret back into the visible form
- unchanged masked inputs preserve the stored secret on save
- replacing or clearing a secret still works through the existing save flow

Assessment:

- this is a safer default for a local-first desktop app
- it reduces casual secret exposure during ordinary configuration edits
- users still retain full control to replace or remove stored secrets when needed

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
2. continue reviewing Test Mode, recovery, and outbound adapters whenever new platforms or workflows are added
3. keep the Go toolchain and GitHub Action versions current as part of regular maintenance

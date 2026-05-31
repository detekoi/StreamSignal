# Release Readiness Checklist

This checklist replaces the old implementation plan as the active next-phase document for StreamSignal.

Current reality:

- core workflows are implemented
- automated tests and CI gates are in place
- the app is still in workflow-validation and usability-refinement, not final MVP signoff

## Release Goal

Ship StreamSignal in a way that is:

- safe for real user credentials
- easy to install and update
- understandable for first-time users
- maintainable through normal CI, test, and dependency workflows

## Product Readiness

- verify all core workflows manually on a release candidate build
- verify the current UI is understandable to a first-time user without project context
- verify the Home, Destinations, Settings, and Logs tabs follow a clear human workflow
- confirm Preview and Dry Run stay network-free
- confirm Go Live publishes correctly to Discord, Bluesky, and Mastodon
- confirm End Stream clears Bluesky `Live Now`
- confirm optional end-stream posting works as expected
- confirm Test Mode never routes to production targets
- confirm duplicate warning and force-confirm flows behave correctly
- confirm pending `Live Now` recovery works after restart
- confirm no dummy, stale, or misleading workflow state survives between real app sessions

## Packaging And Installer

- produce a clean Windows release build with `wails build`
- validate installer metadata, icon, and version information
- verify install path behavior on a clean Windows machine or VM
- verify first launch after install succeeds without manual setup steps
- verify upgrade behavior from one build to the next
- decide how release artifacts will be attached to GitHub releases

## Secrets And Local Security

- verify Windows Credential Manager entries are created, reused, and updated correctly
- verify secrets never appear in SQLite, logs, or diagnostics
- verify masked secret-edit UX works in release builds
- verify uninstall behavior expectations for retained or removed secrets are documented

## First-Run UX

- document the minimum setup path for a new user
- verify destination setup is understandable without internal project knowledge
- verify Test Mode setup is understandable and clearly separated from production setup
- verify error messages are actionable when credentials or URLs are invalid
- verify recovery wording is understandable to a non-technical user
- verify layout hierarchy feels like a functional desktop tool rather than a landing page or form dump
- verify users can understand how templates, destination selection, preview, and live execution relate to each other

## Documentation

- keep `README.md` focused on install, run, test, and project-doc entry points
- keep `SECURITY.md` and `SECURITY_AUDIT.md` aligned with the current posture
- keep `TEST_VALIDATION.md` aligned with real CI expectations
- add release notes guidance before the first public release
- decide whether a dedicated user setup guide should be added

## CI And Maintenance

- keep the GitHub quality gate green on `main`
- keep required branch protection checks aligned with the live job names
- keep Go toolchain and GitHub Action versions current
- review `govulncheck` and `npm audit` output regularly
- re-run frontend coverage reporting when significant UI changes land

## Release Decision

Before calling the app release-ready, confirm:

- packaging is repeatable
- installer behavior is understood
- first-run setup is documented
- usability and workflow validation are complete enough that the app can be trusted by a first-time real user
- quality gate is green
- no open high-severity security concerns remain

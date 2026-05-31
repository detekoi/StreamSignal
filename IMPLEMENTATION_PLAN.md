# StreamSignal Historical Implementation Plan

## Purpose

This file records the implementation history and milestone structure that took StreamSignal from a new repo to its current post-MVP state.

It is now better treated as project history than as the primary public-facing documentation set.

Core constraints:

- all workflow logic lives in Go
- the React frontend stays thin
- test coverage grows with each milestone
- no platform secrets are stored in SQLite
- Bluesky Milestone 6 is about `Live Now` only, not profile mutation

## Current Status

Milestones 0 through 9 are complete.

Completed outcomes:

- Wails app shell and React frontend are in place
- SQLite persistence, migrations, and CRUD services are working
- preview, validation, duplicate protection, Test Mode, Dry Run, and Go Live orchestration are implemented
- real Discord, Bluesky post, and Mastodon publishers are wired
- backend and frontend tests are already part of the workflow, not deferred to the end
- Milestone 7 hardening closed the biggest MVP UX gaps around diagnostics, partial execution clarity, and recovery behavior

Milestone 6 closed on the correct requirement:

- manage Bluesky `Live Now` status only
- do not mutate display name, bio, or other profile fields
- do not use profile backup or restore
- the first dedicated Live Now adapter and service seam is now in place
- set `Live Now` on Go Live
- clear `Live Now` on End Stream
- track pending sessions for restart-safe recovery
- offer manual recovery in the Settings UI
- support optional end-stream posts

Milestone 7 closed with:

- clearer execution-state and recovery messaging
- stronger app-shell, repository, and frontend behavior coverage
- friendlier diagnostics and log labels
- safer test isolation for app-shell stateful behavior
- the project is ready to move into post-MVP quality and security validation

## Product Summary

StreamSignal is a local-first desktop app that lets a streamer prepare one announcement and distribute it safely across multiple destinations.

MVP destinations:

- Discord via webhooks
- Bluesky via API
- Mastodon via API

MVP workflows:

- Preview
- Dry Run
- Go Live
- End Stream
- duplicate warning and confirmation
- Test Mode routing
- Bluesky Live Now lifecycle

## Architecture

### Layers

1. Frontend
   - React + TypeScript
   - forms, preview cards, result panels, settings, logs

2. Desktop shell
   - Wails bindings between UI and Go services

3. Application layer
   - preview, execution, settings, destinations, logs, diagnostics

4. Domain layer
   - models, validation, execution results, duplicate rules

5. Infrastructure
   - SQLite repositories
   - credential storage
   - Discord, Bluesky, Mastodon adapters

### Principles

- preview and dry run must stay network-free
- Test Mode must never hit production targets
- publisher adapters stay small and mockable
- workflow rules are tested at service boundaries

## Repository Shape

```text
StreamSignal/
  app.go
  main.go
  frontend/
  internal/
    app/
    domain/
    duplicate/
    platforms/
      discord/
      bluesky/
      mastodon/
      unavailable/
    ports/
    storage/
      sqlite/
    templates/
```

## Data Model

Current core entities:

- `Announcement`
- `Destination`
- `AppSettings`
- `PreviewItem`
- `ExecutionResult`
- `ExecutionSummary`
- `LogEntry`
- `PostHistoryRecord`

Current SQLite scope:

- `destinations`
- `settings`
- `logs`
- `post_history`

Removed from the plan and codebase:

- Bluesky profile backup records
- profile snapshot/update models
- profile backup repositories

## Storage and Secrets

SQLite stores app state and operational history.

Credential Manager stores secret values such as:

- Discord webhook URLs
- Bluesky app passwords or token-like credentials
- Mastodon access tokens
- Test Mode destination secrets

SQLite should only store references or non-secret config.

## Interface Plan

Key ports:

- `DestinationRepository`
- `SettingsRepository`
- `LogRepository`
- `PostHistoryRepository`
- `DiscordPublisher`
- `BlueskyPublisher`
- `MastodonPublisher`
- `SecretStore`

Current Bluesky publisher scope:

- publish announcement posts
- set `Live Now` status
- clear `Live Now` status

Confirmed `Live Now` write contract:

- write the `app.bsky.actor.status` record at `rkey: "self"`
- use `status: "app.bsky.actor.status#live"`
- attach an `app.bsky.embed.external` card pointing at the stream URL
- clear the status by deleting the same record

## Workflow Design

### Preview

Steps:

1. load enabled destinations
2. render content
3. validate rendered output
4. return preview items

Rules:

- no network calls
- no persistence side effects beyond optional logging

### Dry Run

Steps:

1. load settings and enabled destinations
2. validate destination config
3. render content
4. validate content
5. return simulated execution results

Rules:

- no network calls
- no post history writes

### Go Live

Steps:

1. load settings and enabled destinations
2. validate configs
3. render content
4. run duplicate checks
5. stop for confirmation if required
6. apply Test Mode remapping if enabled
7. publish to each destination independently
8. record post history for successful posts
9. aggregate results and log outcomes

Rules:

- one destination failure must not block the others
- duplicate warnings require explicit confirmation
- Test Mode must never use production targets

### End Stream

Current behavior:

- clear Bluesky `Live Now` status
- remove tracked pending sessions when clear succeeds
- optionally publish end-stream posts to enabled destinations

## Validation

### Announcement validation

- stream title required
- stream URL required unless a default is available
- URL must be absolute and valid

### Destination validation

Discord:

- webhook URL must be valid

Bluesky:

- account identifier required
- credential key required

Mastodon:

- credential key required
- instance URL must be valid

### Settings validation

- duplicate window must be positive
- Test Mode requires complete test targets
- end-stream template is required if end-stream posting is enabled

## Duplicate Protection

Current strategy:

1. normalize rendered content
2. hash it
3. compare against recent post history
4. require confirmation when the duplicate rule trips

This is already implemented and covered by tests.

## Test Mode

Current strategy:

- remap execution targets in the application layer
- keep adapters unaware of environment routing

Current supported remaps:

- Discord test webhook
- Bluesky test account identifier and credential key
- Mastodon test credential key and instance URL

This is already implemented and covered by tests.

## Bluesky Live Now

This implementation is strictly about the dedicated Bluesky `Live Now` feature.

What it does:

- set `Live Now` status during Go Live when enabled
- clear `Live Now` status on End Stream
- support a safe manual clear or recovery path if the app closes mid-stream

What this milestone must not do:

- change display name
- change profile description
- back up or restore profile fields
- treat profile mutation as the Live Now implementation

## Logging and Diagnostics

Current logging scope:

- preview
- dry run
- go live
- duplicate warnings
- validation failures
- integration failures
- Live Now set attempts
- Live Now clear attempts
- recovery/manual clear actions

## Frontend Plan

Current screens:

- Home
- Destinations
- Settings
- Logs

Current MVP UI support:

- end-stream clear action
- manual recovery/clear action
- per-destination execution summaries
- diagnostics copy flow

## Milestones

### Milestone 0

- bootstrap app shell

Status:

- complete

### Milestone 1

- persistence, migrations, repositories, CRUD services

Status:

- complete

### Milestone 2

- template rendering, validation, preview flow

Status:

- complete

### Milestone 3

- destinations, settings, logs, diagnostics UI

Status:

- complete

### Milestone 4

- Dry Run, Go Live, duplicate flow, Test Mode routing

Status:

- complete

### Milestone 5

- real Discord, Bluesky post, and Mastodon publishers

Status:

- complete

### Milestone 6

Goal:

- implement Bluesky `Live Now` lifecycle without profile mutation

Deliverables:

- confirmed adapter contract for Bluesky `Live Now`
- service for setting and clearing Live Now status
- execution integration for Go Live and End Stream
- safe manual clear or recovery path
- service and adapter tests for the lifecycle

Exit criteria:

- Go Live can set Live Now status safely
- End Stream can clear Live Now status safely
- the app can recover from an interrupted live session without profile edits

Status:

- complete

Completed milestone-6 outcomes:

- contract confirmed
- Bluesky adapter can now set and clear Live Now status
- service-level validation seam exists
- Go Live and End Stream orchestration are wired
- pending Live Now sessions are tracked locally for restart-safe manual recovery
- the Settings UI can list and clear pending Live Now sessions
- optional end-stream posting is wired into End Stream

### Milestone 7

Goal:

- final hardening and remaining quality gaps

Deliverables:

- deeper repository coverage where still thin
- better user-facing error handling
- remaining acceptance coverage

Status:

- complete

Completed milestone-7 outcomes:

- clearer execution and recovery UX
- diagnostics and logs use friendlier action labels
- app-shell tests use isolated temp databases
- frontend coverage reporting is in place and verified
- broader repository and app-shell coverage
- final acceptance-quality polish

### Milestone 8

Goal:

- validate that the automated test strategy, test depth, and coverage profile align with best practice for a desktop app with external integrations

Deliverables:

- test-inventory review across backend and frontend
- identification of shallow, missing, redundant, or brittle tests
- coverage review by risk area instead of coverage number alone
- targeted additions or refactors where important behavior is under-protected
- written testing guidance for how future work should be validated

Exit criteria:

- critical workflows have appropriate unit, service, repository, and UI coverage
- flaky or low-signal tests are identified and corrected
- remaining coverage gaps are documented with risk context
- the repo has a clear, intentional testing standard instead of just a growing test count

Status:

- complete

Completed milestone-8 outcomes:

- coverage and seam review completed
- best-practice test validation document added
- app-shell binding coverage improved
- unavailable adapter safety behavior is now explicitly tested
- frontend branch and behavior coverage improved in high-value UI paths
- GitHub PR quality-gate recommendations documented for public collaboration

### Milestone 9

Goal:

- perform a full application security audit and bring the app closer to current best practices for local desktop software and third-party API integrations

Deliverables:

- threat-model review for secrets, local storage, diagnostics, logs, and outbound integrations
- code audit for unsafe handling of credentials, URLs, user input, and persistence
- dependency and configuration review
- OWASP-aligned findings list with severity and remediation plan
- implementation of high-value security fixes discovered during the audit

Exit criteria:

- secrets handling, local persistence, diagnostics, and network calls are reviewed against best practice
- high-severity issues are fixed
- medium-severity issues are either fixed or explicitly documented
- the repo has a concrete security posture summary and next-step backlog

Status:

- complete

Current milestone-9 progress:

- high-severity SQLite secret persistence issue identified
- Windows Credential Manager integration is now wired for runtime secret storage
- SQLite now stores only secret references for destinations, Test Mode settings, and Live Now recovery credentials
- legacy raw-secret rows are migrated on read
- first security audit document and dependency check results are recorded
- diagnostics exports redact pending Live Now session content by default
- secret-edit screens now use masked placeholders instead of rehydrating visible raw secrets
- the GitHub-hosted quality gate has been verified live against the public repository
- CI now builds the frontend before backend tests so Wails embed checks pass in automation
- the repo now declares a current Go toolchain and uses Node 24-capable GitHub Action versions

## Test Strategy

Testing starts with Milestone 1 and continues milestone by milestone.

Priority order:

1. domain logic
2. orchestration rules
3. repository behavior
4. adapter behavior
5. frontend interaction states

Current high-value coverage already exists for:

- preview staying network-free
- Dry Run staying network-free
- duplicate detection and confirmation
- Test Mode routing
- partial failure isolation
- destination/settings validation
- platform publishers

Milestone 6 test expectations:

- no Live Now action during Preview
- no Live Now action during Dry Run
- Go Live sets Live Now only when the feature is enabled/configured
- End Stream clears it safely
- interrupted sessions can be recovered safely
- failures in Live Now handling are visible without hiding post results
- recovery clear failures are surfaced clearly in both the UI and logs

Milestone 8 test-validation focus:

- verify we are testing the highest-risk seams first
- confirm unit tests are not overfitted to implementation details
- review whether adapter, repository, and orchestration tests are balanced correctly
- check frontend tests for behavior focus over snapshot-style fragility
- document what "done" means for future feature testing

## Risks

### Risk: future Bluesky Live Now API changes

Mitigation:

- verify the API contract again before future behavior changes
- keep the service behind a small port
- keep adapter tests around the exact request/response contract

### Risk: security assumptions staying implicit

Mitigation:

- treat Milestone 9 as a real audit, not a casual pass
- review secrets, logs, diagnostics, persistence, and dependency exposure together
- write down findings and remediations so security posture is explicit

### Risk: Test Mode safety regression

Mitigation:

- keep routing in the orchestration layer
- preserve the existing tests that assert test targets are used

### Risk: secrets leaking into persistence or logs

Mitigation:

- keep secrets out of SQLite
- continue redacting diagnostics content where needed

## Recommended Next Step

Return to release-readiness items:

1. packaging and installer flow
2. first-run UX and documentation polish
3. routine maintenance for the security and quality gates

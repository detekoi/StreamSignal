# Test Validation Review

## Purpose

This document records the Milestone 8 review of StreamSignal's automated tests against practical best practice, not just raw coverage totals.

## Current Snapshot

Backend coverage from `go test ./... -coverprofile=coverage.out`:

- root app package: `40.3%`
- `internal/app`: `71.2%`
- `internal/domain`: `72.7%`
- `internal/duplicate`: `90.0%`
- `internal/platforms/bluesky`: `76.9%`
- `internal/platforms/discord`: `86.4%`
- `internal/platforms/mastodon`: `87.0%`
- `internal/platforms/unavailable`: previously `0.0%`, now covered
- `internal/storage/sqlite`: `71.8%`
- `internal/templates`: `100.0%`

Frontend coverage from `npm run test:coverage`:

- statements: `92.33%`
- branches: `83.10%`
- functions: `71.27%`
- `src/App.tsx`: `86.32%` statements, `80%` branches
- API wrapper layer: `100%`

## Best-Practice Assessment

### What is strong

- risky workflow logic is tested at the service layer, where most product behavior actually lives
- adapters for real outbound integrations have dedicated tests instead of being covered only indirectly
- repository coverage is meaningful rather than token-level
- frontend tests are behavior-oriented and exercise real user flows instead of snapshots
- coverage is highest in the most failure-prone backend areas: templates, duplicate protection, execution orchestration, and publishers
- app-shell tests now use isolated temp databases, which reduces test pollution and flakiness

### What we are doing right

- testing by seam, not by file count
- validating safety rules like Preview staying network-free
- protecting destination selection and publishing targets with direct assertions
- covering partial-failure behavior instead of only happy paths
- validating recovery and diagnostics flows, not just posting flows

## Remaining Gaps

### Lower-risk gaps

- several frontend `types/*.ts` files remain uncovered because they only declare interfaces
- some root app bindings are still lightly tested relative to the service layer

### Moderate gaps

- frontend function coverage is good, but not as strong as statement coverage
- `src/App.tsx` still has some unhit conditional branches in less central UI paths
- root package coverage is still lower than the core backend packages because the shell is intentionally thin

### Not currently concerning

- low or zero coverage on pure type-definition files
- lower root-package coverage when the real behavior is already covered more directly in `internal/app`

## Milestone 8 Changes Added In This Pass

- added app-shell tests for:
  - settings persistence and logging
  - preview plus diagnostics binding behavior
  - Go Live binding and summary logging
- added tests for the unavailable publisher adapters so safe fallback semantics are explicit

## Current Bluesky Coverage

Recent workflow work added or updated tests for:

- Preview validation replacing the removed Dry Run path
- explicit production/test destination targeting
- Bluesky rich text link facets and external stream cards
- Bluesky card thumbnail upload from both URL and local-image data URL
- Bluesky `Live Now` duration configuration, set, clear, and recovery behavior
- defensive thumbnail validation for non-image, oversized, and non-HTTP inputs

## Standard For Future Work

Use this rule of thumb before calling a feature "well tested":

1. domain validation and normalization paths are covered
2. orchestration rules are covered with success, warning, and failure behavior
3. repository behavior is covered for persistence that changes workflow state
4. real outbound adapters have request/response or error-path tests
5. frontend tests cover the user-visible decision points and safety messaging

## Recommended PR Quality Gate

For GitHub PR checks, use a quality gate that is strict enough to protect the important paths without blocking contributors on low-value coverage churn.

### Required checks

Every pull request should pass:

- `go test ./...`
- `govulncheck ./...`
- `npm test --prefix frontend`
- `npm run build --prefix frontend`
- `npm audit --prefix frontend`

If a PR changes dependency manifests, add:

- `go mod tidy` with no diff
- frontend install lockfile validation as part of normal CI

Current GitHub Actions implementation:

- [quality-gate.yml](./.github/workflows/quality-gate.yml)

### Coverage posture

Do not gate on a single repo-wide number alone. Prefer a mixed policy:

1. hard pass/fail on the test commands above
2. coverage reporting visible on every PR
3. reviewer attention on changed-file coverage and risky workflow coverage

Recommended baseline thresholds for automated warning, not immediate failure:

- backend overall coverage should not regress materially from the current baseline
- frontend statement coverage should stay above `90%`
- frontend branch coverage should stay above `80%`

Recommended failure conditions:

- a PR lowers coverage in `internal/app`, `internal/duplicate`, `internal/platforms/*`, or `frontend/src/App.tsx` without adding equivalent protection elsewhere
- a PR changes execution, destination targeting, duplicate protection, Live Now, recovery, or diagnostics behavior without adding or updating tests
- a PR introduces snapshot-style UI tests instead of behavior-focused assertions for workflow changes

### Risk-based required tests

Require targeted tests when a PR touches:

- execution orchestration
  - must update service tests in `internal/app`
- publisher adapters
  - must update adapter tests for success and failure paths
- SQLite repositories
  - must add or update round-trip persistence tests
- app-shell bindings
  - should add or update `app_test.go` coverage when behavior changes
- `frontend/src/App.tsx`
  - should add or update interaction tests for the changed user path

### What the gate should avoid

Avoid these anti-patterns:

- blocking PRs only because type-definition files are uncovered
- forcing contributors to raise total coverage when the PR only refactors internals without changing risk
- rewarding superficial tests that only assert implementation details or static rendering

### Suggested review language

When a PR changes behavior, reviewers should ask:

- what user-visible or workflow-risk behavior changed?
- where is that behavior tested now?
- if coverage moved down, was protection added in a better seam?

## Practical Conclusion

StreamSignal is aligned with good testing practice for its current maturity.

That does not mean the product itself is fully validated from a user-workflow perspective yet. The automated test posture is strong, but we are still using release-readiness and hands-on usability review to confirm that implemented behavior is understandable and trustworthy for real users.

The main strength is that the suite protects the highest-risk behavior first. The biggest remaining opportunities are refinement, not rescue:

- keep lifting root app-shell confidence when new bindings are added
- keep adding branch-focused frontend tests when new UI decisions appear
- continue reviewing test quality by risk area instead of chasing a single coverage number

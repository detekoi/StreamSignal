# StreamSignal

StreamSignal is a local-first desktop app for streamers and VTubers that creates one announcement and distributes it to multiple platforms.

Current product status:

- Wails desktop shell, Go backend, and React + TypeScript frontend are all wired together
- Preview, Go Live, End Stream, duplicate confirmation, and explicit production/test destinations are implemented
- Discord, Bluesky, and Mastodon publishers are connected
- Discord webhook posting has been manually validated, including selected-destination targeting, optional additional image embeds, and destination-level End Stream enablement
- Bluesky posting has been validated with app-password auth, rich stream links, external cards, configurable `Live Now` duration, End Stream clearing, and manual recovery if the app closes mid-stream
- Preview is network-free and shows both Go Live messages and destination-level End Stream messages when available
- runtime secrets are stored through Windows Credential Manager rather than directly in SQLite
- automated backend and frontend tests are part of the normal workflow

The app is not yet being treated as fully validated MVP-complete. Core workflows are implemented, but we are still actively validating real-world behavior, usability, and human understanding of the UI before calling it release-ready.

## Stack

- Go backend
- Wails desktop shell
- React + TypeScript frontend
- SQLite for application data
- Windows Credential Manager for secrets

## Development

Run the app in development mode:

```bash
wails dev
```

Build the desktop app:

```bash
wails build
```

Run the automated test suites:

```bash
cd frontend && npm run build
go test ./...
npm test --prefix frontend
```

Run the frontend dependency audit:

```bash
npm audit --prefix frontend
```

## Project Docs

- [SECURITY.md](./SECURITY.md): vulnerability reporting policy
- [SECURITY_AUDIT.md](./SECURITY_AUDIT.md): public-facing security posture summary
- [INTEGRATION_LIMITATIONS.md](./INTEGRATION_LIMITATIONS.md): current platform integration constraints and boundaries
- [TEST_VALIDATION.md](./TEST_VALIDATION.md): testing strategy and PR quality-gate guidance
- [RELEASE_READINESS_CHECKLIST.md](./RELEASE_READINESS_CHECKLIST.md): release and packaging checklist for the next phase
- [.github/workflows/quality-gate.yml](./.github/workflows/quality-gate.yml): GitHub PR checks

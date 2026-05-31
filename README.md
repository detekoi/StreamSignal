# StreamSignal

StreamSignal is a local-first desktop app for streamers and VTubers that creates one announcement and distributes it to multiple platforms.

Current MVP status:

- Wails desktop shell, Go backend, and React + TypeScript frontend are all wired together
- Preview, Dry Run, Go Live, End Stream, duplicate confirmation, and Test Mode routing are implemented
- Discord, Bluesky, and Mastodon publishers are connected
- Bluesky `Live Now` can be set on Go Live, cleared on End Stream, and manually recovered if the app closes mid-stream
- runtime secrets are stored through Windows Credential Manager rather than directly in SQLite
- automated backend and frontend tests are part of the normal workflow

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
go test ./...
npm test --prefix frontend
```

Run the frontend production dependency audit:

```bash
npm audit --omit=dev --prefix frontend
```

## Project Docs

- [SECURITY.md](./SECURITY.md): vulnerability reporting policy
- [SECURITY_AUDIT.md](./SECURITY_AUDIT.md): public-facing security posture summary
- [TEST_VALIDATION.md](./TEST_VALIDATION.md): testing strategy and PR quality-gate guidance
- [.github/workflows/quality-gate.yml](./.github/workflows/quality-gate.yml): GitHub PR checks

Historical project planning notes remain in [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for now, but that file is better treated as build history than core public documentation.

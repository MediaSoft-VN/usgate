You are the primary Build agent for this repository.

## Repository identity

This is **new-api**, a Go AI gateway, not a Fastify/Drizzle/USGate CMS project.
The root module uses Go 1.26, Gin, GORM, SQLite/MySQL/PostgreSQL, Redis, and
React 19/Rsbuild/Bun in `web/`. `relaykit/` and `tokenkit/` are independent Go
modules. Read `AGENTS1.md` and any nearer `AGENTS.md` before working; frontend
changes also require `web/AGENTS.md`.

## Required behavior

- Inspect existing patterns before editing; keep changes small and direct.
- Never invent architecture from the stale CMS instructions in `AGENTS.md`.
- Preserve all supported database dialects for database-affecting changes.
- Use `common.*` JSON wrappers in the root module; use `kitutil.*` in relaykit.
- Read `docs/plugin-api/v1.md` before changing JavaScript task plugins or their host API.
- Read `.agents/rules/billing.md` in full before any billing-related work.
- Follow OWASP guidance for authentication changes.
- Add focused regression tests for behavior changes, then run formatting and relevant checks.
- Use Bun for frontend commands. Do not add documentation under `docs/` unless explicitly requested.
- Do not expose secrets, tokens, credentials, or raw sensitive errors in logs or output.

## Task execution gate

- Work on exactly one backlog task at a time; do not mix unrelated changes.
- Define acceptance criteria and an observable E2E path before editing.
- Implement, test, and verify the current task completely before selecting another.
- Run the relevant E2E test plus focused unit/integration checks and formatting.
- Inspect `git status`, `git diff`, and recent history, stage only this task's files,
  then commit the task. Never commit secrets or unrelated existing user changes.
- Do not move to the next task until the commit succeeds. Report blockers instead
  of bypassing a failed or unavailable E2E check.

## Verification defaults

Use the narrowest relevant checks first:

```text
gofmt -w <modified-go-files>
go test ./<affected-package>/...
GOWORK=off go test ./...          # when relaykit or tokenkit is changed
cd web && bun run typecheck
cd web && bun run lint
cd web && bun run test             # when the frontend has a configured test script
```

Report commands that could not run and why. Do not claim a check passed without
seeing its result.

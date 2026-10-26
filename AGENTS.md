# AGENTS.md — Mandatory Instructions for Coding Agents

Welcome to **Eterno**. This document establishes mandatory operational protocols for all autonomous and pair-programming coding agents working in this repository.

## 1. Prime Directives for Future Agents

1. **Read authoritative context first**: Before planning or making changes, read [`docs/README.md`](file:///home/rajababu/intern/eterno-v1/docs/README.md) and [`docs/memory/CURRENT_STATE.md`](file:///home/rajababu/intern/eterno-v1/docs/memory/CURRENT_STATE.md).
2. **Consult ADRs and Architecture**: Before modifying architecture, scoring, verification logic, or database schemas, consult [`docs/decisions/`](file:///home/rajababu/intern/eterno-v1/docs/decisions/) and [`docs/architecture/`](file:///home/rajababu/intern/eterno-v1/docs/architecture/).
3. **Inspect actual code and test suites**: Never trust documentation blindly; inspect implementation files and run tests to verify actual runtime behavior.
4. **Follow current priorities**: Check [`docs/memory/NEXT_STEPS.md`](file:///home/rajababu/intern/eterno-v1/docs/memory/NEXT_STEPS.md) for the active roadmap.
5. **Preserve conventions and avoid regressions**: Maintain the modular Go monolith pattern, idiomatic Go naming, thin HTTP handlers, and the white-first minimal React design system.
6. **Apply versioned database migrations**: Never execute ad-hoc schema modifications in production code; all schema updates must use ordered SQL migrations in `backend/migrations/`.
7. **Protect secrets and credentials**: Never commit API keys, database credentials, tokens, or live production secrets. Use `.env.example` templates with empty placeholders.
8. **Write deterministic tests**: Write unit and integration tests with mocks for external platforms (e.g. LeetCode GraphQL). Never let test suites depend on external network availability.
9. **Update documentation alongside code**: When you alter an API route, scoring formula, or verification policy, update the corresponding markdown documentation and [`docs/architecture/PROJECT_GRAPH.md`](file:///home/rajababu/intern/eterno-v1/docs/architecture/PROJECT_GRAPH.md).
10. **Record state changes**: Update [`docs/memory/CURRENT_STATE.md`](file:///home/rajababu/intern/eterno-v1/docs/memory/CURRENT_STATE.md), [`docs/memory/NEXT_STEPS.md`](file:///home/rajababu/intern/eterno-v1/docs/memory/NEXT_STEPS.md), and append to [`docs/memory/SESSION_LOG.md`](file:///home/rajababu/intern/eterno-v1/docs/memory/SESSION_LOG.md) before finishing a work session.
11. **Run builds and tests before concluding**: Always verify that `go test ./...`, `go build ./cmd/...`, and `npm run build` pass without error.
12. **Commit frequently**: Commit self-contained units of work with descriptive commit messages following the established repository git history.

---

## 2. Verified Tooling and Environment Commands

### Go Backend
- **Go Version**: `go1.22+` / `go1.26` (Located at `/usr/local/go/bin/go`)
- **Run API Server**: `cd backend && go run ./cmd/api`
- **Run Background Worker**: `cd backend && go run ./cmd/worker`
- **Run Unit and Integration Tests**: `cd backend && go test -v -race ./...`
- **Build Backend Binaries**: `cd backend && go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker`

### React Frontend
- **Node & npm**: Node `v22+`, npm `10+`
- **Install Dependencies**: `cd frontend && npm install --prefer-offline`
- **Run Development Server**: `cd frontend && npm run dev`
- **Typecheck and Build**: `cd frontend && npm run build`

### Database
- **PostgreSQL**: PostgreSQL 16
- **Local Dev Container**: `docker run -d --name eterno-postgres -e POSTGRES_DB=eterno -e POSTGRES_USER=eterno -e POSTGRES_PASSWORD=eterno_secret -p 127.0.0.1:5434:5432 postgres:16-alpine`
- **Database URL**: `postgres://eterno:eterno_secret@127.0.0.1:5434/eterno?sslmode=disable`

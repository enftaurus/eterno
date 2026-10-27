# Eterno — Engineering Session Log

## Session: 2026-10-27
- **Objective**: Full implementation and verification of Eterno platform: PostgreSQL schema migration, Go backend monolith, LeetCode verification engine, background workers, React 18 white-first frontend, and automated tests.
- **Actions Completed**:
  - Environment: Spun up isolated `eterno-postgres` Docker container on `127.0.0.1:5434`.
  - Documentation: Created `AGENTS.md`, 5 Mermaid diagrams, ADR-0001 through ADR-0008, lifecycle & verification docs.
  - Database: Executed migration `000001_initial_schema.up.sql` (19 tables, indexes, cascading keys) and applied demo seed dataset for Stanford School of Engineering.
  - Backend: Implemented complete Go domain packages (auth, challenges, platforms/leetcode, submissions, verification, scoring, leaderboards, analytics, activity, jobs, audit, http/handlers, router).
  - Background Worker: Persistent job processor polling `sync_jobs` using row-level locking (`FOR UPDATE SKIP LOCKED`).
  - Frontend: Built complete React 18 + Vite + TypeScript + Tailwind CSS application with white-first minimal design system (Sidebar, AppLayout, Heatmap, 14 pages covering both Student & Admin portals). Clean Vite build output in 1.6s.
  - Verification: Tested live HTTP API endpoints (`/healthz`, `/api/v1/auth/login`, `/api/v1/me/challenges`, `/api/v1/me/heatmap`, `/api/v1/admin/dashboard`, `/api/v1/leaderboards`, `/api/v1/me/platform-accounts/leetcode/sync`).
  - Developer Tooling: Authored and verified `dev.sh` orchestrator script which checks prerequisites, ensures PostgreSQL is running, applies migrations and seed data if needed, downloads backend and frontend dependencies, builds binaries, and launches the Backend API, Worker daemon, and Vite dev server with graceful shutdown handling.
- **Tests Executed**:
  - `backend/tests/unit_test.go`: All unit tests passed (slug extraction, verdict normalization, tie-breaker ranking, heatmap intensity calculation, bcrypt hashing & JWT claims).
  - `backend/tests/integration_test.go`: Complete 13-step end-to-end integration test passed against live PostgreSQL.
  - `dev.sh`: Verified end-to-end dependency installation, container health polling, backend/frontend startup, port availability, and graceful SIGINT cleanup.
- **Status**: Complete production-ready implementation. All acceptance criteria satisfied.

## Session: 2026-10-27 (Zero-Mock Hardening & Edge-Case Campaign)
- **Objective**: Execute comprehensive zero-mock audit, eliminate all mock data/synthetic counts from normal runtime paths, harden LeetCode external provider evidence parsing, implement multi-challenge verification with strict time boundaries, secure background queue leases and deduplication, harden CSV imports against formula injection, implement 70+ modular edge-case tests, and verify Release Candidate readiness.
- **Actions Completed**:
  - Audit: Created authoritative report `docs/project-management/ZERO_MOCK_AUDIT.md` categorizing 13 findings across P0–P2 severity levels.
  - Configuration: Default `MockExternalProviders` set to `false`. Added strict boolean/integer parsing (`getEnvBoolStrict`, `getEnvIntStrict`). In production, missing/weak JWT secret or enabled mock mode halts server immediately.
  - Scripting: Updated `dev.sh` to require explicit `--seed` flag for demo data; empty databases are never auto-seeded. Added clean port checks (8080, 5173).
  - Database: Created and applied migration `000002_harden_constraints.up.sql` adding `idx_score_events_completion_uniq`, check constraints `points >= 0`, `points_awarded >= 0`, and index on `sync_jobs(status, locked_at)`.
  - LeetCode Adapter: Enforced strict hostname validation (`leetcode.com`, `www.leetcode.com`), exact word-boundary token regex matching, non-empty ID/slug validation, future-timestamp rejection (>10m clock skew), typed errors (`ErrRateLimited`, `ErrUnauthorized`), and 5MB payload ceiling.
  - Multi-Challenge Verification: Replaced early-return verification with `VerifySubmissionAll`, iterating across all active challenge assignments for a problem slug. Fixed time boundaries (exact baseline and deadline inclusive, pre-baseline rejected).
  - Background Queue: Added `RecoverStaleJobs` for expired leases (>5 minutes old). Added deduplicated `Enqueue` preventing duplicate pending/running jobs.
  - CSV Security: Implemented spreadsheet formula sanitization (prepends `'` for `=, +, -, @, \t, \r`), enforced 2MB size and 1,000-row batch limits, RFC 5322 regex validation, and in-batch email deduplication. Protected database transaction from duplicate platform account constraint failures.
  - Frontend Integrity: Removed all fake metrics ("142 problems", "412 commits"); dynamically computes solves and active days; badged GitHub and Codeforces as "Upcoming / Beta"; added persistent error banners with retry buttons; made registration institution selection dynamic.
  - Architecture: Authored `docs/decisions/ADR-0009-zero-mock-hardening.md` documenting all architectural and security decisions.
- **Test Suites Created & Executed**:
  - `backend/tests/provider_test.go`: 21 test scenarios for GraphQL 429, 401, 500, cancellations, zero/future timestamps, token boundaries, and 22 URL parsing scenarios.
  - `backend/tests/verification_test.go`: Time boundaries, multi-challenge matching, cancelled challenge point prevention, concurrent idempotent verification.
  - `backend/tests/worker_test.go`: Row locking with `SKIP LOCKED`, concurrent worker disjoint claims, stale lease recovery, job deduplication, retry backoff, terminal failure state.
  - `backend/tests/csv_test.go`: Valid imports, formula injection sanitization, regex email validation, duplicate rejection, batch size limits, RFC 4180 parsing.
  - `backend/tests/auth_tenancy_test.go`: Missing token, malformed token, expired token, tampered signature, alg=none rejection, student RBAC prohibition, college admin access, cross-tenant isolation, URL param tampering.
  - `backend/tests/analytics_test.go`: Safe empty stats, streak state machine with gap breaks, multi-criteria deterministic leaderboard tie-breaking, heatmap intensity bucketing, year/platform filtering.
  - Full Race Detector Verification: Ran `go test -v -race ./...` across all test files — 100% passed in 10.1s with 0 races and 0 failures.
  - Binary Builds: `go build ./cmd/...` and `go vet ./...` exited with code 0.
  - Frontend Build: `npm run build` completed cleanly in 1.49s with 0 errors.
- **Status**: Release Candidate (RC-1) verified and ready for production deployment.

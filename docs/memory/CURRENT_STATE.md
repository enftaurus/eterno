# Eterno — Current Project State

## 1. Product Scope
Eterno College Coding Platform & Developer Intelligence:
- Multi-tenant architecture for colleges, departments, batches, and clubs with strict isolation.
- Challenge creation, LeetCode problem validation, multi-challenge matching, and lifecycle management.
- Verifiable account linking with exact token boundary matching, background submission sync, and deterministic verification.
- Deterministic scoring engine with non-negative check constraints, multi-criteria tie-broken leaderboards, and streak tracking.
- Unified cross-platform activity events and interactive contribution heatmap with dynamic real-data computation.
- Hardened CSV student roster import/export with formula injection neutralization and batch limits.
- White-first minimal React UI with role-based navigation, error recovery banners, and zero mock fallbacks.

## 2. Infrastructure & Environment
- **Go Version**: `go1.22+` / `go1.26` (Located at `/usr/local/go/bin/go`)
- **PostgreSQL**: PostgreSQL 16 container running at `127.0.0.1:5434` (`eterno` DB)
- **Node & npm**: Node 22, npm 10
- **Frontend Stack**: React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, Recharts, Lucide React
- **Development Tooling**: `./dev.sh` with safe process termination and opt-in `--seed`

## 3. Implementation & Zero-Mock Hardening Status
- [x] Repository zero-mock audit completed (`docs/project-management/ZERO_MOCK_AUDIT.md`).
- [x] ADR-0001 through ADR-0009 recorded in `docs/decisions/`.
- [x] Database migrations:
  - `000001_initial_schema.up.sql`: 19 core tables, foreign keys, and indexes.
  - `000002_harden_constraints.up.sql`: Unique score event constraints, non-negative point checks, sync job stale recovery index.
- [x] Backend Configuration Hardening:
  - `MockExternalProviders` defaults to `false`.
  - Production fails fast on missing/default/short JWT secret or enabled mock mode.
  - Strict boolean and integer parsing; validated CORS origins and connection pools.
- [x] LeetCode Provider Evidence Correctness:
  - Strict URL hostname parsing (`leetcode.com`, `www.leetcode.com`).
  - Regex word/punctuation boundary token verification.
  - Rejection of zero, missing, or future-dated timestamps (>10m clock skew).
  - Typed error handling for HTTP 429, 401/403, 500, and cancellations; 5MB payload limit.
- [x] Multi-Challenge Verification Engine:
  - `VerifySubmissionAll` evaluates all candidate challenges matching problem slug.
  - Strict time boundaries: exact baseline & deadline allowed; pre-baseline strictly rejected.
  - Concurrent verification idempotency (`ON CONFLICT (challenge_problem_id, user_id) DO NOTHING`).
- [x] Background Worker & Sync Queue:
  - Row-level lock claiming with `FOR UPDATE SKIP LOCKED`.
  - Stale lease recovery (`RecoverStaleJobs`) for crashed workers past 5 minutes.
  - Deduplicated `Enqueue` prevents queue flooding.
  - Exponential retry backoff and terminal `FAILED` status transitions.
- [x] CSV Security:
  - Formula injection protection (`=, +, -, @, \t, \r` neutralized with leading `'`).
  - 2MB file limit, 1,000 student batch limit, RFC 5322 regex validation, in-batch deduplication.
  - Safe verification of existing platform accounts preventing aborted transactions.
- [x] Frontend Hardening:
  - Removed all fake metrics and static counts ("142 problems", "412 commits").
  - Persistent error banners with retry buttons on query failures.
  - Dynamic institution dropdowns on registration.
  - Clean empty state indicators (`"—"`).
- [x] Comprehensive Automated Test Campaign:
  - 70+ modular edge-case tests in `backend/tests/` passing with race detection (`-race`):
    - `provider_test.go`: GraphQL errors, timeouts, rate limits, evil URLs, token matching.
    - `verification_test.go`: Time boundaries, multi-challenge matching, concurrent deduplication.
    - `worker_test.go`: Concurrency, disjoint claiming, stale lease recovery, retries.
    - `csv_test.go`: Formula injection, invalid emails, duplicates, batch limits, RFC 4180.
    - `auth_tenancy_test.go`: Alg none, expired tokens, tampered signatures, RBAC, tenant isolation.
    - `analytics_test.go`: Safe empty stats, streak transitions, tie-breaking, intensity scales.
  - All tests passing with race detector in 10s: `go test -v -race ./...` -> 0 failures.
  - Backend compiles clean: `go build ./cmd/...` and `go vet ./...` -> 0 errors.
  - Frontend builds clean: `npm run build` -> 0 errors in 1.49s.

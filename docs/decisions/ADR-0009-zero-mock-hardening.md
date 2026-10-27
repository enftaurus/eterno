# ADR-0009: Zero-Mock Hardening, Evidence Correctness, and Reliability Controls

## Status
Accepted

## Date
2026-10-27

## Context
Transitioning Eterno from an initial implementation to a reliable Release Candidate requires removing all mock data, synthetic metrics, and demo assumptions from application runtime paths. Furthermore, external platform evidence handling (LeetCode GraphQL), time-boundary verification, background worker queue concurrency, CSV import/export security, and frontend reporting needed hardening against adversarial and edge-case behaviors.

## Decisions

### 1. Hardened Configuration & Strict Zero-Mock Runtime
- `MockExternalProviders` now defaults strictly to `false` in configuration.
- In production (`APP_ENV=production`), startup immediately fails if `MockExternalProviders` is enabled or if `JWT_SECRET` is missing, default, or fewer than 32 characters.
- Strict parsing functions (`getEnvBoolStrict`, `getEnvIntStrict`) reject malformed boolean or integer environment values rather than defaulting silently.
- Database pool bounds (`max_open_conns`, `max_idle_conns`) and CORS origins are strictly validated at bootstrap.

### 2. Test Fixture & Migration Isolation
- Schema migrations (`000001_initial_schema.up.sql`, `000002_harden_constraints.up.sql`) contain zero demo users, institutions, challenges, or score events.
- Demo data is decoupled into `backend/migrations/seed.sql` and is only applied when `./dev.sh --seed` is explicitly invoked in local development. Empty databases are never automatically seeded.

### 3. Provider Evidence Correctness & LeetCode Security
- **URL & Slug Parsing**: Strict hostname verification accepts only `leetcode.com` and `www.leetcode.com`. Rejects spoofed domains (`leetcode.com.evil.com`), evil query/fragment attacks, embedded credentials, and unexpected ports.
- **Token Verification**: Token matching uses exact regex word/punctuation boundaries (`\b...`) rather than loose substring checks. Returned username must match requested username.
- **Evidence Validation**: Rejects zero timestamps, missing IDs/slugs, and future-dated timestamps (>10m clock skew). Never substitutes `time.Now()` for missing external evidence.
- **Transport Resilience**: Strict 5MB limit on GraphQL responses (`io.LimitReader`), typed error handling for HTTP 429 (`ErrRateLimited`), HTTP 401/403 (`ErrUnauthorized`), HTTP 500 upstream server error, and context cancellation propagation.

### 4. Multi-Challenge Verification & Invariant Enforcement
- `VerifySubmissionAll` evaluates all active challenges matching a problem slug without early return, ensuring that an ineligible challenge (e.g., past deadline) does not suppress scoring on another eligible challenge.
- Strict time boundaries: exact baseline and deadline are inclusive; submissions prior to assignment baseline are strictly rejected.
- Unique constraints (`idx_score_events_completion_uniq`) and check constraints (`points >= 0`) guarantee idempotent, non-negative scoring.

### 5. Background Queue Concurrency & Stale Lease Recovery
- `ClaimNextJob` utilizes PostgreSQL `FOR UPDATE SKIP LOCKED` for lock-free parallel worker claiming.
- `RecoverStaleJobs` automatically reclaims jobs left in `RUNNING` status by crashed workers past a 5-minute lease threshold, resetting them to `PENDING` with retry increment.
- Idempotent `Enqueue` returns existing active (`PENDING` or `RUNNING`) jobs to prevent queue flooding.
- Terminal failure transitions jobs to `FAILED` status after exceeding `max_retries`.

### 6. CSV Security & Sanitization
- Neutralizes CSV formula injection by prepending `'` to fields starting with `=`, `+`, `-`, `@`, `\t`, or `\r`.
- Enforces 2MB maximum file size and 1,000 student records per import batch.
- Validates RFC 5322 compliant email regex and deduplicates emails within each import batch.
- Safely verifies existing platform accounts before insertion to prevent unique constraint transaction aborts.

### 7. Frontend Integrity
- Removed all hardcoded metric fallbacks (`|| 0`, "142 problems", "412 commits").
- Empty states display clean dashes (`"—"`) or empty state illustrations.
- Query failures render persistent, visible error banners with explicit retry buttons.
- Dynamic institution fetching on registration and dynamic tenant profiles throughout the UI.
- GitHub and Codeforces accurately badged as "Upcoming / Beta".

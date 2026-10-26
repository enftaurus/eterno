# ADR-0004: Server-Side Submission Verification Engine

## Status
Accepted

## Date
2026-10-27

## Context
Students must not be able to mark an assigned problem as solved by clicking a button or forging client payloads. Verification must be strictly server-validated against external platform evidence.

## Decision
1. **Assignment Baseline**: When a challenge is activated, an assignment baseline timestamp is registered.
2. **Canonical Matching**: Incoming submissions are checked for:
   - Canonical problem slug match (e.g. `two-sum`).
   - Verdict equals `Accepted`.
   - `submitted_at` >= `baseline_at` (unless historical completion is explicitly allowed).
   - `submitted_at` <= `deadline`.
3. **Idempotence**: Submissions are stored with unique constraints on `(platform, provider_submission_id)` and completion records on `(challenge_problem_id, user_id)` preventing duplicate points or completion records.

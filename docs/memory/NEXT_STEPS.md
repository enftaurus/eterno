# Eterno — Next Steps & Engineering Priorities

| Priority | Task | Target Module | Dependencies | Status | Acceptance Criteria |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **P0** | Zero-Mock Repository Audit | `docs/project-management/` | Repo codebase | **COMPLETED** | Comprehensive audit report (`ZERO_MOCK_AUDIT.md`) identifying 13 P0–P2 findings. |
| **P0** | Hardened Configuration & Real Default | `backend/internal/config/` | Config | **COMPLETED** | Real provider default, strict parsing, production fail-safe for weak JWT/mock mode. |
| **P0** | Test Fixture & Migration Separation | `backend/migrations/`, `dev.sh` | DB Layer | **COMPLETED** | Schema migrations free of demo data; opt-in `./dev.sh --seed`. |
| **P0** | LeetCode Evidence Validation | `backend/internal/platforms/leetcode/` | Adapter | **COMPLETED** | Strict domain check, exact token boundary regex, future timestamp rejection, typed errors. |
| **P0** | Multi-Challenge Verification | `backend/internal/verification/` | Service, Migration 000002 | **COMPLETED** | Evaluates all candidate challenges; exact time boundaries; atomic completion. |
| **P0** | Queue Reliability & Stale Leases | `backend/internal/jobs/` | Queue | **COMPLETED** | `RecoverStaleJobs` reclaims crashed leases (>5m); idempotent `Enqueue`; terminal fail. |
| **P0** | CSV Security & Sanitization | `backend/internal/institutions/`, `frontend/src/` | CSV Service | **COMPLETED** | Neutralize formula injection (`=, +, -, @`); enforce 2MB/1000 row bounds; regex email check. |
| **P0** | Frontend Real-Data & Error Hardening | `frontend/src/pages/` | React App | **COMPLETED** | Removed fake static numbers; added error retry banners; dynamic institution endpoints. |
| **P0** | Edge-Case Test Campaign (70+ Tests) | `backend/tests/` | All packages | **COMPLETED** | 100% test pass rate with `-race` across provider, verification, worker, csv, auth, analytics. |

## Release Candidate Readiness
The platform has achieved **Release Candidate (RC-1)** status. All runtime mock behavior, synthetic metrics, and silent fallbacks have been eliminated. All state transitions, scoring events, and platform verifications are backed by validated PostgreSQL records.

## Future Extensibility Roadmap (Post-Release)
1. **GitHub PR & Commit Webhook Receiver**: Ingest live webhook payloads into `activity_events` with HMAC SHA-256 signature verification.
2. **Codeforces Rating & Contest Ingestion**: Adapter polling Codeforces API contest ratings and submissions.
3. **Automated Weekly Email Digest**: Background worker job dispatching student performance reports to college administrators via SMTP / SES.
4. **Redis Cache Layer**: Optional read caching for global leaderboards when tenant concurrency exceeds 500 req/sec.

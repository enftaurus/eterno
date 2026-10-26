# Eterno — Next Steps & Engineering Priorities

| Priority | Task | Target Module | Dependencies | Status | Acceptance Criteria |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **P0** | Database Schema Migration | `backend/migrations/` | PostgreSQL 16 | **COMPLETED** | All 19 tables, foreign keys, indexes, and constraints applied idempotently. |
| **P0** | Go Domain Models & DB Layer | `backend/internal/database/`, `models/` | Migration 000001 | **COMPLETED** | Connection pooling, scan helpers, models matching schema. |
| **P0** | Auth & RBAC Middleware | `backend/internal/auth/`, `middleware/` | Models | **COMPLETED** | JWT signing/claims, password hashing, tenant-scoping middleware. |
| **P0** | Platform Adapters | `backend/internal/platforms/` | Models | **COMPLETED** | LeetCode provider adapter with bio verification token challenge & mock mode. |
| **P0** | Challenge Management & Lifecycle | `backend/internal/challenges/` | DB Layer | **COMPLETED** | Challenge CRUD, LeetCode slug extraction, assignment creation on publish. |
| **P0** | Ingestion & Verification Worker | `backend/internal/verification/`, `cmd/worker/` | DB Layer, Adapters | **COMPLETED** | Submissions persisted, deduplicated, verified against baseline, completions generated. |
| **P0** | Scoring & Leaderboards | `backend/internal/scoring/`, `leaderboards/` | Verification | **COMPLETED** | Score events logged, deterministic tie-breaking rules applied. |
| **P0** | Consistency Analytics | `backend/internal/analytics/` | Verification | **COMPLETED** | Streaks, active days in 7/30/90, completion rates calculated with zero hardcoded values. |
| **P0** | Activity Events & Heatmap API | `backend/internal/activity/` | Models | **COMPLETED** | Normalized cross-platform events and daily aggregates. |
| **P0** | REST API Handlers & Router | `backend/internal/http/`, `cmd/api/` | All internal packages | **COMPLETED** | Full REST surface under `/api/v1` matching spec. |
| **P0** | React Frontend White-First UI | `frontend/src/` | API routes | **COMPLETED** | Student dashboard, challenges, admin portal, analytics, heatmap (clean Vite build). |
| **P0** | Integration & End-to-End Tests | `backend/tests/` | API, Worker | **COMPLETED** | Full 13-step workflow test passes deterministically. |

## Future Extensibility Roadmap (Post-v1)
1. **GitHub PR Webhook Receiver**: Ingest commit and PR merge events directly via webhooks into `activity_events`.
2. **Codeforces Rating Tracking**: Poll Codeforces API user ratings and contest submissions into unified developer profiles.
3. **Automated Weekly Email Digest**: Dispatch weekly consistency reports to college administrators via background worker.

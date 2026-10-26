# Eterno — Next Steps & Engineering Priorities

| Priority | Task | Target Module | Dependencies | Acceptance Criteria |
| :--- | :--- | :--- | :--- | :--- |
| **P0** | Database Schema Migration | `backend/migrations/` | PostgreSQL 16 | All tables, foreign keys, indexes, and constraints applied idempotently. |
| **P0** | Go Domain Models & DB Layer | `backend/internal/database/`, `models/` | Migration 000001 | Connection pooling, scan helpers, models matching schema. |
| **P0** | Auth & RBAC Middleware | `backend/internal/auth/`, `middleware/` | Models | JWT signing/claims, password hashing, tenant-scoping middleware. |
| **P0** | Platform Adapters | `backend/internal/platforms/` | Models | LeetCode provider adapter with verification challenge & mock mode. |
| **P0** | Challenge Management & Lifecycle | `backend/internal/challenges/` | DB Layer | Challenge CRUD, LeetCode slug extraction, assignment creation on publish. |
| **P0** | Ingestion & Verification Worker | `backend/internal/verification/`, `cmd/worker/` | DB Layer, Adapters | Submissions persisted, deduplicated, verified against baseline, completions generated. |
| **P0** | Scoring & Leaderboards | `backend/internal/scoring/`, `leaderboards/` | Verification | Score events logged, tie-breaking rules applied. |
| **P0** | Consistency Analytics | `backend/internal/analytics/` | Verification | Streaks, active days in 7/30/90, completion rates calculated. |
| **P0** | Activity Events & Heatmap API | `backend/internal/activity/` | Models | Normalized cross-platform events and daily aggregates. |
| **P0** | REST API Handlers & Router | `backend/internal/http/`, `cmd/api/` | All internal packages | Full REST surface under `/api/v1` matching spec. |
| **P0** | React Frontend White-First UI | `frontend/src/` | API routes | Student dashboard, challenges, admin portal, analytics, heatmap. |
| **P0** | Integration & End-to-End Tests | `backend/tests/` | API, Worker | Full 13-step workflow test passes deterministically. |

# Eterno — Current Project State

## 1. Product Scope
Eterno College Coding Platform & Developer Intelligence:
- Multi-tenant architecture for colleges, departments, batches, and clubs.
- Challenge creation, LeetCode problem validation, lifecycle management.
- Verifiable account linking, background submission sync, and deterministic verification.
- Deterministic scoring engine, tie-broken leaderboards, and streak tracking.
- Unified cross-platform activity events and interactive contribution heatmap.
- White-first minimal React UI with role-based navigation.

## 2. Infrastructure & Environment
- **Go Version**: `go1.22+` / `go1.26`
- **PostgreSQL**: PostgreSQL 16 container running at `127.0.0.1:5434` (`eterno` DB)
- **Node & npm**: Node 22, npm 10
- **Frontend Stack**: React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, Recharts, Lucide React

## 3. Implementation Status
- [x] Repository audit & environment initialization.
- [x] AGENTS.md and comprehensive documentation tree established.
- [x] ADR-0001 through ADR-0008 recorded.
- [ ] Database migration schema (`000001_initial_schema.up.sql`).
- [ ] Go backend core domain models and database repositories.
- [ ] Authentication, JWT tokens, and RBAC middleware.
- [ ] LeetCode adapter with verification token challenge and submission fetcher.
- [ ] Challenge CRUD, assignment baseline, and lifecycle transitions.
- [ ] Verification engine and persistent job queue worker.
- [ ] Scoring engine and multi-level leaderboards.
- [ ] Consistency streak calculations and admin dashboard analytics.
- [ ] Unified activity event ingestion and heatmap aggregation.
- [ ] React frontend white-first design system and student/admin portals.
- [ ] Automated integration tests covering complete end-to-end workflow.

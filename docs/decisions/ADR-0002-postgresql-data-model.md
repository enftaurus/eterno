# ADR-0002: PostgreSQL 16 Data Model & Migrations

## Status
Accepted

## Date
2026-10-27

## Context
Eterno requires strict multi-tenant isolation across colleges, auditable scoring events, deduplicated submission records, and robust ACID transactions.

## Decision
We utilize **PostgreSQL 16** with versioned forward migrations in `backend/migrations/`. Data isolation is enforced at query and database levels using foreign keys, explicit institution filters, and composite unique indexes (e.g. `(challenge_id, user_id)` for assignments and `(platform, provider_submission_id)` for submissions).

## Consequences
- Guaranteed referential integrity and transactional multi-record state updates.
- Zero risk of cross-tenant data leaks when paired with tenant-scoping middleware.

# ADR-0001: Modular Go Monolith Architecture

## Status
Accepted

## Date
2026-10-27

## Context
Eterno requires high concurrency, low latency, clean domain boundary separation, and production reliability for processing coding challenges, student submissions, and analytics. We needed to choose between microservices, Python/FastAPI, or a modular Go monolith.

## Decision
We build Eterno as a **Modular Go Monolith** with clear domain packages under `backend/internal/` and a standalone background worker executable (`cmd/worker`) alongside the API daemon (`cmd/api`). Chi/Gin is used for HTTP routing, standardizing on versioned REST routes under `/api/v1`.

## Consequences
- **Pros**: Single deployment artifact, direct in-process domain calls, zero network serialization overhead between modules, simplified transactional integrity.
- **Cons**: Requires strict discipline to prevent cyclic package dependencies; background worker shares code via internal domain packages.

# ADR-0008: Persistent Background Job Queue in PostgreSQL

## Status
Accepted

## Date
2026-10-27

## Context
Submission synchronization and account verification jobs must survive application crashes and restarts. Relying on in-memory goroutines risks losing pending jobs.

## Decision
We implement a database-backed job queue in PostgreSQL using `sync_jobs` and `sync_runs`. Workers claim pending jobs with `SELECT ... FOR UPDATE SKIP LOCKED` and lease timestamps. Failed attempts increment `retry_count` and compute exponential backoff with jitter up to `max_retries`.

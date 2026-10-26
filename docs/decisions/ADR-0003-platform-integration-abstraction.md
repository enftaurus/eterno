# ADR-0003: Platform Integration Abstraction

## Status
Accepted

## Date
2026-10-27

## Context
Eterno must verify problem solves on LeetCode today, but will expand to GitHub, Codeforces, and HackerRank without modifying challenge or verification business logic.

## Decision
We define a decoupled Go interface `PlatformAdapter` exposing:
1. `VerifyAccount(ctx, username, token)`: verifies account control using bio tokens.
2. `FetchRecentSubmissions(ctx, username, limit)`: retrieves normalized submission records.
3. `SyncUserSubmissions(ctx, username, since)`: synchronizes submission history.
All platform-specific network communication, schema parsing, and rate limiting remain contained within `internal/platforms/{provider}/`.

# ADR-0005: Scoring and Leaderboards

## Status
Accepted

## Date
2026-10-27

## Context
Competition integrity requires transparent, auditable points and deterministic leaderboard ranking.

## Decision
Points are recorded as discrete ledger entries in `score_events` linked to user, institution, challenge, and completion IDs. Leaderboard ties are resolved deterministically:
1. Total verified points (descending)
2. Total verified challenge completions (descending)
3. Earliest timestamp of reaching the tied score (ascending)

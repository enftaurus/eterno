# ADR-0006: Canonical Activity Model & Unified Heatmap

## Status
Accepted

## Date
2026-10-27

## Context
Eterno unifies developer activity across LeetCode, GitHub, Codeforces, and internal college challenges into a single developer profile heatmap.

## Decision
All activities are normalized into `activity_events` containing `(source, external_id, event_type, occurred_at, metadata)`. Unique constraints on `(source, external_id)` ensure repeated sync never duplicates activity squares or counts. Daily aggregates are bucketed in the user's institution timezone with source attribution.

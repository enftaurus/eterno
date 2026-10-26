# ADR-0007: White-First Minimalist React Design System

## Status
Accepted

## Date
2026-10-27

## Context
Section 23 mandates a clean, lightweight, highly readable white-first interface inspired by Linear and Stripe, avoiding generic dark neon dashboard aesthetics.

## Decision
We build a component library in React 18 + Tailwind CSS using:
- Pure white and off-white background tokens (`bg-white`, `bg-neutral-50/50`)
- Subtle borders (`border-neutral-200`) and charcoal text (`text-neutral-900`)
- Single restrained primary indigo accent (`bg-indigo-600`)
- Lucide React icons
- Accessible data tables, metric cards, loading skeletons, and interactive SVG contribution heatmap.

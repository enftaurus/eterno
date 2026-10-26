# Eterno Documentation Index

Welcome to the comprehensive documentation repository for **Eterno** — the production-grade developer intelligence, college coding competition, and automated submission verification platform.

## Documentation Structure

```text
docs/
├── README.md                           # This index
├── PRODUCT.md                          # Product vision, user stories, and feature matrix
├── architecture/
│   ├── SYSTEM_OVERVIEW.md              # High-level architecture and design principles
│   ├── PROJECT_GRAPH.md                # Mermaid diagrams (Context, Runtime, Verification, DB, Deployment)
│   ├── BACKEND_ARCHITECTURE.md         # Go backend modular monolith architecture
│   ├── FRONTEND_ARCHITECTURE.md        # React TypeScript Vite white-first design architecture
│   ├── DATABASE_SCHEMA.md              # PostgreSQL 16 schema documentation and entity relationships
│   ├── DATA_FLOWS.md                   # End-to-end data lifecycle and synchronization flows
│   ├── INTEGRATIONS.md                 # LeetCode, GitHub, and Codeforces adapters specification
│   └── SECURITY_MODEL.md               # Multi-tenancy, RBAC, password security, and secret handling
├── decisions/
│   ├── README.md                       # Architectural Decision Records (ADR) index
│   ├── ADR-0001-modular-go-monolith.md
│   ├── ADR-0002-postgresql-data-model.md
│   ├── ADR-0003-platform-integration-abstraction.md
│   ├── ADR-0004-submission-verification.md
│   ├── ADR-0005-scoring-and-leaderboards.md
│   ├── ADR-0006-unified-activity-model.md
│   ├── ADR-0007-frontend-design-system.md
│   └── ADR-0008-background-jobs.md
├── product/
│   ├── REQUIREMENTS.md                 # Core system and functional requirements
│   ├── USER_ROLES.md                   # Platform admin, College admin, Group admin, Student
│   ├── CHALLENGE_LIFECYCLE.md          # State machine: DRAFT -> SCHEDULED -> ACTIVE -> COMPLETED
│   ├── SCORING_RULES.md                # Deterministic scoring, base points, difficulty multipliers
│   ├── METRIC_DEFINITIONS.md           # Consistency streaks, active days, completion rate definitions
│   └── LEETCODE_VERIFICATION.md        # Verification protocol, token proof, and rate limiting
├── engineering/
│   ├── DEVELOPMENT_SETUP.md            # Local developer setup, Docker, migrations, testing
│   ├── TESTING_STRATEGY.md             # Unit, integration, and end-to-end testing standards
│   ├── MIGRATION_POLICY.md             # Versioned SQL migrations and rollback procedures
│   ├── ERROR_HANDLING.md               # Error codes, RFC 7807 problem details, retry semantics
│   ├── OBSERVABILITY.md                # Structured JSON logging, health checks, audit logs
│   ├── DEPLOYMENT.md                   # Production deployment with Docker & Caddy reverse proxy
│   └── TROUBLESHOOTING.md              # Common failure modes and resolution playbook
├── project-management/
│   ├── ROADMAP.md                      # Strategic releases and feature milestones
│   ├── IMPLEMENTATION_STATUS.md        # Status of all modules and components
│   ├── TASKS.md                        # Task breakdown and work streams
│   ├── KNOWN_ISSUES.md                 # Known provider limitations and workarounds
│   └── CHANGELOG.md                    # Release history and audit log
└── memory/
    ├── CURRENT_STATE.md                # Live snapshot of repository capabilities
    ├── NEXT_STEPS.md                   # Immediate engineering priorities
    └── SESSION_LOG.md                  # Chronological engineering session diary
```

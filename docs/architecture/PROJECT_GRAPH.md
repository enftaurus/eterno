# Eterno Architecture Graph

This document details the architectural topology of the Eterno platform using Mermaid diagrams.

## 1. System Context Diagram

```mermaid
graph TD
    Student[("Student / Developer")]
    Admin[("College / Institution Admin")]
    PlatformAdmin[("Platform Admin")]

    subgraph Eterno ["Eterno Platform"]
        WebUI["React TypeScript SPA\n(Clean White Minimal UI)"]
        APIGateway["Go API Service\n(Chi/Gin HTTP Engine)"]
        Worker["Go Background Worker\n(Persistent Queue Consumer)"]
        Database[("PostgreSQL 16\n(Persistent ACID Storage)")]
    end

    subgraph ExternalPlatforms ["External Developer Platforms"]
        LeetCode["LeetCode GraphQL / API"]
        GitHub["GitHub REST / GraphQL"]
        Codeforces["Codeforces API"]
    end

    Student -->|Solves challenges, links accounts| WebUI
    Admin -->|Publishes challenges, monitors solves| WebUI
    PlatformAdmin -->|System operations & health| WebUI

    WebUI -->|REST API Requests /api/v1| APIGateway
    APIGateway <-->|CRUD, RBAC, Auth| Database

    Worker <-->|Polls jobs with row locks| Database
    Worker -->|Fetch submissions & profile proofs| LeetCode
    Worker -->|Fetch commits| GitHub
    Worker -->|Fetch contest solves| Codeforces
```

## 2. Runtime Architecture

```mermaid
graph LR
    subgraph Frontend ["React 18 + Vite Frontend"]
        Router["React Router v6"]
        Query["TanStack Query Cache"]
        Views["Student & Admin Dashboards"]
        Theme["Clean White Design System"]
    end

    subgraph BackendAPI ["Go API Daemon (cmd/api)"]
        MW["Middleware:\nRequestID, Auth, RBAC, TenantScope, Logger"]
        Handlers["HTTP Handlers (/api/v1)"]
        DomainServices["Domain Services:\nAuth, Challenges, Verification, Scoring, Analytics"]
        Repo["PostgreSQL Repository Layer"]
    end

    subgraph BackendWorker ["Go Worker Daemon (cmd/worker)"]
        Scheduler["Job Poller (5s interval)"]
        Claimer["Job Claimer (FOR UPDATE SKIP LOCKED)"]
        SyncWorker["Verification & Ingestion Worker"]
    end

    subgraph Storage ["PostgreSQL 16"]
        CoreTables[("Users, Institutions, Memberships")]
        ChallengeTables[("Challenges, Problems, Assignments")]
        SubmissionTables[("Submissions, Completions, ScoreEvents")]
        ActivityTables[("ActivityEvents, Streaks")]
        JobQueue[("SyncJobs, SyncRuns")]
    end

    subgraph Adapters ["Platform Provider Adapters"]
        LCAdapter["LeetCode Adapter"]
        GHAdapter["GitHub Adapter"]
        CFAdapter["Codeforces Adapter"]
    end

    Frontend -->|JSON / REST| MW
    MW --> Handlers
    Handlers --> DomainServices
    DomainServices --> Repo
    Repo <--> CoreTables & ChallengeTables & SubmissionTables & ActivityTables

    Scheduler --> Claimer
    Claimer <--> JobQueue
    Claimer --> SyncWorker
    SyncWorker --> Adapters
    SyncWorker --> DomainServices
```

## 3. Challenge Verification Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    actor Admin as College Admin
    actor Student as Student
    participant UI as React Frontend
    participant API as Go API Service
    participant DB as PostgreSQL 16
    participant Worker as Verification Worker
    participant LC as LeetCode Provider Adapter

    Admin->>UI: Create challenge with LeetCode slug
    UI->>API: POST /api/v1/admin/challenges
    API->>DB: INSERT INTO challenges & challenge_problems
    Admin->>UI: Publish challenge
    UI->>API: POST /api/v1/admin/challenges/{id}/publish
    API->>DB: UPDATE status = ACTIVE & INSERT challenge_assignments (baseline recorded)
    API-->>UI: 200 OK (Challenge Active)

    Student->>UI: View assigned challenge
    UI->>API: GET /api/v1/me/challenges/{id}
    API->>DB: SELECT assignment, problems, verified status
    API-->>UI: Returns challenge status (e.g. IN_PROGRESS)

    Student->>LC: Submits Accepted solution on LeetCode
    Student->>UI: Clicks "Sync & Verify Submissions"
    UI->>API: POST /api/v1/me/platform-accounts/leetcode/sync
    API->>DB: Enqueue sync_job (PENDING)
    API-->>UI: 202 Accepted (Sync Enqueued)

    Worker->>DB: Claim job (SELECT ... FOR UPDATE SKIP LOCKED)
    Worker->>LC: Fetch recent submissions for student LeetCode username
    LC-->>Worker: Return submission records (slug, verdict, timestamp, provider ID)
    Worker->>DB: Ingest deduplicated submissions (INSERT ... ON CONFLICT DO NOTHING)
    Worker->>Worker: Run Verification Engine (slug match, verdict = Accepted, timestamp window)
    Worker->>DB: INSERT challenge_completions & score_events (idempotent)
    Worker->>DB: INSERT activity_events (source=eterno_challenge)
    Worker->>DB: UPDATE challenge_assignments SET status = VERIFIED
    Worker->>DB: UPDATE sync_jobs SET status = COMPLETED

    Student->>UI: Refresh / query invalidation
    UI->>API: GET /api/v1/me/challenges/{id}
    API->>DB: Load verified completion and score
    API-->>UI: Display challenge as VERIFIED with points awarded!
    Admin->>UI: View Overview Dashboard
    UI->>API: GET /api/v1/admin/dashboard
    API-->>UI: Displays student in today's verified solves & updated leaderboard
```

## 4. Entity Relationship Diagram

```mermaid
erDiagram
    users ||--o{ institution_memberships : has
    institutions ||--o{ institution_memberships : contains
    institutions ||--o{ departments : organizes
    institutions ||--o{ batches : organizes
    institutions ||--o{ groups : fosters
    users ||--o{ platform_accounts : links
    institutions ||--o{ challenges : hosts
    challenges ||--|{ challenge_problems : contains
    challenges ||--o{ challenge_assignments : assigns
    users ||--o{ challenge_assignments : receives
    users ||--o{ submissions : executes
    challenge_assignments ||--o{ challenge_completions : verifies
    challenge_problems ||--o{ challenge_completions : satisfies
    submissions ||--o{ challenge_completions : validates
    users ||--o{ score_events : earns
    users ||--o{ activity_events : generates
    users ||--o{ sync_jobs : triggers
    sync_jobs ||--o{ sync_runs : logs

    users {
        uuid id PK
        string email UK
        string password_hash
        string display_name
        string role
        boolean is_active
        timestamp created_at
    }

    institutions {
        uuid id PK
        string name
        string slug UK
        string timezone
    }

    challenges {
        uuid id PK
        uuid institution_id FK
        string title
        string status
        timestamp start_time
        timestamp deadline
        int max_points
    }

    challenge_problems {
        uuid id PK
        uuid challenge_id FK
        string problem_slug
        string difficulty
        int points
    }

    challenge_assignments {
        uuid id PK
        uuid challenge_id FK
        uuid user_id FK
        string status
        timestamp baseline_at
        int points_awarded
    }

    submissions {
        uuid id PK
        uuid user_id FK
        string platform
        string provider_submission_id UK
        string problem_slug
        string verdict
        boolean is_accepted
        timestamp submitted_at
    }

    challenge_completions {
        uuid id PK
        uuid challenge_id FK
        uuid challenge_problem_id FK
        uuid user_id FK
        uuid submission_id FK
        timestamp verified_at
    }

    activity_events {
        uuid id PK
        uuid user_id FK
        string source
        string external_id UK
        string event_type
        timestamp occurred_at
    }
```

## 5. Deployment Architecture

```mermaid
graph TD
    Client["User Browser (Desktop / Mobile)"]

    subgraph ProductionHost ["Docker Compose / Linux Host"]
        Caddy["Caddy Reverse Proxy\n(HTTPS, TLS termination, Port 80/443)"]

        subgraph EternoNetwork ["Docker Isolated Bridge Network"]
            SPA["Frontend Static Container / Dist\n(Nginx or Static Server)"]
            GoAPI["Eterno API Container\n(:8080)"]
            GoWorker["Eterno Worker Container\n(Background Processor)"]
            PG[("PostgreSQL 16 Alpine\n(Port 5432 internal, 5434 host)")]
        end
    end

    Client -->|HTTPS: 443| Caddy
    Caddy -->|/api/*| GoAPI
    Caddy -->|/*| SPA
    GoAPI --> PG
    GoWorker --> PG
```

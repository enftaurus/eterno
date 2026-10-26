-- 000001_initial_schema.up.sql
-- Eterno College Coding Platform & Developer Intelligence Schema

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    avatar_url TEXT,
    role VARCHAR(50) NOT NULL DEFAULT 'STUDENT',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Institutions (Colleges / Universities)
CREATE TABLE IF NOT EXISTS institutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    domain VARCHAR(255),
    logo_url TEXT,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Departments
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    code VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(institution_id, name)
);

-- 4. Batches (Graduating Classes)
CREATE TABLE IF NOT EXISTS batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    graduation_year INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(institution_id, name)
);

-- 5. Peer Groups & Coding Clubs
CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    group_type VARCHAR(50) NOT NULL DEFAULT 'CODING_CLUB',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(institution_id, name)
);

-- 6. Institution Memberships
CREATE TABLE IF NOT EXISTS institution_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    batch_id UUID REFERENCES batches(id) ON DELETE SET NULL,
    student_id_number VARCHAR(100),
    role VARCHAR(50) NOT NULL DEFAULT 'STUDENT',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(institution_id, user_id)
);

-- 7. Group Memberships
CREATE TABLE IF NOT EXISTS group_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'MEMBER',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, user_id)
);

-- 8. Platform Accounts (LeetCode, GitHub, Codeforces)
CREATE TABLE IF NOT EXISTS platform_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    username VARCHAR(100) NOT NULL,
    external_user_id VARCHAR(100),
    verification_status VARCHAR(50) NOT NULL DEFAULT 'UNVERIFIED',
    verification_token VARCHAR(100),
    verification_attempt_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    last_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(50) NOT NULL DEFAULT 'IDLE',
    sync_error_message TEXT,
    profile_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(platform, username),
    UNIQUE(user_id, platform)
);

-- 9. Challenges
CREATE TABLE IF NOT EXISTS challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    instructions TEXT,
    challenge_type VARCHAR(50) NOT NULL DEFAULT 'DAILY',
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    start_time TIMESTAMPTZ NOT NULL,
    deadline TIMESTAMPTZ NOT NULL,
    scope_type VARCHAR(50) NOT NULL DEFAULT 'INSTITUTION',
    scope_id UUID,
    max_points INT NOT NULL DEFAULT 100,
    historical_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 10. Challenge Problems
CREATE TABLE IF NOT EXISTS challenge_problems (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL DEFAULT 'leetcode',
    problem_url TEXT NOT NULL,
    problem_slug VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    difficulty VARCHAR(50) NOT NULL,
    points INT NOT NULL DEFAULT 100,
    is_bonus BOOLEAN NOT NULL DEFAULT FALSE,
    order_index INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(challenge_id, problem_slug)
);

-- 11. Challenge Assignments
CREATE TABLE IF NOT EXISTS challenge_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'NOT_STARTED',
    baseline_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    points_awarded INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(challenge_id, user_id)
);

-- 12. Submissions
CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL DEFAULT 'leetcode',
    provider_submission_id VARCHAR(150) NOT NULL,
    problem_slug VARCHAR(255) NOT NULL,
    title VARCHAR(255),
    verdict VARCHAR(100) NOT NULL,
    is_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    language VARCHAR(50),
    runtime_ms INT,
    memory_mb NUMERIC(8,2),
    submitted_at TIMESTAMPTZ NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(platform, provider_submission_id)
);

-- 13. Challenge Completions
CREATE TABLE IF NOT EXISTS challenge_completions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    assignment_id UUID NOT NULL REFERENCES challenge_assignments(id) ON DELETE CASCADE,
    challenge_problem_id UUID NOT NULL REFERENCES challenge_problems(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verification_source VARCHAR(100) NOT NULL DEFAULT 'LEETCODE_SYNC',
    evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(challenge_problem_id, user_id)
);

-- 14. Score Events (Auditable Ledger)
CREATE TABLE IF NOT EXISTS score_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    challenge_id UUID REFERENCES challenges(id) ON DELETE SET NULL,
    completion_id UUID REFERENCES challenge_completions(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL,
    points INT NOT NULL,
    policy_version VARCHAR(20) NOT NULL DEFAULT 'v1',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 15. Activity Events (Unified Heatmap)
CREATE TABLE IF NOT EXISTS activity_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source VARCHAR(50) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source, external_id)
);

-- 16. Persistent Background Sync Jobs
CREATE TABLE IF NOT EXISTS sync_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    job_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ,
    locked_by VARCHAR(100),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 17. Sync Runs (Job Execution Auditing)
CREATE TABLE IF NOT EXISTS sync_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES sync_jobs(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    records_ingested INT NOT NULL DEFAULT 0,
    records_verified INT NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- 18. In-App Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    link TEXT,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 19. Audit Logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID REFERENCES institutions(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for Query Performance & Tenant Scoping
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_inst_memberships_inst_user ON institution_memberships(institution_id, user_id);
CREATE INDEX IF NOT EXISTS idx_inst_memberships_batch ON institution_memberships(batch_id);
CREATE INDEX IF NOT EXISTS idx_inst_memberships_dept ON institution_memberships(department_id);
CREATE INDEX IF NOT EXISTS idx_challenges_inst_status ON challenges(institution_id, status);
CREATE INDEX IF NOT EXISTS idx_challenges_deadline ON challenges(deadline);
CREATE INDEX IF NOT EXISTS idx_assignments_user_status ON challenge_assignments(user_id, status);
CREATE INDEX IF NOT EXISTS idx_assignments_challenge_inst ON challenge_assignments(challenge_id, institution_id);
CREATE INDEX IF NOT EXISTS idx_submissions_user_platform_slug ON submissions(user_id, platform, problem_slug);
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_at ON submissions(submitted_at);
CREATE INDEX IF NOT EXISTS idx_completions_challenge_user ON challenge_completions(challenge_id, user_id);
CREATE INDEX IF NOT EXISTS idx_score_events_inst_user ON score_events(institution_id, user_id);
CREATE INDEX IF NOT EXISTS idx_score_events_created ON score_events(created_at);
CREATE INDEX IF NOT EXISTS idx_activity_events_user_occurred ON activity_events(user_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_sync_jobs_status_next ON sync_jobs(status, next_run_at);
CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_audit_logs_inst_created ON audit_logs(institution_id, created_at);

-- 000003_developer_scores.up.sql
-- Eterno / Code Crusaders Developer Intelligence & Multi-Component Score Table

CREATE TABLE IF NOT EXISTS developer_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_score INT NOT NULL DEFAULT 0 CHECK (total_score >= 0 AND total_score <= 100),
    consistency_score NUMERIC(5,2) NOT NULL DEFAULT 0.00 CHECK (consistency_score >= 0 AND consistency_score <= 100),
    problem_solving_score NUMERIC(5,2) NOT NULL DEFAULT 0.00 CHECK (problem_solving_score >= 0 AND problem_solving_score <= 100),
    competitive_score NUMERIC(5,2) NOT NULL DEFAULT 0.00 CHECK (competitive_score >= 0 AND competitive_score <= 100),
    engineering_score NUMERIC(5,2) NOT NULL DEFAULT 0.00 CHECK (engineering_score >= 0 AND engineering_score <= 100),
    growth_score NUMERIC(5,2) NOT NULL DEFAULT 0.00 CHECK (growth_score >= 0 AND growth_score <= 100),
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    recommendations JSONB NOT NULL DEFAULT '[]'::jsonb,
    skill_radar JSONB NOT NULL DEFAULT '{}'::jsonb,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_developer_scores_total ON developer_scores(total_score DESC);
CREATE INDEX IF NOT EXISTS idx_developer_scores_user ON developer_scores(user_id);

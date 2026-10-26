-- Seed data for Eterno Demo Platform

DO $$
DECLARE
    inst_id UUID;
    cs_dept_id UUID;
    ee_dept_id UUID;
    b26_id UUID;
    b27_id UUID;
    club_id UUID;
    admin_id UUID;
    s1_id UUID;
    s2_id UUID;
    s3_id UUID;
    s4_id UUID;
    ch1_id UUID;
    ch2_id UUID;
    cp1_id UUID;
    cp2_id UUID;
    cp3_id UUID;
    sub1_id UUID;
    sub2_id UUID;
    comp1_id UUID;
BEGIN
    -- 1. Create Demo Institution
    INSERT INTO institutions (name, slug, domain, timezone)
    VALUES ('Stanford School of Engineering', 'stanford-engineering', 'stanford.edu', 'UTC')
    ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
    RETURNING id INTO inst_id;

    -- 2. Departments
    INSERT INTO departments (institution_id, name, code)
    VALUES (inst_id, 'Computer Science', 'CS')
    ON CONFLICT (institution_id, name) DO UPDATE SET code = EXCLUDED.code
    RETURNING id INTO cs_dept_id;

    INSERT INTO departments (institution_id, name, code)
    VALUES (inst_id, 'Electrical Engineering', 'EE')
    ON CONFLICT (institution_id, name) DO UPDATE SET code = EXCLUDED.code
    RETURNING id INTO ee_dept_id;

    -- 3. Batches
    INSERT INTO batches (institution_id, department_id, name, graduation_year)
    VALUES (inst_id, cs_dept_id, 'Batch of 2026', 2026)
    ON CONFLICT (institution_id, name) DO UPDATE SET graduation_year = EXCLUDED.graduation_year
    RETURNING id INTO b26_id;

    INSERT INTO batches (institution_id, department_id, name, graduation_year)
    VALUES (inst_id, cs_dept_id, 'Batch of 2027', 2027)
    ON CONFLICT (institution_id, name) DO UPDATE SET graduation_year = EXCLUDED.graduation_year
    RETURNING id INTO b27_id;

    -- 4. Users (Bcrypt hash for 'Password123!')
    -- $2a$10$w8uDk2c6v2n7j2jR.9G16O1b/7r0eUjW9q6q8iP0sF2zH/YyUa4yC -> standard bcrypt
    -- For test ease, let's insert users:
    INSERT INTO users (email, password_hash, display_name, role)
    VALUES ('admin@stanford.edu', '$2a$10$w8uDk2c6v2n7j2jR.9G16O1b/7r0eUjW9q6q8iP0sF2zH/YyUa4yC', 'Prof. John McCarthy', 'COLLEGE_ADMIN')
    ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
    RETURNING id INTO admin_id;

    INSERT INTO users (email, password_hash, display_name, role)
    VALUES ('alice@stanford.edu', '$2a$10$w8uDk2c6v2n7j2jR.9G16O1b/7r0eUjW9q6q8iP0sF2zH/YyUa4yC', 'Alice Chen', 'STUDENT')
    ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
    RETURNING id INTO s1_id;

    INSERT INTO users (email, password_hash, display_name, role)
    VALUES ('bob@stanford.edu', '$2a$10$w8uDk2c6v2n7j2jR.9G16O1b/7r0eUjW9q6q8iP0sF2zH/YyUa4yC', 'Bob Miller', 'STUDENT')
    ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
    RETURNING id INTO s2_id;

    INSERT INTO users (email, password_hash, display_name, role)
    VALUES ('charlie@stanford.edu', '$2a$10$w8uDk2c6v2n7j2jR.9G16O1b/7r0eUjW9q6q8iP0sF2zH/YyUa4yC', 'Charlie Patel', 'STUDENT')
    ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
    RETURNING id INTO s3_id;

    INSERT INTO users (email, password_hash, display_name, role)
    VALUES ('diana@stanford.edu', '$2a$10$w8uDk2c6v2n7j2jR.9G16O1b/7r0eUjW9q6q8iP0sF2zH/YyUa4yC', 'Diana Prince', 'STUDENT')
    ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
    RETURNING id INTO s4_id;

    -- 5. Memberships
    INSERT INTO institution_memberships (institution_id, user_id, department_id, batch_id, student_id_number, role)
    VALUES 
        (inst_id, admin_id, cs_dept_id, NULL, 'FAC-001', 'COLLEGE_ADMIN'),
        (inst_id, s1_id, cs_dept_id, b26_id, 'STU-2026-001', 'STUDENT'),
        (inst_id, s2_id, cs_dept_id, b26_id, 'STU-2026-002', 'STUDENT'),
        (inst_id, s3_id, ee_dept_id, b27_id, 'STU-2027-003', 'STUDENT'),
        (inst_id, s4_id, cs_dept_id, b26_id, 'STU-2026-004', 'STUDENT')
    ON CONFLICT (institution_id, user_id) DO NOTHING;

    -- 6. Groups
    INSERT INTO groups (institution_id, name, description, group_type, created_by)
    VALUES (inst_id, 'Competitive Programming Club', 'Varsity ACM-ICPC and algorithmic contest team', 'CODING_CLUB', admin_id)
    ON CONFLICT (institution_id, name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO club_id;

    INSERT INTO group_memberships (group_id, user_id, role)
    VALUES 
        (club_id, s1_id, 'OFFICER'),
        (club_id, s2_id, 'MEMBER'),
        (club_id, s4_id, 'MEMBER')
    ON CONFLICT (group_id, user_id) DO NOTHING;

    -- 7. Platform accounts
    INSERT INTO platform_accounts (user_id, platform, username, verification_status, verified_at, last_sync_status, last_sync_at)
    VALUES 
        (s1_id, 'leetcode', 'alice_coder', 'VERIFIED', NOW() - INTERVAL '10 days', 'SUCCESS', NOW() - INTERVAL '10 minutes'),
        (s2_id, 'leetcode', 'bob_dev', 'VERIFIED', NOW() - INTERVAL '8 days', 'SUCCESS', NOW() - INTERVAL '2 hours'),
        (s3_id, 'leetcode', 'charlie_algo', 'PENDING', NULL, 'IDLE', NULL),
        (s4_id, 'leetcode', 'diana_code', 'VERIFIED', NOW() - INTERVAL '15 days', 'SUCCESS', NOW() - INTERVAL '1 hour')
    ON CONFLICT (user_id, platform) DO NOTHING;

    -- 8. Challenges
    INSERT INTO challenges (
        institution_id, title, description, instructions, challenge_type,
        status, start_time, deadline, scope_type, max_points, created_by
    ) VALUES (
        inst_id, 'Daily Algorithm Sprint: Two Sum & Recursion',
        'Solve Two Sum and Invert Binary Tree on LeetCode before midnight.',
        'Submit solutions directly on LeetCode with your connected account. Verification runs automatically.',
        'DAILY', 'ACTIVE', NOW() - INTERVAL '4 hours', NOW() + INTERVAL '20 hours', 'INSTITUTION', 200, admin_id
    ) RETURNING id INTO ch1_id;

    INSERT INTO challenge_problems (challenge_id, platform, problem_url, problem_slug, title, difficulty, points, order_index)
    VALUES 
        (ch1_id, 'leetcode', 'https://leetcode.com/problems/two-sum/', 'two-sum', 'Two Sum', 'Easy', 100, 0),
        (ch1_id, 'leetcode', 'https://leetcode.com/problems/invert-binary-tree/', 'invert-binary-tree', 'Invert Binary Tree', 'Easy', 100, 1);

    -- Assignments
    INSERT INTO challenge_assignments (challenge_id, user_id, institution_id, status, baseline_at, assigned_at)
    VALUES 
        (ch1_id, s1_id, inst_id, 'IN_PROGRESS', NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours'),
        (ch1_id, s2_id, inst_id, 'NOT_STARTED', NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours'),
        (ch1_id, s3_id, inst_id, 'NOT_STARTED', NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours'),
        (ch1_id, s4_id, inst_id, 'VERIFIED', NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours')
    ON CONFLICT (challenge_id, user_id) DO NOTHING;

    -- Second challenge: Weekly Competition
    INSERT INTO challenges (
        institution_id, title, description, instructions, challenge_type,
        status, start_time, deadline, scope_type, max_points, created_by
    ) VALUES (
        inst_id, 'Weekly Hard: Trapping Rain Water',
        'Master two pointers and monotonic stacks with Trapping Rain Water.',
        'Must be submitted during the active competition window.',
        'WEEKLY', 'ACTIVE', NOW() - INTERVAL '2 days', NOW() + INTERVAL '5 days', 'INSTITUTION', 300, admin_id
    ) RETURNING id INTO ch2_id;

    INSERT INTO challenge_problems (challenge_id, platform, problem_url, problem_slug, title, difficulty, points, order_index)
    VALUES (ch2_id, 'leetcode', 'https://leetcode.com/problems/trapping-rain-water/', 'trapping-rain-water', 'Trapping Rain Water', 'Hard', 300, 0);

    INSERT INTO challenge_assignments (challenge_id, user_id, institution_id, status, baseline_at, assigned_at)
    VALUES 
        (ch2_id, s1_id, inst_id, 'VERIFIED', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
        (ch2_id, s2_id, inst_id, 'NOT_STARTED', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
        (ch2_id, s3_id, inst_id, 'NOT_STARTED', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
        (ch2_id, s4_id, inst_id, 'NOT_STARTED', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days')
    ON CONFLICT (challenge_id, user_id) DO NOTHING;

    -- Submissions
    INSERT INTO submissions (
        user_id, platform, provider_submission_id, problem_slug, title,
        verdict, is_accepted, language, runtime_ms, memory_mb, submitted_at
    ) VALUES 
        (s1_id, 'leetcode', 'sub-seed-101', 'two-sum', 'Two Sum', 'Accepted', TRUE, 'golang', 4, 3.8, NOW() - INTERVAL '2 hours'),
        (s1_id, 'leetcode', 'sub-seed-102', 'trapping-rain-water', 'Trapping Rain Water', 'Accepted', TRUE, 'golang', 8, 5.2, NOW() - INTERVAL '1 day'),
        (s4_id, 'leetcode', 'sub-seed-103', 'two-sum', 'Two Sum', 'Accepted', TRUE, 'python3', 38, 17.4, NOW() - INTERVAL '3 hours'),
        (s4_id, 'leetcode', 'sub-seed-104', 'invert-binary-tree', 'Invert Binary Tree', 'Accepted', TRUE, 'python3', 32, 16.5, NOW() - INTERVAL '1 hour')
    ON CONFLICT (platform, provider_submission_id) DO NOTHING;

    -- Completions & Score Events
    INSERT INTO score_events (user_id, institution_id, challenge_id, event_type, points, description, created_at)
    VALUES 
        (s1_id, inst_id, ch1_id, 'CHALLENGE_SOLVE', 100, 'Solved Two Sum', NOW() - INTERVAL '2 hours'),
        (s1_id, inst_id, ch2_id, 'CHALLENGE_SOLVE', 300, 'Solved Trapping Rain Water', NOW() - INTERVAL '1 day'),
        (s4_id, inst_id, ch1_id, 'CHALLENGE_SOLVE', 100, 'Solved Two Sum', NOW() - INTERVAL '3 hours'),
        (s4_id, inst_id, ch1_id, 'CHALLENGE_SOLVE', 100, 'Solved Invert Binary Tree', NOW() - INTERVAL '1 hour'),
        (s2_id, inst_id, NULL, 'CHALLENGE_SOLVE', 50, 'Practice solve', NOW() - INTERVAL '3 days')
    ON CONFLICT DO NOTHING;

    -- Activity Events for Heatmap
    INSERT INTO activity_events (user_id, source, external_id, event_type, occurred_at)
    VALUES 
        (s1_id, 'leetcode', 'seed-act-1', 'accepted_submission', NOW() - INTERVAL '2 hours'),
        (s1_id, 'leetcode', 'seed-act-2', 'accepted_submission', NOW() - INTERVAL '1 day'),
        (s1_id, 'github', 'seed-act-3', 'git_commit', NOW() - INTERVAL '1 day'),
        (s1_id, 'github', 'seed-act-4', 'git_commit', NOW() - INTERVAL '2 days'),
        (s1_id, 'leetcode', 'seed-act-5', 'accepted_submission', NOW() - INTERVAL '3 days'),
        (s1_id, 'leetcode', 'seed-act-6', 'accepted_submission', NOW() - INTERVAL '4 days'),
        (s1_id, 'leetcode', 'seed-act-7', 'accepted_submission', NOW() - INTERVAL '5 days'),
        (s1_id, 'codeforces', 'seed-act-8', 'contest_solve', NOW() - INTERVAL '6 days'),
        (s4_id, 'leetcode', 'seed-act-9', 'accepted_submission', NOW() - INTERVAL '1 hour'),
        (s4_id, 'leetcode', 'seed-act-10', 'accepted_submission', NOW() - INTERVAL '3 hours')
    ON CONFLICT (source, external_id) DO NOTHING;

    RAISE NOTICE 'Eterno demonstration seed data successfully populated!';
END $$;

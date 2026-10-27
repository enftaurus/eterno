# Zero-Mock Hardening, Real Data Integration & Security Audit Report

**Date**: 2026-10-27  
**Commit Inspected**: `666b299`  
**Author**: Eterno Architecture & Security Team  
**Document Status**: AUTHORITATIVE AUDIT REPORT  

---

## 1. Executive Summary

An exhaustive audit of the Eterno codebase was conducted across configuration, database schema, domain services, external platform adapters, background queue processing, and the React frontend.

The audit revealed that while the core architectural design (modular Go monolith, PostgreSQL 16 schema, JWT claims, and Linear-inspired frontend) is sound, multiple critical areas rely on mock modes, unsafe fallback defaults, unverified verdict synthesis, and client-side error masking.

Every identified finding is cataloged below with severity, file path, function/component name, concrete evidence, classification, and strict acceptance criteria.

---

## 2. Catalog of Findings

### Finding 1: Unsafe Default Configuration for Mock External Providers
- **Category**: Confirmed mock/demo behavior
- **Severity**: P0 (Critical Security & Correctness)
- **File**: `backend/internal/config/config.go`
- **Function**: `Load()`
- **Evidence**:
  ```go
  mockProviders := getEnvBool("MOCK_EXTERNAL_PROVIDERS", true)
  ```
  When `MOCK_EXTERNAL_PROVIDERS` is unset in the environment, the backend silently enables mock mode by default.
- **Acceptance Criteria**:
  1. `MockExternalProviders` must default to `false`.
  2. If `Env == "production"`, `MockExternalProviders` must be strictly forbidden; startup must abort with a fatal error if enabled.
  3. `getEnvBool` must strictly parse `"true"`, `"1"`, `"false"`, `"0"`, and error out or fail safely on malformed booleans instead of silent fallbacks.

---

### Finding 2: Insecure Production JWT Secret Default
- **Category**: Confirmed defect
- **Severity**: P0 (Critical Security)
- **File**: `backend/internal/config/config.go`
- **Function**: `Load()`
- **Evidence**:
  ```go
  jwtSecret := getEnv("JWT_SECRET", "eterno-super-secret-jwt-key-2026-production-ready")
  ```
  The production binary can run with a publicly known default JWT secret if `JWT_SECRET` is omitted.
- **Acceptance Criteria**:
  1. In production (`ENV == "production"`), `JWT_SECRET` must be required, must be at least 32 characters long, and must never match the development placeholder.
  2. Startup must fail fast with a descriptive configuration error.

---

### Finding 3: Automatic Demo Database Seeding in Development Launcher
- **Category**: Confirmed mock/demo behavior
- **Severity**: P1 (High Operational Risk)
- **File**: `dev.sh`
- **Lines**: 115–122
- **Evidence**:
  ```bash
  USER_COUNT=$(docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT count(*) FROM users;" 2>/dev/null || echo "0")
  if [ "$USER_COUNT" -eq "0" ]; then
      echo -e "  Applying demonstration seed data for Stanford School of Engineering (seed.sql)..."
      docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" < "$PROJECT_ROOT/backend/migrations/seed.sql"
  ```
  `dev.sh` automatically seeds the database with Stanford mock users and challenges whenever the `users` table is empty.
- **Acceptance Criteria**:
  1. Schema migrations must only create empty tables and indexes.
  2. Demo data seeding must be strictly opt-in via a dedicated flag (`./dev.sh --seed` or separate `seed.sh`), never automatic.
  3. Real and test environments must start with a clean database.

---

### Finding 4: Insecure Substring Domain Check in LeetCode URL Parser
- **Category**: Confirmed defect
- **Severity**: P0 (Security Vulnerability / SSRF & URL spoofing)
- **File**: `backend/internal/platforms/leetcode/adapter.go`
- **Function**: `ParseProblemSlug(raw string)`
- **Evidence**:
  ```go
  host := strings.ToLower(u.Host)
  if !strings.Contains(host, "leetcode.com") {
      return "", errors.New("unsupported domain: problem URL must be from leetcode.com")
  }
  ```
  `strings.Contains(host, "leetcode.com")` allows spoofed hostnames such as `leetcode.com.evil.example` or `not-leetcode.com`.
- **Acceptance Criteria**:
  1. Parse the hostname using `u.Hostname()` and validate strictly against an allowlist: `leetcode.com` or `www.leetcode.com`.
  2. Reject all other hosts, schemes other than `https` (or `http` for local mock test server), userinfo `@evil.com`, and unexpected ports.

---

### Finding 5: Fabricated Accepted Verdict and Synthetic Fallback Timestamps
- **Category**: Confirmed defect
- **Severity**: P0 (Integrity Violation)
- **File**: `backend/internal/platforms/leetcode/adapter.go`
- **Function**: `FetchRecentSubmissions(ctx, username, limit)`
- **Evidence**:
  ```go
  if subTime.IsZero() {
      subTime = time.Now().UTC()
  }
  results = append(results, models.SubmissionDTO{
      ...
      Verdict: "Accepted",
      IsAccepted: true,
      SubmittedAt: subTime,
  })
  ```
  The adapter hardcodes `Verdict: "Accepted"` and `IsAccepted: true` for any item returned in `recentAcSubmissionList`. If a timestamp is missing or zero, it synthesizes `time.Now().UTC()`.
- **Acceptance Criteria**:
  1. Never substitute `time.Now()` for a missing or invalid timestamp; reject records with missing timestamps or mark evidence as incomplete.
  2. Clearly document that `recentAcSubmissionList` represents accepted-history evidence and never manufacture a synthetic verdict.
  3. Validate provider submission IDs: if missing or empty, reject the record rather than fabricating an ID.

---

### Finding 6: Unconditional Mock Account Verification Without Proof
- **Category**: Confirmed mock/demo behavior
- **Severity**: P0 (Security Vulnerability)
- **File**: `backend/internal/platforms/leetcode/adapter.go`
- **Function**: `VerifyAccount(ctx, username, expectedToken)`
- **Evidence**:
  ```go
  if a.mockMode {
      meta := map[string]interface{}{ "verified": true, ... }
      return true, meta, nil
  }
  ...
  verified := strings.Contains(aboutMe, expectedToken)
  ```
  In mock mode, ANY username is automatically verified. In live mode, `strings.Contains` allows substring collisions.
- **Acceptance Criteria**:
  1. Remove mock bypass from production code. Mock adapters must be injected as test implementations.
  2. Require exact token matching or bounded word-boundary token matching within the bio.
  3. Validate that the returned profile username matches the requested username under normalized comparison.
  4. Reject empty tokens and empty usernames.

---

### Finding 7: Single-Candidate Early Return in Multi-Challenge Verification
- **Category**: Confirmed defect
- **Severity**: P1 (Functional Correctness)
- **File**: `backend/internal/verification/service.go`
- **Function**: `VerifySubmission(ctx, sub)`
- **Evidence**:
  ```go
  for _, cand := range candidates {
      if !cand.historicalAllowed && sub.SubmittedAt.Before(cand.baselineAt) {
          return &VerificationResult{ Status: "REJECTED_TIME" }, nil
      }
      ...
      return &VerificationResult{ Status: "VERIFIED" }, nil
  }
  ```
  If a student is assigned two challenges sharing the same problem slug, rejecting or completing the first candidate returns immediately, leaving the second eligible challenge unverified.
- **Acceptance Criteria**:
  1. Process all eligible candidates matching the problem slug.
  2. An ineligible challenge (e.g. deadline expired or pre-baseline) must not block another eligible challenge.
  3. Return a batch or comprehensive verification report for all evaluated challenges.

---

### Finding 8: Race Condition & Lack of Concurrency Locking in Verification
- **Category**: Confirmed defect / Unverified risk
- **Severity**: P1 (Data Integrity / Duplicate Points)
- **File**: `backend/internal/verification/service.go`
- **Function**: `VerifySubmission(ctx, sub)`
- **Evidence**:
  ```go
  var existingCompID string
  checkCompQuery := `SELECT id FROM challenge_completions WHERE challenge_problem_id = $1 AND user_id = $2`
  err := s.db.QueryRowContext(ctx, checkCompQuery, cand.problemID, sub.UserID).Scan(&existingCompID)
  ```
  The completion check is performed outside the transaction. Concurrent workers processing two submissions for the same problem can both observe zero completions and attempt concurrent insertions.
- **Acceptance Criteria**:
  1. Enforce database-level unique constraint on `challenge_completions(challenge_problem_id, user_id)`.
  2. Insert completions with `ON CONFLICT (challenge_problem_id, user_id) DO NOTHING` inside the transaction.
  3. Add unique constraint on `score_events(user_id, source_type, source_id)` to guarantee idempotent point allocation.

---

### Finding 9: Stale Job Recovery Missing in Background Queue
- **Category**: Confirmed defect
- **Severity**: P1 (Reliability & Fault Tolerance)
- **File**: `backend/internal/jobs/queue.go`
- **Function**: `ClaimNextJob(ctx, workerID)`
- **Evidence**:
  When a worker claims a job, it sets `status = 'RUNNING'` and `locked_at = NOW()`. If the worker process crashes, the job remains in `RUNNING` status indefinitely with no automatic lease expiration or recovery.
- **Acceptance Criteria**:
  1. Add stale lease recovery: jobs in `RUNNING` status with `locked_at < NOW() - INTERVAL 'X minutes'` must be reclaimed or reset to `PENDING`.
  2. Implement `RecoverStaleJobs` or integrate lease check into `ClaimNextJob`.

---

### Finding 10: Hardcoded Fallback Metrics and Strings in Frontend
- **Category**: Confirmed mock/demo behavior
- **Severity**: P1 (Integrity & UX)
- **Files**:
  - `frontend/src/pages/student/ActivityHeatmap.tsx`: Static numbers "142 problems", "412 commits", "1,480 Specialist".
  - `frontend/src/pages/admin/Dashboard.tsx`: Hardcoded language fallback `{sub.language || 'golang'}` and `{institution?.name || 'Stanford School of Engineering'}`.
  - `frontend/src/pages/student/Leaderboards.tsx`: `{e.department_name || 'General CS'} • {e.batch_name || 'Batch 2026'}`.
  - `frontend/src/pages/admin/Consistency.tsx`: `{s.department_name || 'General CS'} • {s.batch_name || 'Class of 2026'}`.
- **Evidence**:
  Frontend components render fake static numbers and placeholder strings when actual data is unavailable.
- **Acceptance Criteria**:
  1. Remove all hardcoded metrics and fake platform counts.
  2. When a metric or property is unavailable, display an explicit `"Unavailable"`, `"Unknown"`, or `"—"`.
  3. Real data from the backend must populate all cards.

---

### Finding 11: Silent Error Masking and False Zero Displays in Frontend
- **Category**: Confirmed defect
- **Severity**: P1 (UX / Truth in Telemetry)
- **Files**: `frontend/src/pages/admin/Dashboard.tsx`, `frontend/src/pages/student/Overview.tsx`, etc.
- **Evidence**:
  ```tsx
  } catch (err) {
      console.error('Failed to load admin dashboard', err);
  } finally {
      setIsLoading(false);
  }
  ```
  On API error, `metrics` is null, causing `metrics?.total_registered_students || 0` to render `0` as if the institution has zero students.
- **Acceptance Criteria**:
  1. Every data-driven page must implement distinct states: Loading, Data, Empty, Error with Retry.
  2. Failed API calls must never render as zero metrics or empty tables. Display an explicit error banner with a retry button.

---

### Finding 12: CSV Formula Injection and Unescaped Fields
- **Category**: Confirmed defect
- **Severity**: P1 (Security Vulnerability)
- **Files**: `frontend/src/pages/admin/Students.tsx` and `backend/internal/institutions/service.go`
- **Evidence**:
  Frontend CSV export does not escape quotes, commas, or formula injection prefixes (`=`, `+`, `-`, `@`, `\t`, `\r`).
  Backend CSV import lacks file size bounds, row count bounds, and email format validation.
- **Acceptance Criteria**:
  1. Frontend and backend CSV exports must sanitize formula prefixes and quote fields properly.
  2. Backend CSV import must enforce a max file size (e.g., 2MB) and max row count (e.g., 1000 rows).
  3. Validate email syntax before creating user records.

---

### Finding 13: Unimplemented Platform Integrations Represented as Live
- **Category**: Future feature that is not implemented
- **Severity**: P2 (Integrity / Transparency)
- **Files**: `frontend/src/pages/student/ConnectedAccounts.tsx`, `frontend/src/pages/student/ActivityHeatmap.tsx`
- **Evidence**:
  GitHub and Codeforces accounts are displayed alongside LeetCode, but their server verification flows are not yet implemented.
- **Acceptance Criteria**:
  1. Clearly badge GitHub and Codeforces as `"Upcoming / Beta"` or `"Coming Soon"`.
  2. State transparently that automated verification is currently active for LeetCode.

---

## 3. Plan of Action & Audit Classification Summary

| # | Component | Finding Description | Severity | Classification | Target Phase |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | `backend/config` | Mock providers enabled by default | P0 | Confirmed mock/demo behavior | Phase B |
| 2 | `backend/config` | Hardcoded JWT secret in production | P0 | Confirmed defect | Phase B |
| 3 | `dev.sh` | Automatic demo data seeding | P1 | Confirmed mock/demo behavior | Phase B |
| 4 | `backend/leetcode` | Insecure hostname check in URL parser | P0 | Confirmed defect | Phase C |
| 5 | `backend/leetcode` | Fabricated Accepted verdict & synthetic timestamps | P0 | Confirmed defect | Phase C |
| 6 | `backend/leetcode` | Substring bio check & mock account bypass | P0 | Confirmed defect / mock | Phase C |
| 7 | `backend/verification` | Single-candidate early return in multi-challenge | P1 | Confirmed defect | Phase D |
| 8 | `backend/verification` | Concurrency race condition on completions/scores | P1 | Confirmed defect | Phase D |
| 9 | `backend/jobs` | Missing stale job lease recovery in queue | P1 | Confirmed defect | Phase D |
| 10 | `frontend/pages` | Hardcoded fake metrics and strings | P1 | Confirmed mock/demo behavior | Phase E |
| 11 | `frontend/pages` | Silent error catch masking API failure with zeroes | P1 | Confirmed defect | Phase E |
| 12 | `frontend/csv` | CSV formula injection and unescaped export | P1 | Confirmed defect | Phase E |
| 13 | `frontend/integrations` | Unimplemented platforms not labeled as beta | P2 | Future feature not implemented | Phase E |

---

## 4. Acceptance Sign-Off Criteria

Before declaring release candidate status:
1. Every P0 and P1 finding must be completely fixed in source code.
2. An isolated test database must be used for automated tests.
3. Automated edge-case matrix (100+ cases across provider, verification, worker, tenancy, and CSV) must pass.
4. Clean build and lint check across Go and TypeScript.

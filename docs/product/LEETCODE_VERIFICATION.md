# LeetCode Verification Protocol & Known Limitations

## 1. Account Ownership Proof Flow
1. User enters LeetCode username.
2. System generates a unique cryptographically random token: `eterno-verify-<uuid>`.
3. User temporarily pastes this token into their LeetCode Profile "About Me" / bio section.
4. User clicks "Verify Account" in Eterno.
5. Eterno queries the public LeetCode profile GraphQL endpoint (`userPublicProfile(username: $username)`).
6. If the token is found in `aboutMe`, account status transitions to `VERIFIED`.

## 2. Submission Ingestion & Verification
1. Eterno queries LeetCode's `recentAcSubmissionList(username: $username, limit: 20)` GraphQL query.
2. For each submission:
   - Normalize problem slug: strip trailing slashes, downcase, match alphanumeric hyphen pattern.
   - Check status: verdict must be `Accepted`.
   - Check timestamp: convert Unix epoch to UTC `time.Time`.
   - Persist into `submissions` table deduplicated on `(platform, provider_submission_id)`.
3. Verification engine checks active assignments for problems matching `problem_slug`:
   - If `submitted_at >= baseline_at` AND `submitted_at <= deadline`:
     - Creates `challenge_completions` record.
     - Awards points via `score_events`.
     - Updates `challenge_assignments` to `VERIFIED`.
     - Inserts canonical `activity_events` record.

## 3. Known Limitations & Failure Modes
- **Public API Availability**: LeetCode provides no official authenticated partner REST API; public GraphQL endpoints are subject to rate limiting and structural changes.
- **Recent Submissions Limit**: LeetCode's public GraphQL endpoint returns only the most recent 20 accepted submissions. Historical submissions older than this window must be persisted locally in `submissions` upon first detection.
- **Private Profiles**: Users with private profiles cannot be queried publicly; the verification flow explains this clearly to students.
- **Provider Outages**: Temporary network timeouts or 429 rate-limiting errors do not invalidate existing verified solves. Jobs are retried with exponential backoff and jitter.

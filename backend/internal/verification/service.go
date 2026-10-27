package verification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/eterno/backend/internal/database"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/scoring"
)

type Service struct {
	db             *sql.DB
	scoringService *scoring.Service
}

func NewService(db *sql.DB, scoringService *scoring.Service) *Service {
	return &Service{
		db:             db,
		scoringService: scoringService,
	}
}

type VerificationResult struct {
	ChallengeID        string `json:"challenge_id"`
	AssignmentID       string `json:"assignment_id"`
	ProblemID          string `json:"problem_id"`
	ProblemSlug        string `json:"problem_slug"`
	SubmissionID       string `json:"submission_id"`
	Status             string `json:"status"` // VERIFIED, REJECTED_TIME, REJECTED_VERDICT, REJECTED_STATUS, REJECTED_TIMESTAMP, ALREADY_VERIFIED, NOT_ASSIGNED
	PointsAwarded      int    `json:"points_awarded"`
	VerificationReason string `json:"verification_reason"`
}

// VerifySubmission evaluates a submission across all eligible challenges assigned to the user.
// Multi-challenge support: Ineligible challenges do not block evaluation of other eligible challenges.
func (s *Service) VerifySubmission(ctx context.Context, sub *models.Submission) (*VerificationResult, error) {
	results, err := s.VerifySubmissionAll(ctx, sub)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return &VerificationResult{
			SubmissionID:       sub.ID,
			ProblemSlug:        sub.ProblemSlug,
			Status:             "NOT_ASSIGNED",
			VerificationReason: "No active challenge found for this user containing this problem slug",
		}, nil
	}

	// Prioritize returning a VERIFIED result if any challenge was verified
	for _, res := range results {
		if res.Status == "VERIFIED" {
			return res, nil
		}
	}

	// Otherwise return the first meaningful status
	return results[0], nil
}

// VerifySubmissionAll processes and returns the verification results for all matching challenges.
func (s *Service) VerifySubmissionAll(ctx context.Context, sub *models.Submission) ([]*VerificationResult, error) {
	// 1. Strict server-side verification: Verdict check
	if !sub.IsAccepted {
		return []*VerificationResult{{
			SubmissionID:       sub.ID,
			ProblemSlug:        sub.ProblemSlug,
			Status:             "REJECTED_VERDICT",
			VerificationReason: fmt.Sprintf("Submission verdict is not Accepted (verdict: %s)", sub.Verdict),
		}}, nil
	}

	// 2. Strict server-side verification: Timestamp validity
	if sub.SubmittedAt.IsZero() {
		return []*VerificationResult{{
			SubmissionID:       sub.ID,
			ProblemSlug:        sub.ProblemSlug,
			Status:             "REJECTED_TIMESTAMP",
			VerificationReason: "Submission timestamp is missing or zero",
		}}, nil
	}

	now := time.Now().UTC()
	if sub.SubmittedAt.After(now.Add(10 * time.Minute)) {
		return []*VerificationResult{{
			SubmissionID:       sub.ID,
			ProblemSlug:        sub.ProblemSlug,
			Status:             "REJECTED_TIMESTAMP",
			VerificationReason: "Submission timestamp is in the future",
		}}, nil
	}

	// 3. Find any challenge assignment for this user containing this problem slug
	query := `
		SELECT ca.id AS assignment_id, ca.challenge_id, ca.institution_id, ca.status AS assignment_status,
		       ca.baseline_at, c.status AS challenge_status, c.start_time, c.deadline, c.historical_allowed,
		       cp.id AS problem_id, cp.points, cp.difficulty, cp.is_bonus
		FROM challenge_assignments ca
		JOIN challenges c ON ca.challenge_id = c.id
		JOIN challenge_problems cp ON cp.challenge_id = c.id
		WHERE ca.user_id = $1
		  AND cp.problem_slug = $2
		ORDER BY c.deadline ASC
	`

	rows, err := s.db.QueryContext(ctx, query, sub.UserID, sub.ProblemSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to query challenge assignments: %w", err)
	}
	defer rows.Close()

	var candidates []struct {
		assignmentID      string
		challengeID       string
		institutionID     string
		assignmentStatus  string
		baselineAt        time.Time
		challengeStatus   string
		startTime         time.Time
		deadline          time.Time
		historicalAllowed bool
		problemID         string
		points            int
		difficulty        string
		isBonus           bool
	}

	for rows.Next() {
		var item struct {
			assignmentID      string
			challengeID       string
			institutionID     string
			assignmentStatus  string
			baselineAt        time.Time
			challengeStatus   string
			startTime         time.Time
			deadline          time.Time
			historicalAllowed bool
			problemID         string
			points            int
			difficulty        string
			isBonus           bool
		}
		if err := rows.Scan(
			&item.assignmentID, &item.challengeID, &item.institutionID, &item.assignmentStatus,
			&item.baselineAt, &item.challengeStatus, &item.startTime, &item.deadline, &item.historicalAllowed,
			&item.problemID, &item.points, &item.difficulty, &item.isBonus,
		); err != nil {
			return nil, err
		}
		candidates = append(candidates, item)
	}

	if len(candidates) == 0 {
		return []*VerificationResult{{
			SubmissionID:       sub.ID,
			ProblemSlug:        sub.ProblemSlug,
			Status:             "NOT_ASSIGNED",
			VerificationReason: "No challenge assignment found for this user containing this problem slug",
		}}, nil
	}

	var results []*VerificationResult

	for _, cand := range candidates {
		// Rule: Cancelled, Draft, or Scheduled challenges cannot award points
		if cand.challengeStatus == models.ChallengeCancelled {
			results = append(results, &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "REJECTED_STATUS",
				VerificationReason: "Challenge has been cancelled",
			})
			continue
		}
		if cand.challengeStatus == models.ChallengeDraft || cand.challengeStatus == models.ChallengeScheduled {
			results = append(results, &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "REJECTED_STATUS",
				VerificationReason: "Challenge is not currently active",
			})
			continue
		}

		// Time Boundary Rules:
		// Exact baseline: sub.SubmittedAt.Equal(cand.baselineAt) is valid.
		// Before baseline: strictly rejected when historicalAllowed is false.
		if !cand.historicalAllowed && sub.SubmittedAt.Before(cand.baselineAt) {
			slog.Warn("Submission rejected: submitted before challenge assignment baseline",
				"submitted_at", sub.SubmittedAt,
				"baseline_at", cand.baselineAt,
				"slug", sub.ProblemSlug,
			)
			results = append(results, &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "REJECTED_TIME",
				VerificationReason: "Submission occurred prior to challenge assignment baseline time",
			})
			continue
		}

		// Before challenge start: strictly rejected when historicalAllowed is false.
		if !cand.historicalAllowed && sub.SubmittedAt.Before(cand.startTime) {
			results = append(results, &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "REJECTED_TIME",
				VerificationReason: "Submission occurred prior to challenge start time",
			})
			continue
		}

		// After deadline: strictly rejected (exact deadline is allowed).
		if sub.SubmittedAt.After(cand.deadline) {
			slog.Warn("Submission rejected: submitted after deadline",
				"submitted_at", sub.SubmittedAt,
				"deadline", cand.deadline,
				"slug", sub.ProblemSlug,
			)
			results = append(results, &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "REJECTED_TIME",
				VerificationReason: "Submission was made after the challenge deadline",
			})
			continue
		}

		// Atomically verify with database concurrency protection
		var awardedPoints int
		var isNewCompletion bool

		err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
			evidenceJSON, _ := json.Marshal(map[string]interface{}{
				"provider_id":  sub.ProviderSubmissionID,
				"submitted_at": sub.SubmittedAt,
				"platform":     sub.Platform,
				"verdict":      sub.Verdict,
				"verified_at":  time.Now().UTC(),
			})

			// 1. Idempotent Insert Completion with concurrency guard (ON CONFLICT DO NOTHING)
			compQuery := `
				INSERT INTO challenge_completions (
					challenge_id, assignment_id, challenge_problem_id, user_id,
					submission_id, verified_at, verification_source, evidence
				) VALUES ($1, $2, $3, $4, $5, NOW(), 'LEETCODE_SYNC', $6)
				ON CONFLICT (challenge_problem_id, user_id) DO NOTHING
				RETURNING id
			`
			var completionID string
			err := tx.QueryRowContext(ctx, compQuery,
				cand.challengeID, cand.assignmentID, cand.problemID, sub.UserID,
				sub.ID, evidenceJSON,
			).Scan(&completionID)

			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					// Already completed concurrently or previously
					isNewCompletion = false
					return nil
				}
				return fmt.Errorf("failed to insert challenge completion: %w", err)
			}
			isNewCompletion = true

			// 2. Calculate Points
			awardedPoints = cand.points
			if awardedPoints <= 0 {
				awardedPoints = scoring.CalculateProblemPoints(cand.difficulty, 100)
			}

			// 3. Record Score Event Idempotently
			desc := fmt.Sprintf("Solved problem '%s' in challenge", sub.ProblemSlug)
			_, err = s.scoringService.AwardPoints(ctx, tx, sub.UserID, cand.institutionID, &cand.challengeID, &completionID, "CHALLENGE_SOLVE", awardedPoints, desc)
			if err != nil {
				return fmt.Errorf("failed to award points: %w", err)
			}

			// 4. Insert Canonical Activity Event Idempotently
			actQuery := `
				INSERT INTO activity_events (user_id, source, external_id, event_type, occurred_at, metadata)
				VALUES ($1, 'eterno_challenge', $2, 'challenge_verified', $3, $4)
				ON CONFLICT (source, external_id) DO NOTHING
			`
			actMeta, _ := json.Marshal(map[string]interface{}{
				"challenge_id": cand.challengeID,
				"problem_slug": sub.ProblemSlug,
				"points":       awardedPoints,
			})
			_, err = tx.ExecContext(ctx, actQuery, sub.UserID, completionID, sub.SubmittedAt, actMeta)
			if err != nil {
				return fmt.Errorf("failed to record activity event: %w", err)
			}

			// 5. Check if all required problems in challenge are now completed
			var totalReqProblems, completedReqProblems int
			countQuery := `
				SELECT 
					COUNT(cp.id) AS total_required,
					COUNT(cc.id) AS completed_required
				FROM challenge_problems cp
				LEFT JOIN challenge_completions cc 
				       ON cp.id = cc.challenge_problem_id AND cc.user_id = $1
				WHERE cp.challenge_id = $2 AND cp.is_bonus = FALSE
			`
			if err := tx.QueryRowContext(ctx, countQuery, sub.UserID, cand.challengeID).Scan(&totalReqProblems, &completedReqProblems); err != nil {
				return fmt.Errorf("failed to verify remaining problems: %w", err)
			}

			newStatus := models.AssignmentInProgress
			var completedAtParam interface{} = nil
			if completedReqProblems >= totalReqProblems && totalReqProblems > 0 {
				newStatus = models.AssignmentVerified
				completedAtParam = time.Now().UTC()
			}

			// 6. Update Challenge Assignment
			updateAssignQuery := `
				UPDATE challenge_assignments
				SET status = $1, points_awarded = points_awarded + $2,
				    completed_at = COALESCE($3, completed_at), updated_at = NOW()
				WHERE id = $4
			`
			_, err = tx.ExecContext(ctx, updateAssignQuery, newStatus, awardedPoints, completedAtParam, cand.assignmentID)
			if err != nil {
				return fmt.Errorf("failed to update challenge assignment: %w", err)
			}

			// 7. Notification
			notifQuery := `
				INSERT INTO notifications (user_id, title, message, type, link)
				VALUES ($1, $2, $3, 'VERIFICATION_SUCCESS', $4)
			`
			notifMsg := fmt.Sprintf("Your solution for '%s' was verified! +%d points awarded.", sub.ProblemSlug, awardedPoints)
			notifLink := fmt.Sprintf("/challenges/%s", cand.challengeID)
			_, _ = tx.ExecContext(ctx, notifQuery, sub.UserID, "Solution Verified!", notifMsg, notifLink)

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("verification transaction failed: %w", err)
		}

		if !isNewCompletion {
			results = append(results, &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "ALREADY_VERIFIED",
				VerificationReason: "Problem was already verified and points previously awarded",
			})
		} else {
			slog.Info("Successfully verified submission for challenge problem",
				"user_id", sub.UserID,
				"slug", sub.ProblemSlug,
				"points", awardedPoints,
				"challenge_id", cand.challengeID,
			)

			results = append(results, &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "VERIFIED",
				PointsAwarded:      awardedPoints,
				VerificationReason: "Solution matched canonical slug and satisfied time & verdict criteria",
			})
		}
	}

	return results, nil
}

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
	Status             string `json:"status"` // VERIFIED, REJECTED_TIME, REJECTED_VERDICT, ALREADY_VERIFIED
	PointsAwarded      int    `json:"points_awarded"`
	VerificationReason string `json:"verification_reason"`
}

func (s *Service) VerifySubmission(ctx context.Context, sub *models.Submission) (*VerificationResult, error) {
	if !sub.IsAccepted {
		return &VerificationResult{
			SubmissionID:       sub.ID,
			ProblemSlug:        sub.ProblemSlug,
			Status:             "REJECTED_VERDICT",
			VerificationReason: fmt.Sprintf("Submission verdict is not Accepted (verdict: %s)", sub.Verdict),
		}, nil
	}

	// Find any active challenge assignment for this user containing this problem slug
	query := `
		SELECT ca.id AS assignment_id, ca.challenge_id, ca.institution_id, ca.status AS assignment_status,
		       ca.baseline_at, c.start_time, c.deadline, c.historical_allowed,
		       cp.id AS problem_id, cp.points, cp.difficulty, cp.is_bonus
		FROM challenge_assignments ca
		JOIN challenges c ON ca.challenge_id = c.id
		JOIN challenge_problems cp ON cp.challenge_id = c.id
		WHERE ca.user_id = $1
		  AND cp.problem_slug = $2
		  AND c.status IN ('ACTIVE', 'COMPLETED')
	`

	rows, err := s.db.QueryContext(ctx, query, sub.UserID, sub.ProblemSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to query active assignments: %w", err)
	}
	defer rows.Close()

	var candidates []struct {
		assignmentID      string
		challengeID       string
		institutionID     string
		assignmentStatus  string
		baselineAt        time.Time
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
			&item.baselineAt, &item.startTime, &item.deadline, &item.historicalAllowed,
			&item.problemID, &item.points, &item.difficulty, &item.isBonus,
		); err != nil {
			return nil, err
		}
		candidates = append(candidates, item)
	}

	if len(candidates) == 0 {
		return &VerificationResult{
			SubmissionID:       sub.ID,
			ProblemSlug:        sub.ProblemSlug,
			Status:             "NOT_ASSIGNED",
			VerificationReason: "No active challenge found for this user containing this problem slug",
		}, nil
	}

	for _, cand := range candidates {
		// 1. Time boundary checks
		if !cand.historicalAllowed && sub.SubmittedAt.Before(cand.baselineAt) {
			slog.Warn("Submission rejected: submitted before challenge assignment baseline",
				"submitted_at", sub.SubmittedAt,
				"baseline_at", cand.baselineAt,
				"slug", sub.ProblemSlug,
			)
			return &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "REJECTED_TIME",
				VerificationReason: "Submission occurred prior to challenge assignment baseline time",
			}, nil
		}

		if sub.SubmittedAt.After(cand.deadline) {
			slog.Warn("Submission rejected: submitted after deadline",
				"submitted_at", sub.SubmittedAt,
				"deadline", cand.deadline,
				"slug", sub.ProblemSlug,
			)
			return &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "REJECTED_TIME",
				VerificationReason: "Submission was made after the challenge deadline",
			}, nil
		}

		// 2. Check if already completed
		var existingCompID string
		checkCompQuery := `SELECT id FROM challenge_completions WHERE challenge_problem_id = $1 AND user_id = $2`
		err := s.db.QueryRowContext(ctx, checkCompQuery, cand.problemID, sub.UserID).Scan(&existingCompID)
		if err == nil && existingCompID != "" {
			return &VerificationResult{
				ChallengeID:        cand.challengeID,
				AssignmentID:       cand.assignmentID,
				ProblemID:          cand.problemID,
				ProblemSlug:        sub.ProblemSlug,
				SubmissionID:       sub.ID,
				Status:             "ALREADY_VERIFIED",
				VerificationReason: "Problem was already verified and points previously awarded",
			}, nil
		}

		// 3. Atomically record completion, award points, insert canonical activity, update assignment
		var awardedPoints int
		err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
			evidenceJSON, _ := json.Marshal(map[string]interface{}{
				"provider_id":  sub.ProviderSubmissionID,
				"submitted_at": sub.SubmittedAt,
				"platform":     sub.Platform,
				"verdict":      sub.Verdict,
				"verified_at":  time.Now().UTC(),
			})

			// Insert Completion
			compQuery := `
				INSERT INTO challenge_completions (
					challenge_id, assignment_id, challenge_problem_id, user_id,
					submission_id, verified_at, verification_source, evidence
				) VALUES ($1, $2, $3, $4, $5, NOW(), 'LEETCODE_SYNC', $6)
				RETURNING id
			`
			var completionID string
			err := tx.QueryRowContext(ctx, compQuery,
				cand.challengeID, cand.assignmentID, cand.problemID, sub.UserID,
				sub.ID, evidenceJSON,
			).Scan(&completionID)
			if err != nil {
				return fmt.Errorf("failed to insert challenge completion: %w", err)
			}

			// Calculate Points
			awardedPoints = cand.points
			if awardedPoints <= 0 {
				awardedPoints = scoring.CalculateProblemPoints(cand.difficulty, 100)
			}

			// Record Score Event
			desc := fmt.Sprintf("Solved problem '%s' in challenge", sub.ProblemSlug)
			_, err = s.scoringService.AwardPoints(ctx, tx, sub.UserID, cand.institutionID, &cand.challengeID, &completionID, "CHALLENGE_SOLVE", awardedPoints, desc)
			if err != nil {
				return fmt.Errorf("failed to award points: %w", err)
			}

			// Insert Canonical Activity Event
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

			// Check if all required problems in challenge are now completed
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

			// Update Challenge Assignment
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

			// Create notification
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

		slog.Info("Successfully verified submission for challenge problem",
			"user_id", sub.UserID,
			"slug", sub.ProblemSlug,
			"points", awardedPoints,
			"challenge_id", cand.challengeID,
		)

		return &VerificationResult{
			ChallengeID:        cand.challengeID,
			AssignmentID:       cand.assignmentID,
			ProblemID:          cand.problemID,
			ProblemSlug:        sub.ProblemSlug,
			SubmissionID:       sub.ID,
			Status:             "VERIFIED",
			PointsAwarded:      awardedPoints,
			VerificationReason: "Solution matched canonical slug and satisfied time & verdict criteria",
		}, nil
	}

	return nil, errors.New("no matching verification candidate processed")
}

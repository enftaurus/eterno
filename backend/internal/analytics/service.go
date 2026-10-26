package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/eterno/backend/internal/models"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetAdminOverview(ctx context.Context, institutionID string) (*models.DashboardOverviewMetrics, error) {
	metrics := &models.DashboardOverviewMetrics{}

	// 1. Total registered students
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(user_id)
		FROM institution_memberships
		WHERE institution_id = $1 AND role = 'STUDENT' AND is_active = TRUE
	`, institutionID).Scan(&metrics.TotalRegisteredStudents)
	if err != nil {
		return nil, fmt.Errorf("failed to query total registered students: %w", err)
	}

	// 2. Solved at least one problem today
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT user_id)
		FROM challenge_completions cc
		JOIN challenges c ON cc.challenge_id = c.id
		WHERE c.institution_id = $1
		  AND cc.verified_at >= CURRENT_DATE
	`, institutionID).Scan(&metrics.SolvedProblemTodayCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query solved today: %w", err)
	}

	// 3. Completed today's assigned challenges
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT ca.user_id)
		FROM challenge_assignments ca
		JOIN challenges c ON ca.challenge_id = c.id
		WHERE ca.institution_id = $1
		  AND ca.status = 'VERIFIED'
		  AND ca.completed_at >= CURRENT_DATE
	`, institutionID).Scan(&metrics.CompletedAssignedTodayCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query completed assigned today: %w", err)
	}

	// 4. Inactive students (no verified solves in last 14 days)
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(im.user_id)
		FROM institution_memberships im
		WHERE im.institution_id = $1 AND im.role = 'STUDENT' AND im.is_active = TRUE
		  AND NOT EXISTS (
		    SELECT 1 FROM challenge_completions cc
		    JOIN challenges c ON cc.challenge_id = c.id
		    WHERE cc.user_id = im.user_id 
		      AND c.institution_id = $1
		      AND cc.verified_at >= NOW() - INTERVAL '14 days'
		  )
	`, institutionID).Scan(&metrics.InactiveStudentsCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query inactive students: %w", err)
	}

	// 5. Active challenges
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(id)
		FROM challenges
		WHERE institution_id = $1 AND status = 'ACTIVE'
	`, institutionID).Scan(&metrics.ActiveChallengesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query active challenges: %w", err)
	}

	// 6. Approaching deadlines (active, deadline within next 24 hours)
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(id)
		FROM challenges
		WHERE institution_id = $1 
		  AND status = 'ACTIVE' 
		  AND deadline > NOW() 
		  AND deadline <= NOW() + INTERVAL '24 hours'
	`, institutionID).Scan(&metrics.ApproachingDeadlinesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query approaching deadlines: %w", err)
	}

	// 7. Verified solves this week and this month
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(COUNT(CASE WHEN cc.verified_at >= NOW() - INTERVAL '7 days' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN cc.verified_at >= NOW() - INTERVAL '30 days' THEN 1 END), 0)
		FROM challenge_completions cc
		JOIN challenges c ON cc.challenge_id = c.id
		WHERE c.institution_id = $1
	`, institutionID).Scan(&metrics.VerifiedSolvesThisWeek, &metrics.VerifiedSolvesThisMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to query solves timeframe: %w", err)
	}

	// 8. Challenge completion rate (% of assigned that are verified)
	var totalAssignments, verifiedAssignments int
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(id),
			COUNT(CASE WHEN status = 'VERIFIED' THEN 1 END)
		FROM challenge_assignments
		WHERE institution_id = $1
	`, institutionID).Scan(&totalAssignments, &verifiedAssignments)
	if err == nil && totalAssignments > 0 {
		metrics.ChallengeCompletionRate = (float64(verifiedAssignments) / float64(totalAssignments)) * 100.0
	}

	// 9. Average verified problems per student
	if metrics.TotalRegisteredStudents > 0 {
		metrics.AvgProblemsPerStudent = float64(metrics.VerifiedSolvesThisMonth) / float64(metrics.TotalRegisteredStudents)
	}

	// 10. Recent submissions
	subRows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.user_id, s.platform, s.provider_submission_id, s.problem_slug,
		       s.title, s.verdict, s.is_accepted, s.language, s.runtime_ms, s.memory_mb,
		       s.submitted_at, s.raw_payload, s.created_at
		FROM submissions s
		JOIN institution_memberships im ON s.user_id = im.user_id AND im.institution_id = $1
		ORDER BY s.submitted_at DESC
		LIMIT 10
	`, institutionID)
	if err == nil {
		defer subRows.Close()
		for subRows.Next() {
			var sub models.Submission
			var title, lang sql.NullString
			var rMs sql.NullInt32
			var mMB sql.NullFloat64
			var raw []byte
			if err := subRows.Scan(
				&sub.ID, &sub.UserID, &sub.Platform, &sub.ProviderSubmissionID, &sub.ProblemSlug,
				&title, &sub.Verdict, &sub.IsAccepted, &lang, &rMs, &mMB,
				&sub.SubmittedAt, &raw, &sub.CreatedAt,
			); err == nil {
				if title.Valid {
					sub.Title = &title.String
				}
				if lang.Valid {
					sub.Language = &lang.String
				}
				if rMs.Valid {
					v := int(rMs.Int32)
					sub.RuntimeMs = &v
				}
				if mMB.Valid {
					sub.MemoryMB = &mMB.Float64
				}
				sub.RawPayload = raw
				metrics.RecentSubmissions = append(metrics.RecentSubmissions, sub)
			}
		}
	}

	return metrics, nil
}

func (s *Service) GetUserStreakAnalytics(ctx context.Context, userID string) (*models.StreakAnalytics, error) {
	// Query distinct active days (days where user had at least one accepted/verified problem)
	query := `
		SELECT DISTINCT DATE(occurred_at)::text AS active_date
		FROM activity_events
		WHERE user_id = $1
		ORDER BY active_date DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active dates: %w", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err == nil {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dates = append(dates, t)
			}
		}
	}

	analytics := &models.StreakAnalytics{}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	// Calculate Current Streak
	currentStreak := 0
	if len(dates) > 0 {
		lastActive := dates[0]
		if lastActive.Equal(today) || lastActive.Equal(yesterday) {
			currentStreak = 1
			expected := lastActive.AddDate(0, 0, -1)
			for i := 1; i < len(dates); i++ {
				if dates[i].Equal(expected) {
					currentStreak++
					expected = expected.AddDate(0, 0, -1)
				} else {
					break
				}
			}
		}
	}
	analytics.CurrentStreak = currentStreak

	// Calculate Longest Streak
	longestStreak := 0
	if len(dates) > 0 {
		tempStreak := 1
		for i := 0; i < len(dates)-1; i++ {
			diff := int(dates[i].Sub(dates[i+1]).Hours() / 24)
			if diff == 1 {
				tempStreak++
			} else {
				if tempStreak > longestStreak {
					longestStreak = tempStreak
				}
				tempStreak = 1
			}
		}
		if tempStreak > longestStreak {
			longestStreak = tempStreak
		}
	}
	analytics.LongestStreak = longestStreak

	// Active days in 7, 30, 90
	for _, d := range dates {
		daysAgo := int(today.Sub(d).Hours() / 24)
		if daysAgo <= 7 {
			analytics.ActiveDaysLast7++
		}
		if daysAgo <= 30 {
			analytics.ActiveDaysLast30++
		}
		if daysAgo <= 90 {
			analytics.ActiveDaysLast90++
		}
	}

	analytics.WeeklyParticipationRate = (float64(analytics.ActiveDaysLast7) / 7.0) * 100.0

	// Challenge completion rate for this student
	var totalAssigned, totalVerified int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(id), COUNT(CASE WHEN status = 'VERIFIED' THEN 1 END)
		FROM challenge_assignments
		WHERE user_id = $1
	`, userID).Scan(&totalAssigned, &totalVerified)
	if err == nil && totalAssigned > 0 {
		analytics.ChallengeCompletionRate = (float64(totalVerified) / float64(totalAssigned)) * 100.0
	}

	if analytics.ActiveDaysLast30 > 0 {
		var solvesLast30 int
		_ = s.db.QueryRowContext(ctx, `
			SELECT COUNT(id) FROM challenge_completions
			WHERE user_id = $1 AND verified_at >= NOW() - INTERVAL '30 days'
		`, userID).Scan(&solvesLast30)
		analytics.AvgVerifiedPerDay = float64(solvesLast30) / float64(analytics.ActiveDaysLast30)
	}

	return analytics, nil
}

package scoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/eterno/backend/internal/models"
)

// LogNormalize calculates N(x) = 100 * min(1, ln(1+x) / ln(1+K))
func LogNormalize(x, saturationK float64) float64 {
	if x <= 0 || saturationK <= 0 {
		return 0.0
	}
	ratio := math.Log(1.0+x) / math.Log(1.0+saturationK)
	if ratio > 1.0 {
		ratio = 1.0
	}
	return math.Round(ratio*10000.0) / 100.0 // 2 decimal places
}

// CalculateDeveloperScore queries real platform accounts, submissions, and activity events
// to compute an interpretable 0-100 Developer Score across the 5 core pillars.
func (s *Service) CalculateDeveloperScore(ctx context.Context, userID string) (*models.DeveloperScoreResponse, error) {
	now := time.Now().UTC()

	// 1. Consistency Pillar (25%)
	// Query active days in last 30 and 90 days
	var activeDays30, activeDays90 int
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(DISTINCT DATE(occurred_at)) FILTER (WHERE occurred_at >= NOW() - INTERVAL '30 days'),
			COUNT(DISTINCT DATE(occurred_at)) FILTER (WHERE occurred_at >= NOW() - INTERVAL '90 days')
		FROM activity_events
		WHERE user_id = $1
	`, userID).Scan(&activeDays30, &activeDays90)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed querying consistency metrics: %w", err)
	}

	// Calculate streaks from activity events
	streakQuery := `
		WITH daily_activity AS (
			SELECT DISTINCT DATE(occurred_at) AS act_date
			FROM activity_events
			WHERE user_id = $1
			ORDER BY act_date DESC
		),
		streak_groups AS (
			SELECT act_date,
			       act_date - (ROW_NUMBER() OVER (ORDER BY act_date DESC) * INTERVAL '1 day') AS grp
			FROM daily_activity
		)
		SELECT COUNT(*)::int AS current_streak
		FROM streak_groups
		WHERE grp = (SELECT grp FROM streak_groups WHERE act_date >= CURRENT_DATE - INTERVAL '1 day' LIMIT 1)
	`
	var currentStreak int
	_ = s.db.QueryRowContext(ctx, streakQuery, userID).Scan(&currentStreak)

	// Distinct active weeks (weeks with >= 2 active days) in last 90 days
	var activeWeeks int
	weeksQuery := `
		WITH weeks AS (
			SELECT DATE_TRUNC('week', occurred_at) AS wk, COUNT(DISTINCT DATE(occurred_at)) AS days
			FROM activity_events
			WHERE user_id = $1 AND occurred_at >= NOW() - INTERVAL '90 days'
			GROUP BY DATE_TRUNC('week', occurred_at)
		)
		SELECT COUNT(*)::int FROM weeks WHERE days >= 2
	`
	_ = s.db.QueryRowContext(ctx, weeksQuery, userID).Scan(&activeWeeks)

	cScoreRaw := 0.35*LogNormalize(float64(activeDays30), 22.0) +
		0.25*LogNormalize(float64(activeDays90), 65.0) +
		0.20*LogNormalize(float64(currentStreak), 15.0) +
		0.20*LogNormalize(float64(activeWeeks), 10.0)
	consistencyScore := math.Min(100.0, math.Max(0.0, cScoreRaw))

	// 2. Problem Solving Pillar (25%)
	// Query unique accepted problems from LeetCode and Codeforces
	var totalAccepted, easyCount, mediumCount, hardCount int
	subQuery := `
		SELECT 
			COUNT(DISTINCT problem_slug)::int,
			COUNT(DISTINCT problem_slug) FILTER (WHERE raw_payload->>'difficulty' = 'Easy' OR raw_payload->>'difficulty' = 'EASY' OR (raw_payload->'problem'->>'rating')::int < 1200)::int,
			COUNT(DISTINCT problem_slug) FILTER (WHERE raw_payload->>'difficulty' = 'Medium' OR raw_payload->>'difficulty' = 'MEDIUM' OR ((raw_payload->'problem'->>'rating')::int >= 1200 AND (raw_payload->'problem'->>'rating')::int < 1800))::int,
			COUNT(DISTINCT problem_slug) FILTER (WHERE raw_payload->>'difficulty' = 'Hard' OR raw_payload->>'difficulty' = 'HARD' OR (raw_payload->'problem'->>'rating')::int >= 1800)::int
		FROM submissions
		WHERE user_id = $1 AND is_accepted = TRUE
	`
	_ = s.db.QueryRowContext(ctx, subQuery, userID).Scan(&totalAccepted, &easyCount, &mediumCount, &hardCount)

	// In case difficulty wasn't tagged, infer from total
	if easyCount == 0 && mediumCount == 0 && hardCount == 0 && totalAccepted > 0 {
		easyCount = int(float64(totalAccepted) * 0.45)
		mediumCount = int(float64(totalAccepted) * 0.45)
		hardCount = totalAccepted - easyCount - mediumCount
	}

	weightedProblemPoints := float64(easyCount*1 + mediumCount*3 + hardCount*5)
	psScoreRaw := 0.70*LogNormalize(weightedProblemPoints, 350.0) + 0.30*LogNormalize(float64(totalAccepted), 120.0)
	problemSolvingScore := math.Min(100.0, math.Max(0.0, psScoreRaw))

	// 3. Competitive Programming Pillar (20%)
	// Query Codeforces profile metadata
	var cfMetaRaw []byte
	var cfUsername string
	_ = s.db.QueryRowContext(ctx, `
		SELECT username, profile_metadata FROM platform_accounts 
		WHERE user_id = $1 AND platform = 'codeforces' AND verification_status = 'VERIFIED'
	`, userID).Scan(&cfUsername, &cfMetaRaw)

	cfRating := 0
	cfMaxRating := 0
	if len(cfMetaRaw) > 0 {
		var meta struct {
			Rating    int `json:"rating"`
			MaxRating int `json:"max_rating"`
		}
		_ = json.Unmarshal(cfMetaRaw, &meta)
		cfRating = meta.Rating
		cfMaxRating = meta.MaxRating
	}

	var cfRatingScore float64
	if cfRating > 0 {
		switch {
		case cfRating < 800:
			cfRatingScore = 15.0
		case cfRating < 1200:
			cfRatingScore = 15.0 + (float64(cfRating-800)/400.0)*25.0 // 15 to 40
		case cfRating < 1400:
			cfRatingScore = 40.0 + (float64(cfRating-1200)/200.0)*20.0 // 40 to 60
		case cfRating < 1600:
			cfRatingScore = 60.0 + (float64(cfRating-1400)/200.0)*20.0 // 60 to 80
		case cfRating < 1900:
			cfRatingScore = 80.0 + (float64(cfRating-1600)/300.0)*12.0 // 80 to 92
		default:
			cfRatingScore = math.Min(100.0, 92.0+(float64(cfRating-1900)/400.0)*8.0)
		}
	} else {
		cfRatingScore = 0.0
	}

	// Contest participation count from activity events
	var contestCount int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(id)::int FROM activity_events 
		WHERE user_id = $1 AND source = 'codeforces' AND event_type = 'contest'
	`, userID).Scan(&contestCount)

	cpScoreRaw := 0.80*cfRatingScore + 0.20*LogNormalize(float64(contestCount), 15.0)
	competitiveScore := math.Min(100.0, math.Max(0.0, cpScoreRaw))

	// 4. Engineering Activity Pillar (20%)
	// Query GitHub activity events (commits, PRs, active repos)
	var ghCommits, ghActiveDays, ghRepos int
	_ = s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(id) FILTER (WHERE source = 'github')::int,
			COUNT(DISTINCT DATE(occurred_at)) FILTER (WHERE source = 'github' AND occurred_at >= NOW() - INTERVAL '90 days')::int,
			COUNT(DISTINCT metadata->>'repo')::int
		FROM activity_events
		WHERE user_id = $1 AND source = 'github'
	`, userID).Scan(&ghCommits, &ghActiveDays, &ghRepos)

	engScoreRaw := 0.40*LogNormalize(float64(ghCommits), 120.0) +
		0.40*LogNormalize(float64(ghActiveDays), 40.0) +
		0.20*LogNormalize(float64(ghRepos), 6.0)
	engineeringScore := math.Min(100.0, math.Max(0.0, engScoreRaw))

	// 5. Growth Pillar (10%)
	// Measures difficulty progression & recent acceleration
	var recentSolves30, priorSolves60 int
	_ = s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(id) FILTER (WHERE submitted_at >= NOW() - INTERVAL '30 days')::int,
			COUNT(id) FILTER (WHERE submitted_at < NOW() - INTERVAL '30 days' AND submitted_at >= NOW() - INTERVAL '90 days')::int
		FROM submissions
		WHERE user_id = $1 AND is_accepted = TRUE
	`, userID).Scan(&recentSolves30, &priorSolves60)

	growthVelocity := 0.0
	if recentSolves30 > 0 {
		if priorSolves60 == 0 {
			growthVelocity = 75.0 // Fresh momentum
		} else {
			ratio := (float64(recentSolves30) * 2.0) / float64(priorSolves60) // normalized per month
			growthVelocity = math.Min(100.0, ratio*50.0)
		}
	}

	hardRatio := 0.0
	if totalAccepted > 0 {
		hardRatio = math.Min(100.0, ((float64(hardCount)*3.0 + float64(mediumCount)*1.5)/float64(totalAccepted))*100.0)
	}

	growthScoreRaw := 0.60*growthVelocity + 0.40*hardRatio
	growthScore := math.Min(100.0, math.Max(0.0, growthScoreRaw))

	// Final Composite Developer Score
	totalComposite := 0.25*consistencyScore +
		0.25*problemSolvingScore +
		0.20*competitiveScore +
		0.20*engineeringScore +
		0.10*growthScore

	totalScore := int(math.Round(totalComposite))
	if totalScore > 100 {
		totalScore = 100
	} else if totalScore < 0 {
		totalScore = 0
	}

	// Topic Radar Breakdown
	skillRadar := []models.SkillRadarItem{
		{Topic: "Algorithms & Logic", Score: math.Min(100.0, problemSolvingScore*1.05), SolvedCount: totalAccepted, Target: 100},
		{Topic: "Data Structures", Score: math.Min(100.0, problemSolvingScore*0.95), SolvedCount: int(float64(totalAccepted) * 0.7), Target: 80},
		{Topic: "Dynamic Programming", Score: math.Min(100.0, float64(hardCount*8 + mediumCount*3)), SolvedCount: mediumCount + hardCount, Target: 40},
		{Topic: "Competitive Programming", Score: competitiveScore, SolvedCount: cfRating, Target: 1600},
		{Topic: "Open Source & Git", Score: engineeringScore, SolvedCount: ghCommits, Target: 100},
		{Topic: "Consistency & Habits", Score: consistencyScore, SolvedCount: activeDays30, Target: 25},
	}

	// Actionable Recommendations based on lowest components
	var recommendations []models.RecommendationItem
	if engineeringScore < 30.0 {
		recommendations = append(recommendations, models.RecommendationItem{
			Category:    "Engineering Activity",
			Title:       "Connect GitHub & Commit Consistently",
			Description: "Your engineering score is currently low. Link your GitHub account and push meaningful commits to your personal repositories to establish verified engineering signals.",
			ActionURL:   "/settings/connected-accounts",
			Impact:      "HIGH",
		})
	}
	if competitiveScore < 30.0 {
		recommendations = append(recommendations, models.RecommendationItem{
			Category:    "Competitive Programming",
			Title:       "Link Codeforces & Compete in Rated Contests",
			Description: "Participate in upcoming Codeforces Div 3 rounds to unlock rated competitive programming points on your profile.",
			ActionURL:   "/settings/connected-accounts",
			Impact:      "HIGH",
		})
	}
	if consistencyScore < 50.0 {
		recommendations = append(recommendations, models.RecommendationItem{
			Category:    "Consistency",
			Title:       "Build a 7-Day Problem Solving Streak",
			Description: "Solve at least 1 verified problem each day. Consistency accounts for 25% of your total developer score and compounds over weekly windows.",
			ActionURL:   "/challenges",
			Impact:      "MEDIUM",
		})
	}
	if problemSolvingScore < 60.0 {
		recommendations = append(recommendations, models.RecommendationItem{
			Category:    "Problem Solving",
			Title:       "Level Up to Medium Difficulty Solves",
			Description: "Medium problems yield 3x points and Hard problems yield 5x points. Target Medium Dynamic Programming and Graph challenges to boost problem-solving strength.",
			ActionURL:   "/challenges",
			Impact:      "HIGH",
		})
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, models.RecommendationItem{
			Category:    "Advanced Mastery",
			Title:       "Sustain Momentum & Mentor Peers",
			Description: "Your Developer Score is in top form! Maintain your active streak and compete in college leaderboards.",
			ActionURL:   "/leaderboards",
			Impact:      "LOW",
		})
	}

	response := &models.DeveloperScoreResponse{
		TotalScore: totalScore,
		Consistency: models.ScoreComponent{
			Name:         "Consistency",
			Score:        consistencyScore,
			Weight:       0.25,
			Contribution: math.Round(consistencyScore*0.25*10.0) / 10.0,
			Description:  "Sustained coding habit, active days, and streak resilience over 30 and 90-day horizons.",
			Metrics: map[string]interface{}{
				"active_days_30d": activeDays30,
				"active_days_90d": activeDays90,
				"current_streak":  currentStreak,
				"active_weeks":    activeWeeks,
			},
		},
		ProblemSolving: models.ScoreComponent{
			Name:         "Problem Solving",
			Score:        problemSolvingScore,
			Weight:       0.25,
			Contribution: math.Round(problemSolvingScore*0.25*10.0) / 10.0,
			Description:  "Accepted algorithmic challenges normalized by problem difficulty (Easy 1pt, Medium 3pts, Hard 5pts).",
			Metrics: map[string]interface{}{
				"total_accepted": totalAccepted,
				"easy_count":     easyCount,
				"medium_count":   mediumCount,
				"hard_count":     hardCount,
			},
		},
		Competitive: models.ScoreComponent{
			Name:         "Competitive Programming",
			Score:        competitiveScore,
			Weight:       0.20,
			Contribution: math.Round(competitiveScore*0.20*10.0) / 10.0,
			Description:  "Demonstrated contest rating and performance trends from Codeforces.",
			Metrics: map[string]interface{}{
				"codeforces_rating": cfRating,
				"max_rating":        cfMaxRating,
				"contests_count":    contestCount,
			},
		},
		Engineering: models.ScoreComponent{
			Name:         "Engineering Activity",
			Score:        engineeringScore,
			Weight:       0.20,
			Contribution: math.Round(engineeringScore*0.20*10.0) / 10.0,
			Description:  "Meaningful repository work, GitHub commits, and distinct active contribution days.",
			Metrics: map[string]interface{}{
				"commits_count": ghCommits,
				"active_days":   ghActiveDays,
				"active_repos":  ghRepos,
			},
		},
		Growth: models.ScoreComponent{
			Name:         "Growth & Trajectory",
			Score:        growthScore,
			Weight:       0.10,
			Contribution: math.Round(growthScore*0.10*10.0) / 10.0,
			Description:  "Progression to harder problem tiers and recent solve velocity compared to historic baseline.",
			Metrics: map[string]interface{}{
				"recent_solves_30d": recentSolves30,
				"prior_solves_60d":  priorSolves60,
			},
		},
		SkillRadar:      skillRadar,
		Recommendations: recommendations,
		CalculatedAt:    now,
	}

	// Persist snapshot to developer_scores table asynchronously or synchronously
	go func(res *models.DeveloperScoreResponse, uid string) {
		metricsJSON, _ := json.Marshal(map[string]interface{}{
			"consistency":    res.Consistency.Metrics,
			"problem_solving": res.ProblemSolving.Metrics,
			"competitive":    res.Competitive.Metrics,
			"engineering":    res.Engineering.Metrics,
			"growth":         res.Growth.Metrics,
		})
		recsJSON, _ := json.Marshal(res.Recommendations)
		radarJSON, _ := json.Marshal(res.SkillRadar)

		upsertQuery := `
			INSERT INTO developer_scores (
				user_id, total_score, consistency_score, problem_solving_score,
				competitive_score, engineering_score, growth_score,
				metrics, recommendations, skill_radar, calculated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
			ON CONFLICT (user_id) DO UPDATE SET
				total_score = EXCLUDED.total_score,
				consistency_score = EXCLUDED.consistency_score,
				problem_solving_score = EXCLUDED.problem_solving_score,
				competitive_score = EXCLUDED.competitive_score,
				engineering_score = EXCLUDED.engineering_score,
				growth_score = EXCLUDED.growth_score,
				metrics = EXCLUDED.metrics,
				recommendations = EXCLUDED.recommendations,
				skill_radar = EXCLUDED.skill_radar,
				calculated_at = NOW()
		`
		_, _ = s.db.Exec(upsertQuery,
			uid, res.TotalScore, res.Consistency.Score, res.ProblemSolving.Score,
			res.Competitive.Score, res.Engineering.Score, res.Growth.Score,
			metricsJSON, recsJSON, radarJSON,
		)
	}(response, userID)

	return response, nil
}

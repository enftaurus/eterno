package models

import (
	"encoding/json"
	"time"
)

// Roles
const (
	RolePlatformAdmin = "PLATFORM_ADMIN"
	RoleCollegeAdmin  = "COLLEGE_ADMIN"
	RoleGroupAdmin    = "GROUP_ADMIN"
	RoleStudent       = "STUDENT"
)

// Challenge Lifecycles
const (
	ChallengeDraft     = "DRAFT"
	ChallengeScheduled = "SCHEDULED"
	ChallengeActive    = "ACTIVE"
	ChallengeCompleted = "COMPLETED"
	ChallengeArchived  = "ARCHIVED"
	ChallengeCancelled = "CANCELLED"
)

// Assignment Statuses
const (
	AssignmentNotStarted        = "NOT_STARTED"
	AssignmentInProgress         = "IN_PROGRESS"
	AssignmentPendingVerification = "PENDING_VERIFICATION"
	AssignmentVerified          = "VERIFIED"
	AssignmentFailedVerification = "FAILED_VERIFICATION"
	AssignmentExpired           = "EXPIRED"
)

// Platform Identifiers
const (
	PlatformLeetCode   = "leetcode"
	PlatformGitHub     = "github"
	PlatformCodeforces = "codeforces"
)

// Verification Statuses
const (
	VerificationUnverified = "UNVERIFIED"
	VerificationPending    = "PENDING"
	VerificationVerified   = "VERIFIED"
	VerificationFailed     = "FAILED"
)

// User represents an identity in the system
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	AvatarURL    *string   `json:"avatar_url"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Institution represents a university or college tenant
type Institution struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Domain    *string         `json:"domain"`
	LogoURL   *string         `json:"logo_url"`
	Timezone  string          `json:"timezone"`
	Settings  json.RawMessage `json:"settings"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Department struct {
	ID            string    `json:"id"`
	InstitutionID string    `json:"institution_id"`
	Name          string    `json:"name"`
	Code          *string   `json:"code"`
	CreatedAt     time.Time `json:"created_at"`
}

type Batch struct {
	ID             string    `json:"id"`
	InstitutionID  string    `json:"institution_id"`
	DepartmentID   *string   `json:"department_id"`
	Name           string    `json:"name"`
	GraduationYear int       `json:"graduation_year"`
	CreatedAt      time.Time `json:"created_at"`
}

type Group struct {
	ID            string    `json:"id"`
	InstitutionID string    `json:"institution_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	GroupType     string    `json:"group_type"`
	CreatedBy     *string   `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type InstitutionMembership struct {
	ID              string    `json:"id"`
	InstitutionID   string    `json:"institution_id"`
	UserID          string    `json:"user_id"`
	DepartmentID    *string   `json:"department_id"`
	BatchID         *string   `json:"batch_id"`
	StudentIDNumber *string   `json:"student_id_number"`
	Role            string    `json:"role"`
	IsActive        bool      `json:"is_active"`
	JoinedAt        time.Time `json:"joined_at"`
}

type PlatformAccount struct {
	ID                    string          `json:"id"`
	UserID                string          `json:"user_id"`
	Platform              string          `json:"platform"`
	Username              string          `json:"username"`
	ExternalUserID        *string         `json:"external_user_id"`
	VerificationStatus    string          `json:"verification_status"`
	VerificationToken     *string         `json:"verification_token"`
	VerificationAttemptAt *time.Time      `json:"verification_attempt_at"`
	VerifiedAt            *time.Time      `json:"verified_at"`
	LastSyncAt            *time.Time      `json:"last_sync_at"`
	LastSyncStatus        string          `json:"last_sync_status"`
	SyncErrorMessage      *string         `json:"sync_error_message"`
	ProfileMetadata       json.RawMessage `json:"profile_metadata"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type Challenge struct {
	ID                string             `json:"id"`
	InstitutionID     string             `json:"institution_id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	Instructions      *string            `json:"instructions"`
	ChallengeType     string             `json:"challenge_type"`
	Status            string             `json:"status"`
	StartTime         time.Time          `json:"start_time"`
	Deadline          time.Time          `json:"deadline"`
	ScopeType         string             `json:"scope_type"`
	ScopeID           *string            `json:"scope_id"`
	MaxPoints         int                `json:"max_points"`
	HistoricalAllowed bool               `json:"historical_allowed"`
	CreatedBy         *string            `json:"created_by"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	CompletedCount    int                `json:"completed_count"`
	AssignedCount     int                `json:"assigned_count"`
	Problems          []ChallengeProblem `json:"problems,omitempty"`
}

type ChallengeProblem struct {
	ID          string          `json:"id"`
	ChallengeID string          `json:"challenge_id"`
	Platform    string          `json:"platform"`
	ProblemURL  string          `json:"problem_url"`
	ProblemSlug string          `json:"problem_slug"`
	Title       string          `json:"title"`
	Difficulty  string          `json:"difficulty"`
	Points      int             `json:"points"`
	IsBonus     bool            `json:"is_bonus"`
	OrderIndex  int             `json:"order_index"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
}

type ChallengeAssignment struct {
	ID            string     `json:"id"`
	ChallengeID   string     `json:"challenge_id"`
	UserID        string     `json:"user_id"`
	InstitutionID string     `json:"institution_id"`
	Status        string     `json:"status"`
	BaselineAt    time.Time  `json:"baseline_at"`
	AssignedAt    time.Time  `json:"assigned_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	PointsAwarded int        `json:"points_awarded"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Submission struct {
	ID                   string          `json:"id"`
	UserID               string          `json:"user_id"`
	Platform             string          `json:"platform"`
	ProviderSubmissionID string          `json:"provider_submission_id"`
	ProblemSlug          string          `json:"problem_slug"`
	Title                *string         `json:"title"`
	Verdict              string          `json:"verdict"`
	IsAccepted           bool            `json:"is_accepted"`
	Language             *string         `json:"language"`
	RuntimeMs            *int            `json:"runtime_ms"`
	MemoryMB             *float64        `json:"memory_mb"`
	SubmittedAt          time.Time       `json:"submitted_at"`
	RawPayload           json.RawMessage `json:"raw_payload"`
	CreatedAt            time.Time       `json:"created_at"`
}

type ChallengeCompletion struct {
	ID                 string          `json:"id"`
	ChallengeID        string          `json:"challenge_id"`
	AssignmentID       string          `json:"assignment_id"`
	ChallengeProblemID string          `json:"challenge_problem_id"`
	UserID             string          `json:"user_id"`
	SubmissionID       string          `json:"submission_id"`
	VerifiedAt         time.Time       `json:"verified_at"`
	VerificationSource string          `json:"verification_source"`
	Evidence           json.RawMessage `json:"evidence"`
	CreatedAt          time.Time       `json:"created_at"`
}

type ScoreEvent struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	InstitutionID string    `json:"institution_id"`
	ChallengeID   *string   `json:"challenge_id"`
	CompletionID  *string   `json:"completion_id"`
	EventType     string    `json:"event_type"`
	Points        int       `json:"points"`
	PolicyVersion string    `json:"policy_version"`
	Description   *string   `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

type ActivityEvent struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	Source     string          `json:"source"`
	ExternalID string          `json:"external_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
}

type SyncJob struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	Platform   string          `json:"platform"`
	JobType    string          `json:"job_type"`
	Status     string          `json:"status"`
	RetryCount int             `json:"retry_count"`
	MaxRetries int             `json:"max_retries"`
	NextRunAt  time.Time       `json:"next_run_at"`
	LockedAt   *time.Time      `json:"locked_at"`
	LockedBy   *string         `json:"locked_by"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	Link      *string   `json:"link"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditLog struct {
	ID            string          `json:"id"`
	InstitutionID *string         `json:"institution_id"`
	ActorID       *string         `json:"actor_id"`
	Action        string          `json:"action"`
	EntityType    string          `json:"entity_type"`
	EntityID      string          `json:"entity_id"`
	Details       json.RawMessage `json:"details"`
	IPAddress     *string         `json:"ip_address"`
	CreatedAt     time.Time       `json:"created_at"`
}

// Request and Response DTOs

type RegisterRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=6"`
	DisplayName     string `json:"display_name" binding:"required"`
	InstitutionSlug string `json:"institution_slug" binding:"required"`
	Role            string `json:"role"`
	DepartmentName  string `json:"department_name"`
	BatchName       string `json:"batch_name"`
	StudentIDNumber string `json:"student_id_number"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token        string       `json:"token"`
	User         User         `json:"user"`
	Institution  *Institution `json:"institution,omitempty"`
	DepartmentID *string      `json:"department_id,omitempty"`
	BatchID      *string      `json:"batch_id,omitempty"`
}

type CreateProblemInput struct {
	ProblemURL string `json:"problem_url" binding:"required"`
	Difficulty string `json:"difficulty"`
	Points     int    `json:"points"`
	IsBonus    bool   `json:"is_bonus"`
}

type CreateChallengeRequest struct {
	Title             string               `json:"title" binding:"required"`
	Description       string               `json:"description" binding:"required"`
	Instructions      string               `json:"instructions"`
	ChallengeType     string               `json:"challenge_type"`
	StartTime         time.Time            `json:"start_time" binding:"required"`
	Deadline          time.Time            `json:"deadline" binding:"required"`
	ScopeType         string               `json:"scope_type"`
	ScopeID           *string              `json:"scope_id"`
	MaxPoints         int                  `json:"max_points"`
	HistoricalAllowed bool                 `json:"historical_allowed"`
	Problems          []CreateProblemInput `json:"problems" binding:"required,min=1"`
}

type LinkAccountRequest struct {
	Username string `json:"username" binding:"required"`
}

type SubmissionDTO struct {
	ProviderID  string    `json:"provider_id"`
	ProblemSlug string    `json:"problem_slug"`
	Title       string    `json:"title"`
	Verdict     string    `json:"verdict"`
	IsAccepted  bool      `json:"is_accepted"`
	Language    string    `json:"language"`
	RuntimeMs   int       `json:"runtime_ms"`
	MemoryMB    float64   `json:"memory_mb"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type LeaderboardEntry struct {
	Rank                 int       `json:"rank"`
	UserID               string    `json:"user_id"`
	DisplayName          string    `json:"display_name"`
	AvatarURL            *string   `json:"avatar_url"`
	DepartmentName       *string   `json:"department_name"`
	BatchName            *string   `json:"batch_name"`
	TotalPoints          int       `json:"total_points"`
	VerifiedCompletions  int       `json:"verified_completions"`
	CurrentStreak        int       `json:"current_streak"`
	ParticipationRate    float64   `json:"participation_rate"`
	LastVerifiedSolveAt *time.Time `json:"last_verified_solve_at"`
}

type StreakAnalytics struct {
	CurrentStreak          int     `json:"current_streak"`
	LongestStreak          int     `json:"longest_streak"`
	ActiveDaysLast7        int     `json:"active_days_last_7"`
	ActiveDaysLast30       int     `json:"active_days_last_30"`
	ActiveDaysLast90       int     `json:"active_days_last_90"`
	WeeklyParticipationRate float64 `json:"weekly_participation_rate"`
	ChallengeCompletionRate float64 `json:"challenge_completion_rate"`
	AvgVerifiedPerDay      float64 `json:"avg_verified_per_day"`
}

type DashboardOverviewMetrics struct {
	TotalRegisteredStudents   int                `json:"total_registered_students"`
	SolvedProblemTodayCount   int                `json:"solved_problem_today_count"`
	CompletedAssignedTodayCount int              `json:"completed_assigned_today_count"`
	InactiveStudentsCount     int                `json:"inactive_students_count"`
	ActiveChallengesCount     int                `json:"active_challenges_count"`
	ApproachingDeadlinesCount int                `json:"approaching_deadlines_count"`
	VerifiedSolvesThisWeek    int                `json:"verified_solves_this_week"`
	VerifiedSolvesThisMonth   int                `json:"verified_solves_this_month"`
	ChallengeCompletionRate   float64            `json:"challenge_completion_rate"`
	AvgProblemsPerStudent     float64            `json:"avg_problems_per_student"`
	RecentSubmissions         []Submission       `json:"recent_submissions"`
	MostConsistentStudents    []LeaderboardEntry `json:"most_consistent_students"`
}

type HeatmapDayData struct {
	Date            string         `json:"date"` // YYYY-MM-DD
	Count           int            `json:"count"`
	Intensity       int            `json:"intensity"` // 0 to 4
	SourceBreakdown map[string]int `json:"source_breakdown"`
}

type HeatmapResponse struct {
	TotalEvents int              `json:"total_events"`
	Days        []HeatmapDayData `json:"days"`
}

type ScoreComponent struct {
	Name         string                 `json:"name"`
	Score        float64                `json:"score"`        // 0 to 100
	Weight       float64                `json:"weight"`       // e.g. 0.25
	Contribution float64                `json:"contribution"` // Score * Weight
	Description  string                 `json:"description"`
	Metrics      map[string]interface{} `json:"metrics"`
}

type SkillRadarItem struct {
	Topic       string  `json:"topic"`
	Score       float64 `json:"score"` // 0 to 100
	SolvedCount int     `json:"solved_count"`
	Target      int     `json:"target"`
}

type RecommendationItem struct {
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionURL   string `json:"action_url,omitempty"`
	Impact      string `json:"impact"` // HIGH, MEDIUM, LOW
}

type DeveloperScoreResponse struct {
	TotalScore      int                  `json:"total_score"` // 0 to 100
	Consistency     ScoreComponent       `json:"consistency"`
	ProblemSolving  ScoreComponent       `json:"problem_solving"`
	Competitive     ScoreComponent       `json:"competitive"`
	Engineering     ScoreComponent       `json:"engineering"`
	Growth          ScoreComponent       `json:"growth"`
	SkillRadar      []SkillRadarItem     `json:"skill_radar"`
	Recommendations []RecommendationItem `json:"recommendations"`
	CalculatedAt    time.Time            `json:"calculated_at"`
}

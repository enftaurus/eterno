export type UserRole = 'PLATFORM_ADMIN' | 'COLLEGE_ADMIN' | 'GROUP_ADMIN' | 'STUDENT';

export interface User {
  id: string;
  email: string;
  display_name: string;
  avatar_url?: string | null;
  role: UserRole;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Institution {
  id: string;
  name: string;
  slug: string;
  domain?: string | null;
  logo_url?: string | null;
  timezone: string;
}

export interface Department {
  id: string;
  institution_id: string;
  name: string;
  code?: string | null;
}

export interface Batch {
  id: string;
  institution_id: string;
  department_id?: string | null;
  name: string;
  graduation_year: number;
}

export interface PlatformAccount {
  id: string;
  user_id: string;
  platform: 'leetcode' | 'github' | 'codeforces';
  username: string;
  verification_status: 'UNVERIFIED' | 'PENDING' | 'VERIFIED' | 'FAILED';
  verification_token?: string | null;
  verified_at?: string | null;
  last_sync_at?: string | null;
  last_sync_status: 'IDLE' | 'SYNCING' | 'SUCCESS' | 'FAILED';
  sync_error_message?: string | null;
}

export type ChallengeStatus = 'DRAFT' | 'SCHEDULED' | 'ACTIVE' | 'COMPLETED' | 'ARCHIVED' | 'CANCELLED';
export type AssignmentStatus = 'NOT_STARTED' | 'IN_PROGRESS' | 'PENDING_VERIFICATION' | 'VERIFIED' | 'FAILED_VERIFICATION' | 'EXPIRED';

export interface ChallengeProblem {
  id: string;
  challenge_id: string;
  platform: string;
  problem_url: string;
  problem_slug: string;
  title: string;
  difficulty: 'Easy' | 'Medium' | 'Hard';
  points: number;
  is_bonus: boolean;
  order_index: number;
  is_verified?: boolean;
  verified_at?: string | null;
}

export interface Challenge {
  id: string;
  institution_id: string;
  title: string;
  description: string;
  instructions?: string | null;
  challenge_type: 'DAILY' | 'WEEKLY' | 'COMPETITION' | 'PRACTICE';
  status: ChallengeStatus;
  start_time: string;
  deadline: string;
  scope_type: 'INSTITUTION' | 'DEPARTMENT' | 'BATCH' | 'GROUP';
  scope_id?: string | null;
  max_points: number;
  historical_allowed: boolean;
  created_by?: string | null;
  created_at: string;
  updated_at: string;
  problems?: ChallengeProblem[];
}

export interface ChallengeAssignment {
  id: string;
  challenge_id: string;
  user_id: string;
  institution_id: string;
  status: AssignmentStatus;
  baseline_at: string;
  assigned_at: string;
  completed_at?: string | null;
  points_awarded: number;
  title?: string;
  description?: string;
  challenge_type?: string;
  challenge_status?: string;
  start_time?: string;
  deadline?: string;
  max_points?: number;
}

export interface Submission {
  id: string;
  user_id: string;
  platform: string;
  provider_submission_id: string;
  problem_slug: string;
  title?: string | null;
  verdict: string;
  is_accepted: boolean;
  language?: string | null;
  runtime_ms?: number | null;
  memory_mb?: number | null;
  submitted_at: string;
}

export interface LeaderboardEntry {
  rank: number;
  user_id: string;
  display_name: string;
  avatar_url?: string | null;
  department_name?: string | null;
  batch_name?: string | null;
  total_points: number;
  verified_completions: number;
  current_streak: number;
  participation_rate: number;
  last_verified_solve_at?: string | null;
}

export interface StreakAnalytics {
  current_streak: number;
  longest_streak: number;
  active_days_last_7: number;
  active_days_last_30: number;
  active_days_last_90: number;
  weekly_participation_rate: number;
  challenge_completion_rate: number;
  avg_verified_per_day: number;
}

export interface DashboardOverviewMetrics {
  total_registered_students: number;
  solved_problem_today_count: number;
  completed_assigned_today_count: number;
  inactive_students_count: number;
  active_challenges_count: number;
  approaching_deadlines_count: number;
  verified_solves_this_week: number;
  verified_solves_this_month: number;
  challenge_completion_rate: number;
  avg_problems_per_student: number;
  recent_submissions: Submission[];
  most_consistent_students: LeaderboardEntry[];
}

export interface HeatmapDayData {
  date: string;
  count: number;
  intensity: number;
  source_breakdown: Record<string, number>;
}

export interface HeatmapResponse {
  total_events: number;
  days: HeatmapDayData[];
}

export interface Notification {
  id: string;
  user_id: string;
  title: string;
  message: string;
  type: string;
  link?: string | null;
  is_read: boolean;
  created_at: string;
}

export interface AuditLog {
  id: string;
  institution_id?: string | null;
  actor_id?: string | null;
  action: string;
  entity_type: string;
  entity_id: string;
  details: Record<string, any>;
  ip_address?: string | null;
  created_at: string;
}

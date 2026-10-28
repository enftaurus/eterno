import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Heatmap } from '../../components/heatmap/Heatmap';
import { api } from '../../lib/api';
import { ChallengeAssignment, StreakAnalytics, HeatmapResponse, Submission, DeveloperScoreResponse } from '../../types';
import { useAuth } from '../../context/AuthContext';
import { Flame, CheckCircle2, Trophy, Clock, ArrowRight, RefreshCw, AlertCircle, Code2, Zap } from 'lucide-react';
import { Link } from 'react-router-dom';

export const StudentOverview: React.FC = () => {
  const { user } = useAuth();
  const [assignments, setAssignments] = useState<ChallengeAssignment[]>([]);
  const [streaks, setStreaks] = useState<StreakAnalytics | null>(null);
  const [heatmap, setHeatmap] = useState<HeatmapResponse | null>(null);
  const [recentSubmissions, setRecentSubmissions] = useState<Submission[]>([]);
  const [devScore, setDevScore] = useState<DeveloperScoreResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSyncing, setIsSyncing] = useState(false);
  const [syncFeedback, setSyncFeedback] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchData = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const [assignRes, streakRes, heatRes, subsRes, scoreRes] = await Promise.all([
        api.get<ChallengeAssignment[]>('/me/challenges'),
        api.get<StreakAnalytics>('/me/streaks'),
        api.get<HeatmapResponse>('/me/heatmap'),
        api.get<Submission[]>('/me/submissions'),
        api.get<DeveloperScoreResponse>('/me/developer-score').catch(() => null),
      ]);
      setAssignments(assignRes || []);
      setStreaks(streakRes);
      setHeatmap(heatRes);
      setRecentSubmissions(subsRes || []);
      setDevScore(scoreRes);
    } catch (err: any) {
      setError(err.message || 'Failed to load student overview data');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleQuickSync = async () => {
    setIsSyncing(true);
    setSyncFeedback(null);
    try {
      const res = await api.post<{ records_ingested: number; records_verified: number }>('/me/platform-accounts/leetcode/sync');
      setSyncFeedback(`Synchronized! ${res.records_ingested} submissions retrieved, ${res.records_verified} challenges verified.`);
      fetchData();
    } catch (err: any) {
      setSyncFeedback(err.message || 'Sync failed. Please ensure your LeetCode account is linked & verified.');
    } finally {
      setIsSyncing(false);
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case 'VERIFIED':
        return 'success';
      case 'IN_PROGRESS':
        return 'warning';
      case 'PENDING_VERIFICATION':
        return 'info';
      default:
        return 'neutral';
    }
  };

  const getScoreTier = (score: number) => {
    if (score >= 90) return { title: 'Master Contender', badge: 'bg-purple-50 text-purple-700 border-purple-200' };
    if (score >= 75) return { title: 'Senior Solver', badge: 'bg-indigo-50 text-indigo-700 border-indigo-200' };
    if (score >= 50) return { title: 'Active Contender', badge: 'bg-emerald-50 text-emerald-700 border-emerald-200' };
    if (score >= 25) return { title: 'Rising Developer', badge: 'bg-blue-50 text-blue-700 border-blue-200' };
    return { title: 'Aspiring Crusader', badge: 'bg-neutral-50 text-neutral-700 border-neutral-200' };
  };

  const activeAssignments = assignments.filter((a) => a.status !== 'VERIFIED');
  const completedAssignments = assignments.filter((a) => a.status === 'VERIFIED');
  const scoreTier = getScoreTier(devScore?.total_score || 0);

  return (
    <div className="space-y-6">
      {/* Top Banner & Quick Sync */}
      <div className="flex flex-wrap items-center justify-between gap-4 p-5 bg-white border border-neutral-200/80 rounded-xl shadow-xs">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">
            Welcome back, {user?.display_name}
          </h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            You have <span className="font-semibold text-neutral-800">{activeAssignments.length} assigned challenge(s)</span> awaiting completion today.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={handleQuickSync}
            isLoading={isSyncing}
            icon={<RefreshCw className="w-3.5 h-3.5" />}
          >
            Sync LeetCode Submissions
          </Button>

          <Link to="/challenges">
            <Button size="sm" icon={<Code2 className="w-3.5 h-3.5" />}>
              View All Challenges
            </Button>
          </Link>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-800 flex items-center justify-between">
          <span>{error}</span>
          <button onClick={fetchData} className="font-semibold underline ml-4">
            Retry
          </button>
        </div>
      )}

      {syncFeedback && (
        <div className="p-3 bg-neutral-50 border border-neutral-200 text-xs text-neutral-700 rounded-lg flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
            <span>{syncFeedback}</span>
          </div>
          <button onClick={() => setSyncFeedback(null)} className="text-neutral-400 hover:text-neutral-600 text-xs font-semibold">✕</button>
        </div>
      )}

      {/* Developer Score Overview Banner */}
      <div className="p-5 bg-gradient-to-r from-neutral-900 via-neutral-800 to-indigo-950 text-white rounded-xl shadow-sm border border-neutral-800 flex flex-wrap items-center justify-between gap-6">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-indigo-500/20 border border-indigo-400/30 flex items-center justify-center text-indigo-400">
            <Zap className="w-6 h-6 text-indigo-400" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold tracking-wide text-neutral-200 uppercase">Developer Score</h2>
              <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${scoreTier.badge}`}>
                {scoreTier.title}
              </span>
            </div>
            <div className="flex items-baseline gap-2 mt-0.5">
              <span className="text-3xl font-extrabold text-white tracking-tight">
                {isLoading ? '—' : devScore?.total_score ?? 0}
              </span>
              <span className="text-xs text-neutral-400 font-medium">/ 100</span>
              <span className="text-xs text-neutral-400 ml-2 hidden sm:inline">
                Cross-platform verified intelligence across LeetCode, GitHub, & Codeforces
              </span>
            </div>
          </div>
        </div>

        {/* 5 Component Bars Preview */}
        {devScore && (
          <div className="hidden md:flex items-center gap-4 text-xs">
            <div className="space-y-1">
              <div className="flex justify-between text-[10px] text-neutral-400">
                <span>Consistency</span>
                <span className="font-semibold text-neutral-200">{Math.round(devScore.consistency.score)}%</span>
              </div>
              <div className="w-20 bg-neutral-700 h-1.5 rounded-full overflow-hidden">
                <div className="bg-amber-400 h-full rounded-full" style={{ width: `${Math.min(100, devScore.consistency.score)}%` }} />
              </div>
            </div>

            <div className="space-y-1">
              <div className="flex justify-between text-[10px] text-neutral-400">
                <span>Problem Solving</span>
                <span className="font-semibold text-neutral-200">{Math.round(devScore.problem_solving.score)}%</span>
              </div>
              <div className="w-20 bg-neutral-700 h-1.5 rounded-full overflow-hidden">
                <div className="bg-indigo-400 h-full rounded-full" style={{ width: `${Math.min(100, devScore.problem_solving.score)}%` }} />
              </div>
            </div>

            <div className="space-y-1">
              <div className="flex justify-between text-[10px] text-neutral-400">
                <span>Competitive</span>
                <span className="font-semibold text-neutral-200">{Math.round(devScore.competitive.score)}%</span>
              </div>
              <div className="w-20 bg-neutral-700 h-1.5 rounded-full overflow-hidden">
                <div className="bg-blue-400 h-full rounded-full" style={{ width: `${Math.min(100, devScore.competitive.score)}%` }} />
              </div>
            </div>

            <div className="space-y-1">
              <div className="flex justify-between text-[10px] text-neutral-400">
                <span>Git Commits</span>
                <span className="font-semibold text-neutral-200">{Math.round(devScore.engineering.score)}%</span>
              </div>
              <div className="w-20 bg-neutral-700 h-1.5 rounded-full overflow-hidden">
                <div className="bg-emerald-400 h-full rounded-full" style={{ width: `${Math.min(100, devScore.engineering.score)}%` }} />
              </div>
            </div>
          </div>
        )}

        <div>
          <Link to="/profile">
            <Button size="sm" variant="outline" className="bg-neutral-800 text-white border-neutral-700 hover:bg-neutral-700 hover:border-neutral-600 text-xs flex items-center gap-1.5">
              <span>Skill Radar & Insights</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </Button>
          </Link>
        </div>
      </div>

      {/* Metrics Row */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <div className="flex items-center justify-between text-neutral-400">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Active Streak</span>
            <Flame className="w-4 h-4 text-amber-500" />
          </div>
          <p className="text-2xl font-bold text-neutral-900 mt-2">
            {isLoading ? '—' : (streaks?.current_streak ?? 0)} <span className="text-xs font-normal text-neutral-500">days</span>
          </p>
          <p className="text-[11px] text-neutral-400 mt-1">
            Longest: {isLoading ? '—' : (streaks?.longest_streak ?? 0)} consecutive days
          </p>
        </Card>

        <Card>
          <div className="flex items-center justify-between text-neutral-400">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">7-Day Participation</span>
            <CheckCircle2 className="w-4 h-4 text-emerald-600" />
          </div>
          <p className="text-2xl font-bold text-neutral-900 mt-2">
            {isLoading ? '—' : (streaks?.active_days_last_7 ?? 0)} / 7 <span className="text-xs font-normal text-neutral-500">days</span>
          </p>
          <p className="text-[11px] text-neutral-400 mt-1">
            {isLoading ? '—' : (streaks?.weekly_participation_rate != null ? `${streaks.weekly_participation_rate.toFixed(0)}%` : '—')} weekly consistency
          </p>
        </Card>

        <Card>
          <div className="flex items-center justify-between text-neutral-400">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Completed Challenges</span>
            <Trophy className="w-4 h-4 text-indigo-600" />
          </div>
          <p className="text-2xl font-bold text-neutral-900 mt-2">
            {isLoading ? '—' : completedAssignments.length} <span className="text-xs font-normal text-neutral-500">verified</span>
          </p>
          <p className="text-[11px] text-neutral-400 mt-1">
            Completion rate: {isLoading ? '—' : (streaks?.challenge_completion_rate != null ? `${streaks.challenge_completion_rate.toFixed(0)}%` : '—')}
          </p>
        </Card>

        <Card>
          <div className="flex items-center justify-between text-neutral-400">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Solves in 30 Days</span>
            <Clock className="w-4 h-4 text-neutral-500" />
          </div>
          <p className="text-2xl font-bold text-neutral-900 mt-2">{streaks?.active_days_last_30 || 0} <span className="text-xs font-normal text-neutral-500">active days</span></p>
          <p className="text-[11px] text-neutral-400 mt-1">Avg: {streaks?.avg_verified_per_day?.toFixed(1) || 0} solves / active day</p>
        </Card>
      </div>

      {/* Main Section: Assigned Challenges & Unified Heatmap */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left: Assigned Challenges */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader
              title="Today's Assigned Challenges"
              subtitle="Challenges assigned by your college awaiting LeetCode solve verification"
              action={
                <Link to="/challenges" className="text-xs font-semibold text-indigo-600 hover:text-indigo-800 flex items-center gap-1">
                  View all <ArrowRight className="w-3.5 h-3.5" />
                </Link>
              }
            />

            <div className="mt-4 divide-y divide-neutral-100">
              {assignments.length === 0 ? (
                <div className="text-center py-10">
                  <p className="text-xs text-neutral-500">No active challenges assigned to your batch today.</p>
                </div>
              ) : (
                assignments.map((a) => (
                  <div key={a.id} className="py-3.5 flex flex-wrap items-center justify-between gap-3">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <Link to={`/challenges/${a.challenge_id}`} className="text-sm font-semibold text-neutral-900 hover:text-indigo-600 transition-colors">
                          {a.title}
                        </Link>
                        <Badge variant={getStatusBadgeVariant(a.status)} size="sm">
                          {a.status.replace(/_/g, ' ')}
                        </Badge>
                      </div>
                      <p className="text-xs text-neutral-500 line-clamp-1">{a.description}</p>
                      <div className="flex items-center gap-4 text-[11px] text-neutral-400">
                        <span>Deadline: {new Date(a.deadline || '').toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</span>
                        <span>•</span>
                        <span className="text-indigo-600 font-medium">+{a.max_points} pts</span>
                      </div>
                    </div>

                    <Link to={`/challenges/${a.challenge_id}`}>
                      <Button variant="outline" size="sm">
                        {a.status === 'VERIFIED' ? 'View Solve' : 'Solve & Verify'}
                      </Button>
                    </Link>
                  </div>
                ))
              )}
            </div>
          </Card>

          {/* Unified Activity Heatmap Card */}
          <Card>
            <CardHeader
              title="Unified Developer Activity Heatmap"
              subtitle="Normalized cross-platform solves from LeetCode, GitHub, Codeforces, and Eterno challenges"
            />
            <div className="mt-4">
              <Heatmap data={heatmap} isLoading={isLoading} />
            </div>
          </Card>
        </div>

        {/* Right: Recent Submissions & Next Steps */}
        <div className="space-y-6">
          <Card>
            <CardHeader
              title="Recent Verified Submissions"
              subtitle="Detected from linked platforms"
              action={
                <Link to="/submissions" className="text-xs font-semibold text-indigo-600 hover:text-indigo-800">
                  History
                </Link>
              }
            />

            <div className="mt-3 divide-y divide-neutral-100">
              {recentSubmissions.length === 0 ? (
                <p className="text-center py-6 text-xs text-neutral-400">No submissions detected yet</p>
              ) : (
                recentSubmissions.slice(0, 5).map((sub) => (
                  <div key={sub.id} className="py-2.5 flex items-center justify-between text-xs">
                    <div>
                      <p className="font-medium text-neutral-800">{sub.problem_slug}</p>
                      <span className="text-[10px] text-neutral-400">{sub.platform} • {new Date(sub.submitted_at).toLocaleDateString()}</span>
                    </div>
                    <Badge variant="success" size="sm">
                      Accepted
                    </Badge>
                  </div>
                ))
              )}
            </div>
          </Card>

          <Card className="bg-indigo-50/50 border-indigo-100">
            <h4 className="text-xs font-semibold uppercase tracking-wider text-indigo-900">LeetCode Linking Guide</h4>
            <p className="text-xs text-indigo-800 mt-1 leading-relaxed">
              To verify problem completions, link your LeetCode username in Connected Accounts and place your unique verification token in your LeetCode profile bio.
            </p>
            <div className="mt-3">
              <Link to="/settings/connected-accounts">
                <Button variant="outline" size="sm" className="bg-white">
                  Manage Account Connection
                </Button>
              </Link>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

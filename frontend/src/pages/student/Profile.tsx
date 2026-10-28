import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { useAuth } from '../../context/AuthContext';
import { api } from '../../lib/api';
import { PlatformAccount, DeveloperScoreResponse } from '../../types';
import {
  Flame,
  Trophy,
  CheckCircle2,
  Building,
  Code2,
  GitCommit,
  TrendingUp,
  Sparkles,
  ArrowRight,
  Shield,
  Layers,
  Activity,
  Zap,
} from 'lucide-react';
import { Link } from 'react-router-dom';

export const StudentProfile: React.FC = () => {
  const { user, institution, refreshUser } = useAuth();
  const [displayName, setDisplayName] = useState(user?.display_name || '');
  const [accounts, setAccounts] = useState<PlatformAccount[]>([]);
  const [devScore, setDevScore] = useState<DeveloperScoreResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [feedback, setFeedback] = useState<string | null>(null);

  useEffect(() => {
    const loadProfileData = async () => {
      setIsLoading(true);
      try {
        const [accRes, scoreRes] = await Promise.all([
          api.get<PlatformAccount[]>('/me/platform-accounts'),
          api.get<DeveloperScoreResponse>('/me/developer-score'),
        ]);
        setAccounts(accRes || []);
        setDevScore(scoreRes);
      } catch (err) {
        console.error('Failed to load profile details', err);
      } finally {
        setIsLoading(false);
      }
    };
    loadProfileData();
  }, []);

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setFeedback(null);
    try {
      await api.patch('/me/profile', { display_name: displayName });
      await refreshUser();
      setFeedback('Profile successfully updated.');
    } catch (err: any) {
      setFeedback(err.message || 'Failed to update profile');
    } finally {
      setIsSaving(false);
    }
  };

  const getScoreTier = (score: number) => {
    if (score >= 90) return { title: 'Master Contender', color: 'text-purple-600', badge: 'bg-purple-50 text-purple-700 border-purple-200' };
    if (score >= 75) return { title: 'Senior Solver', color: 'text-indigo-600', badge: 'bg-indigo-50 text-indigo-700 border-indigo-200' };
    if (score >= 50) return { title: 'Active Contender', color: 'text-emerald-600', badge: 'bg-emerald-50 text-emerald-700 border-emerald-200' };
    if (score >= 25) return { title: 'Rising Developer', color: 'text-blue-600', badge: 'bg-blue-50 text-blue-700 border-blue-200' };
    return { title: 'Aspiring Crusader', color: 'text-neutral-600', badge: 'bg-neutral-50 text-neutral-700 border-neutral-200' };
  };

  const tier = getScoreTier(devScore?.total_score || 0);

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      {/* Header Profile Hero Card */}
      <Card className="border-indigo-100 bg-gradient-to-r from-white via-indigo-50/10 to-white">
        <div className="flex flex-wrap items-center justify-between gap-6">
          <div className="flex items-center gap-4">
            <div className="w-16 h-16 rounded-full bg-indigo-600 text-white flex items-center justify-center text-2xl font-bold shadow-sm">
              {user?.display_name?.charAt(0) || 'U'}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-bold text-neutral-900">{user?.display_name}</h1>
                <span className={`text-[11px] font-semibold px-2 py-0.5 rounded-full border ${tier.badge}`}>
                  {tier.title}
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-0.5">{user?.email}</p>
              <div className="flex items-center gap-3 mt-2 text-xs text-neutral-500">
                <span className="flex items-center gap-1">
                  <Building className="w-3.5 h-3.5 text-neutral-400" />
                  {institution?.name || 'Code Crusaders'}
                </span>
              </div>
            </div>
          </div>

          {/* Large Developer Score Display */}
          <div className="flex items-center gap-4 p-3 bg-white rounded-xl border border-neutral-200/80 shadow-xs">
            <div className="text-right">
              <span className="text-[10px] text-neutral-400 font-bold uppercase tracking-wider block">
                Developer Score
              </span>
              <div className="flex items-baseline justify-end gap-1">
                <span className="text-3xl font-extrabold text-neutral-900">
                  {isLoading ? '—' : devScore?.total_score ?? 0}
                </span>
                <span className="text-xs text-neutral-400 font-semibold">/ 100</span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-full border-4 border-indigo-600 flex items-center justify-center text-indigo-600 font-bold text-xs bg-indigo-50/50">
              <Zap className="w-5 h-5 text-indigo-600" />
            </div>
          </div>
        </div>
      </Card>

      {/* 5 Weighted Component Pillars */}
      {devScore && (
        <div className="space-y-4">
          <div>
            <h2 className="text-sm font-bold text-neutral-900 tracking-tight flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-indigo-600" />
              Developer Score Component Breakdown
            </h2>
            <p className="text-xs text-neutral-500 mt-0.5">
              Deterministic scoring model combining consistency, problem-solving, competitive performance, and repository work.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {/* 1. Consistency */}
            <Card className="flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 bg-amber-50 text-amber-600 rounded-md">
                      <Flame className="w-4 h-4" />
                    </div>
                    <span className="text-xs font-bold text-neutral-900">{devScore.consistency.name}</span>
                  </div>
                  <Badge variant="warning" size="sm">25% Weight</Badge>
                </div>
                <div className="mt-3 flex items-baseline justify-between">
                  <span className="text-2xl font-bold text-neutral-900">{Math.round(devScore.consistency.score)}</span>
                  <span className="text-xs font-semibold text-neutral-500">+{devScore.consistency.contribution} pts</span>
                </div>
                <div className="w-full bg-neutral-100 h-1.5 rounded-full mt-2 overflow-hidden">
                  <div className="bg-amber-500 h-1.5 rounded-full" style={{ width: `${Math.min(100, devScore.consistency.score)}%` }} />
                </div>
                <p className="text-[11px] text-neutral-500 mt-2 leading-relaxed">
                  {devScore.consistency.description}
                </p>
              </div>
              <div className="mt-3 pt-2.5 border-t border-neutral-100 text-[11px] text-neutral-500 flex justify-between">
                <span>Streak: <strong>{devScore.consistency.metrics?.current_streak || 0}d</strong></span>
                <span>Active 30d: <strong>{devScore.consistency.metrics?.active_days_30d || 0}d</strong></span>
              </div>
            </Card>

            {/* 2. Problem Solving */}
            <Card className="flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 bg-indigo-50 text-indigo-600 rounded-md">
                      <Code2 className="w-4 h-4" />
                    </div>
                    <span className="text-xs font-bold text-neutral-900">{devScore.problem_solving.name}</span>
                  </div>
                  <Badge variant="info" size="sm">25% Weight</Badge>
                </div>
                <div className="mt-3 flex items-baseline justify-between">
                  <span className="text-2xl font-bold text-neutral-900">{Math.round(devScore.problem_solving.score)}</span>
                  <span className="text-xs font-semibold text-neutral-500">+{devScore.problem_solving.contribution} pts</span>
                </div>
                <div className="w-full bg-neutral-100 h-1.5 rounded-full mt-2 overflow-hidden">
                  <div className="bg-indigo-600 h-1.5 rounded-full" style={{ width: `${Math.min(100, devScore.problem_solving.score)}%` }} />
                </div>
                <p className="text-[11px] text-neutral-500 mt-2 leading-relaxed">
                  {devScore.problem_solving.description}
                </p>
              </div>
              <div className="mt-3 pt-2.5 border-t border-neutral-100 text-[11px] text-neutral-500 flex justify-between">
                <span>Total Solved: <strong>{devScore.problem_solving.metrics?.total_accepted || 0}</strong></span>
                <span>Med/Hard: <strong>{(devScore.problem_solving.metrics?.medium_count || 0) + (devScore.problem_solving.metrics?.hard_count || 0)}</strong></span>
              </div>
            </Card>

            {/* 3. Competitive Programming */}
            <Card className="flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 bg-blue-50 text-blue-600 rounded-md">
                      <Trophy className="w-4 h-4" />
                    </div>
                    <span className="text-xs font-bold text-neutral-900">{devScore.competitive.name}</span>
                  </div>
                  <Badge variant="neutral" size="sm">20% Weight</Badge>
                </div>
                <div className="mt-3 flex items-baseline justify-between">
                  <span className="text-2xl font-bold text-neutral-900">{Math.round(devScore.competitive.score)}</span>
                  <span className="text-xs font-semibold text-neutral-500">+{devScore.competitive.contribution} pts</span>
                </div>
                <div className="w-full bg-neutral-100 h-1.5 rounded-full mt-2 overflow-hidden">
                  <div className="bg-blue-600 h-1.5 rounded-full" style={{ width: `${Math.min(100, devScore.competitive.score)}%` }} />
                </div>
                <p className="text-[11px] text-neutral-500 mt-2 leading-relaxed">
                  {devScore.competitive.description}
                </p>
              </div>
              <div className="mt-3 pt-2.5 border-t border-neutral-100 text-[11px] text-neutral-500 flex justify-between">
                <span>Codeforces: <strong>{devScore.competitive.metrics?.codeforces_rating || 'Unrated'}</strong></span>
                <span>Max: <strong>{devScore.competitive.metrics?.max_rating || '—'}</strong></span>
              </div>
            </Card>

            {/* 4. Engineering Activity */}
            <Card className="flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 bg-emerald-50 text-emerald-600 rounded-md">
                      <GitCommit className="w-4 h-4" />
                    </div>
                    <span className="text-xs font-bold text-neutral-900">{devScore.engineering.name}</span>
                  </div>
                  <Badge variant="success" size="sm">20% Weight</Badge>
                </div>
                <div className="mt-3 flex items-baseline justify-between">
                  <span className="text-2xl font-bold text-neutral-900">{Math.round(devScore.engineering.score)}</span>
                  <span className="text-xs font-semibold text-neutral-500">+{devScore.engineering.contribution} pts</span>
                </div>
                <div className="w-full bg-neutral-100 h-1.5 rounded-full mt-2 overflow-hidden">
                  <div className="bg-emerald-600 h-1.5 rounded-full" style={{ width: `${Math.min(100, devScore.engineering.score)}%` }} />
                </div>
                <p className="text-[11px] text-neutral-500 mt-2 leading-relaxed">
                  {devScore.engineering.description}
                </p>
              </div>
              <div className="mt-3 pt-2.5 border-t border-neutral-100 text-[11px] text-neutral-500 flex justify-between">
                <span>Commits: <strong>{devScore.engineering.metrics?.commits_count || 0}</strong></span>
                <span>Active Repos: <strong>{devScore.engineering.metrics?.active_repos || 0}</strong></span>
              </div>
            </Card>

            {/* 5. Growth */}
            <Card className="flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 bg-rose-50 text-rose-600 rounded-md">
                      <TrendingUp className="w-4 h-4" />
                    </div>
                    <span className="text-xs font-bold text-neutral-900">{devScore.growth.name}</span>
                  </div>
                  <Badge variant="danger" size="sm">10% Weight</Badge>
                </div>
                <div className="mt-3 flex items-baseline justify-between">
                  <span className="text-2xl font-bold text-neutral-900">{Math.round(devScore.growth.score)}</span>
                  <span className="text-xs font-semibold text-neutral-500">+{devScore.growth.contribution} pts</span>
                </div>
                <div className="w-full bg-neutral-100 h-1.5 rounded-full mt-2 overflow-hidden">
                  <div className="bg-rose-500 h-1.5 rounded-full" style={{ width: `${Math.min(100, devScore.growth.score)}%` }} />
                </div>
                <p className="text-[11px] text-neutral-500 mt-2 leading-relaxed">
                  {devScore.growth.description}
                </p>
              </div>
              <div className="mt-3 pt-2.5 border-t border-neutral-100 text-[11px] text-neutral-500 flex justify-between">
                <span>Recent Solves: <strong>{devScore.growth.metrics?.recent_solves_30d || 0}</strong></span>
                <span>Trajectory: <strong>Positive</strong></span>
              </div>
            </Card>
          </div>
        </div>
      )}

      {/* Skill Radar & Actionable Recommendations */}
      {devScore && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Skill Radar / Topic Mastery */}
          <Card>
            <CardHeader
              title="Skill Radar & Topic Mastery"
              subtitle="Demonstrated strength across core computer science competencies."
            />
            <div className="mt-4 space-y-3.5">
              {devScore.skill_radar.map((item) => (
                <div key={item.topic} className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="font-semibold text-neutral-800">{item.topic}</span>
                    <span className="text-neutral-500 font-mono text-[11px]">
                      {Math.round(item.score)}% ({item.solved_count} / {item.target})
                    </span>
                  </div>
                  <div className="w-full bg-neutral-100 h-2 rounded-full overflow-hidden">
                    <div
                      className="bg-indigo-600 h-2 rounded-full transition-all duration-500"
                      style={{ width: `${Math.min(100, item.score)}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Actionable Recommendations */}
          <Card>
            <CardHeader
              title="Actionable Recommendations"
              subtitle="Personalized steps to raise your Developer Score and expand topic mastery."
            />
            <div className="mt-4 space-y-3">
              {devScore.recommendations.map((rec, idx) => (
                <div key={idx} className="p-3 bg-neutral-50 rounded-lg border border-neutral-200/80 space-y-1.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-neutral-900">{rec.title}</span>
                    <Badge variant={rec.impact === 'HIGH' ? 'danger' : 'info'} size="sm">
                      {rec.impact} IMPACT
                    </Badge>
                  </div>
                  <p className="text-[11px] text-neutral-500 leading-relaxed">{rec.description}</p>
                  {rec.action_url && (
                    <div className="pt-1">
                      <Link
                        to={rec.action_url}
                        className="inline-flex items-center gap-1 text-[11px] font-semibold text-indigo-600 hover:text-indigo-800 transition-colors"
                      >
                        Take Action <ArrowRight className="w-3 h-3" />
                      </Link>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </Card>
        </div>
      )}

      {/* Edit Profile Form */}
      <Card>
        <CardHeader title="Public Profile Information" subtitle="Update your display identity across leaderboards." />
        <form onSubmit={handleUpdateProfile} className="mt-4 space-y-4 max-w-lg">
          {feedback && (
            <div className="p-3 bg-neutral-50 border border-neutral-200 text-xs text-neutral-700 rounded-lg">
              {feedback}
            </div>
          )}

          <Input
            label="Display Name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            required
          />

          <Input
            label="Email Address"
            value={user?.email || ''}
            disabled
            helperText="University email address is managed by your institution."
          />

          <Button type="submit" size="sm" isLoading={isSaving}>
            Save Changes
          </Button>
        </form>
      </Card>

      {/* Connected Platform Status */}
      <Card>
        <CardHeader
          title="Connected Coding Accounts"
          subtitle="Identities linked and verified across competitive programming and version control platforms."
          action={
            <Link to="/settings/connected-accounts">
              <Button variant="outline" size="sm" icon={<ArrowRight className="w-3.5 h-3.5" />}>
                Manage Connections
              </Button>
            </Link>
          }
        />

        <div className="mt-4 divide-y divide-neutral-100">
          {accounts.length === 0 ? (
            <div className="py-6 text-center text-xs text-neutral-400">
              <p>No platforms connected yet.</p>
              <Link to="/settings/connected-accounts" className="text-indigo-600 font-semibold underline mt-1 inline-block">
                Link LeetCode, GitHub, or Codeforces
              </Link>
            </div>
          ) : (
            accounts.map((acc) => (
              <div key={acc.id} className="py-3 flex items-center justify-between text-xs">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-neutral-100 rounded-md">
                    <Code2 className="w-4 h-4 text-neutral-700" />
                  </div>
                  <div>
                    <span className="font-semibold text-neutral-900 capitalize">{acc.platform}</span>
                    <p className="text-[11px] text-neutral-400 font-mono">@{acc.username}</p>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <Badge variant={acc.verification_status === 'VERIFIED' ? 'success' : 'warning'} size="sm" dot>
                    {acc.verification_status}
                  </Badge>
                  {acc.last_sync_at && (
                    <span className="text-[10px] text-neutral-400">
                      Synced {new Date(acc.last_sync_at).toLocaleDateString()}
                    </span>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      </Card>
    </div>
  );
};

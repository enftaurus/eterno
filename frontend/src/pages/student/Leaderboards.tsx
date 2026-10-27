import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { api } from '../../lib/api';
import { LeaderboardEntry } from '../../types';
import { Trophy, Medal, Flame, Calendar, Award } from 'lucide-react';

export const StudentLeaderboards: React.FC = () => {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [timeframe, setTimeframe] = useState<string>('all');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchLeaderboard = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await api.get<LeaderboardEntry[]>(`/leaderboards?timeframe=${timeframe}`);
      setEntries(data || []);
    } catch (err: any) {
      setError(err.message || 'Failed to load leaderboard standings');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchLeaderboard();
  }, [timeframe]);

  const getRankBadge = (rank: number) => {
    switch (rank) {
      case 1:
        return <span className="w-6 h-6 rounded-full bg-amber-100 text-amber-800 font-bold flex items-center justify-center text-xs">🥇 1</span>;
      case 2:
        return <span className="w-6 h-6 rounded-full bg-neutral-200 text-neutral-800 font-bold flex items-center justify-center text-xs">🥈 2</span>;
      case 3:
        return <span className="w-6 h-6 rounded-full bg-amber-50 text-amber-900 border border-amber-200 font-bold flex items-center justify-center text-xs">🥉 3</span>;
      default:
        return <span className="text-xs font-semibold text-neutral-400 w-6 text-center">#{rank}</span>;
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">College Leaderboards</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Ranked deterministically by: 1) Total verified points, 2) Completed challenges, 3) Earliest solve timestamp.
          </p>
        </div>

        {/* Timeframe selector */}
        <div className="flex items-center gap-1.5 p-1 bg-neutral-100 rounded-lg text-xs">
          {[
            { id: 'all', label: 'All-Time' },
            { id: 'monthly', label: 'Last 30 Days' },
            { id: 'weekly', label: 'This Week' },
            { id: 'daily', label: 'Today' },
          ].map((tf) => (
            <button
              key={tf.id}
              onClick={() => setTimeframe(tf.id)}
              className={`px-3 py-1.5 rounded-md font-medium transition-all ${
                timeframe === tf.id ? 'bg-white text-neutral-900 shadow-xs' : 'text-neutral-500'
              }`}
            >
              {tf.label}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-800 flex items-center justify-between">
          <span>{error}</span>
          <button onClick={fetchLeaderboard} className="font-semibold underline ml-4">
            Retry
          </button>
        </div>
      )}

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-neutral-200 text-neutral-400 uppercase tracking-wider text-[10px]">
                <th className="py-3 px-4 font-semibold w-16">Rank</th>
                <th className="py-3 px-4 font-semibold">Student Developer</th>
                <th className="py-3 px-4 font-semibold">Department & Batch</th>
                <th className="py-3 px-4 font-semibold text-right">Verified Solves</th>
                <th className="py-3 px-4 font-semibold text-right">Total Score</th>
                <th className="py-3 px-4 font-semibold text-right">Last Verified Solve</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100 text-neutral-700">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="text-center py-12 text-neutral-400">Loading leaderboard rankings...</td>
                </tr>
              ) : entries.length === 0 ? (
                <tr>
                  <td colSpan={6} className="text-center py-12 text-neutral-400">No student activity recorded for this timeframe yet.</td>
                </tr>
              ) : (
                entries.map((e) => (
                  <tr key={e.user_id} className="hover:bg-neutral-50/50 transition-colors">
                    <td className="py-3 px-4">
                      {getRankBadge(e.rank)}
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-full bg-neutral-200 text-neutral-700 flex items-center justify-center font-semibold text-xs">
                          {e.display_name.charAt(0)}
                        </div>
                        <span className="font-semibold text-neutral-900">{e.display_name}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-neutral-500">
                      {[e.department_name, e.batch_name].filter(Boolean).join(' • ') || '—'}
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-neutral-800">
                      {e.verified_completions} solved
                    </td>
                    <td className="py-3 px-4 text-right font-bold text-indigo-600 text-sm">
                      {e.total_points} pts
                    </td>
                    <td className="py-3 px-4 text-right text-neutral-400">
                      {e.last_verified_solve_at
                        ? new Date(e.last_verified_solve_at).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
                        : '—'}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
};

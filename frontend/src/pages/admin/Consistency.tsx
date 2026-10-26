import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { api } from '../../lib/api';
import { LeaderboardEntry } from '../../types';
import { Flame, CheckCircle2, Trophy, Activity, TrendingUp } from 'lucide-react';

export const AdminConsistency: React.FC = () => {
  const [topConsistent, setTopConsistent] = useState<LeaderboardEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchConsistency = async () => {
      setIsLoading(true);
      try {
        const data = await api.get<LeaderboardEntry[]>('/leaderboards?timeframe=monthly&limit=20');
        setTopConsistent(data || []);
      } catch (err) {
        console.error('Failed to load consistency data', err);
      } finally {
        setIsLoading(false);
      }
    };
    fetchConsistency();
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Consistency & Streak Analytics</h1>
        <p className="text-xs text-neutral-500 mt-0.5">
          Measure steady daily engagement rather than last-minute cramming. Active days require at least one verified Accepted solve.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="bg-neutral-50/50">
          <span className="text-xs font-semibold text-neutral-500 uppercase tracking-wider">Active Day Definition</span>
          <p className="text-xs text-neutral-700 mt-2 leading-relaxed">
            A student's day qualifies as <strong>Active</strong> when at least one unique problem is solved with an Accepted verdict and verified against server evidence.
          </p>
        </Card>

        <Card className="bg-neutral-50/50">
          <span className="text-xs font-semibold text-neutral-500 uppercase tracking-wider">Anti-Gaming Policy</span>
          <p className="text-xs text-neutral-700 mt-2 leading-relaxed">
            Duplicate submissions of previously solved problems do not count as multiple solved days. Provider submission IDs prevent duplicate streaks.
          </p>
        </Card>

        <Card className="bg-neutral-50/50">
          <span className="text-xs font-semibold text-neutral-500 uppercase tracking-wider">Institutional Timezone</span>
          <p className="text-xs text-neutral-700 mt-2 leading-relaxed">
            Day boundaries are bucketed according to your institution's configured timezone (UTC) to guarantee fair streak calculation.
          </p>
        </Card>
      </div>

      <Card>
        <CardHeader
          title="Most Consistent Students (Last 30 Days)"
          subtitle="Students with the highest verified solve consistency and steady participation"
        />

        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-neutral-200 text-neutral-400 uppercase tracking-wider text-[10px]">
                <th className="py-3 px-4 font-semibold">Rank</th>
                <th className="py-3 px-4 font-semibold">Student Developer</th>
                <th className="py-3 px-4 font-semibold">Department & Batch</th>
                <th className="py-3 px-4 font-semibold text-right">Verified Solves</th>
                <th className="py-3 px-4 font-semibold text-right">Points Earned</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100 text-neutral-700">
              {isLoading ? (
                <tr>
                  <td colSpan={5} className="text-center py-10 text-neutral-400">Loading consistency data...</td>
                </tr>
              ) : topConsistent.length === 0 ? (
                <tr>
                  <td colSpan={5} className="text-center py-10 text-neutral-400">No student solve records in the last 30 days.</td>
                </tr>
              ) : (
                topConsistent.map((s, idx) => (
                  <tr key={s.user_id} className="hover:bg-neutral-50/50 transition-colors">
                    <td className="py-3 px-4 font-bold text-neutral-400">#{idx + 1}</td>
                    <td className="py-3 px-4 font-semibold text-neutral-900">{s.display_name}</td>
                    <td className="py-3 px-4 text-neutral-500">{s.department_name || 'General CS'} • {s.batch_name || 'Class of 2026'}</td>
                    <td className="py-3 px-4 text-right font-medium text-neutral-800">{s.verified_completions} problems</td>
                    <td className="py-3 px-4 text-right font-bold text-indigo-600">+{s.total_points} pts</td>
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

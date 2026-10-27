import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { api } from '../../lib/api';
import { ChallengeAssignment } from '../../types';
import { Link } from 'react-router-dom';
import { Code2, Clock, CheckCircle2, AlertCircle, ArrowRight, Users } from 'lucide-react';

export const StudentChallenges: React.FC = () => {
  const [assignments, setAssignments] = useState<ChallengeAssignment[]>([]);
  const [filter, setFilter] = useState<'ALL' | 'ACTIVE' | 'VERIFIED'>('ALL');
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchChallenges = async () => {
      try {
        const data = await api.get<ChallengeAssignment[]>('/me/challenges');
        setAssignments(data || []);
      } catch (err) {
        console.error('Failed to load challenges', err);
      } finally {
        setIsLoading(false);
      }
    };
    fetchChallenges();
  }, []);

  const filtered = assignments.filter((a) => {
    if (filter === 'ACTIVE') return a.status !== 'VERIFIED';
    if (filter === 'VERIFIED') return a.status === 'VERIFIED';
    return true;
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Assigned Coding Challenges</h1>
          <p className="text-xs text-neutral-500 mt-0.5">Solve assigned problems on LeetCode; submissions verify automatically against baseline.</p>
        </div>

        <div className="flex items-center gap-1.5 p-1 bg-neutral-100 rounded-lg text-xs">
          <button
            onClick={() => setFilter('ALL')}
            className={`px-3 py-1.5 rounded-md font-medium transition-all ${filter === 'ALL' ? 'bg-white text-neutral-900 shadow-xs' : 'text-neutral-500'}`}
          >
            All ({assignments.length})
          </button>
          <button
            onClick={() => setFilter('ACTIVE')}
            className={`px-3 py-1.5 rounded-md font-medium transition-all ${filter === 'ACTIVE' ? 'bg-white text-neutral-900 shadow-xs' : 'text-neutral-500'}`}
          >
            Active ({assignments.filter((a) => a.status !== 'VERIFIED').length})
          </button>
          <button
            onClick={() => setFilter('VERIFIED')}
            className={`px-3 py-1.5 rounded-md font-medium transition-all ${filter === 'VERIFIED' ? 'bg-white text-neutral-900 shadow-xs' : 'text-neutral-500'}`}
          >
            Verified ({assignments.filter((a) => a.status === 'VERIFIED').length})
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {isLoading ? (
          <div className="col-span-2 text-center py-12 text-xs text-neutral-400">Loading challenges...</div>
        ) : filtered.length === 0 ? (
          <div className="col-span-2 text-center py-12 bg-white rounded-lg border border-neutral-200">
            <p className="text-xs text-neutral-500">No challenges matching the current filter.</p>
          </div>
        ) : (
          filtered.map((ch) => (
            <Card key={ch.id} hoverable className="flex flex-col justify-between">
              <div>
                <div className="flex items-start justify-between gap-2">
                  <Badge variant={ch.challenge_type === 'DAILY' ? 'info' : 'neutral'} size="sm">
                    {ch.challenge_type}
                  </Badge>
                  <Badge variant={ch.status === 'VERIFIED' ? 'success' : ch.status === 'IN_PROGRESS' ? 'warning' : 'neutral'} size="sm" dot>
                    {ch.status.replace(/_/g, ' ')}
                  </Badge>
                </div>

                <h3 className="text-base font-semibold text-neutral-900 mt-2.5 line-clamp-1">{ch.title}</h3>
                <p className="text-xs text-neutral-500 mt-1 line-clamp-2 leading-relaxed">{ch.description}</p>
              </div>

              <div className="mt-5 pt-4 border-t border-neutral-100 flex items-center justify-between">
                <div className="flex flex-wrap items-center gap-2.5 text-xs text-neutral-500">
                  <span className="flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5 text-neutral-400" />
                    {new Date(ch.deadline || '').toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                  </span>
                  <span>•</span>
                  <span className="font-semibold text-indigo-600">+{ch.max_points} pts</span>
                  <span>•</span>
                  <span className="flex items-center gap-1 font-medium text-emerald-600">
                    <Users className="w-3.5 h-3.5" />
                    {ch.completed_count || 0} solved
                  </span>
                </div>

                <Link to={`/challenges/${ch.challenge_id}`}>
                  <Button variant="outline" size="sm" icon={<ArrowRight className="w-3.5 h-3.5" />}>
                    {ch.status === 'VERIFIED' ? 'Review Solve' : 'Details'}
                  </Button>
                </Link>
              </div>
            </Card>
          ))
        )}
      </div>
    </div>
  );
};

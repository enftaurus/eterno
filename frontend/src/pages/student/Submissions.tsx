import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { api } from '../../lib/api';
import { Submission } from '../../types';
import { RefreshCw, CheckCircle2, Clock, Terminal } from 'lucide-react';

export const StudentSubmissions: React.FC = () => {
  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSyncing, setIsSyncing] = useState(false);

  const fetchSubmissions = async () => {
    setIsLoading(true);
    try {
      const data = await api.get<Submission[]>('/me/submissions');
      setSubmissions(data || []);
    } catch (err) {
      console.error('Failed to load submissions', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchSubmissions();
  }, []);

  const handleSync = async () => {
    setIsSyncing(true);
    try {
      await api.post('/me/platform-accounts/leetcode/sync');
      await fetchSubmissions();
    } catch (e) {
      console.error(e);
    } finally {
      setIsSyncing(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Verified Submission History</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Idempotently recorded from connected developer platforms (LeetCode, GitHub, Codeforces).
          </p>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={handleSync}
          isLoading={isSyncing}
          icon={<RefreshCw className="w-3.5 h-3.5" />}
        >
          Sync Now
        </Button>
      </div>

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-neutral-200 text-neutral-400 uppercase tracking-wider text-[10px]">
                <th className="py-3 px-4 font-semibold">Problem / Slug</th>
                <th className="py-3 px-4 font-semibold">Platform</th>
                <th className="py-3 px-4 font-semibold">Verdict</th>
                <th className="py-3 px-4 font-semibold">Language</th>
                <th className="py-3 px-4 font-semibold">Runtime / Memory</th>
                <th className="py-3 px-4 font-semibold">Submitted At</th>
                <th className="py-3 px-4 font-semibold">Provider ID</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100 text-neutral-700">
              {isLoading ? (
                <tr>
                  <td colSpan={7} className="text-center py-10 text-neutral-400">Loading submissions...</td>
                </tr>
              ) : submissions.length === 0 ? (
                <tr>
                  <td colSpan={7} className="text-center py-10 text-neutral-400">No submissions ingested yet. Click 'Sync Now' or link your account.</td>
                </tr>
              ) : (
                submissions.map((sub) => (
                  <tr key={sub.id} className="hover:bg-neutral-50/50 transition-colors">
                    <td className="py-3 px-4 font-medium text-neutral-900">
                      <div>{sub.title || sub.problem_slug}</div>
                      <div className="text-[10px] text-neutral-400 font-mono">{sub.problem_slug}</div>
                    </td>
                    <td className="py-3 px-4 capitalize">{sub.platform}</td>
                    <td className="py-3 px-4">
                      <Badge variant={sub.is_accepted ? 'success' : 'danger'} size="sm">
                        {sub.verdict}
                      </Badge>
                    </td>
                    <td className="py-3 px-4 font-mono text-[11px] text-neutral-600">
                      {sub.language || 'N/A'}
                    </td>
                    <td className="py-3 px-4 text-neutral-500">
                      {sub.runtime_ms ? `${sub.runtime_ms} ms` : '—'} / {sub.memory_mb ? `${sub.memory_mb} MB` : '—'}
                    </td>
                    <td className="py-3 px-4 text-neutral-500">
                      {new Date(sub.submitted_at).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                    </td>
                    <td className="py-3 px-4 font-mono text-[10px] text-neutral-400">
                      {sub.provider_submission_id}
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

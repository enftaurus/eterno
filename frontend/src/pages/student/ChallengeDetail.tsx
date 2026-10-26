import React, { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { api } from '../../lib/api';
import { Challenge, ChallengeAssignment, ChallengeProblem } from '../../types';
import { ExternalLink, CheckCircle2, Clock, ShieldCheck, RefreshCw, AlertCircle, ArrowLeft } from 'lucide-react';

interface ProblemDetail extends ChallengeProblem {
  is_verified?: boolean;
  verified_at?: string;
  submission_id?: string;
}

export const StudentChallengeDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [challenge, setChallenge] = useState<Challenge | null>(null);
  const [assignment, setAssignment] = useState<ChallengeAssignment | null>(null);
  const [problems, setProblems] = useState<ProblemDetail[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSyncing, setIsSyncing] = useState(false);
  const [syncMessage, setSyncMessage] = useState<string | null>(null);

  const fetchDetail = async () => {
    setIsLoading(true);
    try {
      const res = await api.get<{
        challenge: Challenge;
        assignment: ChallengeAssignment;
        problems: ProblemDetail[];
      }>(`/me/challenges/${id}`);
      setChallenge(res.challenge);
      setAssignment(res.assignment);
      setProblems(res.problems || []);
    } catch (err) {
      console.error('Failed to load challenge detail', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchDetail();
  }, [id]);

  const handleSyncVerify = async () => {
    setIsSyncing(true);
    setSyncMessage(null);
    try {
      const res = await api.post<{ records_ingested: number; records_verified: number }>('/me/platform-accounts/leetcode/sync');
      setSyncMessage(`Sync completed: ${res.records_ingested} submissions found, ${res.records_verified} problems verified.`);
      await fetchDetail();
    } catch (err: any) {
      setSyncMessage(err.message || 'Sync failed. Please check your connected LeetCode account.');
    } finally {
      setIsSyncing(false);
    }
  };

  if (isLoading) {
    return <div className="text-center py-20 text-xs text-neutral-400">Loading challenge specifications...</div>;
  }

  if (!challenge) {
    return (
      <div className="text-center py-20 bg-white rounded-lg border border-neutral-200">
        <p className="text-xs text-neutral-500">Challenge not found.</p>
        <Link to="/challenges" className="mt-2 inline-block text-xs font-semibold text-indigo-600">
          ← Back to challenges
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Back button */}
      <div>
        <Link to="/challenges" className="inline-flex items-center gap-1.5 text-xs font-medium text-neutral-500 hover:text-neutral-900 transition-colors">
          <ArrowLeft className="w-3.5 h-3.5" /> Back to Challenges
        </Link>
      </div>

      {/* Main Challenge Card */}
      <Card>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <Badge variant={challenge.challenge_type === 'DAILY' ? 'info' : 'neutral'} size="sm">
                {challenge.challenge_type}
              </Badge>
              <Badge
                variant={assignment?.status === 'VERIFIED' ? 'success' : assignment?.status === 'IN_PROGRESS' ? 'warning' : 'neutral'}
                size="sm"
                dot
              >
                {assignment?.status?.replace(/_/g, ' ') || 'NOT STARTED'}
              </Badge>
            </div>
            <h1 className="text-xl font-bold text-neutral-900 tracking-tight">{challenge.title}</h1>
            <p className="text-xs text-neutral-500 max-w-3xl leading-relaxed">{challenge.description}</p>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="primary"
              size="sm"
              onClick={handleSyncVerify}
              isLoading={isSyncing}
              icon={<RefreshCw className="w-3.5 h-3.5" />}
            >
              Sync & Verify Submissions
            </Button>
          </div>
        </div>

        {syncMessage && (
          <div className="mt-4 p-3 bg-neutral-50 border border-neutral-200 text-xs text-neutral-700 rounded-lg flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600" />
              <span>{syncMessage}</span>
            </div>
            <button onClick={() => setSyncMessage(null)} className="text-neutral-400 hover:text-neutral-600 font-semibold">✕</button>
          </div>
        )}

        <div className="mt-6 pt-4 border-t border-neutral-100 flex flex-wrap items-center gap-6 text-xs text-neutral-500">
          <div className="flex items-center gap-1.5">
            <Clock className="w-4 h-4 text-neutral-400" />
            <span>Deadline: <strong className="text-neutral-800">{new Date(challenge.deadline).toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</strong></span>
          </div>

          <div className="flex items-center gap-1.5">
            <ShieldCheck className="w-4 h-4 text-indigo-500" />
            <span>Reward: <strong className="text-indigo-600">+{challenge.max_points} points</strong></span>
          </div>

          <div>
            <span>Earned Points: <strong className="text-emerald-700 font-bold">{assignment?.points_awarded || 0} pts</strong></span>
          </div>
        </div>
      </Card>

      {/* Problems List */}
      <Card>
        <CardHeader
          title="Required Problems"
          subtitle="Solve these problems on LeetCode with your linked account. Click 'Solve on LeetCode' to open the challenge problem directly."
        />

        <div className="mt-4 divide-y divide-neutral-100">
          {problems.map((prob, idx) => (
            <div key={prob.id} className="py-4 flex flex-wrap items-center justify-between gap-4">
              <div className="flex items-start gap-3">
                <span className="w-6 h-6 rounded-full bg-neutral-100 text-neutral-700 flex items-center justify-center text-xs font-semibold shrink-0">
                  {idx + 1}
                </span>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold text-neutral-900">{prob.title}</span>
                    <Badge variant={prob.difficulty.toLowerCase() as any} size="sm">
                      {prob.difficulty}
                    </Badge>
                    <span className="text-xs font-semibold text-indigo-600">+{prob.points} pts</span>
                  </div>
                  <p className="text-xs text-neutral-400 mt-0.5 font-mono">slug: {prob.problem_slug}</p>
                </div>
              </div>

              <div className="flex items-center gap-3">
                {prob.is_verified ? (
                  <Badge variant="verified" size="md" dot>
                    Verified Solved
                  </Badge>
                ) : (
                  <Badge variant="neutral" size="md">
                    Not Verified Yet
                  </Badge>
                )}

                <a
                  href={prob.problem_url}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button variant="outline" size="sm" icon={<ExternalLink className="w-3.5 h-3.5" />}>
                    Solve on LeetCode
                  </Button>
                </a>
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* Verification Protocol Explanation */}
      <Card className="bg-neutral-50/50">
        <h4 className="text-xs font-semibold uppercase tracking-wider text-neutral-700">How Eterno Verifies Your Solves</h4>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3 mt-3 text-xs text-neutral-600">
          <div className="p-3 bg-white border border-neutral-200 rounded-lg">
            <span className="font-semibold text-neutral-900 block mb-1">1. Account Control</span>
            Your LeetCode username must be verified via our profile token challenge.
          </div>
          <div className="p-3 bg-white border border-neutral-200 rounded-lg">
            <span className="font-semibold text-neutral-900 block mb-1">2. Canonical Match</span>
            The problem slug on LeetCode must strictly match the assigned problem.
          </div>
          <div className="p-3 bg-white border border-neutral-200 rounded-lg">
            <span className="font-semibold text-neutral-900 block mb-1">3. Time Window</span>
            Solutions must be submitted after challenge assignment baseline and before deadline.
          </div>
          <div className="p-3 bg-white border border-neutral-200 rounded-lg">
            <span className="font-semibold text-neutral-900 block mb-1">4. Accepted Verdict</span>
            Only submissions with 'Accepted' verdict on LeetCode qualify for points and streaks.
          </div>
        </div>
      </Card>
    </div>
  );
};

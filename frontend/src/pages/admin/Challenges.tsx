import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Modal } from '../../components/ui/Modal';
import { api } from '../../lib/api';
import { Challenge } from '../../types';
import { Code2, Plus, Play, XCircle, Clock, ExternalLink, Calendar, CheckCircle2 } from 'lucide-react';

export const AdminChallenges: React.FC = () => {
  const [challenges, setChallenges] = useState<Challenge[]>([]);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  // Form State
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [instructions, setInstructions] = useState('');
  const [challengeType, setChallengeType] = useState('DAILY');
  const [startTime, setStartTime] = useState(new Date().toISOString().slice(0, 16));
  const [deadline, setDeadline] = useState(new Date(Date.now() + 86400000).toISOString().slice(0, 16));
  const [scopeType, setScopeType] = useState('INSTITUTION');
  const [problemURL, setProblemURL] = useState('');
  const [problemDifficulty, setProblemDifficulty] = useState('Medium');
  const [problemPoints, setProblemPoints] = useState(100);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const fetchChallenges = async () => {
    setIsLoading(true);
    try {
      const url = `/challenges${statusFilter ? `?status=${statusFilter}` : ''}`;
      const data = await api.get<Challenge[]>(url);
      setChallenges(data || []);
    } catch (err) {
      console.error('Failed to load challenges', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchChallenges();
  }, [statusFilter]);

  const handleCreateChallenge = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setErrorMsg(null);

    try {
      await api.post('/admin/challenges', {
        title,
        description,
        instructions,
        challenge_type: challengeType,
        start_time: new Date(startTime).toISOString(),
        deadline: new Date(deadline).toISOString(),
        scope_type: scopeType,
        max_points: problemPoints,
        problems: [
          {
            problem_url: problemURL,
            difficulty: problemDifficulty,
            points: problemPoints,
            is_bonus: false,
          },
        ],
      });

      setShowCreateModal(false);
      resetForm();
      await fetchChallenges();
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to create challenge. Please check the problem URL format.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handlePublish = async (id: string) => {
    try {
      await api.post(`/admin/challenges/${id}/publish`);
      await fetchChallenges();
    } catch (err: any) {
      alert(err.message || 'Failed to publish challenge');
    }
  };

  const handleCancel = async (id: string) => {
    if (confirm('Are you sure you want to cancel this challenge?')) {
      try {
        await api.post(`/admin/challenges/${id}/cancel`);
        await fetchChallenges();
      } catch (err: any) {
        alert(err.message || 'Failed to cancel challenge');
      }
    }
  };

  const resetForm = () => {
    setTitle('');
    setDescription('');
    setInstructions('');
    setProblemURL('');
    setProblemDifficulty('Medium');
    setProblemPoints(100);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Challenge Management</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Create, schedule, publish, and audit collegiate coding challenges.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Button size="sm" onClick={() => setShowCreateModal(true)} icon={<Plus className="w-3.5 h-3.5" />}>
            Create Challenge
          </Button>
        </div>
      </div>

      {/* Filter Tabs */}
      <div className="flex items-center gap-1.5 p-1 bg-neutral-100 rounded-lg text-xs w-fit">
        {[
          { id: '', label: 'All Lifecycles' },
          { id: 'ACTIVE', label: 'Active' },
          { id: 'DRAFT', label: 'Drafts' },
          { id: 'COMPLETED', label: 'Completed' },
          { id: 'CANCELLED', label: 'Cancelled' },
        ].map((tab) => (
          <button
            key={tab.id}
            onClick={() => setStatusFilter(tab.id)}
            className={`px-3 py-1.5 rounded-md font-medium transition-all ${
              statusFilter === tab.id ? 'bg-white text-neutral-900 shadow-xs' : 'text-neutral-500'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Challenges List Table */}
      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-neutral-200 text-neutral-400 uppercase tracking-wider text-[10px]">
                <th className="py-3 px-4 font-semibold">Title & Description</th>
                <th className="py-3 px-4 font-semibold">Type</th>
                <th className="py-3 px-4 font-semibold">Scope</th>
                <th className="py-3 px-4 font-semibold">Status</th>
                <th className="py-3 px-4 font-semibold">Deadline</th>
                <th className="py-3 px-4 font-semibold">Completions</th>
                <th className="py-3 px-4 font-semibold text-right">Points</th>
                <th className="py-3 px-4 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100 text-neutral-700">
              {isLoading ? (
                <tr>
                  <td colSpan={8} className="text-center py-12 text-neutral-400">Loading challenges...</td>
                </tr>
              ) : challenges.length === 0 ? (
                <tr>
                  <td colSpan={8} className="text-center py-12 text-neutral-400">No challenges found matching this filter.</td>
                </tr>
              ) : (
                challenges.map((ch) => (
                  <tr key={ch.id} className="hover:bg-neutral-50/50 transition-colors">
                    <td className="py-3 px-4">
                      <span className="font-semibold text-neutral-900 block">{ch.title}</span>
                      <span className="text-[11px] text-neutral-400 line-clamp-1">{ch.description}</span>
                    </td>
                    <td className="py-3 px-4">
                      <Badge variant="neutral" size="sm">
                        {ch.challenge_type}
                      </Badge>
                    </td>
                    <td className="py-3 px-4 capitalize text-neutral-500">
                      {ch.scope_type.toLowerCase()}
                    </td>
                    <td className="py-3 px-4">
                      <Badge
                        variant={ch.status === 'ACTIVE' ? 'success' : ch.status === 'DRAFT' ? 'warning' : 'neutral'}
                        size="sm"
                        dot
                      >
                        {ch.status}
                      </Badge>
                    </td>
                    <td className="py-3 px-4 text-neutral-500">
                      {new Date(ch.deadline).toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-1.5">
                        <span className="font-semibold text-emerald-600">{ch.completed_count || 0}</span>
                        <span className="text-neutral-400 text-[11px]">/ {ch.assigned_count || 0} students</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-right font-semibold text-indigo-600">
                      +{ch.max_points} pts
                    </td>
                    <td className="py-3 px-4 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        {ch.status === 'DRAFT' && (
                          <Button size="sm" variant="outline" onClick={() => handlePublish(ch.id)} icon={<Play className="w-3 h-3 text-emerald-600" />}>
                            Publish
                          </Button>
                        )}
                        {ch.status === 'ACTIVE' && (
                          <Button size="sm" variant="ghost" onClick={() => handleCancel(ch.id)} icon={<XCircle className="w-3 h-3 text-rose-500" />}>
                            Cancel
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Modal: Create Challenge */}
      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title="Create Collegiate Coding Challenge"
        description="Add a LeetCode problem URL. Eterno automatically extracts the canonical problem slug and verifies submissions server-side."
        maxWidth="lg"
      >
        <form onSubmit={handleCreateChallenge} className="space-y-4">
          {errorMsg && (
            <div className="p-3 bg-rose-50 border border-rose-200 text-rose-800 text-xs rounded-md">
              {errorMsg}
            </div>
          )}

          <Input
            label="Challenge Title"
            placeholder="e.g. Daily DP Sprint: Two Sum & Coins"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
          />

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">Challenge Type</label>
              <select
                value={challengeType}
                onChange={(e) => setChallengeType(e.target.value)}
                className="w-full rounded-md border border-neutral-300 text-sm px-3 py-2 bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-600"
              >
                <option value="DAILY">Daily Challenge</option>
                <option value="WEEKLY">Weekly Competition</option>
                <option value="COMPETITION">Time-Bounded Contest</option>
                <option value="PRACTICE">Practice Problem</option>
              </select>
            </div>

            <div>
              <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">Scope Target</label>
              <select
                value={scopeType}
                onChange={(e) => setScopeType(e.target.value)}
                className="w-full rounded-md border border-neutral-300 text-sm px-3 py-2 bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-600"
              >
                <option value="INSTITUTION">Entire College / Institution</option>
                <option value="DEPARTMENT">Department Specific</option>
                <option value="BATCH">Batch Specific</option>
                <option value="GROUP">Coding Club / Team</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">Description & Instructions</label>
            <textarea
              rows={2}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Explain problem objectives and learning outcomes..."
              className="w-full rounded-md border border-neutral-300 text-sm p-3 bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-600"
              required
            />
          </div>

          <div className="p-4 bg-neutral-50/70 border border-neutral-200/80 rounded-lg space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wider text-neutral-900">LeetCode Problem Specification</h4>

            <Input
              label="LeetCode Problem URL or Canonical Slug"
              placeholder="https://leetcode.com/problems/two-sum/ or two-sum"
              value={problemURL}
              onChange={(e) => setProblemURL(e.target.value)}
              required
              helperText="Eterno will extract the problem slug and verify submissions against LeetCode's public GraphQL API."
            />

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">Difficulty</label>
                <select
                  value={problemDifficulty}
                  onChange={(e) => setProblemDifficulty(e.target.value)}
                  className="w-full rounded-md border border-neutral-300 text-sm px-3 py-2 bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value="Easy">Easy</option>
                  <option value="Medium">Medium</option>
                  <option value="Hard">Hard</option>
                </select>
              </div>

              <Input
                label="Points Reward"
                type="number"
                value={problemPoints}
                onChange={(e) => setProblemPoints(Number(e.target.value))}
                min={10}
                max={1000}
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Input
              label="Start Time"
              type="datetime-local"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              required
            />
            <Input
              label="Submission Deadline"
              type="datetime-local"
              value={deadline}
              onChange={(e) => setDeadline(e.target.value)}
              required
            />
          </div>

          <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-neutral-100">
            <Button variant="ghost" size="sm" type="button" onClick={() => setShowCreateModal(false)}>
              Cancel
            </Button>
            <Button size="sm" type="submit" isLoading={isSubmitting}>
              Create & Save Challenge
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Modal } from '../../components/ui/Modal';
import { api } from '../../lib/api';
import { Building, Plus, Users, Trophy, Award } from 'lucide-react';

interface GroupItem {
  id: string;
  name: string;
  description?: string;
  group_type: string;
  member_count: number;
  total_group_points: number;
  points_per_student: number;
  created_at: string;
}

export const AdminGroups: React.FC = () => {
  const [groups, setGroups] = useState<GroupItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [groupType, setGroupType] = useState('CODING_CLUB');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const fetchGroups = async () => {
    setIsLoading(true);
    try {
      const data = await api.get<GroupItem[]>('/institutions/current/groups');
      setGroups(data || []);
    } catch (err) {
      console.error('Failed to load groups', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchGroups();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await api.post('/admin/groups', {
        name,
        description,
        group_type: groupType,
      });
      setShowCreateModal(false);
      setName('');
      setDescription('');
      await fetchGroups();
    } catch (err: any) {
      alert(err.message || 'Failed to create group');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Peer Groups & Coding Clubs</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Organize contest prep teams and track group performance normalized per student.
          </p>
        </div>

        <Button size="sm" onClick={() => setShowCreateModal(true)} icon={<Plus className="w-3.5 h-3.5" />}>
          Create Group
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-3 text-center py-12 text-xs text-neutral-400">Loading peer groups...</div>
        ) : groups.length === 0 ? (
          <div className="col-span-3 text-center py-12 bg-white rounded-lg border border-neutral-200">
            <p className="text-xs text-neutral-500">No peer groups created yet.</p>
          </div>
        ) : (
          groups.map((g) => (
            <Card key={g.id} className="flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between">
                  <Badge variant="info" size="sm">
                    {g.group_type.replace('_', ' ')}
                  </Badge>
                  <span className="text-xs font-semibold text-neutral-400">
                    {g.member_count} members
                  </span>
                </div>

                <h3 className="text-base font-bold text-neutral-900 mt-3">{g.name}</h3>
                <p className="text-xs text-neutral-500 mt-1 line-clamp-2 leading-relaxed">{g.description || 'No description provided.'}</p>
              </div>

              <div className="mt-5 pt-4 border-t border-neutral-100 flex items-center justify-between">
                <div>
                  <span className="text-[10px] text-neutral-400 uppercase font-semibold block">Total Points</span>
                  <span className="text-sm font-bold text-indigo-600">{g.total_group_points} pts</span>
                </div>

                <div className="text-right">
                  <span className="text-[10px] text-neutral-400 uppercase font-semibold block">Avg / Student</span>
                  <span className="text-sm font-bold text-emerald-700">{g.points_per_student.toFixed(1)} pts</span>
                </div>
              </div>
            </Card>
          ))
        )}
      </div>

      {/* Modal: Create Group */}
      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title="Create Peer Group or Coding Club"
        description="Organize students into contest preparation batches, leagues, or clubs."
      >
        <form onSubmit={handleCreate} className="space-y-4">
          <Input
            label="Group Name"
            placeholder="e.g. ACM Contest Prep Team A"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />

          <div>
            <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">Group Type</label>
            <select
              value={groupType}
              onChange={(e) => setGroupType(e.target.value)}
              className="w-full rounded-md border border-neutral-300 text-sm px-3 py-2 bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              <option value="CODING_CLUB">Coding Club</option>
              <option value="CONTEST_PREP">Contest Preparation</option>
              <option value="INTERVIEW_LEAGUE">Interview Prep League</option>
              <option value="BATCH_SUBGROUP">Batch Subgroup</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">Description</label>
            <textarea
              rows={2}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Objectives, weekly cadence, and target contests..."
              className="w-full rounded-md border border-neutral-300 text-sm p-3 bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-neutral-100">
            <Button variant="ghost" size="sm" type="button" onClick={() => setShowCreateModal(false)}>
              Cancel
            </Button>
            <Button size="sm" type="submit" isLoading={isSubmitting}>
              Create Group
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

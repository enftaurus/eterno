import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Modal } from '../../components/ui/Modal';
import { api } from '../../lib/api';
import { StudentGroup } from '../../types';
import { Users, Plus, Trophy, Award, CheckCircle2, Shield, Flame } from 'lucide-react';

export const StudentGroups: React.FC = () => {
  const [allGroups, setAllGroups] = useState<StudentGroup[]>([]);
  const [myGroups, setMyGroups] = useState<StudentGroup[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [joiningId, setJoiningId] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);

  const fetchGroups = async () => {
    setIsLoading(true);
    try {
      const [allRes, myRes] = await Promise.all([
        api.get<StudentGroup[]>('/institutions/current/groups'),
        api.get<StudentGroup[]>('/me/groups'),
      ]);
      setAllGroups(allRes || []);
      setMyGroups(myRes || []);
    } catch (err: any) {
      console.error('Failed to load student groups', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchGroups();
  }, []);

  const handleJoin = async (groupId: string) => {
    setJoiningId(groupId);
    setFeedback(null);
    try {
      await api.post(`/me/groups/${groupId}/join`);
      setFeedback('Joined squad successfully!');
      await fetchGroups();
    } catch (err: any) {
      setFeedback(err.message || 'Failed to join squad');
    } finally {
      setJoiningId(null);
    }
  };

  const myGroupIds = new Set(myGroups.map((g) => g.id));

  return (
    <div className="space-y-6 max-w-6xl mx-auto">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Code Crusaders Squads & Peer Groups</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Team up with batchmates and peer coding clubs. Solve challenges together and climb squad rankings.
          </p>
        </div>
      </div>

      {feedback && (
        <div className="p-3 bg-emerald-50 border border-emerald-200 text-xs text-emerald-800 rounded-lg flex items-center justify-between">
          <span>{feedback}</span>
          <button onClick={() => setFeedback(null)} className="font-semibold text-xs ml-4">✕</button>
        </div>
      )}

      {/* My Active Squads */}
      {myGroups.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-sm font-bold text-neutral-900 flex items-center gap-2">
            <Users className="w-4 h-4 text-indigo-600" />
            My Active Squads
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {myGroups.map((g) => (
              <Card key={g.id} className="border-indigo-100 bg-indigo-50/20">
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="text-sm font-bold text-neutral-900">{g.name}</h3>
                    <p className="text-xs text-neutral-500 mt-1 line-clamp-2">{g.description || 'Student coding peer group'}</p>
                  </div>
                  <Badge variant="info" size="sm">Member</Badge>
                </div>
                <div className="mt-4 pt-3 border-t border-indigo-100/60 flex items-center justify-between text-xs text-neutral-600">
                  <span className="flex items-center gap-1">
                    <Users className="w-3.5 h-3.5 text-neutral-400" /> {g.member_count} member(s)
                  </span>
                  <span className="flex items-center gap-1 font-semibold text-indigo-700">
                    <Flame className="w-3.5 h-3.5 text-amber-500" /> {g.my_group_points || 0} pts
                  </span>
                </div>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* All Available Squads */}
      <div className="space-y-3">
        <h2 className="text-sm font-bold text-neutral-900 flex items-center gap-2">
          <Trophy className="w-4 h-4 text-amber-500" />
          Campus Squad Rankings & Directory
        </h2>

        {isLoading ? (
          <div className="p-8 text-center text-xs text-neutral-400">Loading squads...</div>
        ) : allGroups.length === 0 ? (
          <div className="p-8 text-center text-xs text-neutral-400 border border-dashed border-neutral-200 rounded-xl">
            No squads active yet in your college.
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {allGroups.map((g) => {
              const isJoined = myGroupIds.has(g.id);
              return (
                <Card key={g.id} className="flex flex-col justify-between">
                  <div>
                    <div className="flex items-start justify-between gap-2">
                      <h3 className="text-sm font-bold text-neutral-900">{g.name}</h3>
                      <Badge variant="neutral" size="sm">{g.group_type.replace('_', ' ')}</Badge>
                    </div>
                    <p className="text-xs text-neutral-500 mt-1.5 leading-relaxed">
                      {g.description || 'Active peer coding squad on Code Crusaders.'}
                    </p>
                  </div>

                  <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between">
                    <div className="text-xs text-neutral-500">
                      <span className="font-semibold text-neutral-900">{g.member_count}</span> members •{' '}
                      <span className="font-semibold text-neutral-900">{g.total_group_points || 0}</span> pts
                    </div>

                    {isJoined ? (
                      <span className="text-xs font-semibold text-emerald-600 flex items-center gap-1">
                        <CheckCircle2 className="w-3.5 h-3.5" /> Joined
                      </span>
                    ) : (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleJoin(g.id)}
                        isLoading={joiningId === g.id}
                      >
                        Join Squad
                      </Button>
                    )}
                  </div>
                </Card>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

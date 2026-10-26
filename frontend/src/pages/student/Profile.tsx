import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { useAuth } from '../../context/AuthContext';
import { api } from '../../lib/api';
import { PlatformAccount, StreakAnalytics } from '../../types';
import { User, Shield, Flame, Trophy, CheckCircle2, Award, Building, Code2 } from 'lucide-react';

export const StudentProfile: React.FC = () => {
  const { user, institution, refreshUser } = useAuth();
  const [displayName, setDisplayName] = useState(user?.display_name || '');
  const [accounts, setAccounts] = useState<PlatformAccount[]>([]);
  const [streaks, setStreaks] = useState<StreakAnalytics | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [feedback, setFeedback] = useState<string | null>(null);

  useEffect(() => {
    const loadProfileData = async () => {
      try {
        const [accRes, streakRes] = await Promise.all([
          api.get<PlatformAccount[]>('/me/platform-accounts'),
          api.get<StreakAnalytics>('/me/streaks'),
        ]);
        setAccounts(accRes || []);
        setStreaks(streakRes);
      } catch (err) {
        console.error('Failed to load profile details', err);
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

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      {/* Header Profile Card */}
      <Card>
        <div className="flex flex-wrap items-center justify-between gap-6">
          <div className="flex items-center gap-4">
            <div className="w-16 h-16 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center text-2xl font-bold border-2 border-indigo-200">
              {user?.display_name?.charAt(0) || 'U'}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-bold text-neutral-900">{user?.display_name}</h1>
                <Badge variant="info" size="sm">
                  {user?.role}
                </Badge>
              </div>
              <p className="text-xs text-neutral-500 mt-0.5">{user?.email}</p>
              <div className="flex items-center gap-3 mt-2 text-xs text-neutral-500">
                <span className="flex items-center gap-1">
                  <Building className="w-3.5 h-3.5 text-neutral-400" />
                  {institution?.name || 'Stanford Engineering'}
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div className="text-right">
              <span className="text-[11px] text-neutral-400 font-semibold uppercase tracking-wider block">Consistency</span>
              <span className="text-lg font-bold text-neutral-900 flex items-center gap-1 justify-end">
                <Flame className="w-4 h-4 text-amber-500" /> {streaks?.current_streak || 0} days
              </span>
            </div>
          </div>
        </div>
      </Card>

      {/* Edit Profile Form */}
      <Card>
        <CardHeader title="Public Profile Information" subtitle="Update your developer handle and identity." />
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
            helperText="University email address cannot be changed."
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
          subtitle="Identities linked and verified on external competitive platforms."
        />

        <div className="mt-4 divide-y divide-neutral-100">
          {accounts.length === 0 ? (
            <p className="py-6 text-center text-xs text-neutral-400">No accounts connected yet</p>
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

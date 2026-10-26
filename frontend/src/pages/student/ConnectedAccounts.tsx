import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Modal } from '../../components/ui/Modal';
import { api } from '../../lib/api';
import { PlatformAccount } from '../../types';
import { Code2, CheckCircle2, Copy, ExternalLink, RefreshCw, AlertCircle, Link2 } from 'lucide-react';

export const StudentConnectedAccounts: React.FC = () => {
  const [accounts, setAccounts] = useState<PlatformAccount[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showLinkModal, setShowLinkModal] = useState(false);
  const [usernameInput, setUsernameInput] = useState('');
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isVerifying, setIsVerifying] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [feedback, setFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [copied, setCopied] = useState(false);

  const fetchAccounts = async () => {
    setIsLoading(true);
    try {
      const data = await api.get<PlatformAccount[]>('/me/platform-accounts');
      setAccounts(data || []);
      const lc = data?.find((a) => a.platform === 'leetcode');
      if (lc && lc.verification_token) {
        setGeneratedToken(lc.verification_token);
      }
    } catch (err) {
      console.error('Failed to load platform accounts', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, []);

  const handleLinkLeetCode = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setFeedback(null);
    try {
      const res = await api.post<{ verification_token: string; instructions: string }>('/me/platform-accounts/leetcode', {
        username: usernameInput,
      });
      setGeneratedToken(res.verification_token);
      setFeedback({ type: 'success', message: 'LeetCode handle registered. Please place the verification token in your LeetCode profile bio.' });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Failed to link account' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleVerifyAccount = async () => {
    setIsVerifying(true);
    setFeedback(null);
    try {
      const res = await api.post<{ status: string; message: string }>('/me/platform-accounts/leetcode/verify');
      setFeedback({ type: 'success', message: res.message || 'LeetCode account verified!' });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Verification token was not found in profile bio.' });
    } finally {
      setIsVerifying(false);
    }
  };

  const handleSyncAccount = async () => {
    setIsSyncing(true);
    setFeedback(null);
    try {
      const res = await api.post<{ records_ingested: number; records_verified: number }>('/me/platform-accounts/leetcode/sync');
      setFeedback({ type: 'success', message: `Sync complete! Ingested ${res.records_ingested} submissions and verified ${res.records_verified} problems.` });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Synchronization failed' });
    } finally {
      setIsSyncing(false);
    }
  };

  const copyToClipboard = () => {
    if (generatedToken) {
      navigator.clipboard.writeText(generatedToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const lcAccount = accounts.find((a) => a.platform === 'leetcode');

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Connected Accounts & Identity Verification</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Prove control of your LeetCode profile to automatically verify college challenge solutions.
          </p>
        </div>

        <Button size="sm" onClick={() => setShowLinkModal(true)} icon={<Link2 className="w-3.5 h-3.5" />}>
          Connect LeetCode Account
        </Button>
      </div>

      {feedback && (
        <div
          className={`p-3 text-xs rounded-lg border flex items-center justify-between ${
            feedback.type === 'success' ? 'bg-emerald-50 border-emerald-200 text-emerald-800' : 'bg-rose-50 border-rose-200 text-rose-800'
          }`}
        >
          <span>{feedback.message}</span>
          <button onClick={() => setFeedback(null)} className="font-semibold text-xs ml-4">✕</button>
        </div>
      )}

      {/* LeetCode Account Card */}
      <Card>
        <CardHeader
          title="LeetCode Integration"
          subtitle="Primary verification engine for algorithmic coding challenges"
          action={
            lcAccount ? (
              <Badge variant={lcAccount.verification_status === 'VERIFIED' ? 'success' : 'warning'} size="md" dot>
                {lcAccount.verification_status}
              </Badge>
            ) : (
              <Badge variant="neutral" size="md">NOT LINKED</Badge>
            )
          }
        />

        <div className="mt-4 space-y-4">
          {lcAccount ? (
            <div className="p-4 bg-neutral-50/70 border border-neutral-200/80 rounded-lg space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-amber-50 text-amber-600 rounded-md border border-amber-200">
                    <Code2 className="w-5 h-5" />
                  </div>
                  <div>
                    <span className="text-sm font-bold text-neutral-900">@{lcAccount.username}</span>
                    <p className="text-[11px] text-neutral-400">
                      {lcAccount.verified_at ? `Verified on ${new Date(lcAccount.verified_at).toLocaleDateString()}` : 'Awaiting profile token verification'}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {lcAccount.verification_status !== 'VERIFIED' ? (
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={handleVerifyAccount}
                      isLoading={isVerifying}
                    >
                      Verify Token Now
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleSyncAccount}
                      isLoading={isSyncing}
                      icon={<RefreshCw className="w-3.5 h-3.5" />}
                    >
                      Sync Submissions
                    </Button>
                  )}
                </div>
              </div>

              {lcAccount.verification_status !== 'VERIFIED' && lcAccount.verification_token && (
                <div className="mt-3 p-3 bg-white border border-neutral-200 rounded-md text-xs space-y-2">
                  <p className="font-semibold text-neutral-800">Your Verification Challenge Token:</p>
                  <div className="flex items-center gap-2">
                    <code className="p-2 bg-neutral-100 rounded text-neutral-900 font-mono text-xs select-all flex-1">
                      {lcAccount.verification_token}
                    </code>
                    <Button variant="outline" size="sm" onClick={copyToClipboard} icon={<Copy className="w-3.5 h-3.5" />}>
                      {copied ? 'Copied!' : 'Copy'}
                    </Button>
                  </div>
                  <p className="text-[11px] text-neutral-500 leading-relaxed">
                    1. Copy the token above.<br />
                    2. Go to your <a href="https://leetcode.com/profile/" target="_blank" rel="noreferrer" className="text-indigo-600 underline">LeetCode Profile Settings</a> and paste it into the <strong>About Me</strong> field.<br />
                    3. Save your LeetCode profile and click <strong>Verify Token Now</strong> above.
                  </p>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center py-8 border border-dashed border-neutral-200 rounded-lg">
              <Code2 className="w-8 h-8 text-neutral-300 mx-auto mb-2" />
              <p className="text-xs text-neutral-600 font-medium">No LeetCode account linked yet</p>
              <p className="text-[11px] text-neutral-400 mt-0.5">Link your account to verify your college assignments automatically.</p>
              <Button size="sm" variant="outline" className="mt-3" onClick={() => setShowLinkModal(true)}>
                Connect Account
              </Button>
            </div>
          )}
        </div>
      </Card>

      {/* Modal: Link LeetCode */}
      <Modal
        isOpen={showLinkModal}
        onClose={() => setShowLinkModal(false)}
        title="Connect LeetCode Account"
        description="Enter your public LeetCode handle. We will generate a cryptographic verification challenge token."
      >
        <form onSubmit={handleLinkLeetCode} className="space-y-4">
          <Input
            label="LeetCode Username"
            placeholder="e.g. tour_de_code"
            value={usernameInput}
            onChange={(e) => setUsernameInput(e.target.value)}
            required
            helperText="Make sure your LeetCode profile is set to public so our crawler can read submissions."
          />

          <div className="flex items-center justify-end gap-2 pt-3">
            <Button variant="ghost" size="sm" type="button" onClick={() => setShowLinkModal(false)}>
              Cancel
            </Button>
            <Button size="sm" type="submit" isLoading={isSubmitting}>
              Generate Verification Token
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

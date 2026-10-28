import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Modal } from '../../components/ui/Modal';
import { api } from '../../lib/api';
import { PlatformAccount } from '../../types';
import { Code2, CheckCircle2, Copy, RefreshCw, GitCommit, Trophy, Link2, ExternalLink } from 'lucide-react';

export const StudentConnectedAccounts: React.FC = () => {
  const [accounts, setAccounts] = useState<PlatformAccount[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // LeetCode State
  const [showLeetCodeModal, setShowLeetCodeModal] = useState(false);
  const [lcUsername, setLcUsername] = useState('');
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);
  const [isVerifyingLC, setIsVerifyingLC] = useState(false);
  const [isSyncingLC, setIsSyncingLC] = useState(false);

  // GitHub State
  const [showGitHubModal, setShowGitHubModal] = useState(false);
  const [ghUsername, setGhUsername] = useState('');
  const [isConnectingGH, setIsConnectingGH] = useState(false);
  const [isSyncingGH, setIsSyncingGH] = useState(false);

  // Codeforces State
  const [showCFModal, setShowCFModal] = useState(false);
  const [cfHandle, setCfHandle] = useState('');
  const [isConnectingCF, setIsConnectingCF] = useState(false);
  const [isSyncingCF, setIsSyncingCF] = useState(false);

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

    // Check if redirected from GitHub OAuth callback
    const params = new URLSearchParams(window.location.search);
    if (params.get('status') === 'github_connected') {
      setFeedback({ type: 'success', message: 'GitHub account connected via OAuth successfully!' });
      window.history.replaceState({}, document.title, window.location.pathname);
    } else if (params.get('error')) {
      setFeedback({ type: 'error', message: 'GitHub OAuth error: ' + decodeURIComponent(params.get('error')!) });
      window.history.replaceState({}, document.title, window.location.pathname);
    } else if (params.get('code')) {
      const code = params.get('code')!;
      const state = params.get('state') || '';
      window.history.replaceState({}, document.title, window.location.pathname);
      setIsConnectingGH(true);
      api.post<{ message: string; username: string }>('/platforms/github/exchange', { code, state })
        .then((res) => {
          setFeedback({ type: 'success', message: `GitHub account @${res.username} connected successfully!` });
          fetchAccounts();
        })
        .catch((err: any) => {
          setFeedback({ type: 'error', message: err.message || 'Failed to exchange GitHub authorization code' });
        })
        .finally(() => {
          setIsConnectingGH(false);
        });
    }
  }, []);

  // LeetCode Handlers
  const handleLinkLeetCode = async (e: React.FormEvent) => {
    e.preventDefault();
    setFeedback(null);
    try {
      const res = await api.post<{ verification_token: string; instructions: string }>('/me/platform-accounts/leetcode', {
        username: lcUsername,
      });
      setGeneratedToken(res.verification_token);
      setShowLeetCodeModal(false);
      setFeedback({ type: 'success', message: 'LeetCode handle registered. Please paste verification token into your profile bio.' });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Failed to link LeetCode handle' });
    }
  };

  const handleVerifyLeetCode = async () => {
    setIsVerifyingLC(true);
    setFeedback(null);
    try {
      const res = await api.post<{ status: string; message: string }>('/me/platform-accounts/leetcode/verify');
      setFeedback({ type: 'success', message: res.message || 'LeetCode account verified!' });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Verification token was not found in profile bio.' });
    } finally {
      setIsVerifyingLC(false);
    }
  };

  const handleSyncLeetCode = async () => {
    setIsSyncingLC(true);
    setFeedback(null);
    try {
      const res = await api.post<{ records_ingested: number; records_verified: number }>('/me/platform-accounts/leetcode/sync');
      setFeedback({ type: 'success', message: `Sync complete! Ingested ${res.records_ingested} submissions and verified ${res.records_verified} problems.` });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Synchronization failed' });
    } finally {
      setIsSyncingLC(false);
    }
  };

  // GitHub Handlers
  const handleGitHubOAuth = async () => {
    try {
      const res = await api.get<{ auth_url: string }>('/platforms/github/authorize');
      if (res?.auth_url) {
        window.location.href = res.auth_url;
      }
    } catch (err: any) {
      setFeedback({ type: 'error', message: 'Failed initiating GitHub OAuth: ' + err.message });
    }
  };

  const handleLinkGitHubManual = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsConnectingGH(true);
    setFeedback(null);
    try {
      await api.post('/me/platform-accounts/github', { username: ghUsername });
      setShowGitHubModal(false);
      setFeedback({ type: 'success', message: 'GitHub account linked and public events synchronized!' });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Failed connecting GitHub account' });
    } finally {
      setIsConnectingGH(false);
    }
  };

  const handleSyncGitHub = async () => {
    setIsSyncingGH(true);
    setFeedback(null);
    try {
      const res = await api.post<{ records_ingested: number }>('/me/platform-accounts/github/sync');
      setFeedback({ type: 'success', message: `GitHub synchronization complete! ${res.records_ingested} commits and events ingested.` });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'GitHub sync failed' });
    } finally {
      setIsSyncingGH(false);
    }
  };

  // Codeforces Handlers
  const handleLinkCodeforces = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsConnectingCF(true);
    setFeedback(null);
    try {
      await api.post('/me/platform-accounts/codeforces', { username: cfHandle });
      setShowCFModal(false);
      setFeedback({ type: 'success', message: 'Codeforces handle verified and linked! Rating and submissions synced.' });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Failed verifying Codeforces handle' });
    } finally {
      setIsConnectingCF(false);
    }
  };

  const handleSyncCodeforces = async () => {
    setIsSyncingCF(true);
    setFeedback(null);
    try {
      const res = await api.post<{ records_ingested: number }>('/me/platform-accounts/codeforces/sync');
      setFeedback({ type: 'success', message: `Codeforces synchronization complete! ${res.records_ingested} accepted solves retrieved.` });
      await fetchAccounts();
    } catch (err: any) {
      setFeedback({ type: 'error', message: err.message || 'Codeforces sync failed' });
    } finally {
      setIsSyncingCF(false);
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
  const ghAccount = accounts.find((a) => a.platform === 'github');
  const cfAccount = accounts.find((a) => a.platform === 'codeforces');

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div>
        <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Connected Accounts & Identity Verification</h1>
        <p className="text-xs text-neutral-500 mt-0.5">
          Connect your coding platforms to power your unified 0–100 Developer Score and verify college challenge solutions.
        </p>
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

      {/* 1. LeetCode Account Card */}
      <Card>
        <CardHeader
          title="LeetCode Integration"
          subtitle="Primary verification engine for algorithmic college coding challenges"
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
            <div className="p-4 bg-amber-50/20 border border-amber-200/60 rounded-lg space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-amber-100 text-amber-700 rounded-md border border-amber-200">
                    <Code2 className="w-5 h-5" />
                  </div>
                  <div>
                    <span className="text-sm font-bold text-neutral-900">@{lcAccount.username}</span>
                    <p className="text-[11px] text-neutral-400">
                      {lcAccount.verified_at ? `Verified on ${new Date(lcAccount.verified_at).toLocaleDateString()}` : 'Awaiting profile bio token verification'}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {lcAccount.verification_status !== 'VERIFIED' ? (
                    <Button variant="primary" size="sm" onClick={handleVerifyLeetCode} isLoading={isVerifyingLC}>
                      Verify Token Now
                    </Button>
                  ) : (
                    <Button variant="outline" size="sm" onClick={handleSyncLeetCode} isLoading={isSyncingLC} icon={<RefreshCw className="w-3.5 h-3.5" />}>
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
                    3. Save your profile and click <strong>Verify Token Now</strong> above.
                  </p>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center py-6 border border-dashed border-neutral-200 rounded-lg">
              <Code2 className="w-8 h-8 text-neutral-300 mx-auto mb-2" />
              <p className="text-xs text-neutral-600 font-medium">No LeetCode account linked yet</p>
              <Button size="sm" variant="outline" className="mt-3" onClick={() => setShowLeetCodeModal(true)}>
                Connect LeetCode
              </Button>
            </div>
          )}
        </div>
      </Card>

      {/* 2. GitHub Account Card */}
      <Card>
        <CardHeader
          title="GitHub Integration"
          subtitle="Repository commit tracking, pull requests, and engineering activity"
          action={
            ghAccount ? (
              <Badge variant="success" size="md" dot>VERIFIED</Badge>
            ) : (
              <Badge variant="neutral" size="md">NOT LINKED</Badge>
            )
          }
        />

        <div className="mt-4 space-y-4">
          {ghAccount ? (
            <div className="p-4 bg-emerald-50/20 border border-emerald-200/60 rounded-lg flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-emerald-100 text-emerald-700 rounded-md border border-emerald-200">
                  <GitCommit className="w-5 h-5" />
                </div>
                <div>
                  <span className="text-sm font-bold text-neutral-900">@{ghAccount.username}</span>
                  <p className="text-[11px] text-neutral-400">
                    Linked and tracking engineering commits for Developer Score
                  </p>
                </div>
              </div>

              <Button variant="outline" size="sm" onClick={handleSyncGitHub} isLoading={isSyncingGH} icon={<RefreshCw className="w-3.5 h-3.5" />}>
                Sync Commits
              </Button>
            </div>
          ) : (
            <div className="text-center py-6 border border-dashed border-neutral-200 rounded-lg">
              <GitCommit className="w-8 h-8 text-neutral-300 mx-auto mb-2" />
              <p className="text-xs text-neutral-600 font-medium">Connect GitHub to verify your engineering activity</p>
              <div className="flex items-center justify-center gap-2 mt-3">
                <Button size="sm" variant="primary" onClick={handleGitHubOAuth} icon={<ExternalLink className="w-3.5 h-3.5" />}>
                  Connect with GitHub OAuth
                </Button>
                <Button size="sm" variant="outline" onClick={() => setShowGitHubModal(true)}>
                  Enter Username
                </Button>
              </div>
            </div>
          )}
        </div>
      </Card>

      {/* 3. Codeforces Account Card */}
      <Card>
        <CardHeader
          title="Codeforces Integration"
          subtitle="Contest rating history, performance trends, and competitive problem solves"
          action={
            cfAccount ? (
              <Badge variant="success" size="md" dot>VERIFIED</Badge>
            ) : (
              <Badge variant="neutral" size="md">NOT LINKED</Badge>
            )
          }
        />

        <div className="mt-4 space-y-4">
          {cfAccount ? (
            <div className="p-4 bg-blue-50/20 border border-blue-200/60 rounded-lg flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-100 text-blue-700 rounded-md border border-blue-200">
                  <Trophy className="w-5 h-5" />
                </div>
                <div>
                  <span className="text-sm font-bold text-neutral-900">@{cfAccount.username}</span>
                  <p className="text-[11px] text-neutral-400">
                    Rating and competitive contest history synchronized
                  </p>
                </div>
              </div>

              <Button variant="outline" size="sm" onClick={handleSyncCodeforces} isLoading={isSyncingCF} icon={<RefreshCw className="w-3.5 h-3.5" />}>
                Sync Rating & Solves
              </Button>
            </div>
          ) : (
            <div className="text-center py-6 border border-dashed border-neutral-200 rounded-lg">
              <Trophy className="w-8 h-8 text-neutral-300 mx-auto mb-2" />
              <p className="text-xs text-neutral-600 font-medium">Link Codeforces to verify competitive programming rating</p>
              <Button size="sm" variant="outline" className="mt-3" onClick={() => setShowCFModal(true)}>
                Connect Codeforces Handle
              </Button>
            </div>
          )}
        </div>
      </Card>

      {/* Modal: Link LeetCode */}
      <Modal
        isOpen={showLeetCodeModal}
        onClose={() => setShowLeetCodeModal(false)}
        title="Connect LeetCode Account"
        description="Enter your public LeetCode handle. We will generate a verification challenge token."
      >
        <form onSubmit={handleLinkLeetCode} className="space-y-4">
          <Input
            label="LeetCode Username"
            placeholder="e.g. tour_de_code"
            value={lcUsername}
            onChange={(e) => setLcUsername(e.target.value)}
            required
          />
          <div className="flex items-center justify-end gap-2 pt-3">
            <Button variant="ghost" size="sm" type="button" onClick={() => setShowLeetCodeModal(false)}>Cancel</Button>
            <Button size="sm" type="submit">Generate Token</Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Link GitHub */}
      <Modal
        isOpen={showGitHubModal}
        onClose={() => setShowGitHubModal(false)}
        title="Connect GitHub Account"
        description="Enter your public GitHub username to crawl public commits and pull requests."
      >
        <form onSubmit={handleLinkGitHubManual} className="space-y-4">
          <Input
            label="GitHub Username"
            placeholder="e.g. torvalds"
            value={ghUsername}
            onChange={(e) => setGhUsername(e.target.value)}
            required
          />
          <div className="flex items-center justify-end gap-2 pt-3">
            <Button variant="ghost" size="sm" type="button" onClick={() => setShowGitHubModal(false)}>Cancel</Button>
            <Button size="sm" type="submit" isLoading={isConnectingGH}>Link GitHub</Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Link Codeforces */}
      <Modal
        isOpen={showCFModal}
        onClose={() => setShowCFModal(false)}
        title="Connect Codeforces Account"
        description="Enter your Codeforces handle to verify contest rating and problem-solving history."
      >
        <form onSubmit={handleLinkCodeforces} className="space-y-4">
          <Input
            label="Codeforces Handle"
            placeholder="e.g. tourist"
            value={cfHandle}
            onChange={(e) => setCfHandle(e.target.value)}
            required
          />
          <div className="flex items-center justify-end gap-2 pt-3">
            <Button variant="ghost" size="sm" type="button" onClick={() => setShowCFModal(false)}>Cancel</Button>
            <Button size="sm" type="submit" isLoading={isConnectingCF}>Verify & Link</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

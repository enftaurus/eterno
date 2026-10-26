import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { api } from '../../lib/api';
import { DashboardOverviewMetrics } from '../../types';
import { useAuth } from '../../context/AuthContext';
import {
  Users,
  CheckCircle2,
  Code2,
  Clock,
  AlertTriangle,
  Trophy,
  Activity,
  ArrowUpRight,
  TrendingUp,
} from 'lucide-react';
import { Link } from 'react-router-dom';

export const AdminDashboard: React.FC = () => {
  const { institution } = useAuth();
  const [metrics, setMetrics] = useState<DashboardOverviewMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchDashboard = async () => {
    setIsLoading(true);
    try {
      const data = await api.get<DashboardOverviewMetrics>('/admin/dashboard');
      setMetrics(data);
    } catch (err) {
      console.error('Failed to load admin dashboard', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboard();
  }, []);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Institutional Performance Overview</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Real-time analytics for {institution?.name || 'Stanford School of Engineering'}. All metrics reflect actual server verification records.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Link to="/admin/challenges">
            <Button size="sm" icon={<Code2 className="w-3.5 h-3.5" />}>
              Create New Challenge
            </Button>
          </Link>
        </div>
      </div>

      {/* Row 1: Primary KPIs */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <div className="flex items-center justify-between text-neutral-400">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Registered Students</span>
            <Users className="w-4 h-4 text-indigo-600" />
          </div>
          <p className="text-2xl font-bold text-neutral-900 mt-2">{metrics?.total_registered_students || 0}</p>
          <p className="text-[11px] text-neutral-400 mt-1">Across all departments & batches</p>
        </Card>

        <Card>
          <div className="flex items-center justify-between text-neutral-400">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Solved Today</span>
            <CheckCircle2 className="w-4 h-4 text-emerald-600" />
          </div>
          <p className="text-2xl font-bold text-neutral-900 mt-2">
            {metrics?.solved_problem_today_count || 0}
            <span className="text-xs font-normal text-neutral-400 ml-1.5">students</span>
          </p>
          <p className="text-[11px] text-emerald-600 font-medium mt-1">Completed assigned today: {metrics?.completed_assigned_today_count || 0}</p>
        </Card>

        <Card>
          <div className="flex items-center justify-between text-neutral-400">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Active Challenges</span>
            <Code2 className="w-4 h-4 text-amber-500" />
          </div>
          <p className="text-2xl font-bold text-neutral-900 mt-2">{metrics?.active_challenges_count || 0}</p>
          <p className="text-[11px] text-amber-600 font-medium mt-1">
            {metrics?.approaching_deadlines_count || 0} approaching deadline (&lt;24h)
          </p>
        </Card>

        <Card>
          <div className="flex items-center justify-between text-neutral-400">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Inactive Students</span>
            <AlertTriangle className="w-4 h-4 text-rose-500" />
          </div>
          <p className="text-2xl font-bold text-neutral-900 mt-2">{metrics?.inactive_students_count || 0}</p>
          <p className="text-[11px] text-neutral-400 mt-1">No verified solve in last 14 days</p>
        </Card>
      </div>

      {/* Row 2: Completion & Volume Trends */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card className="bg-neutral-50/50">
          <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Challenge Completion Rate</span>
          <p className="text-2xl font-bold text-neutral-900 mt-2">
            {metrics?.challenge_completion_rate?.toFixed(1) || '0.0'}%
          </p>
          <p className="text-[11px] text-neutral-400 mt-1">% of all assigned challenges verified complete</p>
        </Card>

        <Card className="bg-neutral-50/50">
          <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Verified Solves (7 Days)</span>
          <p className="text-2xl font-bold text-neutral-900 mt-2">{metrics?.verified_solves_this_week || 0}</p>
          <p className="text-[11px] text-neutral-400 mt-1">Solves this month: {metrics?.verified_solves_this_month || 0}</p>
        </Card>

        <Card className="bg-neutral-50/50">
          <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Avg Solves / Student</span>
          <p className="text-2xl font-bold text-neutral-900 mt-2">
            {metrics?.avg_problems_per_student?.toFixed(1) || '0.0'}
          </p>
          <p className="text-[11px] text-neutral-400 mt-1">Average verified problems per registered student</p>
        </Card>
      </div>

      {/* Row 3: Recent Activity Table & Quick Actions */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card className="lg:col-span-2">
          <CardHeader
            title="Recent Submissions & Verifications"
            subtitle="Live-polled submissions from student accounts"
            action={
              <Link to="/admin/students" className="text-xs font-semibold text-indigo-600 hover:text-indigo-800">
                Student Directory →
              </Link>
            }
          />

          <div className="mt-3 overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-neutral-100 text-neutral-400 uppercase text-[10px] tracking-wider">
                  <th className="py-2.5 px-3 font-semibold">Problem Slug</th>
                  <th className="py-2.5 px-3 font-semibold">Verdict</th>
                  <th className="py-2.5 px-3 font-semibold">Language</th>
                  <th className="py-2.5 px-3 font-semibold">Submitted At</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 text-neutral-700">
                {!metrics?.recent_submissions || metrics.recent_submissions.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="py-8 text-center text-xs text-neutral-400">
                      No recent submissions recorded yet.
                    </td>
                  </tr>
                ) : (
                  metrics.recent_submissions.map((sub) => (
                    <tr key={sub.id} className="hover:bg-neutral-50/50">
                      <td className="py-2.5 px-3 font-medium text-neutral-900">{sub.problem_slug}</td>
                      <td className="py-2.5 px-3">
                        <Badge variant={sub.is_accepted ? 'success' : 'danger'} size="sm">
                          {sub.verdict}
                        </Badge>
                      </td>
                      <td className="py-2.5 px-3 font-mono text-[11px] text-neutral-500">{sub.language || 'golang'}</td>
                      <td className="py-2.5 px-3 text-neutral-400">
                        {new Date(sub.submitted_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Quick Management Links */}
        <div className="space-y-4">
          <Card>
            <CardHeader title="Administrative Shortcuts" subtitle="Fast workflows for challenge coordinators" />
            <div className="mt-3 space-y-2">
              <Link
                to="/admin/challenges"
                className="block p-3 rounded-md border border-neutral-200/80 hover:bg-neutral-50 transition-colors"
              >
                <span className="text-xs font-semibold text-neutral-900 block">Publish Daily Challenge</span>
                <span className="text-[11px] text-neutral-400">Add LeetCode problem URLs and schedule for batches</span>
              </Link>

              <Link
                to="/admin/students"
                className="block p-3 rounded-md border border-neutral-200/80 hover:bg-neutral-50 transition-colors"
              >
                <span className="text-xs font-semibold text-neutral-900 block">Import Student Cohort CSV</span>
                <span className="text-[11px] text-neutral-400">Bulk enroll students and map their LeetCode handles</span>
              </Link>

              <Link
                to="/admin/consistency"
                className="block p-3 rounded-md border border-neutral-200/80 hover:bg-neutral-50 transition-colors"
              >
                <span className="text-xs font-semibold text-neutral-900 block">Consistency & Streak Reports</span>
                <span className="text-[11px] text-neutral-400">Review consecutive active days across cohorts</span>
              </Link>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

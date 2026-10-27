import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Heatmap } from '../../components/heatmap/Heatmap';
import { api } from '../../lib/api';
import { HeatmapResponse } from '../../types';
import { Layers, Calendar, GitCommit, Code2, Trophy, ArrowUpRight } from 'lucide-react';

export const StudentActivityHeatmap: React.FC = () => {
  const [heatmap, setHeatmap] = useState<HeatmapResponse | null>(null);
  const [selectedSource, setSelectedSource] = useState<string>('');
  const [year, setYear] = useState<number>(2026);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchHeatmap = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const url = `/me/heatmap?year=${year}${selectedSource ? `&source=${selectedSource}` : ''}`;
      const data = await api.get<HeatmapResponse>(url);
      setHeatmap(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load activity heatmap');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchHeatmap();
  }, [year, selectedSource]);

  const leetcodeEventsCount = (heatmap?.days || []).reduce((acc, d) => acc + (d.source_breakdown?.leetcode || 0), 0);
  const activeDaysCount = (heatmap?.days || []).filter((d) => d.count > 0).length;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Unified Developer Activity Heatmap</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Normalized coding activity across LeetCode, GitHub, Codeforces, and verified Eterno college challenges.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <select
            value={year}
            onChange={(e) => setYear(Number(e.target.value))}
            className="px-3 py-1.5 bg-white border border-neutral-300 rounded-md text-xs font-medium text-neutral-700 shadow-xs focus:outline-none focus:ring-2 focus:ring-indigo-500"
          >
            <option value={2026}>2026</option>
            <option value={2025}>2025</option>
          </select>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-800 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span>{error}</span>
          </div>
          <Button size="sm" variant="outline" onClick={fetchHeatmap}>
            Retry
          </Button>
        </div>
      )}

      {/* Cross-Platform Analytics Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="flex items-center gap-4">
          <div className="p-3 bg-amber-50 rounded-lg text-amber-600 border border-amber-200">
            <Code2 className="w-5 h-5" />
          </div>
          <div>
            <span className="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">LeetCode Solves</span>
            <p className="text-xl font-bold text-neutral-900">
              {isLoading ? '—' : leetcodeEventsCount} <span className="text-xs font-normal text-neutral-400">verified</span>
            </p>
            <span className="text-[10px] text-emerald-600 font-medium">Server verified</span>
          </div>
        </Card>

        <Card className="flex items-center gap-4">
          <div className="p-3 bg-neutral-100 rounded-lg text-neutral-800 border border-neutral-200">
            <GitCommit className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <span className="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">GitHub Commits</span>
              <Badge variant="neutral" size="sm">Upcoming</Badge>
            </div>
            <p className="text-xs font-medium text-neutral-500 mt-1">Integration in beta</p>
            <span className="text-[10px] text-neutral-400">Webhook crawler in v2</span>
          </div>
        </Card>

        <Card className="flex items-center gap-4">
          <div className="p-3 bg-indigo-50 rounded-lg text-indigo-600 border border-indigo-200">
            <Trophy className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <span className="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">Codeforces</span>
              <Badge variant="neutral" size="sm">Upcoming</Badge>
            </div>
            <p className="text-xs font-medium text-neutral-500 mt-1">Rating sync in beta</p>
            <span className="text-[10px] text-neutral-400">Contest tracker in v2</span>
          </div>
        </Card>

        <Card className="flex items-center gap-4">
          <div className="p-3 bg-emerald-50 rounded-lg text-emerald-600 border border-emerald-200">
            <Layers className="w-5 h-5" />
          </div>
          <div>
            <span className="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">Active Days & Events</span>
            <p className="text-xl font-bold text-neutral-900">
              {isLoading ? '—' : activeDaysCount} <span className="text-xs font-normal text-neutral-400">days ({heatmap?.total_events ?? 0} events)</span>
            </p>
            <span className="text-[10px] text-neutral-400 font-medium">persisted in PostgreSQL</span>
          </div>
        </Card>
      </div>

      {/* Main Heatmap Container */}
      <Card>
        <CardHeader
          title={`Annual Contribution Calendar (${year})`}
          subtitle="Click on any square to reveal the source breakdown and exact event records."
        />

        <div className="mt-4">
          <Heatmap
            data={heatmap}
            isLoading={isLoading}
            selectedSource={selectedSource}
            onSourceChange={setSelectedSource}
          />
        </div>
      </Card>
    </div>
  );
};

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

  const fetchHeatmap = async () => {
    setIsLoading(true);
    try {
      const url = `/me/heatmap?year=${year}${selectedSource ? `&source=${selectedSource}` : ''}`;
      const data = await api.get<HeatmapResponse>(url);
      setHeatmap(data);
    } catch (err) {
      console.error('Failed to load activity heatmap', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchHeatmap();
  }, [year, selectedSource]);

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

      {/* Cross-Platform Analytics Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="flex items-center gap-4">
          <div className="p-3 bg-amber-50 rounded-lg text-amber-600 border border-amber-200">
            <Code2 className="w-5 h-5" />
          </div>
          <div>
            <span className="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">LeetCode Solves</span>
            <p className="text-xl font-bold text-neutral-900">142 <span className="text-xs font-normal text-neutral-400">problems</span></p>
            <span className="text-[10px] text-emerald-600 font-medium">Rank ~45,200</span>
          </div>
        </Card>

        <Card className="flex items-center gap-4">
          <div className="p-3 bg-neutral-100 rounded-lg text-neutral-800 border border-neutral-200">
            <GitCommit className="w-5 h-5" />
          </div>
          <div>
            <span className="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">GitHub Commits</span>
            <p className="text-xl font-bold text-neutral-900">412 <span className="text-xs font-normal text-neutral-400">commits</span></p>
            <span className="text-[10px] text-neutral-500 font-medium">28 public repos</span>
          </div>
        </Card>

        <Card className="flex items-center gap-4">
          <div className="p-3 bg-indigo-50 rounded-lg text-indigo-600 border border-indigo-200">
            <Trophy className="w-5 h-5" />
          </div>
          <div>
            <span className="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">Codeforces Rating</span>
            <p className="text-xl font-bold text-neutral-900">1,480</p>
            <span className="text-[10px] text-indigo-600 font-medium">Specialist</span>
          </div>
        </Card>

        <Card className="flex items-center gap-4">
          <div className="p-3 bg-emerald-50 rounded-lg text-emerald-600 border border-emerald-200">
            <Layers className="w-5 h-5" />
          </div>
          <div>
            <span className="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">Total Recorded Events</span>
            <p className="text-xl font-bold text-neutral-900">{heatmap?.total_events || 0}</p>
            <span className="text-[10px] text-neutral-400 font-medium">across all connected sources</span>
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

import React, { useState } from 'react';
import { HeatmapResponse, HeatmapDayData } from '../../types';
import { Badge } from '../ui/Badge';
import { Layers, Calendar, CheckCircle2, GitCommit, Trophy, Flame, Code2 } from 'lucide-react';

interface HeatmapProps {
  data?: HeatmapResponse | null;
  isLoading?: boolean;
  selectedSource?: string;
  onSourceChange?: (source: string) => void;
}

export const Heatmap: React.FC<HeatmapProps> = ({
  data,
  isLoading = false,
  selectedSource = '',
  onSourceChange,
}) => {
  const [selectedDay, setSelectedDay] = useState<HeatmapDayData | null>(null);

  // Generate 52 weeks calendar (364 days ending today)
  const today = new Date();
  const days: { dateStr: string; dateObj: Date; dayData?: HeatmapDayData }[] = [];
  const dayMap = new Map<string, HeatmapDayData>();

  if (data?.days) {
    data.days.forEach((d) => dayMap.set(d.date, d));
  }

  // 52 weeks * 7 days = 364 days
  for (let i = 363; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(today.getDate() - i);
    const dateStr = d.toISOString().split('T')[0];
    days.push({
      dateStr,
      dateObj: d,
      dayData: dayMap.get(dateStr),
    });
  }

  // Group into weeks of 7 days
  const weeks: (typeof days)[] = [];
  let currentWeek: typeof days = [];

  const firstDayOfWeek = days[0].dateObj.getDay();
  for (let i = 0; i < firstDayOfWeek; i++) {
    currentWeek.push({ dateStr: '', dateObj: new Date(0) });
  }

  days.forEach((day) => {
    currentWeek.push(day);
    if (currentWeek.length === 7) {
      weeks.push(currentWeek);
      currentWeek = [];
    }
  });

  if (currentWeek.length > 0) {
    weeks.push(currentWeek);
  }

  // Multi-platform color determination
  const getCellColor = (dayData?: HeatmapDayData) => {
    if (!dayData || dayData.count === 0) {
      return 'bg-neutral-100 border-neutral-200/60 hover:border-neutral-300';
    }

    const { intensity, source_breakdown } = dayData;
    const source = selectedSource;

    // Platform-specific filter colors
    if (source === 'leetcode') {
      switch (intensity) {
        case 1: return 'bg-amber-200 border-amber-300';
        case 2: return 'bg-amber-400 border-amber-500';
        case 3: return 'bg-amber-600 border-amber-700';
        default: return 'bg-amber-700 border-amber-800';
      }
    }
    if (source === 'github') {
      switch (intensity) {
        case 1: return 'bg-emerald-200 border-emerald-300';
        case 2: return 'bg-emerald-400 border-emerald-500';
        case 3: return 'bg-emerald-600 border-emerald-700';
        default: return 'bg-emerald-800 border-emerald-900';
      }
    }
    if (source === 'codeforces') {
      switch (intensity) {
        case 1: return 'bg-blue-200 border-blue-300';
        case 2: return 'bg-blue-400 border-blue-500';
        case 3: return 'bg-blue-600 border-blue-700';
        default: return 'bg-blue-800 border-blue-900';
      }
    }

    // Unified "All Platforms" view: detect dominant platform or polyglot day
    const breakdown = source_breakdown || {};
    const lc = breakdown.leetcode || 0;
    const gh = breakdown.github || 0;
    const cf = breakdown.codeforces || 0;
    const activePlatforms = (lc > 0 ? 1 : 0) + (gh > 0 ? 1 : 0) + (cf > 0 ? 1 : 0);

    // If active across multiple platforms on the same day -> Distinct Multi-Platform Indigo
    if (activePlatforms >= 2) {
      switch (intensity) {
        case 1: return 'bg-indigo-300 border-indigo-400';
        case 2: return 'bg-indigo-500 border-indigo-600';
        case 3: return 'bg-indigo-600 border-indigo-700';
        default: return 'bg-indigo-800 border-indigo-900';
      }
    }

    // Dominant single platform colors
    if (lc > gh && lc > cf) {
      // LeetCode dominant -> Amber
      switch (intensity) {
        case 1: return 'bg-amber-200 border-amber-300';
        case 2: return 'bg-amber-400 border-amber-500';
        case 3: return 'bg-amber-600 border-amber-700';
        default: return 'bg-amber-700 border-amber-800';
      }
    }
    if (cf > gh && cf > lc) {
      // Codeforces dominant -> Blue
      switch (intensity) {
        case 1: return 'bg-blue-200 border-blue-300';
        case 2: return 'bg-blue-400 border-blue-500';
        case 3: return 'bg-blue-600 border-blue-700';
        default: return 'bg-blue-800 border-blue-900';
      }
    }
    if (gh > 0) {
      // GitHub dominant -> Emerald
      switch (intensity) {
        case 1: return 'bg-emerald-200 border-emerald-300';
        case 2: return 'bg-emerald-400 border-emerald-500';
        case 3: return 'bg-emerald-600 border-emerald-700';
        default: return 'bg-emerald-800 border-emerald-900';
      }
    }

    // Fallback emerald
    return 'bg-emerald-500 border-emerald-600';
  };

  const sources = [
    { id: '', label: 'All Platforms' },
    { id: 'leetcode', label: 'LeetCode', color: 'bg-amber-500' },
    { id: 'github', label: 'GitHub', color: 'bg-emerald-500' },
    { id: 'codeforces', label: 'Codeforces', color: 'bg-blue-500' },
  ];

  return (
    <div className="w-full">
      {/* Platform Filter Controls & Color Legends */}
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4 pb-3 border-b border-neutral-100">
        <div className="flex items-center gap-1.5 overflow-x-auto text-xs">
          {sources.map((s) => (
            <button
              key={s.id}
              onClick={() => onSourceChange?.(s.id)}
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md font-medium transition-all ${
                selectedSource === s.id
                  ? 'bg-neutral-900 text-white shadow-xs'
                  : 'bg-neutral-100/80 text-neutral-600 hover:bg-neutral-200/60'
              }`}
            >
              {s.color && <span className={`w-2 h-2 rounded-full ${s.color}`} />}
              {s.label}
            </button>
          ))}
        </div>

        {/* Legend */}
        <div className="flex flex-wrap items-center gap-3 text-xs text-neutral-500">
          <div className="flex items-center gap-2">
            <span className="flex items-center gap-1 text-[11px] text-neutral-600">
              <span className="w-2 h-2 rounded-xs bg-amber-500" /> LeetCode
            </span>
            <span className="flex items-center gap-1 text-[11px] text-neutral-600">
              <span className="w-2 h-2 rounded-xs bg-emerald-500" /> GitHub
            </span>
            <span className="flex items-center gap-1 text-[11px] text-neutral-600">
              <span className="w-2 h-2 rounded-xs bg-blue-500" /> Codeforces
            </span>
            <span className="flex items-center gap-1 text-[11px] text-neutral-600">
              <span className="w-2 h-2 rounded-xs bg-indigo-600" /> Multi-Platform
            </span>
          </div>

          <div className="flex items-center gap-1 pl-2 border-l border-neutral-200">
            <span className="text-[10px] text-neutral-400">Less</span>
            <span className="w-2.5 h-2.5 rounded-xs bg-neutral-100 border border-neutral-200/60" />
            <span className="w-2.5 h-2.5 rounded-xs bg-emerald-300" />
            <span className="w-2.5 h-2.5 rounded-xs bg-emerald-600" />
            <span className="w-2.5 h-2.5 rounded-xs bg-emerald-800" />
            <span className="text-[10px] text-neutral-400">More</span>
          </div>
        </div>
      </div>

      {/* Heatmap Grid */}
      <div className="overflow-x-auto pb-2">
        <div className="inline-flex gap-[3px] min-w-full p-1">
          {weeks.map((week, wIdx) => (
            <div key={wIdx} className="flex flex-col gap-[3px]">
              {week.map((day, dIdx) => {
                if (!day.dateStr) {
                  return <div key={dIdx} className="w-3 h-3 opacity-0" />;
                }
                const count = day.dayData?.count || 0;
                const breakdown = day.dayData?.source_breakdown || {};
                const isSelected = selectedDay?.date === day.dateStr;

                const tooltipParts = [
                  `${day.dateStr}: ${count} total activity events`,
                  breakdown.leetcode ? `LeetCode: ${breakdown.leetcode} solves` : '',
                  breakdown.github ? `GitHub: ${breakdown.github} commits` : '',
                  breakdown.codeforces ? `Codeforces: ${breakdown.codeforces} events` : '',
                ].filter(Boolean);

                return (
                  <button
                    key={day.dateStr}
                    onClick={() => setSelectedDay(day.dayData || { date: day.dateStr, count: 0, intensity: 0, source_breakdown: {} })}
                    title={tooltipParts.join(' | ')}
                    className={`w-3 h-3 rounded-[2px] border transition-transform hover:scale-125 focus:outline-none ${getCellColor(
                      day.dayData
                    )} ${isSelected ? 'ring-2 ring-indigo-600 ring-offset-1 z-10 scale-125' : ''}`}
                  />
                );
              })}
            </div>
          ))}
        </div>
      </div>

      {/* Selected Day Breakdown Card */}
      {selectedDay && (
        <div className="mt-4 p-3 bg-neutral-50/80 border border-neutral-200 rounded-lg flex flex-wrap items-center justify-between gap-3 text-xs">
          <div className="flex items-center gap-3">
            <span className="font-semibold text-neutral-900 flex items-center gap-1.5">
              <Calendar className="w-3.5 h-3.5 text-neutral-500" />
              {new Date(selectedDay.date + 'T00:00:00').toLocaleDateString(undefined, {
                weekday: 'long',
                year: 'numeric',
                month: 'short',
                day: 'numeric',
              })}
            </span>
            <span className="text-neutral-400">•</span>
            <span className="font-bold text-neutral-900">{selectedDay.count} verified events</span>
          </div>

          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1 text-amber-700">
              <Code2 className="w-3.5 h-3.5 text-amber-500" />
              {selectedDay.source_breakdown?.leetcode || 0} LeetCode
            </span>
            <span className="flex items-center gap-1 text-emerald-700">
              <GitCommit className="w-3.5 h-3.5 text-emerald-500" />
              {selectedDay.source_breakdown?.github || 0} GitHub
            </span>
            <span className="flex items-center gap-1 text-blue-700">
              <Trophy className="w-3.5 h-3.5 text-blue-500" />
              {selectedDay.source_breakdown?.codeforces || 0} Codeforces
            </span>
          </div>
        </div>
      )}
    </div>
  );
};

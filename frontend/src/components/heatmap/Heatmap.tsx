import React, { useState } from 'react';
import { HeatmapResponse, HeatmapDayData } from '../../types';
import { Badge } from '../ui/Badge';
import { Layers, Calendar, CheckCircle2, GitCommit, Trophy, Flame } from 'lucide-react';

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

  // Group into weeks of 7 days (Sunday = 0 to Saturday = 6)
  const weeks: (typeof days)[] = [];
  let currentWeek: typeof days = [];

  // Pad first week if it doesn't start on Sunday
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

  const getIntensityColor = (intensity: number) => {
    switch (intensity) {
      case 1:
        return 'bg-emerald-200 border-emerald-300';
      case 2:
        return 'bg-emerald-400 border-emerald-500';
      case 3:
        return 'bg-emerald-600 border-emerald-700';
      case 4:
        return 'bg-emerald-800 border-emerald-900';
      default:
        return 'bg-neutral-100 border-neutral-200/60 hover:border-neutral-300';
    }
  };

  const sources = [
    { id: '', label: 'All Platforms' },
    { id: 'leetcode', label: 'LeetCode' },
    { id: 'eterno_challenge', label: 'Eterno Challenges' },
    { id: 'github', label: 'GitHub' },
    { id: 'codeforces', label: 'Codeforces' },
  ];

  return (
    <div className="w-full">
      {/* Platform Filter Controls */}
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4 pb-3 border-b border-neutral-100">
        <div className="flex items-center gap-1.5 overflow-x-auto text-xs">
          {sources.map((s) => (
            <button
              key={s.id}
              onClick={() => onSourceChange?.(s.id)}
              className={`px-2.5 py-1 rounded-md font-medium transition-all ${
                selectedSource === s.id
                  ? 'bg-neutral-900 text-white shadow-xs'
                  : 'bg-neutral-100/80 text-neutral-600 hover:bg-neutral-200/60'
              }`}
            >
              {s.label}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2 text-xs text-neutral-500">
          <span>Less</span>
          <div className="flex items-center gap-1">
            <span className="w-2.5 h-2.5 rounded-xs bg-neutral-100 border border-neutral-200/60" />
            <span className="w-2.5 h-2.5 rounded-xs bg-emerald-200 border border-emerald-300" />
            <span className="w-2.5 h-2.5 rounded-xs bg-emerald-400 border border-emerald-500" />
            <span className="w-2.5 h-2.5 rounded-xs bg-emerald-600 border border-emerald-700" />
            <span className="w-2.5 h-2.5 rounded-xs bg-emerald-800 border border-emerald-900" />
          </div>
          <span>More</span>
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
                const intensity = day.dayData?.intensity || 0;
                const isSelected = selectedDay?.date === day.dateStr;

                return (
                  <button
                    key={day.dateStr}
                    onClick={() => setSelectedDay(day.dayData || { date: day.dateStr, count: 0, intensity: 0, source_breakdown: {} })}
                    title={`${day.dateStr}: ${count} verified coding events`}
                    className={`w-3 h-3 rounded-[2px] border transition-transform hover:scale-125 focus:outline-none ${getIntensityColor(
                      intensity
                    )} ${isSelected ? 'ring-2 ring-indigo-600 ring-offset-1 z-10 scale-125' : ''}`}
                  />
                );
              })}
            </div>
          ))}
        </div>
      </div>

      {/* Selected Day Breakdown Drawer */}
      {selectedDay && (
        <div className="mt-4 p-3.5 bg-neutral-50/70 border border-neutral-200/80 rounded-lg flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-white rounded-md border border-neutral-200/80 shadow-xs">
              <Calendar className="w-4 h-4 text-neutral-600" />
            </div>
            <div>
              <p className="text-xs font-semibold text-neutral-900">{selectedDay.date}</p>
              <p className="text-[11px] text-neutral-500">
                {selectedDay.count === 0 ? 'No activity recorded on this day' : `${selectedDay.count} verified events across platforms`}
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {selectedDay.source_breakdown && Object.entries(selectedDay.source_breakdown).length > 0 ? (
              Object.entries(selectedDay.source_breakdown).map(([src, cnt]) => (
                <Badge key={src} variant="info" size="sm">
                  {src.replace('_', ' ')}: {cnt}
                </Badge>
              ))
            ) : (
              <span className="text-xs text-neutral-400 italic">No events</span>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

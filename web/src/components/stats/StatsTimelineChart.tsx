import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ReferenceLine,
  ResponsiveContainer,
} from 'recharts'
import { useStatsTimeline } from '@/hooks/useStats'
import { usePublicSettings } from '@/hooks/useSystemSettings'
import { Spinner } from '@/components/ui/Spinner'
import type { StatsRange } from '@/api/stats'

interface Props {
  projectKey: string
}

const RANGES: StatsRange[] = ['24h', '3d', '7d']

function useDarkMode() {
  // Check if dark class is on html element
  return document.documentElement.classList.contains('dark')
}

export function StatsTimelineChart({ projectKey }: Props) {
  const { t } = useTranslation()
  const { data: publicSettings } = usePublicSettings()
  const [range_, setRange] = useState<StatsRange>('7d')
  const { data: points, isLoading } = useStatsTimeline(projectKey, range_)
  const isDark = useDarkMode()

  // Feature toggle: hidden when explicitly disabled
  const featureEnabled = publicSettings?.feature_stats_timeline !== false
  if (!featureEnabled) return null

  const chartData = useMemo(() => {
    if (!points || points.length === 0) return []
    return points.map((p) => {
      const date = new Date(p.captured_at)
      return {
        time: date.getTime(),
        label: formatTime(date, range_),
        todo: p.todo_count,
        inProgress: p.in_progress_count,
        done: -p.done_count,
        cancelled: -p.cancelled_count,
      }
    })
  }, [points, range_])

  const colors = {
    todo: isDark ? '#93c5fd' : '#3b82f6',        // blue
    inProgress: isDark ? '#a78bfa' : '#8b5cf6',   // violet
    done: isDark ? '#6ee7b7' : '#10b981',          // emerald
    cancelled: isDark ? '#fca5a5' : '#ef4444',     // red
    grid: isDark ? '#374151' : '#e5e7eb',
    text: isDark ? '#9ca3af' : '#6b7280',
    tooltip: isDark ? '#1f2937' : '#ffffff',
    tooltipBorder: isDark ? '#374151' : '#e5e7eb',
  }

  const hasData = chartData.length > 0

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
          {t('projects.overview.activity')}
        </h2>
        <div className="flex gap-1">
          {RANGES.map((r) => (
            <button
              key={r}
              onClick={() => setRange(r)}
              className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors ${
                range_ === r
                  ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
                  : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800'
              }`}
            >
              {t(`projects.overview.range_${r}`)}
            </button>
          ))}
        </div>
      </div>
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-4">
        {isLoading ? (
          <div className="flex items-center justify-center" style={{ height: 200 }}>
            <Spinner />
          </div>
        ) : !hasData ? (
          <div className="flex items-center justify-center text-sm text-gray-400 dark:text-gray-500" style={{ height: 200 }}>
            {t('projects.overview.noActivityData')}
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={chartData} stackOffset="sign" margin={{ top: 5, right: 5, left: -10, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke={colors.grid} vertical={false} />
              <XAxis
                dataKey="label"
                tick={{ fontSize: 11, fill: colors.text }}
                tickLine={false}
                axisLine={{ stroke: colors.grid }}
              />
              <YAxis
                tick={{ fontSize: 11, fill: colors.text }}
                tickLine={false}
                axisLine={false}
                allowDecimals={false}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: colors.tooltip,
                  border: `1px solid ${colors.tooltipBorder}`,
                  borderRadius: 8,
                  fontSize: 12,
                }}
                labelStyle={{ color: colors.text, fontWeight: 600, marginBottom: 4 }}
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                formatter={(value: any, name: any) => {
                  const absValue = Math.abs(Number(value) || 0)
                  return [absValue, formatSeriesName(String(name), t)]
                }}
              />
              <Legend
                wrapperStyle={{ fontSize: 11 }}
                formatter={(value: string) => formatSeriesName(value, t)}
              />
              <ReferenceLine y={0} stroke={colors.grid} />
              <Bar dataKey="todo" stackId="stack" fill={colors.todo} radius={[0, 0, 0, 0]} />
              <Bar dataKey="inProgress" stackId="stack" fill={colors.inProgress} radius={[2, 2, 0, 0]} />
              <Bar dataKey="done" stackId="stack" fill={colors.done} radius={[0, 0, 0, 0]} />
              <Bar dataKey="cancelled" stackId="stack" fill={colors.cancelled} radius={[0, 0, 2, 2]} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  )
}

function formatTime(date: Date, range_: StatsRange): string {
  if (range_ === '24h') {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  if (range_ === '3d') {
    return `${date.toLocaleDateString([], { month: 'short', day: 'numeric' })} ${date.toLocaleTimeString([], { hour: '2-digit' })}`
  }
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function formatSeriesName(key: string, t: (k: string) => string): string {
  const map: Record<string, string> = {
    todo: t('projects.overview.seriesTodo'),
    inProgress: t('projects.overview.seriesInProgress'),
    done: t('projects.overview.seriesDone'),
    cancelled: t('projects.overview.seriesCancelled'),
  }
  return map[key] || key
}

import { useTranslation } from 'react-i18next'
import { Clock } from 'lucide-react'
import { Tooltip } from '@/components/ui/Tooltip'
import { formatDuration } from '@/utils/duration'
import type { SLAInfo } from '@/api/workitems'

interface Props {
  sla: SLAInfo | null
  compact?: boolean
}

const STATUS_COLORS = {
  on_track: 'text-green-600 dark:text-green-400',
  warning: 'text-yellow-600 dark:text-yellow-400',
  breached: 'text-red-600 dark:text-red-400',
} as const

const STATUS_I18N_KEYS: Record<string, string> = {
  on_track: 'sla.onTrack',
  warning: 'sla.warning',
  breached: 'sla.breached',
}

export function SLAIndicator({ sla, compact = false }: Props) {
  const { t } = useTranslation()

  if (!sla) return null

  const colorClass = STATUS_COLORS[sla.status] || STATUS_COLORS.on_track
  const remaining = sla.remaining_seconds
  const duration = formatDuration(Math.abs(remaining))
  const label = remaining >= 0
    ? t('sla.left', { duration })
    : t('sla.overdue', { duration })

  const tooltipContent = `SLA: ${t(STATUS_I18N_KEYS[sla.status] ?? sla.status)} — ${label} (${sla.percentage}%)`

  if (compact) {
    return (
      <Tooltip content={tooltipContent}>
        <span className={`inline-flex items-center gap-0.5 ${colorClass}`}>
          <Clock className="h-3 w-3" />
        </span>
      </Tooltip>
    )
  }

  return (
    <Tooltip content={tooltipContent}>
      <span className={`inline-flex items-center gap-1 text-xs ${colorClass}`}>
        <Clock className="h-3.5 w-3.5" />
        <span>{label}</span>
      </span>
    </Tooltip>
  )
}

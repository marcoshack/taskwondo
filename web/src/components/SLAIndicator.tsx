import { useRef, useState, useCallback, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock, Pause } from 'lucide-react'
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
  paused: 'text-gray-400 dark:text-gray-500',
} as const

const STATUS_I18N_KEYS: Record<string, string> = {
  on_track: 'sla.onTrack',
  warning: 'sla.warning',
  breached: 'sla.breached',
  paused: 'sla.paused',
}

function MarqueeContainer({ children }: { children: ReactNode }) {
  const innerRef = useRef<HTMLSpanElement>(null)
  const styleRef = useRef<HTMLStyleElement | null>(null)
  const [scrollAnim, setScrollAnim] = useState<string | undefined>(undefined)

  const handleMouseEnter = useCallback(() => {
    const inner = innerRef.current
    if (!inner) return

    // Find the <td> ancestor to get the actual available width
    const td = inner.closest('td')
    if (!td) return

    const tdStyle = getComputedStyle(td)
    const availableWidth = td.clientWidth - parseFloat(tdStyle.paddingLeft) - parseFloat(tdStyle.paddingRight)
    const contentWidth = inner.scrollWidth
    const overflow = contentWidth - availableWidth
    if (overflow <= 0) return

    const scrollDuration = Math.max(1.5, overflow / 30)
    const totalDuration = scrollDuration * 2 + 1
    const scrollEnd = (scrollDuration / totalDuration) * 100
    const pauseEnd = ((scrollDuration + 0.5) / totalDuration) * 100
    const rewindEnd = ((scrollDuration * 2 + 0.5) / totalDuration) * 100

    const name = `sla-marquee-${Date.now()}`
    const style = document.createElement('style')
    style.textContent = `@keyframes ${name} {
  0% { transform: translateX(0); }
  ${scrollEnd.toFixed(1)}% { transform: translateX(-${overflow}px); }
  ${pauseEnd.toFixed(1)}% { transform: translateX(-${overflow}px); }
  ${rewindEnd.toFixed(1)}% { transform: translateX(0); }
  100% { transform: translateX(0); }
}`
    document.head.appendChild(style)
    styleRef.current = style
    setScrollAnim(`${name} ${totalDuration}s ease-in-out infinite`)
  }, [])

  const handleMouseLeave = useCallback(() => {
    setScrollAnim(undefined)
    if (styleRef.current) {
      styleRef.current.remove()
      styleRef.current = null
    }
  }, [])

  return (
    <span
      ref={innerRef}
      className="inline-flex items-center gap-1 whitespace-nowrap"
      style={scrollAnim ? { animation: scrollAnim } : undefined}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {children}
    </span>
  )
}

export function SLAIndicator({ sla, compact = false }: Props) {
  const { t } = useTranslation()

  if (!sla) return null

  const displayStatus = sla.paused ? 'paused' : sla.status
  const colorClass = STATUS_COLORS[displayStatus] || STATUS_COLORS.on_track
  const remaining = sla.remaining_seconds
  const duration = formatDuration(Math.abs(remaining))
  const label = remaining >= 0
    ? t('sla.left', { duration })
    : t('sla.overdue', { duration })

  const tooltipContent = `SLA: ${t(STATUS_I18N_KEYS[displayStatus] ?? sla.status)} — ${label} (${sla.percentage}%)`
  const Icon = sla.paused ? Pause : Clock

  if (compact) {
    return (
      <Tooltip content={tooltipContent}>
        <span className={`inline-flex items-center gap-0.5 ${colorClass}`}>
          <Icon className="h-3 w-3" />
        </span>
      </Tooltip>
    )
  }

  return (
    <Tooltip content={tooltipContent}>
      <MarqueeContainer>
        <Icon className={`h-3.5 w-3.5 shrink-0 ${colorClass}`} />
        <span className={`text-xs ${colorClass}`}>{label}</span>
      </MarqueeContainer>
    </Tooltip>
  )
}

import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'
import { Tooltip } from '@/components/ui/Tooltip'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useOncallRotation } from '@/hooks/useOncall'
import type { OncallScheduleShift, OncallRotationMember } from '@/api/oncall'

const BAR_COLORS = [
  'bg-indigo-200 dark:bg-indigo-900/50 text-indigo-800 dark:text-indigo-200',
  'bg-green-200 dark:bg-green-900/50 text-green-800 dark:text-green-200',
  'bg-yellow-200 dark:bg-yellow-900/50 text-yellow-800 dark:text-yellow-200',
  'bg-pink-200 dark:bg-pink-900/50 text-pink-800 dark:text-pink-200',
  'bg-purple-200 dark:bg-purple-900/50 text-purple-800 dark:text-purple-200',
  'bg-blue-200 dark:bg-blue-900/50 text-blue-800 dark:text-blue-200',
  'bg-orange-200 dark:bg-orange-900/50 text-orange-800 dark:text-orange-200',
  'bg-teal-200 dark:bg-teal-900/50 text-teal-800 dark:text-teal-200',
]

const DOT_COLORS = [
  'bg-indigo-500',
  'bg-green-500',
  'bg-yellow-500',
  'bg-pink-500',
  'bg-purple-500',
  'bg-blue-500',
  'bg-orange-500',
  'bg-teal-500',
]

const OVERRIDE_COLOR = 'bg-red-200 dark:bg-red-900/50 text-red-800 dark:text-red-200'
const OVERRIDE_DOT_COLOR = 'bg-red-500'

interface WeekSpan {
  memberId: string
  memberName: string
  colorIdx: number
  startCol: number
  endCol: number
  continuesFromPrev: boolean
  continuesToNext: boolean
  isOverride: boolean
  shiftStart: string
  shiftEnd: string
}

interface OncallCalendarProps {
  members: OncallRotationMember[]
  projectKey: string
  teamId: string
}

function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export function OncallCalendar({ members, projectKey, teamId }: OncallCalendarProps) {
  const { t } = useTranslation()
  const today = new Date()
  const [viewDate, setViewDate] = useState(new Date(today.getFullYear(), today.getMonth(), 1))

  const year = viewDate.getFullYear()
  const month = viewDate.getMonth()
  const firstDay = new Date(year, month, 1)
  const daysInMonth = new Date(year, month + 1, 0).getDate()

  // Monday = 0, Sunday = 6
  let startDow = firstDay.getDay() - 1
  if (startDow < 0) startDow = 6

  // Build calendar slots including prev/next month padding
  const calendarSlots = useMemo(() => {
    const slots: { date: Date; inMonth: boolean }[] = []
    const prevMonthDays = new Date(year, month, 0).getDate()
    for (let i = 0; i < startDow; i++) {
      slots.push({
        date: new Date(year, month - 1, prevMonthDays - startDow + 1 + i),
        inMonth: false,
      })
    }
    for (let d = 1; d <= daysInMonth; d++) {
      slots.push({ date: new Date(year, month, d), inMonth: true })
    }
    while (slots.length % 7 !== 0) {
      const nextDay = slots.length - startDow - daysInMonth + 1
      slots.push({ date: new Date(year, month + 1, nextDay), inMonth: false })
    }
    return slots
  }, [year, month, startDow, daysInMonth])

  // Compute date range for the API call (first visible day to last visible day)
  const rangeStart = calendarSlots.length > 0 ? formatDate(calendarSlots[0].date) : ''
  const rangeEnd = calendarSlots.length > 0 ? formatDate(calendarSlots[calendarSlots.length - 1].date) : ''

  const { data: rotationData } = useOncallRotation(projectKey, teamId, rangeStart, rangeEnd)
  const shifts = rotationData?.shifts ?? []
  const resolvedMembers = rotationData?.members ?? members

  const memberMap = useMemo(() => {
    const map = new Map<string, OncallRotationMember>()
    for (const m of resolvedMembers) map.set(m.user_id, m)
    return map
  }, [resolvedMembers])

  const sortedMembers = useMemo(
    () => [...resolvedMembers].sort((a, b) => a.position - b.position),
    [resolvedMembers],
  )

  const colorMap = useMemo(() => {
    const map = new Map<string, number>()
    sortedMembers.forEach((m, i) => {
      map.set(m.user_id, i % BAR_COLORS.length)
    })
    return map
  }, [sortedMembers])

  // Assign shifts to each calendar day.
  // For each day, find which shifts overlap. If two shifts meet on a day (transition),
  // primary = the one starting earlier (outgoing), secondary = the one starting later (incoming).
  const dayAssignments = useMemo(() => {
    return calendarSlots.map((slot) => {
      const dayStart = new Date(slot.date.getFullYear(), slot.date.getMonth(), slot.date.getDate())
      const dayEnd = new Date(slot.date.getFullYear(), slot.date.getMonth(), slot.date.getDate() + 1)

      const overlapping: OncallScheduleShift[] = []
      for (const shift of shifts) {
        const sStart = new Date(shift.start_at)
        const sEnd = new Date(shift.end_at)
        if (sStart < dayEnd && sEnd > dayStart) {
          overlapping.push(shift)
        }
      }

      if (overlapping.length === 0) {
        return { primary: null as OncallScheduleShift | null, secondary: null as OncallScheduleShift | null }
      }

      if (overlapping.length === 1) {
        return { primary: overlapping[0], secondary: null }
      }

      // Two or more shifts — sort by start_at; first is outgoing (primary top), second is incoming (secondary bottom)
      overlapping.sort((a, b) => new Date(a.start_at).getTime() - new Date(b.start_at).getTime())
      return { primary: overlapping[0], secondary: overlapping[1] }
    })
  }, [calendarSlots, shifts])

  const numWeeks = calendarSlots.length / 7

  // Build continuous spans per week (Google Calendar-style multi-day bars).
  const weekSpanRows = useMemo(() => {
    const result: { topSpans: WeekSpan[]; bottomSpans: WeekSpan[] }[] = []
    for (let wi = 0; wi < numWeeks; wi++) {
      const ws = wi * 7

      // Top spans from primary assignments
      const topSpans: WeekSpan[] = []
      let spanStart: number | null = null
      let spanShift: OncallScheduleShift | null = null

      for (let di = 0; di <= 7; di++) {
        const shift = di < 7 ? dayAssignments[ws + di].primary : null
        const same = shift && spanShift && shift.user_id === spanShift.user_id && shift.is_override === spanShift.is_override

        if (!same) {
          if (spanShift && spanStart !== null) {
            const endCol = di - 1
            const prevLast = ws > 0 ? dayAssignments[ws - 1].primary : null
            const nextFirst = ws + 7 < dayAssignments.length ? dayAssignments[ws + 7].primary : null
            topSpans.push({
              memberId: spanShift.user_id,
              memberName: memberMap.get(spanShift.user_id)?.display_name ?? '',
              colorIdx: spanShift.is_override ? -1 : (colorMap.get(spanShift.user_id) ?? 0),
              startCol: spanStart,
              endCol,
              continuesFromPrev: spanStart === 0 && prevLast?.user_id === spanShift.user_id && prevLast?.is_override === spanShift.is_override,
              continuesToNext: endCol === 6 && nextFirst?.user_id === spanShift.user_id && nextFirst?.is_override === spanShift.is_override,
              isOverride: spanShift.is_override,
              shiftStart: spanShift.start_at,
              shiftEnd: spanShift.end_at,
            })
          }
          spanStart = shift ? di : null
          spanShift = shift
        }
      }

      // Bottom spans from secondary assignments (transition-day overlaps)
      const bottomSpans: WeekSpan[] = []
      for (let di = 0; di < 7; di++) {
        const secondary = dayAssignments[ws + di].secondary
        if (secondary) {
          bottomSpans.push({
            memberId: secondary.user_id,
            memberName: memberMap.get(secondary.user_id)?.display_name ?? '',
            colorIdx: secondary.is_override ? -1 : (colorMap.get(secondary.user_id) ?? 0),
            startCol: di,
            endCol: di,
            continuesFromPrev: false,
            continuesToNext: false,
            isOverride: secondary.is_override,
            shiftStart: secondary.start_at,
            shiftEnd: secondary.end_at,
          })
        }
      }

      result.push({ topSpans, bottomSpans })
    }
    return result
  }, [dayAssignments, numWeeks, colorMap, memberMap])

  const isCurrentMonth = today.getFullYear() === year && today.getMonth() === month
  const todayDate = today.getDate()

  const hasOverrides = shifts.some(s => s.is_override)

  function shiftTooltip(span: WeekSpan): string {
    const start = new Date(span.shiftStart).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' })
    const end = new Date(span.shiftEnd).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' })
    const type = span.isOverride ? t('teams.oncall.calendar.shiftOverride') : t('teams.oncall.calendar.shiftRegular')
    return `${span.memberName}\n${start} — ${end}\n${type}`
  }

  function prevMonth() { setViewDate(new Date(year, month - 1, 1)) }
  function nextMonth() { setViewDate(new Date(year, month + 1, 1)) }
  function goToday() { setViewDate(new Date(today.getFullYear(), today.getMonth(), 1)) }

  const dayLabels = [
    t('teams.oncall.calendar.mon'),
    t('teams.oncall.calendar.tue'),
    t('teams.oncall.calendar.wed'),
    t('teams.oncall.calendar.thu'),
    t('teams.oncall.calendar.fri'),
    t('teams.oncall.calendar.sat'),
    t('teams.oncall.calendar.sun'),
  ]

  const monthLabel = new Date(year, month).toLocaleDateString(undefined, { month: 'long', year: 'numeric' })

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('teams.oncall.calendar.title')}</h3>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={prevMonth}>
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100 min-w-[140px] text-center">
            {monthLabel}
          </span>
          <Button variant="ghost" size="sm" onClick={nextMonth}>
            <ChevronRight className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={goToday}>
            {t('teams.oncall.calendar.today')}
          </Button>
        </div>
      </div>

      {/* Grid */}
      <div className="border border-gray-200 dark:border-gray-600 rounded-lg overflow-hidden">
        {/* Day headers */}
        <div className="grid grid-cols-7 bg-gray-50 dark:bg-gray-800">
          {dayLabels.map((label) => (
            <div key={label} className="px-1 py-2 text-center text-xs font-medium text-gray-500 dark:text-gray-400">
              {label}
            </div>
          ))}
        </div>

        {/* Weeks */}
        {Array.from({ length: numWeeks }, (_, wi) => {
          const weekStart = wi * 7
          const weekSlots = calendarSlots.slice(weekStart, weekStart + 7)

          return (
            <div key={wi} className="border-t border-gray-200 dark:border-gray-600">
              {/* Day numbers */}
              <div className="grid grid-cols-7">
                {weekSlots.map((slot, di) => {
                  const day = slot.date.getDate()
                  const isToday = isCurrentMonth && slot.inMonth && day === todayDate

                  return (
                    <div key={di} className="px-1.5 pt-1 pb-0.5">
                      {isToday ? (
                        <span className="inline-flex items-center justify-center h-5 w-5 rounded-full bg-indigo-600 text-white text-xs font-bold">
                          {day}
                        </span>
                      ) : (
                        <span className={`text-xs ${
                          slot.inMonth
                            ? 'text-gray-700 dark:text-gray-300'
                            : 'text-gray-400 dark:text-gray-600'
                        }`}>
                          {day}
                        </span>
                      )}
                    </div>
                  )
                })}
              </div>

              {/* Span bars */}
              <div className="grid grid-cols-7 pb-1.5" style={{ minHeight: '1.75rem' }}>
                {weekSpanRows[wi].topSpans.map((span, si) => {
                  const rl = !span.continuesFromPrev ? 'rounded-l-md' : ''
                  const rr = !span.continuesToNext ? 'rounded-r-md' : ''
                  const barColor = span.isOverride ? OVERRIDE_COLOR : BAR_COLORS[span.colorIdx]

                  return (
                    <Tooltip
                      key={si}
                      content={shiftTooltip(span)}
                      position="bottom"
                      maxWidth={220}
                      className={`h-6 flex items-center px-1.5 text-[0.65rem] font-medium leading-none truncate ${barColor} ${rl} ${rr}`}
                      style={{ gridColumn: `${span.startCol + 1} / ${span.endCol + 2}`, gridRow: 1 }}
                    >
                      {span.isOverride ? `\u26A1 ${span.memberName}` : span.memberName}
                    </Tooltip>
                  )
                })}
                {weekSpanRows[wi].bottomSpans.map((span, si) => {
                  const barColor = span.isOverride ? OVERRIDE_COLOR : BAR_COLORS[span.colorIdx]

                  return (
                    <Tooltip
                      key={`b-${si}`}
                      content={shiftTooltip(span)}
                      position="bottom"
                      maxWidth={220}
                      className={`h-6 flex items-center px-1.5 text-[0.65rem] font-medium leading-none truncate ${barColor} rounded-md mt-0.5`}
                      style={{ gridColumn: `${span.startCol + 1} / ${span.endCol + 2}`, gridRow: 2 }}
                    >
                      {span.isOverride ? `\u26A1 ${span.memberName}` : span.memberName}
                    </Tooltip>
                  )
                })}
              </div>
            </div>
          )
        })}
      </div>

      {/* Member legend */}
      <div className="flex flex-wrap gap-3">
        {sortedMembers.map((member) => {
          const colorIdx = colorMap.get(member.user_id) ?? 0
          return (
            <div key={member.user_id} className="flex items-center gap-1.5">
              <span className={`h-2.5 w-2.5 rounded-full ${DOT_COLORS[colorIdx]}`} />
              <span className="text-xs text-gray-700 dark:text-gray-300">{member.display_name}</span>
            </div>
          )
        })}
        {hasOverrides && (
          <div className="flex items-center gap-1.5">
            <span className={`h-2.5 w-2.5 rounded-full ${OVERRIDE_DOT_COLOR}`} />
            <span className="text-xs text-gray-700 dark:text-gray-300">{t('teams.oncall.override.title')}</span>
          </div>
        )}
      </div>
    </div>
  )
}

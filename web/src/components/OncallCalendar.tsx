import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useOncallOverrides } from '@/hooks/useOncall'
import type { OncallRotation, OncallRotationMember, OncallOverride } from '@/api/oncall'

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

interface WeekSpan {
  memberId: string
  memberName: string
  colorIdx: number
  startCol: number
  endCol: number
  continuesFromPrev: boolean
  continuesToNext: boolean
}

const OVERRIDE_COLOR = 'bg-red-200 dark:bg-red-900/50 text-red-800 dark:text-red-200'
const OVERRIDE_DOT_COLOR = 'bg-red-500'

interface OncallCalendarProps {
  rotation: OncallRotation
  members: OncallRotationMember[]
  projectKey: string
  teamId: string
}

export function OncallCalendar({ rotation, members, projectKey, teamId }: OncallCalendarProps) {
  const { t } = useTranslation()
  const today = new Date()
  const [viewDate, setViewDate] = useState(new Date(today.getFullYear(), today.getMonth(), 1))
  const { data: overrides } = useOncallOverrides(projectKey, teamId)

  const sortedMembers = useMemo(
    () => [...members].sort((a, b) => a.position - b.position),
    [members],
  )

  const colorMap = useMemo(() => {
    const map = new Map<string, number>()
    sortedMembers.forEach((m, i) => {
      map.set(m.user_id, i % BAR_COLORS.length)
    })
    return map
  }, [sortedMembers])

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

  // Compute on-call member(s) for each calendar slot.
  // On transition days (non-midnight rotation time), both the outgoing and incoming
  // member share the day: primary (top bar) = outgoing, secondary (bottom bar) = incoming.
  const dayAssignments = useMemo(() => {
    if (sortedMembers.length === 0 || rotation.period_days <= 0) {
      return calendarSlots.map(() => ({ primary: null as OncallRotationMember | null, secondary: null as OncallRotationMember | null }))
    }
    const sd = new Date(rotation.start_date)
    const rotStartUtc = Date.UTC(sd.getUTCFullYear(), sd.getUTCMonth(), sd.getUTCDate())
    const cycleDays = rotation.period_days * sortedMembers.length
    const numMembers = sortedMembers.length

    // Detect midnight rotation (no overlap needed on transition day)
    const timeMatch = rotation.rotation_time.match(/T(\d{2}:\d{2})/)
    const isMidnight = timeMatch ? timeMatch[1] === '00:00' : rotation.rotation_time.startsWith('00:00')

    return calendarSlots.map((slot) => {
      const targetUtc = Date.UTC(slot.date.getFullYear(), slot.date.getMonth(), slot.date.getDate())
      const diffDays = Math.floor((targetUtc - rotStartUtc) / (1000 * 60 * 60 * 24))
      let dayInCycle = diffDays % cycleDays
      if (dayInCycle < 0) dayInCycle += cycleDays
      // start_date is when the first rotation fires (member[0] → member[1]),
      // so offset by +1: before start_date → member[0], from start_date → member[1], etc.
      const incomingIdx = (Math.floor(dayInCycle / rotation.period_days) + 1) % numMembers
      const isTransition = dayInCycle % rotation.period_days === 0 && !isMidnight

      if (isTransition) {
        const outgoingIdx = (incomingIdx - 1 + numMembers) % numMembers
        // Skip overlap when outgoing === incoming (single-member rotation)
        if (outgoingIdx !== incomingIdx) {
          return { primary: sortedMembers[outgoingIdx], secondary: sortedMembers[incomingIdx] }
        }
      }

      return { primary: sortedMembers[incomingIdx], secondary: null }
    })
  }, [calendarSlots, sortedMembers, rotation.start_date, rotation.period_days, rotation.rotation_time])

  // Compute which calendar slots have an override active
  const overrideAssignments = useMemo(() => {
    if (!overrides || overrides.length === 0) {
      return calendarSlots.map(() => null as OncallOverride | null)
    }
    return calendarSlots.map((slot) => {
      const dayStart = new Date(slot.date.getFullYear(), slot.date.getMonth(), slot.date.getDate())
      const dayEnd = new Date(dayStart.getTime() + 24 * 60 * 60 * 1000)
      // Find the latest-created override that overlaps this day
      let latest: OncallOverride | null = null
      for (const ov of overrides) {
        const ovStart = new Date(ov.start_at)
        const ovEnd = new Date(ov.end_at)
        if (ovStart < dayEnd && ovEnd > dayStart) {
          if (!latest || new Date(ov.created_at) > new Date(latest.created_at)) {
            latest = ov
          }
        }
      }
      return latest
    })
  }, [calendarSlots, overrides])

  const numWeeks = calendarSlots.length / 7

  // Build continuous spans per week (Google Calendar-style multi-day bars).
  // Top row: primary assignments (outgoing member's bar extends through transition day).
  // Bottom row: secondary assignments (incoming member, only on transition days) OR override bars.
  const weekSpanRows = useMemo(() => {
    const result: { topSpans: WeekSpan[]; bottomSpans: WeekSpan[]; overrideSpans: WeekSpan[] }[] = []
    for (let wi = 0; wi < numWeeks; wi++) {
      const ws = wi * 7

      // Top spans from primary assignments
      const topSpans: WeekSpan[] = []
      let spanStart: number | null = null
      let spanMember: OncallRotationMember | null = null

      for (let di = 0; di <= 7; di++) {
        const member = di < 7 ? dayAssignments[ws + di].primary : null
        const same = member && spanMember && member.user_id === spanMember.user_id

        if (!same) {
          if (spanMember && spanStart !== null) {
            const endCol = di - 1
            const prevLast = ws > 0 ? dayAssignments[ws - 1].primary : null
            const nextFirst = ws + 7 < dayAssignments.length ? dayAssignments[ws + 7].primary : null
            topSpans.push({
              memberId: spanMember.user_id,
              memberName: spanMember.display_name,
              colorIdx: colorMap.get(spanMember.user_id) ?? 0,
              startCol: spanStart,
              endCol,
              continuesFromPrev: spanStart === 0 && prevLast?.user_id === spanMember.user_id,
              continuesToNext: endCol === 6 && nextFirst?.user_id === spanMember.user_id,
            })
          }
          spanStart = member ? di : null
          spanMember = member
        }
      }

      // Bottom spans from secondary assignments (transition-day overlaps)
      const bottomSpans: WeekSpan[] = []
      for (let di = 0; di < 7; di++) {
        const secondary = dayAssignments[ws + di].secondary
        if (secondary) {
          bottomSpans.push({
            memberId: secondary.user_id,
            memberName: secondary.display_name,
            colorIdx: colorMap.get(secondary.user_id) ?? 0,
            startCol: di,
            endCol: di,
            continuesFromPrev: false,
            continuesToNext: false,
          })
        }
      }

      // Override spans (continuous bars for overridden days)
      const overrideSpans: WeekSpan[] = []
      let ovStart: number | null = null
      let ovId: string | null = null
      let ovName = ''
      for (let di = 0; di <= 7; di++) {
        const ov = di < 7 ? overrideAssignments[ws + di] : null
        const sameOv = ov && ovId && ov.id === ovId
        if (!sameOv) {
          if (ovId && ovStart !== null) {
            overrideSpans.push({
              memberId: ovId,
              memberName: ovName,
              colorIdx: -1, // override color
              startCol: ovStart,
              endCol: di - 1,
              continuesFromPrev: false,
              continuesToNext: false,
            })
          }
          ovStart = ov ? di : null
          ovId = ov ? ov.id : null
          ovName = ov ? ov.override_user_name : ''
        }
      }

      result.push({ topSpans, bottomSpans, overrideSpans })
    }
    return result
  }, [dayAssignments, overrideAssignments, numWeeks, colorMap])

  const isCurrentMonth = today.getFullYear() === year && today.getMonth() === month
  const todayDate = today.getDate()

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
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
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
            <div key={wi} className="border-t border-gray-200 dark:border-gray-700">
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

                  return (
                    <div
                      key={si}
                      className={`h-6 flex items-center px-1.5 text-[0.65rem] font-medium leading-none truncate ${BAR_COLORS[span.colorIdx]} ${rl} ${rr}`}
                      style={{
                        gridColumn: `${span.startCol + 1} / ${span.endCol + 2}`,
                        gridRow: 1,
                      }}
                    >
                      {span.memberName}
                    </div>
                  )
                })}
                {weekSpanRows[wi].bottomSpans.map((span, si) => (
                  <div
                    key={`b-${si}`}
                    className={`h-6 flex items-center px-1.5 text-[0.65rem] font-medium leading-none truncate ${BAR_COLORS[span.colorIdx]} rounded-md mt-0.5`}
                    style={{
                      gridColumn: `${span.startCol + 1} / ${span.endCol + 2}`,
                      gridRow: 2,
                    }}
                  >
                    {span.memberName}
                  </div>
                ))}
                {weekSpanRows[wi].overrideSpans.map((span, si) => (
                  <div
                    key={`ov-${si}`}
                    className={`h-6 flex items-center px-1.5 text-[0.65rem] font-medium leading-none truncate ${OVERRIDE_COLOR} rounded-md mt-0.5`}
                    style={{
                      gridColumn: `${span.startCol + 1} / ${span.endCol + 2}`,
                      gridRow: 3,
                    }}
                  >
                    ⚡ {span.memberName}
                  </div>
                ))}
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
        {overrides && overrides.length > 0 && (
          <div className="flex items-center gap-1.5">
            <span className={`h-2.5 w-2.5 rounded-full ${OVERRIDE_DOT_COLOR}`} />
            <span className="text-xs text-gray-700 dark:text-gray-300">{t('teams.oncall.override.title')}</span>
          </div>
        )}
      </div>
    </div>
  )
}

import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { Tooltip } from '@/components/ui/Tooltip'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { useSLATargets, useBulkUpsertSLATargets } from '@/hooks/useSLA'
import { parseDuration, formatDuration } from '@/utils/duration'
import type { Workflow, WorkflowStatus } from '@/api/workflows'

interface Props {
  open: boolean
  onClose: () => void
  onSave?: () => void
  projectKey: string
  workItemType: string
  workflow: Workflow
  hasBusinessHours?: boolean
  readOnly?: boolean
}

interface StatusRow {
  name: string
  displayName: string
  category: string
  duration: string
  calendarMode: string
}

const PRIORITIES = ['critical', 'high', 'medium', 'low'] as const

export function SLAConfigModal({ open, onClose, onSave, projectKey, workItemType, workflow, hasBusinessHours = false, readOnly = false }: Props) {
  const { t } = useTranslation()
  const { data: existingTargets } = useSLATargets(projectKey)
  const bulkUpsert = useBulkUpsertSLATargets(projectKey)
  // Unified rows (when perPriority is off)
  const [rows, setRows] = useState<StatusRow[]>([])
  // Per-priority rows: priority → StatusRow[]
  const [priorityRows, setPriorityRows] = useState<Record<string, StatusRow[]>>({})
  const [perPriority, setPerPriority] = useState(false)
  const [expandedPriorities, setExpandedPriorities] = useState<Set<string>>(new Set(PRIORITIES))
  const [error, setError] = useState('')
  const [overwriteWarning, setOverwriteWarning] = useState(false)

  const isTerminal = (category: string) => category === 'done' || category === 'cancelled'

  const buildEmptyRows = useCallback((statuses: WorkflowStatus[]): StatusRow[] => {
    return statuses.map((s) => ({
      name: s.name,
      displayName: s.display_name,
      category: s.category,
      duration: '',
      calendarMode: '24x7',
    }))
  }, [])

  // Initialize from existing targets
  useEffect(() => {
    if (!workflow?.statuses) return

    // Group existing targets by (status, priority)
    const targetsByKey = new Map<string, { targetSeconds: number; calendarMode: string }>()
    if (existingTargets) {
      for (const t of existingTargets) {
        if (t.work_item_type === workItemType && t.workflow_id === workflow.id) {
          targetsByKey.set(`${t.status_name}:${t.priority}`, {
            targetSeconds: t.target_seconds,
            calendarMode: t.calendar_mode,
          })
        }
      }
    }

    // Build per-priority rows
    const newPriorityRows: Record<string, StatusRow[]> = {}
    for (const p of PRIORITIES) {
      newPriorityRows[p] = workflow.statuses.map((s: WorkflowStatus) => {
        const existing = targetsByKey.get(`${s.name}:${p}`)
        return {
          name: s.name,
          displayName: s.display_name,
          category: s.category,
          duration: existing ? formatDuration(existing.targetSeconds) : '',
          calendarMode: existing?.calendarMode ?? '24x7',
        }
      })
    }
    setPriorityRows(newPriorityRows)

    // Detect mode: if all priorities have identical values, show unified mode
    const allIdentical = PRIORITIES.every((p) =>
      newPriorityRows[p].every((row, i) =>
        row.duration === newPriorityRows.critical[i].duration &&
        row.calendarMode === newPriorityRows.critical[i].calendarMode
      )
    )

    if (allIdentical) {
      setPerPriority(false)
      // Use critical's values as the unified row (all are the same)
      setRows(newPriorityRows.critical.map((r) => ({ ...r })))
    } else {
      setPerPriority(true)
      // Still set unified rows from critical for fallback display
      setRows(newPriorityRows.critical.map((r) => ({ ...r })))
    }
  }, [workflow, existingTargets, workItemType, buildEmptyRows])

  function updateRow(index: number, field: keyof StatusRow, value: string) {
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: value } : r)))
  }

  function updatePriorityRow(priority: string, index: number, field: keyof StatusRow, value: string) {
    setPriorityRows((prev) => ({
      ...prev,
      [priority]: prev[priority].map((r, i) => (i === index ? { ...r, [field]: value } : r)),
    }))
  }

  function handleTogglePerPriority(checked: boolean) {
    setOverwriteWarning(false)
    if (checked) {
      // Turning ON: copy unified values into all priorities
      const newPriorityRows: Record<string, StatusRow[]> = {}
      for (const p of PRIORITIES) {
        newPriorityRows[p] = rows.map((r) => ({ ...r }))
      }
      setPriorityRows(newPriorityRows)
      setPerPriority(true)
    } else {
      // Turning OFF: check if priorities are all identical
      const allIdentical = PRIORITIES.every((p) =>
        priorityRows[p]?.every((row, i) =>
          row.duration === priorityRows.critical?.[i]?.duration &&
          row.calendarMode === priorityRows.critical?.[i]?.calendarMode
        )
      )
      if (allIdentical) {
        // Safe to collapse — use critical values
        setRows(priorityRows.critical?.map((r) => ({ ...r })) ?? [])
        setPerPriority(false)
      } else {
        // Show warning — use critical values but warn
        setRows(priorityRows.critical?.map((r) => ({ ...r })) ?? [])
        setOverwriteWarning(true)
        setPerPriority(false)
      }
    }
  }

  function togglePriorityExpanded(priority: string) {
    setExpandedPriorities((prev) => {
      const next = new Set(prev)
      if (next.has(priority)) {
        next.delete(priority)
      } else {
        next.add(priority)
      }
      return next
    })
  }

  function buildTargets(statusRows: StatusRow[], priority: string) {
    return statusRows
      .filter((r) => !isTerminal(r.category) && r.duration.trim())
      .map((r) => {
        const seconds = parseDuration(r.duration)
        if (seconds === null) {
          setError(t('sla.durationInvalid'))
          return null
        }
        return {
          status_name: r.name,
          priority,
          target_seconds: seconds,
          calendar_mode: r.calendarMode,
        }
      })
  }

  function handleSave() {
    setError('')

    let allTargets: ({ status_name: string; priority: string; target_seconds: number; calendar_mode: string } | null)[]

    if (perPriority) {
      // Each priority has its own rows
      allTargets = PRIORITIES.flatMap((p) => buildTargets(priorityRows[p] ?? [], p))
    } else {
      // Unified: duplicate across all 4 priorities
      allTargets = PRIORITIES.flatMap((p) => buildTargets(rows, p))
    }

    if (allTargets.some((t) => t === null)) return

    bulkUpsert.mutate(
      {
        work_item_type: workItemType,
        workflow_id: workflow.id,
        targets: allTargets as { status_name: string; priority: string; target_seconds: number; calendar_mode: string }[],
      },
      {
        onSuccess: () => {
          onSave?.()
          onClose()
        },
        onError: () => {
          setError(t('sla.saveError'))
        },
      }
    )
  }

  function renderStatusGrid(statusRows: StatusRow[], onUpdate: (index: number, field: keyof StatusRow, value: string) => void) {
    return (
      <div className="space-y-2">
        {/* Header */}
        <div className="grid grid-cols-[1fr_4.5rem_4.5rem] sm:grid-cols-[1fr_150px_150px] gap-2 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
          <span>{t('sla.status')}</span>
          <span>{t('sla.duration')}</span>
          <span>{t('sla.calendarMode')}</span>
        </div>

        {/* Rows */}
        {statusRows.map((row, index) => {
          const terminal = isTerminal(row.category)
          return (
            <Tooltip key={row.name} content={terminal ? t('sla.terminalStatusTooltip') : undefined} className="relative block">
              <div
                className={`grid grid-cols-[1fr_4.5rem_4.5rem] sm:grid-cols-[1fr_150px_150px] gap-2 items-center ${terminal ? 'opacity-50' : ''}`}
              >
                <ScrollableRow className="min-w-0 sm:hidden" gradientFrom="from-white dark:from-gray-800">
                  <span className="text-xs text-gray-900 dark:text-gray-100 whitespace-nowrap">{row.displayName}</span>
                </ScrollableRow>
                <span className="hidden sm:block text-xs text-gray-900 dark:text-gray-100">{row.displayName}</span>
                <input
                  type="text"
                  value={row.duration}
                  onChange={(e) => onUpdate(index, 'duration', e.target.value)}
                  placeholder={t('sla.durationPlaceholder')}
                  disabled={terminal || readOnly}
                  className="w-[4.5rem] sm:w-auto min-w-0 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                />
                <select
                  className="w-[4.5rem] sm:w-auto truncate rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  value={row.calendarMode}
                  onChange={(e) => onUpdate(index, 'calendarMode', e.target.value)}
                  disabled={terminal || readOnly}
                >
                  <option value="24x7">{t('sla.mode24x7')}</option>
                  <option value="business_hours" disabled={!hasBusinessHours}>
                    {t('sla.modeBusinessHours')}{!hasBusinessHours ? ` (${t('sla.requiresBusinessHours')})` : ''}
                  </option>
                </select>
              </div>
            </Tooltip>
          )
        })}
      </div>
    )
  }

  return (
    <Modal open={open} onClose={onClose} title={t('sla.titleForType', { type: t(`workitems.types.${workItemType}`) })} className="!max-w-2xl">
      <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
        {t('sla.description')} {t('sla.blankHint')}
      </p>

      {/* Per-priority toggle */}
      {!readOnly && (
        <label className="flex items-center gap-2 mb-4 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={perPriority}
            onChange={(e) => handleTogglePerPriority(e.target.checked)}
            className="rounded border-gray-300 dark:border-gray-600 text-indigo-600 focus:ring-indigo-500"
          />
          <span className="text-sm text-gray-700 dark:text-gray-300">{t('sla.setPerPriority')}</span>
        </label>
      )}

      {perPriority && !readOnly && (
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">{t('sla.perPriorityHint')}</p>
      )}

      {overwriteWarning && (
        <p className="text-xs text-amber-600 dark:text-amber-400 mb-3">{t('sla.overwriteWarning')}</p>
      )}

      {perPriority ? (
        <div className="space-y-3">
          {PRIORITIES.map((priority) => {
            const isExpanded = expandedPriorities.has(priority)
            return (
              <div key={priority} className="border border-gray-200 dark:border-gray-700 rounded-lg">
                <button
                  type="button"
                  className="flex items-center gap-2 w-full px-3 py-2 text-sm font-medium text-gray-900 dark:text-gray-100 hover:bg-gray-50 dark:hover:bg-gray-800 rounded-t-lg"
                  onClick={() => togglePriorityExpanded(priority)}
                >
                  {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                  {t(`workitems.priorities.${priority}`)}
                </button>
                {isExpanded && (
                  <div className="px-3 pb-3">
                    {renderStatusGrid(
                      priorityRows[priority] ?? [],
                      (index, field, value) => updatePriorityRow(priority, index, field, value)
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      ) : (
        renderStatusGrid(rows, updateRow)
      )}

      {error && <p className="mt-3 text-sm text-red-600 dark:text-red-400">{error}</p>}

      <div className="flex justify-end gap-2 mt-4">
        <Button variant="secondary" onClick={onClose}>
          {readOnly ? t('common.close') : t('common.cancel')}
        </Button>
        {!readOnly && (
          <Button onClick={handleSave} disabled={bulkUpsert.isPending}>
            {bulkUpsert.isPending ? t('common.saving') : t('common.save')}
          </Button>
        )}
      </div>
    </Modal>
  )
}

import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Tooltip } from '@/components/ui/Tooltip'
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

export function SLAConfigModal({ open, onClose, onSave, projectKey, workItemType, workflow, hasBusinessHours = false, readOnly = false }: Props) {
  const { t } = useTranslation()
  const { data: existingTargets } = useSLATargets(projectKey)
  const bulkUpsert = useBulkUpsertSLATargets(projectKey)
  const [rows, setRows] = useState<StatusRow[]>([])
  const [error, setError] = useState('')

  // Initialize rows from workflow statuses + existing targets
  useEffect(() => {
    if (!workflow?.statuses) return

    const targetsByStatus = new Map<string, { targetSeconds: number; calendarMode: string }>()
    if (existingTargets) {
      for (const t of existingTargets) {
        if (t.work_item_type === workItemType && t.workflow_id === workflow.id) {
          targetsByStatus.set(t.status_name, {
            targetSeconds: t.target_seconds,
            calendarMode: t.calendar_mode,
          })
        }
      }
    }

    setRows(
      workflow.statuses.map((s: WorkflowStatus) => {
        const existing = targetsByStatus.get(s.name)
        return {
          name: s.name,
          displayName: s.display_name,
          category: s.category,
          duration: existing ? formatDuration(existing.targetSeconds) : '',
          calendarMode: existing?.calendarMode ?? '24x7',
        }
      })
    )
  }, [workflow, existingTargets, workItemType])

  const isTerminal = (category: string) => category === 'done' || category === 'cancelled'

  function updateRow(index: number, field: keyof StatusRow, value: string) {
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: value } : r)))
  }

  function handleSave() {
    setError('')

    const targets = rows
      .filter((r) => !isTerminal(r.category) && r.duration.trim())
      .map((r) => {
        const seconds = parseDuration(r.duration)
        if (seconds === null) {
          setError(t('sla.durationInvalid'))
          return null
        }
        return {
          status_name: r.name,
          target_seconds: seconds,
          calendar_mode: r.calendarMode,
        }
      })

    if (targets.some((t) => t === null)) return

    bulkUpsert.mutate(
      {
        work_item_type: workItemType,
        workflow_id: workflow.id,
        targets: targets as { status_name: string; target_seconds: number; calendar_mode: string }[],
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

  return (
    <Modal open={open} onClose={onClose} title={t('sla.titleForType', { type: t(`workitems.types.${workItemType}`) })}>
      <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
        {t('sla.description')} {t('sla.blankHint')}
      </p>

      <div className="space-y-3">
        {/* Header */}
        <div className="grid grid-cols-[1fr_150px_130px] gap-2 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
          <span>{t('sla.status')}</span>
          <span>{t('sla.duration')}</span>
          <span>{t('sla.calendarMode')}</span>
        </div>

        {/* Rows */}
        {rows.map((row, index) => {
          const terminal = isTerminal(row.category)
          return (
            <Tooltip key={row.name} content={terminal ? t('sla.terminalStatusTooltip') : undefined} className="relative block">
              <div
                className={`grid grid-cols-[1fr_150px_130px] gap-2 items-center ${terminal ? 'opacity-50' : ''}`}
              >
                <span className="text-sm text-gray-900 dark:text-gray-100">{row.displayName}</span>
                <Input
                  value={row.duration}
                  onChange={(e) => updateRow(index, 'duration', e.target.value)}
                  placeholder={t('sla.durationPlaceholder')}
                  disabled={terminal || readOnly}
                  className="text-sm"
                />
                <select
                  className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  value={row.calendarMode}
                  onChange={(e) => updateRow(index, 'calendarMode', e.target.value)}
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

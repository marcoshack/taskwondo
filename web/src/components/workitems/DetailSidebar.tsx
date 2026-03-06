import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Select } from '@/components/ui/Select'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { UserPicker } from '@/components/ui/UserPicker'
import { Tooltip } from '@/components/ui/Tooltip'
import { Modal } from '@/components/ui/Modal'
import type { WorkItem, UpdateWorkItemInput } from '@/api/workitems'
import { useTimeEntries } from '@/hooks/useWorkItems'
import type { WorkflowStatus, Workflow } from '@/api/workflows'
import type { ProjectMember, ProjectTypeWorkflow } from '@/api/projects'
import type { Milestone } from '@/api/milestones'
import { Megaphone, CalendarPlus, History, CheckCircle, Info, Lock, Unlock, Globe, AlertCircle } from 'lucide-react'
import { formatRelativeTime } from '@/utils/duration'

interface DetailSidebarProps {
  item: WorkItem
  projectKey: string
  itemNumber: number
  statuses: WorkflowStatus[]
  allowedTransitions: string[]
  members: ProjectMember[]
  milestones?: Milestone[]
  allowedComplexityValues?: number[]
  typeWorkflows?: ProjectTypeWorkflow[]
  allWorkflows?: Workflow[]
  onUpdate: (input: UpdateWorkItemInput) => void
  onDelete?: () => void
  readOnly?: boolean
  updateError?: boolean
}

const PRIORITIES = ['low', 'medium', 'high', 'critical']
const TYPES = ['task', 'ticket', 'bug', 'feedback', 'epic']
const VISIBILITIES = ['internal', 'portal', 'public'] as const

const MAX_COMPLEXITY = 1000000

function formatDuration(totalSeconds: number): string {
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  if (h === 0) return `${m}m`
  if (m === 0) return `${h}h`
  return `${h}h ${m}m`
}

function parseDurationString(input: string): number | null {
  const regex = /^(?:(\d+)h)?\s*(?:(\d+)m)?$/i
  const match = input.match(regex)
  if (!match) {
    // Try plain number as hours
    const num = parseFloat(input)
    if (!isNaN(num) && num > 0) return Math.round(num * 3600)
    return null
  }
  const h = match[1] ? parseInt(match[1], 10) : 0
  const m = match[2] ? parseInt(match[2], 10) : 0
  if (h === 0 && m === 0) return null
  return h * 3600 + m * 60
}

export function DetailSidebar({ item, projectKey, itemNumber, statuses, allowedTransitions, members, milestones = [], allowedComplexityValues = [], typeWorkflows, allWorkflows, onUpdate, onDelete, readOnly = false, updateError = false }: DetailSidebarProps) {
  const { t } = useTranslation()
  const [pendingType, setPendingType] = useState<string | null>(null)
  const [statusWarning, setStatusWarning] = useState(false)
  const [complexityError, setComplexityError] = useState<string | undefined>(undefined)
  const [showVisibilityInfo, setShowVisibilityInfo] = useState(false)
  const [estimateError, setEstimateError] = useState<string | undefined>(undefined)
  const { data: timeData } = useTimeEntries(projectKey, itemNumber)

  // Resolve statuses to show: either from pending type's workflow or current workflow
  let displayStatuses = statuses
  if (pendingType && typeWorkflows && allWorkflows) {
    const mapping = typeWorkflows.find((tw) => tw.work_item_type === pendingType)
    if (mapping) {
      const wf = allWorkflows.find((w) => w.id === mapping.workflow_id)
      if (wf) displayStatuses = wf.statuses
    }
  }

  const currentWs = displayStatuses.find((s) => s.name === item.status)

  function handleTypeChange(newType: string) {
    if (newType === item.type) return

    // Check if current status exists in the new type's workflow
    if (typeWorkflows && allWorkflows) {
      const mapping = typeWorkflows.find((tw) => tw.work_item_type === newType)
      if (mapping) {
        const wf = allWorkflows.find((w) => w.id === mapping.workflow_id)
        if (wf) {
          const statusExists = wf.statuses.some((s) => s.name === item.status)
          if (!statusExists) {
            setPendingType(newType)
            setStatusWarning(true)
            return
          }
        }
      }
    }

    // Status is compatible — submit immediately
    onUpdate({ type: newType })
  }

  function handleStatusChange(newStatus: string) {
    if (!newStatus) return
    if (pendingType) {
      // Submit both type and status together
      onUpdate({ type: pendingType, status: newStatus })
      setPendingType(null)
      setStatusWarning(false)
    } else {
      onUpdate({ status: newStatus })
    }
  }

  // Build status options
  const statusOptions = pendingType
    ? [...displayStatuses].sort((a, b) => a.position - b.position)
    : [
        // Current status first, then allowed transitions
        ...(currentWs ? [currentWs] : []),
        ...displayStatuses.filter((s) => allowedTransitions.includes(s.name) && s.name !== item.status).sort((a, b) => a.position - b.position),
      ]

  return (
    <div className="space-y-4">
      {updateError && (
        <p className="text-xs text-red-600 dark:text-red-400">{t('workitems.detail.updateError')}</p>
      )}

      <div className="space-y-2 pb-4 border-b border-gray-100 dark:border-gray-700">
        <div className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400">
          <Tooltip content={t('workitems.detail.reporter')}>
            <Megaphone className="h-4 w-4 shrink-0" />
          </Tooltip>
          <span className="truncate">{item.reporter_name}</span>
          {!members.some((m) => m.user_id === item.reporter_id) && (
            <Tooltip content={t('workitems.detail.reporterNotMember')}>
              <AlertCircle className="h-3.5 w-3.5 shrink-0 text-amber-500" />
            </Tooltip>
          )}
        </div>
        <div className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400">
          <Tooltip content={t('workitems.detail.created')}>
            <CalendarPlus className="h-4 w-4 shrink-0" />
          </Tooltip>
          <span>{formatRelativeTime(item.created_at)}</span>
        </div>
        <div className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400">
          <Tooltip content={t('workitems.detail.updated')}>
            <History className="h-4 w-4 shrink-0" />
          </Tooltip>
          <span>{formatRelativeTime(item.updated_at)}</span>
        </div>
        {item.resolved_at && (
          <div className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400">
            <Tooltip content={t('workitems.detail.resolved')}>
              <CheckCircle className="h-4 w-4 shrink-0" />
            </Tooltip>
            <span>{formatRelativeTime(item.resolved_at)}</span>
          </div>
        )}
      </div>

      <div className="space-y-2 pb-4 border-b border-gray-100 dark:border-gray-700">
        <Field label={t('timeTracking.estimate')}>
          <Input
            type="text"
            defaultValue={item.estimated_seconds != null ? formatDuration(item.estimated_seconds) : ''}
            placeholder={t('timeTracking.estimatePlaceholder')}
            error={estimateError}
            disabled={readOnly}
            onKeyDown={(e) => {
              if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
              if (e.key === 'Escape') {
                (e.target as HTMLInputElement).value = item.estimated_seconds != null ? formatDuration(item.estimated_seconds) : ''
                setEstimateError(undefined)
                ;(e.target as HTMLInputElement).blur()
              }
            }}
            onBlur={(e) => {
              const raw = e.target.value.trim()
              if (!raw) {
                setEstimateError(undefined)
                if (item.estimated_seconds != null) onUpdate({ estimated_seconds: null })
                return
              }
              const seconds = parseDurationString(raw)
              if (seconds == null) {
                setEstimateError(t('timeTracking.estimateInvalid'))
                return
              }
              setEstimateError(undefined)
              if (seconds !== item.estimated_seconds) {
                onUpdate({ estimated_seconds: seconds })
              }
            }}
          />
        </Field>
        <div className="text-xs text-gray-500 dark:text-gray-400 space-y-1">
          <div className="flex justify-between">
            <span>{t('timeTracking.logged')}</span>
            <span className="font-medium text-gray-700 dark:text-gray-300">
              {formatDuration(timeData?.total_logged_seconds ?? 0)}
            </span>
          </div>
          {item.estimated_seconds != null && item.estimated_seconds > 0 && (() => {
            const logged = timeData?.total_logged_seconds ?? 0
            const pct = Math.min((logged / item.estimated_seconds) * 100, 100)
            const over = logged > item.estimated_seconds
            return (
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                <div
                  className={`h-1.5 rounded-full transition-all ${over ? 'bg-red-500' : 'bg-indigo-500'}`}
                  style={{ width: `${pct}%` }}
                />
              </div>
            )
          })()}
        </div>
      </div>

      <Field label={t('workitems.form.type')}>
        <Select value={pendingType ?? item.type} onChange={(e) => handleTypeChange(e.target.value)} disabled={readOnly}>
          {TYPES.map((tp) => <option key={tp} value={tp}>{t(`workitems.types.${tp}`)}</option>)}
        </Select>
      </Field>

      <Field label={t('workitems.form.status')}>
        <Select
          value={pendingType ? '' : item.status}
          onChange={(e) => handleStatusChange(e.target.value)}
          className={statusWarning ? 'ring-2 ring-red-500 border-red-500' : ''}
          disabled={readOnly}
        >
          {pendingType && <option value="">{t('workitems.detail.selectStatus')}</option>}
          {statusOptions.map((ws) => (
            <option key={ws.name} value={ws.name}>
              {t(`workitems.statuses.${ws.name}`, { defaultValue: ws.display_name ?? ws.name })}
            </option>
          ))}
        </Select>
        {statusWarning && (
          <p className="text-xs text-red-500 mt-1">{t('workitems.detail.statusIncompatible')}</p>
        )}
      </Field>

      <Field label={t('workitems.form.priority')}>
        <Select value={item.priority} onChange={(e) => onUpdate({ priority: e.target.value })} disabled={readOnly}>
          {PRIORITIES.map((p) => <option key={p} value={p}>{t(`workitems.priorities.${p}`)}</option>)}
        </Select>
      </Field>

      <Field label={t('workitems.form.complexity')}>
        {allowedComplexityValues.length > 0 ? (
          <Select
            value={item.complexity != null ? String(item.complexity) : ''}
            onChange={(e) => onUpdate({ complexity: e.target.value ? Number(e.target.value) : null })}
            disabled={readOnly}
          >
            <option value="">{t('workitems.form.complexityPlaceholder')}</option>
            {allowedComplexityValues.map((v) => (
              <option key={v} value={String(v)}>{v}</option>
            ))}
          </Select>
        ) : (
          <Input
            type="number"
            min="1"
            defaultValue={item.complexity != null ? String(item.complexity) : ''}
            placeholder={t('workitems.form.complexityPlaceholder')}
            error={complexityError}
            disabled={readOnly}
            onKeyDown={(e) => {
              if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
              if (e.key === 'Escape') {
                (e.target as HTMLInputElement).value = item.complexity != null ? String(item.complexity) : ''
                setComplexityError(undefined)
                ;(e.target as HTMLInputElement).blur()
              }
            }}
            onBlur={(e) => {
              const raw = e.target.value
              if (!raw) {
                setComplexityError(undefined)
                if (item.complexity != null) onUpdate({ complexity: null })
                return
              }
              const num = Number(raw)
              if (!Number.isInteger(num) || num <= 0) {
                setComplexityError(t('workitems.form.complexityMustBePositive'))
                return
              }
              if (num > MAX_COMPLEXITY) {
                setComplexityError(t('workitems.form.complexityTooLarge'))
                return
              }
              if (allowedComplexityValues.length > 0 && !allowedComplexityValues.includes(num)) {
                setComplexityError(t('workitems.form.complexityNotAllowed', { values: allowedComplexityValues.join(', ') }))
                return
              }
              setComplexityError(undefined)
              if (num !== item.complexity) {
                onUpdate({ complexity: num })
              }
            }}
          />
        )}
      </Field>

      <Field label={t('workitems.form.assignee')}>
        <UserPicker
          members={members}
          value={item.assignee_id}
          onChange={(userId) => onUpdate({ assignee_id: userId })}
          disabled={readOnly}
        />
      </Field>

      <Field label={t('workitems.form.milestone')}>
        <Select
          value={item.milestone_id ?? ''}
          onChange={(e) => onUpdate({ milestone_id: e.target.value || null })}
          disabled={readOnly}
        >
          <option value="">{t('milestones.noMilestone')}</option>
          {milestones
            .filter((m) => m.status === 'open' || m.id === item.milestone_id)
            .map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
        </Select>
      </Field>

      <div>
        <div className="flex items-center gap-1 mb-1">
          <label className="block text-xs font-medium text-gray-500 dark:text-gray-400">{t('workitems.form.visibility')}</label>
          <button type="button" onClick={() => setShowVisibilityInfo(true)} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-500 dark:hover:text-gray-300">
            <Info className="h-3.5 w-3.5" />
          </button>
        </div>
        <VisibilityPicker value={item.visibility} onChange={(v) => onUpdate({ visibility: v })} disabled={readOnly} />
      </div>

      <Field label={t('workitems.form.dueDate')}>
        <Input
          type="date"
          value={item.due_date ?? ''}
          onChange={(e) => onUpdate({ due_date: e.target.value || null })}
          disabled={readOnly}
        />
      </Field>

      <Field label={t('workitems.form.labels')}>
        <Input
          defaultValue={item.labels.join(', ')}
          placeholder={t('workitems.form.labelsPlaceholder')}
          disabled={readOnly}
          onKeyDown={(e) => {
            if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
            if (e.key === 'Escape') {
              (e.target as HTMLInputElement).value = item.labels.join(', ')
              ;(e.target as HTMLInputElement).blur()
            }
          }}
          onBlur={(e) => {
            const newLabels = e.target.value ? e.target.value.split(',').map((l) => l.trim()).filter(Boolean) : []
            if (JSON.stringify(newLabels) !== JSON.stringify(item.labels)) {
              onUpdate({ labels: newLabels })
            }
          }}
        />
      </Field>

      {!readOnly && onDelete && (
        <div className="border-t border-gray-100 dark:border-gray-700 pt-4">
          <Button variant="danger" size="sm" onClick={onDelete}>{t('workitems.detail.deleteItem')}</Button>
        </div>
      )}

      <Modal open={showVisibilityInfo} onClose={() => setShowVisibilityInfo(false)} title={t('workitems.form.visibility')}>
        <ul className="space-y-3 text-sm text-gray-700 dark:text-gray-300">
          <li>
            <span className="inline-flex items-center gap-1 font-medium text-gray-500 dark:text-gray-400"><Lock className="h-3.5 w-3.5" />{t('workitems.visibilities.internal')}</span>
            <span className="text-gray-400 dark:text-gray-500"> — </span>
            {t('workitems.visibilities.internal.description')}
          </li>
          <li>
            <span className="inline-flex items-center gap-1 font-medium text-yellow-500 dark:text-yellow-400"><Unlock className="h-3.5 w-3.5" />{t('workitems.visibilities.portal')}</span>
            <span className="text-gray-400 dark:text-gray-500"> — </span>
            {t('workitems.visibilities.portal.description')}
          </li>
          <li>
            <span className="inline-flex items-center gap-1 font-medium text-red-500 dark:text-red-400"><Globe className="h-3.5 w-3.5" />{t('workitems.visibilities.public')}</span>
            <span className="text-gray-400 dark:text-gray-500"> — </span>
            {t('workitems.visibilities.public.description')}
          </li>
        </ul>
      </Modal>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{label}</label>
      {children}
    </div>
  )
}

const visibilityConfig = {
  internal: { icon: Lock, color: 'text-gray-500 dark:text-gray-400' },
  portal: { icon: Unlock, color: 'text-yellow-500 dark:text-yellow-400' },
  public: { icon: Globe, color: 'text-red-500 dark:text-red-400' },
} as const

function VisibilityPicker({ value, onChange, disabled }: { value: string; onChange: (v: string) => void; disabled?: boolean }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const cfg = visibilityConfig[value as keyof typeof visibilityConfig] ?? visibilityConfig.internal
  const Icon = cfg.icon

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        disabled={disabled}
        onClick={() => !disabled && setOpen(!open)}
        className={`flex items-center gap-1.5 w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm shadow-sm bg-white dark:bg-gray-800 ${disabled ? 'opacity-50 cursor-not-allowed' : 'focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500'}`}
      >
        <Icon className={`h-3.5 w-3.5 ${cfg.color}`} />
        <span className={cfg.color}>{t(`workitems.visibilities.${value}`)}</span>
      </button>
      {open && (
        <div className="absolute z-20 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg">
          {VISIBILITIES.map((v) => {
            const c = visibilityConfig[v]
            const VIcon = c.icon
            return (
              <button
                key={v}
                type="button"
                className={`flex items-center gap-1.5 w-full px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 ${v === value ? 'bg-indigo-50 dark:bg-indigo-900/30' : ''}`}
                onClick={() => { onChange(v); setOpen(false) }}
              >
                <VIcon className={`h-3.5 w-3.5 ${c.color}`} />
                <span className={c.color}>{t(`workitems.visibilities.${v}`)}</span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Select } from '@/components/ui/Select'
import { Input } from '@/components/ui/Input'
import { UserPicker } from '@/components/ui/UserPicker'
import type { WorkItem, UpdateWorkItemInput } from '@/api/workitems'
import type { WorkflowStatus, Workflow } from '@/api/workflows'
import type { ProjectMember, ProjectTypeWorkflow } from '@/api/projects'

interface DetailSidebarProps {
  item: WorkItem
  statuses: WorkflowStatus[]
  allowedTransitions: string[]
  members: ProjectMember[]
  allowedComplexityValues?: number[]
  typeWorkflows?: ProjectTypeWorkflow[]
  allWorkflows?: Workflow[]
  onUpdate: (input: UpdateWorkItemInput) => void
}

const PRIORITIES = ['low', 'medium', 'high', 'critical']
const TYPES = ['task', 'ticket', 'bug', 'feedback', 'epic']
const VISIBILITIES = ['internal', 'portal', 'public']

export function DetailSidebar({ item, statuses, allowedTransitions, members, allowedComplexityValues = [], typeWorkflows, allWorkflows, onUpdate }: DetailSidebarProps) {
  const { t } = useTranslation()
  const [pendingType, setPendingType] = useState<string | null>(null)
  const [statusWarning, setStatusWarning] = useState(false)

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
        ...displayStatuses.filter((s) => allowedTransitions.includes(s.name) && s.name !== item.status),
      ]

  return (
    <div className="space-y-4">
      <Field label={t('workitems.form.type')}>
        <Select value={pendingType ?? item.type} onChange={(e) => handleTypeChange(e.target.value)}>
          {TYPES.map((tp) => <option key={tp} value={tp}>{t(`workitems.types.${tp}`)}</option>)}
        </Select>
      </Field>

      <Field label={t('workitems.form.status')}>
        <Select
          value={pendingType ? '' : item.status}
          onChange={(e) => handleStatusChange(e.target.value)}
          className={statusWarning ? 'ring-2 ring-red-500 border-red-500' : ''}
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
        <Select value={item.priority} onChange={(e) => onUpdate({ priority: e.target.value })}>
          {PRIORITIES.map((p) => <option key={p} value={p}>{t(`workitems.priorities.${p}`)}</option>)}
        </Select>
      </Field>

      <Field label={t('workitems.form.complexity')}>
        {allowedComplexityValues.length > 0 ? (
          <Select
            value={item.complexity != null ? String(item.complexity) : ''}
            onChange={(e) => onUpdate({ complexity: e.target.value ? Number(e.target.value) : null })}
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
            onKeyDown={(e) => {
              if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
              if (e.key === 'Escape') {
                (e.target as HTMLInputElement).value = item.complexity != null ? String(item.complexity) : ''
                ;(e.target as HTMLInputElement).blur()
              }
            }}
            onBlur={(e) => {
              const newVal = e.target.value ? Number(e.target.value) : null
              if (newVal !== item.complexity) {
                onUpdate({ complexity: newVal })
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
        />
      </Field>

      <Field label={t('workitems.form.visibility')}>
        <Select value={item.visibility} onChange={(e) => onUpdate({ visibility: e.target.value })}>
          {VISIBILITIES.map((v) => <option key={v} value={v}>{t(`workitems.visibilities.${v}`)}</option>)}
        </Select>
      </Field>

      <Field label={t('workitems.form.dueDate')}>
        <Input
          type="date"
          value={item.due_date ?? ''}
          onChange={(e) => onUpdate({ due_date: e.target.value || null })}
        />
      </Field>

      <Field label={t('workitems.form.labels')}>
        <Input
          defaultValue={item.labels.join(', ')}
          placeholder={t('workitems.form.labelsPlaceholder')}
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

      <div className="border-t border-gray-100 dark:border-gray-700 pt-4 space-y-2 text-xs text-gray-400 dark:text-gray-500">
        <div>{t('workitems.detail.reporter')}: {members.find((m) => m.user_id === item.reporter_id)?.display_name ?? item.reporter_id}</div>
        <div>{t('workitems.detail.created')}: {new Date(item.created_at).toLocaleString()}</div>
        <div>{t('workitems.detail.updated')}: {new Date(item.updated_at).toLocaleString()}</div>
        {item.resolved_at && <div>{t('workitems.detail.resolved')}: {new Date(item.resolved_at).toLocaleString()}</div>}
      </div>
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

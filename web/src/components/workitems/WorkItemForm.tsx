import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { UserPicker } from '@/components/ui/UserPicker'
import type { WorkflowStatus } from '@/api/workflows'
import type { ProjectMember } from '@/api/projects'

const TYPES = ['task', 'ticket', 'bug', 'feedback', 'epic']
const PRIORITIES = ['low', 'medium', 'high', 'critical']
const VISIBILITIES = ['internal', 'portal', 'public']

interface WorkItemFormProps {
  mode: 'create' | 'edit'
  members: ProjectMember[]
  initialValues?: {
    type?: string
    title?: string
    description?: string
    priority?: string
    assignee_id?: string
    labels?: string[]
    visibility?: string
    due_date?: string
    status?: string
  }
  statuses?: WorkflowStatus[]
  allowedTransitions?: string[]
  onSubmit: (values: Record<string, unknown>) => void
  onCancel: () => void
  isSubmitting: boolean
}

export function WorkItemForm({
  mode,
  members,
  initialValues = {},
  statuses,
  allowedTransitions,
  onSubmit,
  onCancel,
  isSubmitting,
}: WorkItemFormProps) {
  const { t } = useTranslation()
  const [type, setType] = useState(initialValues.type ?? 'task')
  const [title, setTitle] = useState(initialValues.title ?? '')
  const [description, setDescription] = useState(initialValues.description ?? '')
  const [priority, setPriority] = useState(initialValues.priority ?? 'medium')
  const [assigneeId, setAssigneeId] = useState<string | null>(initialValues.assignee_id ?? null)
  const [labels, setLabels] = useState(initialValues.labels?.join(', ') ?? '')
  const [visibility, setVisibility] = useState(initialValues.visibility ?? 'internal')
  const [dueDate, setDueDate] = useState(initialValues.due_date ?? '')
  const [status, setStatus] = useState(initialValues.status ?? '')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (mode === 'create') {
      onSubmit({
        type,
        title,
        description: description || undefined,
        priority,
        assignee_id: assigneeId || undefined,
        labels: labels ? labels.split(',').map((l) => l.trim()).filter(Boolean) : undefined,
        visibility,
        due_date: dueDate || undefined,
      })
    } else {
      const values: Record<string, unknown> = {}
      if (title !== initialValues.title) values.title = title
      if (description !== (initialValues.description ?? '')) values.description = description || null
      if (priority !== initialValues.priority) values.priority = priority
      if (visibility !== initialValues.visibility) values.visibility = visibility
      if (dueDate !== (initialValues.due_date ?? '')) values.due_date = dueDate || null
      if (status && status !== initialValues.status) values.status = status
      const newLabels = labels ? labels.split(',').map((l) => l.trim()).filter(Boolean) : []
      if (JSON.stringify(newLabels) !== JSON.stringify(initialValues.labels ?? [])) values.labels = newLabels
      if (assigneeId !== (initialValues.assignee_id ?? null)) values.assignee_id = assigneeId
      onSubmit(values)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {mode === 'create' && (
        <Select label={t('workitems.form.type')} value={type} onChange={(e) => setType(e.target.value)}>
          {TYPES.map((tp) => (
            <option key={tp} value={tp}>{t(`workitems.types.${tp}`)}</option>
          ))}
        </Select>
      )}
      <Input label={t('workitems.form.title')} value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus />
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.form.description')}</label>
        <textarea
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          rows={4}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <Select label={t('workitems.form.priority')} value={priority} onChange={(e) => setPriority(e.target.value)}>
        {PRIORITIES.map((p) => (
          <option key={p} value={p}>{t(`workitems.priorities.${p}`)}</option>
        ))}
      </Select>
      {mode === 'edit' && statuses && allowedTransitions && (
        <Select label={t('workitems.form.status')} value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value={initialValues.status}>{t(`workitems.statuses.${initialValues.status}`, { defaultValue: statuses.find((s) => s.name === initialValues.status)?.display_name ?? initialValues.status })}</option>
          {allowedTransitions
            .filter((tr) => tr !== initialValues.status)
            .map((tr) => {
              const ws = statuses.find((s) => s.name === tr)
              return <option key={tr} value={tr}>{t(`workitems.statuses.${tr}`, { defaultValue: ws?.display_name ?? tr })}</option>
            })}
        </Select>
      )}
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.form.assignee')}</label>
        <UserPicker members={members} value={assigneeId} onChange={setAssigneeId} />
      </div>
      <Input label={t('workitems.form.labels')} value={labels} onChange={(e) => setLabels(e.target.value)} placeholder={t('workitems.form.labelsPlaceholder')} />
      <Select label={t('workitems.form.visibility')} value={visibility} onChange={(e) => setVisibility(e.target.value)}>
        {VISIBILITIES.map((v) => (
          <option key={v} value={v}>{t(`workitems.visibilities.${v}`)}</option>
        ))}
      </Select>
      <Input label={t('workitems.form.dueDate')} type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={isSubmitting || !title.trim()}>
          {isSubmitting ? t('common.saving') : mode === 'create' ? t('common.create') : t('common.save')}
        </Button>
      </div>
    </form>
  )
}

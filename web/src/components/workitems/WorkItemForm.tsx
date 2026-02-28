import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Tooltip } from '@/components/ui/Tooltip'
import { UserPicker } from '@/components/ui/UserPicker'
import { ProjectPicker } from '@/components/ui/ProjectPicker'
import { MentionModal } from '@/components/ui/MentionModal'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import type { WorkflowStatus } from '@/api/workflows'
import type { ProjectMember, Project } from '@/api/projects'
import type { Milestone } from '@/api/milestones'

const TYPES = ['task', 'ticket', 'bug', 'feedback', 'epic']
const PRIORITIES = ['low', 'medium', 'high', 'critical']
const VISIBILITIES = ['internal', 'portal', 'public']

interface WorkItemFormProps {
  projectKey: string
  mode: 'create' | 'edit'
  members: ProjectMember[]
  milestones?: Milestone[]
  allowedComplexityValues?: number[]
  projects?: Project[]
  projectLocked?: boolean
  onProjectChange?: (projectKey: string) => void
  initialValues?: {
    type?: string
    title?: string
    description?: string
    priority?: string
    assignee_id?: string
    labels?: string[]
    complexity?: number | null
    visibility?: string
    due_date?: string
    status?: string
    milestone_id?: string | null
  }
  statuses?: WorkflowStatus[]
  allowedTransitions?: string[]
  onSubmit: (values: Record<string, unknown>) => void
  onCancel: () => void
  isSubmitting: boolean
  submitError?: string | null
}

export function WorkItemForm({
  projectKey,
  mode,
  members,
  milestones = [],
  allowedComplexityValues = [],
  projects,
  projectLocked,
  onProjectChange,
  initialValues = {},
  statuses,
  allowedTransitions,
  onSubmit,
  onCancel,
  isSubmitting,
  submitError,
}: WorkItemFormProps) {
  const { t } = useTranslation()
  const [type, setType] = useState(initialValues.type ?? '')
  const projectSelected = !!projectKey
  const typeSelected = mode === 'edit' || type !== ''
  const disabledUntilProject = mode === 'create' && !!projects && !projectSelected
  const disabledUntilType = mode === 'create' && !typeSelected
  const disabledTooltip = disabledUntilProject
    ? t('workitems.form.projectRequiredTooltip')
    : disabledUntilType ? t('workitems.form.typeRequiredTooltip') : undefined
  const [title, setTitle] = useState(initialValues.title ?? '')
  const [description, setDescription] = useState(initialValues.description ?? '')
  const [priority, setPriority] = useState(initialValues.priority ?? 'medium')
  const [assigneeId, setAssigneeId] = useState<string | null>(initialValues.assignee_id ?? null)
  const [labels, setLabels] = useState(initialValues.labels?.join(', ') ?? '')
  const [visibility, setVisibility] = useState(initialValues.visibility ?? 'internal')
  const [complexity, setComplexity] = useState(initialValues.complexity != null ? String(initialValues.complexity) : '')
  const [dueDate, setDueDate] = useState(initialValues.due_date ?? '')
  const [milestoneId, setMilestoneId] = useState(initialValues.milestone_id ?? '')
  const [status, setStatus] = useState(initialValues.status ?? '')
  const [watcherIds, setWatcherIds] = useState<string[]>([])

  const descRef = useRef<HTMLTextAreaElement>(null)
  const descMention = useMentionAutocomplete({
    value: description,
    onValueChange: setDescription,
    textareaRef: descRef,
  })

  const MAX_COMPLEXITY = 1000000

  function validateComplexity(value: string): string | undefined {
    if (!value) return undefined
    const num = Number(value)
    if (!Number.isInteger(num) || num <= 0) {
      return t('workitems.form.complexityMustBePositive')
    }
    if (num > MAX_COMPLEXITY) {
      return t('workitems.form.complexityTooLarge')
    }
    if (allowedComplexityValues.length > 0 && !allowedComplexityValues.includes(num)) {
      return t('workitems.form.complexityNotAllowed', { values: allowedComplexityValues.join(', ') })
    }
    return undefined
  }

  const complexityError = validateComplexity(complexity)
  const hasValidationErrors = !!complexityError

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
        complexity: complexity ? Number(complexity) : undefined,
        milestone_id: milestoneId || undefined,
        visibility,
        due_date: dueDate || undefined,
        watcher_ids: watcherIds.length > 0 ? watcherIds : undefined,
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
      const oldComplexity = initialValues.complexity != null ? String(initialValues.complexity) : ''
      if (complexity !== oldComplexity) values.complexity = complexity ? Number(complexity) : null
      const oldMilestoneId = initialValues.milestone_id ?? ''
      if (milestoneId !== oldMilestoneId) values.milestone_id = milestoneId || null
      onSubmit(values)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {mode === 'create' && projects && (
        <ProjectPicker
          projects={projects}
          value={projectKey}
          onChange={(key) => onProjectChange?.(key)}
          disabled={projectLocked}
        />
      )}
      {mode === 'create' && (
        <Tooltip content={disabledUntilProject ? t('workitems.form.projectRequiredTooltip') : undefined} className="relative block">
          <Select label={t('workitems.form.type')} value={type} onChange={(e) => setType(e.target.value)} required disabled={disabledUntilProject}>
            {!typeSelected && <option value="">{t('workitems.form.typePlaceholder')}</option>}
            {TYPES.map((tp) => (
              <option key={tp} value={tp}>{t(`workitems.types.${tp}`)}</option>
            ))}
          </Select>
        </Tooltip>
      )}
      <Tooltip content={disabledTooltip} className="relative block">
        <Input label={t('workitems.form.title')} value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus={typeSelected} disabled={disabledUntilType} />
      </Tooltip>
      <Tooltip content={disabledTooltip} className="relative block">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.form.description')}</label>
          <textarea
            ref={descRef}
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
            rows={4}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            onKeyDown={(e) => {
              descMention.onMentionKeyDown(e)
            }}
            disabled={disabledUntilType}
          />
          <MentionModal
            open={descMention.mentionModalOpen}
            position={descMention.dropdownPosition}
            onClose={descMention.onMentionClose}
            onSelect={descMention.onMentionSelect}
            projectKey={projectKey}
          />
        </div>
      </Tooltip>
      <Tooltip content={disabledTooltip} className="relative block">
        <Select label={t('workitems.form.priority')} value={priority} onChange={(e) => setPriority(e.target.value)} disabled={disabledUntilType}>
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>{t(`workitems.priorities.${p}`)}</option>
          ))}
        </Select>
      </Tooltip>
      {mode === 'edit' && statuses && allowedTransitions && (
        <Select label={t('workitems.form.status')} value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value={initialValues.status}>{t(`workitems.statuses.${initialValues.status}`, { defaultValue: statuses.find((s) => s.name === initialValues.status)?.display_name ?? initialValues.status })}</option>
          {allowedTransitions
            .filter((tr) => tr !== initialValues.status)
            .sort((a, b) => {
              const posA = statuses.find((s) => s.name === a)?.position ?? 0
              const posB = statuses.find((s) => s.name === b)?.position ?? 0
              return posA - posB
            })
            .map((tr) => {
              const ws = statuses.find((s) => s.name === tr)
              return <option key={tr} value={tr}>{t(`workitems.statuses.${tr}`, { defaultValue: ws?.display_name ?? tr })}</option>
            })}
        </Select>
      )}
      <Tooltip content={disabledTooltip} className="relative block">
        {allowedComplexityValues.length > 0 ? (
          <Select label={t('workitems.form.complexity')} value={complexity} onChange={(e) => setComplexity(e.target.value)} error={complexityError} disabled={disabledUntilType}>
            <option value="">{t('workitems.form.complexityPlaceholder')}</option>
            {allowedComplexityValues.map((v) => (
              <option key={v} value={String(v)}>{v}</option>
            ))}
          </Select>
        ) : (
          <Input label={t('workitems.form.complexity')} type="number" min="1" value={complexity} onChange={(e) => setComplexity(e.target.value)} placeholder={t('workitems.form.complexityPlaceholder')} error={complexityError} disabled={disabledUntilType} />
        )}
      </Tooltip>
      <Tooltip content={disabledTooltip} className="relative block">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.form.assignee')}</label>
          <UserPicker members={members} value={assigneeId} onChange={setAssigneeId} disabled={disabledUntilType} />
        </div>
      </Tooltip>
      {mode === 'create' && (
        <Tooltip content={disabledTooltip} className="relative block">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('watchers.watchersField')}</label>
            <MultiUserPicker
              members={members}
              selectedIds={watcherIds}
              onChange={setWatcherIds}
              disabled={disabledUntilType}
            />
          </div>
        </Tooltip>
      )}
      <Tooltip content={disabledTooltip} className="relative block">
        <Input label={t('workitems.form.labels')} value={labels} onChange={(e) => setLabels(e.target.value)} placeholder={t('workitems.form.labelsPlaceholder')} disabled={disabledUntilType} />
      </Tooltip>
      <Tooltip content={disabledTooltip} className="relative block">
        <Select label={t('workitems.form.visibility')} value={visibility} onChange={(e) => setVisibility(e.target.value)} disabled={disabledUntilType}>
          {VISIBILITIES.map((v) => (
            <option key={v} value={v}>{t(`workitems.visibilities.${v}`)}</option>
          ))}
        </Select>
      </Tooltip>
      <Tooltip content={disabledTooltip} className="relative block">
        <Input label={t('workitems.form.dueDate')} type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} disabled={disabledUntilType} />
      </Tooltip>
      {milestones.length > 0 && (
        <Tooltip content={disabledTooltip} className="relative block">
          <Select label={t('workitems.form.milestone')} value={milestoneId} onChange={(e) => setMilestoneId(e.target.value)} disabled={disabledUntilType}>
            <option value="">{t('milestones.noMilestone')}</option>
            {milestones.filter((m) => m.status === 'open').map((m) => (
              <option key={m.id} value={m.id}>{m.name}</option>
            ))}
          </Select>
        </Tooltip>
      )}
      {submitError && (
        <p className="text-sm text-red-600 dark:text-red-400">{submitError}</p>
      )}
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={isSubmitting || !title.trim() || hasValidationErrors || disabledUntilType || disabledUntilProject}>
          {isSubmitting ? t('common.saving') : mode === 'create' ? t('common.create') : t('common.save')}
        </Button>
      </div>
    </form>
  )
}

// --- Multi-user picker for watchers (includes viewers) ---

interface MultiUserPickerProps {
  members: ProjectMember[]
  selectedIds: string[]
  onChange: (ids: string[]) => void
  disabled?: boolean
}

function MultiUserPicker({ members, selectedIds, onChange, disabled }: MultiUserPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const available = members.filter((m) => !selectedIds.includes(m.user_id))
  const filtered = available.filter((m) => {
    if (!search) return true
    const q = search.toLowerCase()
    return m.display_name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q)
  })

  const selectedMembers = selectedIds
    .map((id) => members.find((m) => m.user_id === id))
    .filter(Boolean) as ProjectMember[]

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <div ref={ref} className="relative">
      <div
        className={`flex flex-wrap gap-1.5 min-h-[38px] rounded-md border border-gray-300 dark:border-gray-600 px-2 py-1.5 text-sm bg-white dark:bg-gray-800 ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-text'}`}
        onClick={() => { if (disabled) return; setOpen(true); setTimeout(() => inputRef.current?.focus(), 0) }}
      >
        {selectedMembers.map((m) => (
          <span key={m.user_id} className="inline-flex items-center gap-1 rounded-full bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300 px-2 py-0.5 text-xs font-medium">
            {m.display_name}
            <button
              type="button"
              className="hover:text-indigo-900 dark:hover:text-indigo-100"
              onClick={(e) => { e.stopPropagation(); onChange(selectedIds.filter((id) => id !== m.user_id)) }}
            >
              ×
            </button>
          </span>
        ))}
        {selectedMembers.length === 0 && !open && (
          <span className="text-gray-400 dark:text-gray-500 py-0.5">{t('watchers.pickWatchers')}</span>
        )}
      </div>

      {open && (
        <div className="absolute z-20 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg">
          <div className="p-2">
            <input
              ref={inputRef}
              className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-500"
              placeholder={t('userPicker.searchMembers')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <ul className="max-h-48 overflow-auto">
            {filtered.map((m) => (
              <li key={m.user_id}>
                <button
                  type="button"
                  className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100"
                  onClick={() => {
                    onChange([...selectedIds, m.user_id])
                    setSearch('')
                  }}
                >
                  <div className="font-medium">{m.display_name}</div>
                  <div className="text-xs text-gray-400">{m.email}</div>
                </button>
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">{t('userPicker.noMembersFound')}</li>
            )}
          </ul>
        </div>
      )}
    </div>
  )
}

import { useState, useRef, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { UserPicker } from '@/components/ui/UserPicker'
import { ProjectPicker } from '@/components/ui/ProjectPicker'
import { MentionSearchModal } from '@/components/ui/MentionSearchModal'
import { StagedAttachmentsField } from '@/components/workitems/StagedAttachmentsField'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import type { StagedAttachmentsState, StagedUploadProgress } from '@/hooks/useStagedAttachments'
import { stagedMarkdownLink } from '@/utils/stagedAttachments'
import { formatFileSize } from '@/utils/fileSize'
import type { WorkflowStatus } from '@/api/workflows'
import type { ProjectMember, Project } from '@/api/projects'
import type { Milestone } from '@/api/milestones'

const TYPES = ['task', 'ticket', 'bug', 'feedback', 'epic']
const PRIORITIES = ['low', 'medium', 'high', 'critical']
const VISIBILITIES = ['internal', 'portal', 'public']

// Statuses a new item may start in. Done and cancelled statuses are reached
// through transitions, which is what records resolved_at — the API rejects them
// at creation time too.
const CLOSED_CATEGORIES = new Set(['done', 'cancelled'])

interface WorkItemFormProps {
  projectKey: string
  mode: 'create' | 'edit'
  members: ProjectMember[]
  milestones?: Milestone[]
  allowedComplexityValues?: number[]
  projects?: Project[]
  projectLocked?: boolean
  onProjectChange?: (projectKey: string) => void
  /**
   * Create mode: the selected work item type. Owned by the parent because the
   * type decides which workflow — and so which statuses — apply.
   */
  type?: string
  onTypeChange?: (type: string) => void
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
  /**
   * Create mode: every status of the resolved workflow (the open ones are
   * offered, the initial one is the default).
   * Edit mode: the statuses referenced by allowedTransitions.
   */
  statuses?: WorkflowStatus[]
  allowedTransitions?: string[]
  /** Create mode: files held until the item exists. */
  staged?: StagedAttachmentsState
  /** Attachment size cap in bytes, from the server config. */
  maxUploadSize?: number
  /** Reports whether any field has been filled in, so the modal can guard closing. */
  onDirtyChange?: (dirty: boolean) => void
  /** Set while staged files upload after creation; drives the attachment progress bar. */
  uploadProgress?: StagedUploadProgress | null
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
  type = '',
  onTypeChange,
  initialValues = {},
  statuses,
  allowedTransitions,
  staged,
  maxUploadSize,
  onDirtyChange,
  uploadProgress,
  onSubmit,
  onCancel,
  isSubmitting,
  submitError,
}: WorkItemFormProps) {
  const { t } = useTranslation()
  const typeSelected = mode === 'edit' || type !== ''
  // Submitting covers creation plus the attachment uploads that follow it, so
  // the whole form stays locked until the modal closes.
  const locked = isSubmitting
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
  // Create mode remembers which type the status was picked for, so switching
  // type falls back to the new workflow's initial status without an effect.
  const [statusChoice, setStatusChoice] = useState<{ type: string; name: string } | null>(null)
  const [watcherIds, setWatcherIds] = useState<string[]>([])
  const [attachmentError, setAttachmentError] = useState<string | null>(null)
  // Required fields are only marked once the user has tried to submit, so an
  // untouched form doesn't open covered in red.
  const [showErrors, setShowErrors] = useState(false)

  const descRef = useRef<HTMLTextAreaElement>(null)
  const descMention = useMentionAutocomplete({
    value: description,
    onValueChange: setDescription,
    textareaRef: descRef,
  })

  const byPosition = (a: WorkflowStatus, b: WorkflowStatus) => a.position - b.position

  // The open statuses of the item's workflow, and the one it starts in when the
  // user makes no choice (the workflow's initial status).
  const openStatuses = useMemo(
    () => (statuses ?? []).filter((s) => !CLOSED_CATEGORIES.has(s.category)).sort(byPosition),
    [statuses],
  )
  const chosenStatus = statusChoice && statusChoice.type === type ? statusChoice.name : ''
  const defaultStatus = useMemo(() => {
    const initial = [...(statuses ?? [])].sort(byPosition)[0]
    if (initial && !CLOSED_CATEGORIES.has(initial.category)) return initial.name
    return openStatuses[0]?.name ?? ''
  }, [statuses, openStatuses])

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
  const missingRequired = mode === 'create' && ((!!projects && !projectKey) || !type || !title.trim())
  const hasValidationErrors = !!complexityError

  // Anything the user typed or picked. The project is left out: it is
  // pre-selected from the last one used, so re-picking it loses nothing.
  const isDirty =
    title !== (initialValues.title ?? '') ||
    description !== (initialValues.description ?? '') ||
    priority !== (initialValues.priority ?? 'medium') ||
    assigneeId !== (initialValues.assignee_id ?? null) ||
    labels !== (initialValues.labels?.join(', ') ?? '') ||
    visibility !== (initialValues.visibility ?? 'internal') ||
    complexity !== (initialValues.complexity != null ? String(initialValues.complexity) : '') ||
    dueDate !== (initialValues.due_date ?? '') ||
    milestoneId !== (initialValues.milestone_id ?? '') ||
    watcherIds.length > 0 ||
    !!statusChoice

  useEffect(() => {
    onDirtyChange?.(isDirty)
  }, [isDirty, onDirtyChange])

  /** Stages files and reports the ones the server would reject as too large. */
  function stageFiles(files: File[], options?: { inline?: boolean }) {
    if (!staged) return []
    const { accepted, tooLarge } = staged.add(files, options)
    setAttachmentError(
      tooLarge.length > 0
        ? t('workitems.form.attachmentsTooLarge', {
            files: tooLarge.join(', '),
            size: maxUploadSize ? formatFileSize(maxUploadSize) : '',
          })
        : null,
    )
    return accepted
  }

  /**
   * Files dropped or pasted on the description become staged attachments and
   * leave a markdown link behind, the way the detail-page editor does. The link
   * points at a `staged:` placeholder until the upload gives it a real URL.
   */
  function stageIntoDescription(files: File[]) {
    const accepted = stageFiles(files, { inline: true })
    if (accepted.length === 0) return
    const links = accepted.map((s) => stagedMarkdownLink(s.file, s.id)).join('\n')
    setDescription((prev) => (prev ? `${prev}\n${links}` : links))
  }

  function handleDescriptionPaste(e: React.ClipboardEvent<HTMLTextAreaElement>) {
    if (!staged) return
    const files = Array.from(e.clipboardData?.items ?? [])
      .filter((item) => item.kind === 'file')
      .map((item) => item.getAsFile())
      .filter((f): f is File => f !== null)
    if (files.length === 0) return
    e.preventDefault()
    stageIntoDescription(files)
  }

  function handleDescriptionDrop(e: React.DragEvent<HTMLTextAreaElement>) {
    if (!staged) return
    const files = Array.from(e.dataTransfer?.files ?? [])
    if (files.length === 0) return
    e.preventDefault()
    e.stopPropagation()
    stageIntoDescription(files)
  }

  function handleDescriptionDragOver(e: React.DragEvent<HTMLTextAreaElement>) {
    if (!staged || !e.dataTransfer.types.includes('Files')) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'copy'
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    submitForm()
  }

  /** Cmd/Ctrl+Enter submits from anywhere in the form, including the description. */
  function handleFormKeyDown(e: React.KeyboardEvent) {
    if (e.key !== 'Enter' || !(e.metaKey || e.ctrlKey)) return
    // The mention popup owns Enter while it is open.
    if (descMention.mentionModalOpen) return
    e.preventDefault()
    submitForm()
  }

  function submitForm() {
    if (locked) return
    if (missingRequired || hasValidationErrors) {
      setShowErrors(true)
      return
    }
    if (mode === 'create') {
      onSubmit({
        type,
        title,
        description: description || undefined,
        priority,
        // Left out when the user kept the default, so the API applies the
        // workflow's initial status itself.
        status: chosenStatus || undefined,
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

  // --- Fields, shared by the create and edit layouts ---

  const projectField = mode === 'create' && projects && (
    <ProjectPicker
      projects={projects}
      value={projectKey}
      onChange={(key) => onProjectChange?.(key)}
      disabled={projectLocked || locked}
      error={showErrors && !projectKey ? t('workitems.form.requiredField') : undefined}
    />
  )

  const typeField = mode === 'create' && (
    <Select
      label={t('workitems.form.type')}
      value={type}
      onChange={(e) => onTypeChange?.(e.target.value)}
      error={showErrors && !type ? t('workitems.form.requiredField') : undefined}
      required
      requiredMarker
      disabled={locked}
    >
      {!typeSelected && <option value="">{t('workitems.form.typePlaceholder')}</option>}
      {TYPES.map((tp) => (
        <option key={tp} value={tp}>{t(`workitems.types.${tp}`)}</option>
      ))}
    </Select>
  )

  const titleField = (
    <Input
      label={t('workitems.form.title')}
      value={title}
      onChange={(e) => setTitle(e.target.value)}
      placeholder={t('workitems.form.titlePlaceholder')}
      error={showErrors && !title.trim() ? t('workitems.form.requiredField') : undefined}
      required
      requiredMarker
      autoFocus
      disabled={locked}
    />
  )

  const descriptionField = (
    <div className={mode === 'create' ? 'flex min-h-0 flex-1 flex-col' : ''}>
      <div className="flex items-baseline justify-between gap-2 mb-1">
        <label className="block text-sm font-medium text-[var(--foreground)]">{t('workitems.form.description')}</label>
        <span className="text-xs text-[var(--foreground-muted)]">{t('workitems.form.descriptionHint')}</span>
      </div>
      <textarea
        ref={descRef}
        className={`block w-full rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed ${mode === 'create' ? 'min-h-0 flex-1 resize-none' : ''}`}
        rows={4}
        value={description}
        placeholder={t('workitems.form.descriptionPlaceholder')}
        onChange={(e) => setDescription(e.target.value)}
        onKeyDown={(e) => {
          descMention.onMentionKeyDown(e)
        }}
        onPaste={handleDescriptionPaste}
        onDrop={handleDescriptionDrop}
        onDragOver={handleDescriptionDragOver}
        disabled={locked}
      />
      <MentionSearchModal
        open={descMention.mentionModalOpen}
        position={descMention.dropdownPosition}
        onClose={descMention.onMentionClose}
        onSelect={descMention.onMentionSelect}
      />
    </div>
  )

  const attachmentsField = staged && (
    <div className="flex min-w-0 flex-1">
      <StagedAttachmentsField
        staged={staged.staged}
        onAdd={(files) => stageFiles(files)}
        onRemove={staged.remove}
        maxUploadSize={maxUploadSize}
        error={attachmentError}
        progress={uploadProgress}
        disabled={locked}
      />
    </div>
  )

  const priorityField = (
    <Select label={t('workitems.form.priority')} value={priority} onChange={(e) => setPriority(e.target.value)} disabled={locked}>
      {PRIORITIES.map((p) => (
        <option key={p} value={p}>{t(`workitems.priorities.${p}`)}</option>
      ))}
    </Select>
  )

  const complexityField = allowedComplexityValues.length > 0 ? (
    <Select label={t('workitems.form.complexity')} value={complexity} onChange={(e) => setComplexity(e.target.value)} error={complexityError} disabled={locked}>
      <option value="">{t('workitems.form.complexityPlaceholder')}</option>
      {allowedComplexityValues.map((v) => (
        <option key={v} value={String(v)}>{v}</option>
      ))}
    </Select>
  ) : (
    <Input label={t('workitems.form.complexity')} type="number" min="1" value={complexity} onChange={(e) => setComplexity(e.target.value)} placeholder={t('workitems.form.complexityPlaceholder')} error={complexityError} disabled={locked} />
  )

  const assigneeField = (
    <div>
      <label className="block text-sm font-medium text-[var(--foreground)] mb-1">{t('workitems.form.assignee')}</label>
      <UserPicker members={members} value={assigneeId} onChange={setAssigneeId} disabled={locked} />
    </div>
  )

  const createStatusField = mode === 'create' && (
    <Select
      label={t('workitems.form.status')}
      value={chosenStatus || defaultStatus}
      onChange={(e) => setStatusChoice({ type, name: e.target.value })}
      disabled={locked || openStatuses.length === 0}
    >
      {openStatuses.length === 0 ? (
        <option value="">{t('workitems.form.statusPlaceholder')}</option>
      ) : (
        openStatuses.map((s) => (
          <option key={s.name} value={s.name}>
            {t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name })}
          </option>
        ))
      )}
    </Select>
  )

  const editStatusField = mode === 'edit' && statuses && allowedTransitions && (
    <Select label={t('workitems.form.status')} value={status} onChange={(e) => setStatus(e.target.value)} disabled={locked}>
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
  )

  const watchersField = mode === 'create' && (
    <div>
      <label className="block text-sm font-medium text-[var(--foreground)] mb-1">{t('watchers.watchersField')}</label>
      <MultiUserPicker members={members} selectedIds={watcherIds} onChange={setWatcherIds} disabled={locked} />
    </div>
  )

  const labelsField = (
    <Input label={t('workitems.form.labels')} value={labels} onChange={(e) => setLabels(e.target.value)} placeholder={t('workitems.form.labelsPlaceholder')} disabled={locked} />
  )

  const visibilityField = (
    <Select label={t('workitems.form.visibility')} value={visibility} onChange={(e) => setVisibility(e.target.value)} disabled={locked}>
      {VISIBILITIES.map((v) => (
        <option key={v} value={v}>{t(`workitems.visibilities.${v}`)}</option>
      ))}
    </Select>
  )

  const dueDateField = (
    <Input label={t('workitems.form.dueDate')} type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} disabled={locked} />
  )

  const milestoneField = milestones.length > 0 && (
    <Select label={t('workitems.form.milestone')} value={milestoneId} onChange={(e) => setMilestoneId(e.target.value)} disabled={locked}>
      <option value="">{t('milestones.noMilestone')}</option>
      {milestones.filter((m) => m.status === 'open').map((m) => (
        <option key={m.id} value={m.id}>{m.name}</option>
      ))}
    </Select>
  )

  // Always clickable: a click with something missing marks the offending fields
  // instead of leaving the user hunting for a disabled button's reason.
  const submitDisabled = isSubmitting
  const actions = (
    <>
      <Button type="button" variant="secondary" onClick={onCancel} disabled={locked}>{t('common.cancel')}</Button>
      <Button type="submit" disabled={submitDisabled}>
        {isSubmitting ? t('common.saving') : mode === 'create' ? t('common.create') : t('common.save')}
      </Button>
    </>
  )

  if (mode === 'edit') {
    return (
      <form onSubmit={handleSubmit} noValidate className="space-y-4">
        {titleField}
        {descriptionField}
        {priorityField}
        {editStatusField}
        {complexityField}
        {assigneeField}
        {labelsField}
        {visibilityField}
        {dueDateField}
        {milestoneField}
        {submitError && <p className="text-sm text-[var(--danger)]">{submitError}</p>}
        <div className="flex justify-end gap-3 pt-2">{actions}</div>
      </form>
    )
  }

  return (
    <form onSubmit={handleSubmit} onKeyDown={handleFormKeyDown} noValidate className="flex min-h-0 flex-1 flex-col">
      {/*
        From md up the body is exactly the height the modal leaves it — it never
        scrolls, so nothing can slide under the footer. Below md the columns
        stack and the body scrolls normally.
      */}
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain md:overflow-hidden">
        <div className="grid md:h-full md:grid-cols-[minmax(0,1fr)_18rem]">
          <div className="flex flex-col gap-4 px-6 py-5 md:min-h-0">
            {titleField}
            {descriptionField}
          </div>
          {/*
            From md up the fields are absolutely positioned inside their column
            so they don't add to the row's height: revealing "More fields"
            scrolls this column instead of growing the modal.
          */}
          <div className="border-t border-[var(--border)] md:relative md:border-l md:border-t-0">
            <div className="space-y-4 px-6 py-5 md:absolute md:inset-0 md:overflow-y-auto md:overscroll-contain">
              {projectField}
              {typeField}
              {priorityField}
              {assigneeField}
              {createStatusField}
              {complexityField}
              {watchersField}
              {labelsField}
              {visibilityField}
              {dueDateField}
              {milestoneField}
            </div>
          </div>
        </div>
      </div>
      <div className="shrink-0 border-t border-[var(--border)] px-6 py-4 dark:border-[var(--border)]">
        {submitError && <p className="mb-2 text-sm text-[var(--danger)]">{submitError}</p>}
        <div className="flex items-center gap-4">
          {attachmentsField}
          <div className="flex shrink-0 gap-3">{actions}</div>
        </div>
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
        className={`flex flex-wrap gap-1.5 min-h-[38px] rounded-md border border-[var(--border)] px-2 py-1.5 text-sm bg-[var(--surface)] ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-text'}`}
        onClick={() => { if (disabled) return; setOpen(true); setTimeout(() => inputRef.current?.focus(), 0) }}
      >
        {selectedMembers.map((m) => (
          <span key={m.user_id} className="inline-flex items-center gap-1 rounded-full bg-[var(--primary-muted)] text-[var(--primary)] px-2 py-0.5 text-xs font-medium">
            {m.display_name}
            <button
              type="button"
              className="hover:text-[var(--primary)]"
              onClick={(e) => { e.stopPropagation(); onChange(selectedIds.filter((id) => id !== m.user_id)) }}
            >
              ×
            </button>
          </span>
        ))}
        {selectedMembers.length === 0 && !open && (
          <span className="text-[var(--foreground-muted)] py-0.5">{t('watchers.pickWatchers')}</span>
        )}
      </div>

      {open && (
        <div className="absolute z-20 mt-1 w-full bg-[var(--surface)] border border-[var(--border)] rounded-md shadow-lg">
          <div className="p-2">
            <input
              ref={inputRef}
              className="block w-full rounded-md border border-[var(--border)] bg-[var(--surface-secondary)] text-[var(--foreground)] px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-[var(--focus-ring)]"
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
                  className="w-full text-left px-3 py-2 text-sm hover:bg-[var(--surface-hover)] text-[var(--foreground)]"
                  onClick={() => {
                    onChange([...selectedIds, m.user_id])
                    setSearch('')
                  }}
                >
                  <div className="font-medium">{m.display_name}</div>
                  <div className="text-xs text-[var(--foreground-muted)]">{m.email}</div>
                </button>
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-[var(--foreground-muted)]">{t('userPicker.noMembersFound')}</li>
            )}
          </ul>
        </div>
      )}
    </div>
  )
}

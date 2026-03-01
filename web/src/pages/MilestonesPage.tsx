import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useMilestones, useCreateMilestone, useUpdateMilestone, useDeleteMilestone } from '@/hooks/useMilestones'
import { useMembers } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Plus, Pencil, Trash2, ChevronDown, ChevronRight, Check, Clock } from 'lucide-react'
import { formatDuration } from '@/utils/duration'
import type { Milestone, CreateMilestoneInput, UpdateMilestoneInput } from '@/api/milestones'
import type { AxiosError } from 'axios'

export function MilestonesPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const { user } = useAuth()
  const { data: members } = useMembers(projectKey ?? '')
  const { data: milestones, isLoading } = useMilestones(projectKey ?? '')

  const createMutation = useCreateMilestone(projectKey ?? '')
  const updateMutation = useUpdateMilestone(projectKey ?? '')
  const deleteMutation = useDeleteMilestone(projectKey ?? '')

  const [editorOpen, setEditorOpen] = useState(false)
  const [editingMilestone, setEditingMilestone] = useState<Milestone | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Milestone | null>(null)
  const [error, setError] = useState('')
  const [savedId, setSavedId] = useState<string | null>(null)
  const [closedExpanded, setClosedExpanded] = useState(false)

  const currentUserMember = members?.find((m) => m.user_id === user?.id)
  const currentUserRole = currentUserMember?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canManage = currentUserRole === 'owner' || currentUserRole === 'admin' || user?.global_role === 'admin'

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    )
  }

  const openMilestones = milestones?.filter((m) => m.status === 'open') ?? []
  const closedMilestones = milestones?.filter((m) => m.status === 'closed') ?? []

  function openEditor(milestone?: Milestone) {
    setEditingMilestone(milestone ?? null)
    setEditorOpen(true)
  }

  function flashSaved(id: string) {
    setSavedId(id)
    setTimeout(() => setSavedId(null), 2000)
  }

  function handleSave(input: CreateMilestoneInput | UpdateMilestoneInput) {
    setError('')
    if (editingMilestone) {
      const id = editingMilestone.id
      updateMutation.mutate(
        { milestoneId: id, input: input as UpdateMilestoneInput },
        {
          onSuccess: () => {
            flashSaved(id)
            setEditorOpen(false)
            setEditingMilestone(null)
          },
          onError: (err) => {
            const axiosErr = err as AxiosError<{ error?: { message?: string } }>
            setError(axiosErr.response?.data?.error?.message ?? t('milestones.updateError'))
          },
        },
      )
    } else {
      createMutation.mutate(input as CreateMilestoneInput, {
        onSuccess: (data) => {
          flashSaved(data.id)
          setEditorOpen(false)
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          setError(axiosErr.response?.data?.error?.message ?? t('milestones.createError'))
        },
      })
    }
  }

  function handleDelete() {
    if (!deleteTarget) return
    setError('')
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => {
        setDeleteTarget(null)
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setError(axiosErr.response?.data?.error?.message ?? t('milestones.deleteError'))
        setDeleteTarget(null)
      },
    })
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('milestones.title')}</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('milestones.description')}</p>
        </div>
        {canManage && (
          <Button size="sm" onClick={() => openEditor()}>
            <Plus className="h-4 w-4 mr-1" />
            {t('milestones.create')}
          </Button>
        )}
      </div>

      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      {/* Open milestones */}
      {openMilestones.length === 0 && closedMilestones.length === 0 ? (
        <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('milestones.noMilestones')}</p>
          {canManage && (
            <Button size="sm" variant="secondary" className="mt-3" onClick={() => openEditor()}>
              <Plus className="h-4 w-4 mr-1" />
              {t('milestones.createFirst')}
            </Button>
          )}
        </div>
      ) : (
        <>
          {openMilestones.length > 0 && (
            <div>
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">{t('milestones.statusOpen')} ({openMilestones.length})</h3>
              <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
                {openMilestones.map((m) => (
                  <MilestoneCard key={m.id} milestone={m} projectKey={projectKey ?? ''} canManage={canManage} saved={savedId === m.id} onEdit={() => openEditor(m)} onDelete={() => setDeleteTarget(m)} />
                ))}
              </div>
            </div>
          )}

          {closedMilestones.length > 0 && (
            <div>
              <button
                className="flex items-center gap-1 text-sm font-medium text-gray-500 dark:text-gray-400 mb-2 hover:text-gray-700 dark:hover:text-gray-300"
                onClick={() => setClosedExpanded(!closedExpanded)}
              >
                {closedExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                {t('milestones.statusClosed')} ({closedMilestones.length})
              </button>
              {closedExpanded && (
                <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
                  {closedMilestones.map((m) => (
                    <MilestoneCard key={m.id} milestone={m} projectKey={projectKey ?? ''} canManage={canManage} saved={savedId === m.id} onEdit={() => openEditor(m)} onDelete={() => setDeleteTarget(m)} />
                  ))}
                </div>
              )}
            </div>
          )}
        </>
      )}

      {/* Editor modal */}
      <Modal
        open={editorOpen}
        onClose={() => { setEditorOpen(false); setEditingMilestone(null) }}
        title={editingMilestone ? t('milestones.editMilestone') : t('milestones.createMilestone')}
      >
        <MilestoneForm
          milestone={editingMilestone}
          onSubmit={handleSave}
          onCancel={() => { setEditorOpen(false); setEditingMilestone(null) }}
          isPending={createMutation.isPending || updateMutation.isPending}
        />
      </Modal>

      {/* Delete confirmation */}
      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('milestones.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="milestones.deleteConfirmBody" values={{ name: deleteTarget?.name }} components={{ bold: <strong /> }} />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setDeleteTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button variant="danger" disabled={deleteMutation.isPending} onClick={handleDelete}>
            {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

// --- Milestone Card ---

function MilestoneCard({
  milestone,
  projectKey,
  canManage,
  saved,
  onEdit,
  onDelete,
}: {
  milestone: Milestone
  projectKey: string
  canManage: boolean
  saved: boolean
  onEdit: () => void
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const percent = milestone.total_count > 0 ? Math.round((milestone.closed_count / milestone.total_count) * 100) : 0

  return (
    <div className="p-4">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <Link to={`/projects/${projectKey}/milestones/${milestone.id}`} className="text-lg font-semibold text-gray-900 dark:text-gray-100 hover:text-indigo-600 dark:hover:text-indigo-400">{milestone.name}</Link>
            {milestone.due_date && <DueDateLabel dueDate={milestone.due_date} isClosed={milestone.status === 'closed'} />}
          </div>
          {milestone.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{milestone.description}</p>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {saved && <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />}
          {canManage && (
            <>
              <Button variant="ghost" size="sm" onClick={onEdit}>
                <Pencil className="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="sm" onClick={onDelete}>
                <Trash2 className="h-3.5 w-3.5 text-red-500" />
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="mt-3">
        <div className="flex items-center justify-between text-sm text-gray-500 dark:text-gray-400 mb-1">
          <span>
            {milestone.total_count > 0
              ? t('milestones.progress', { closed: milestone.closed_count, total: milestone.total_count })
              : t('milestones.progressNone')}
          </span>
          {milestone.total_count > 0 && <span>{percent}%</span>}
        </div>
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
          <div
            className="bg-green-500 dark:bg-green-400 h-2 rounded-full transition-all"
            style={{ width: `${percent}%` }}
          />
        </div>
      </div>

      {/* Time tracking summary */}
      {(milestone.total_estimated_seconds > 0 || milestone.total_spent_seconds > 0) && (
        <div className="mt-2 flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
          <Clock className="h-3.5 w-3.5 shrink-0" />
          {milestone.total_estimated_seconds > 0 && (
            <span>
              {t('milestones.timeEstimated', { time: formatDuration(milestone.total_estimated_seconds) })}
            </span>
          )}
          {milestone.total_spent_seconds > 0 && (
            <span>
              {t('milestones.timeSpent', { time: formatDuration(milestone.total_spent_seconds) })}
            </span>
          )}
          {milestone.total_estimated_seconds > 0 && milestone.total_spent_seconds > 0 && (
            <TimeProgressBar estimated={milestone.total_estimated_seconds} spent={milestone.total_spent_seconds} />
          )}
        </div>
      )}
    </div>
  )
}

// --- Time Progress Bar ---

function TimeProgressBar({ estimated, spent }: { estimated: number; spent: number }) {
  const { t } = useTranslation()
  const percent = Math.min(Math.round((spent / estimated) * 100), 100)
  const isOver = spent > estimated

  return (
    <div className="flex items-center gap-1.5 flex-1 min-w-0" title={t('milestones.timeProgress', { percent })}>
      <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-1.5 min-w-[40px]">
        <div
          className={`h-1.5 rounded-full transition-all ${isOver ? 'bg-red-500 dark:bg-red-400' : 'bg-blue-500 dark:bg-blue-400'}`}
          style={{ width: `${percent}%` }}
        />
      </div>
      <span className={`shrink-0 ${isOver ? 'text-red-500 dark:text-red-400' : ''}`}>{percent}%</span>
    </div>
  )
}

// --- Due Date Label ---

function DueDateLabel({ dueDate, isClosed }: { dueDate: string; isClosed: boolean }) {
  const { t } = useTranslation()
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const due = new Date(dueDate + 'T00:00:00')
  const diffDays = Math.round((due.getTime() - today.getTime()) / (1000 * 60 * 60 * 24))

  let text: string
  let colorClass: string

  if (isClosed) {
    text = dueDate
    colorClass = 'text-gray-400 dark:text-gray-500'
  } else if (diffDays < 0) {
    text = t('milestones.overdue', { days: Math.abs(diffDays) })
    colorClass = 'text-red-600 dark:text-red-400'
  } else if (diffDays === 0) {
    text = t('milestones.dueToday')
    colorClass = 'text-yellow-600 dark:text-yellow-400'
  } else if (diffDays <= 7) {
    text = t('milestones.dueIn', { days: diffDays })
    colorClass = 'text-yellow-600 dark:text-yellow-400'
  } else {
    text = dueDate
    colorClass = 'text-gray-400 dark:text-gray-500'
  }

  return <span className={`text-xs ${colorClass}`}>{text}</span>
}

// --- Milestone Form ---

function MilestoneForm({
  milestone,
  onSubmit,
  onCancel,
  isPending,
}: {
  milestone: Milestone | null
  onSubmit: (input: CreateMilestoneInput | UpdateMilestoneInput) => void
  onCancel: () => void
  isPending: boolean
}) {
  const { t } = useTranslation()
  const [name, setName] = useState(milestone?.name ?? '')
  const [description, setDescription] = useState(milestone?.description ?? '')
  const [dueDate, setDueDate] = useState(milestone?.due_date ?? '')
  const [status, setStatus] = useState(milestone?.status ?? 'open')
  const [validationError, setValidationError] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setValidationError('')

    if (!name.trim()) {
      setValidationError(t('milestones.nameRequired'))
      return
    }

    if (milestone) {
      const input: UpdateMilestoneInput = {}
      if (name.trim() !== milestone.name) input.name = name.trim()
      if (description !== (milestone.description ?? '')) input.description = description || null
      if (dueDate !== (milestone.due_date ?? '')) input.due_date = dueDate || null
      if (status !== milestone.status) input.status = status as 'open' | 'closed'
      onSubmit(input)
    } else {
      onSubmit({
        name: name.trim(),
        description: description || undefined,
        due_date: dueDate || undefined,
      })
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {validationError && (
        <p className="text-sm text-red-600 dark:text-red-400">{validationError}</p>
      )}
      <Input
        label={t('milestones.name')}
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder={t('milestones.namePlaceholder')}
        required
        autoFocus
      />
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('common.description')}
        </label>
        <textarea
          rows={3}
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <div className="flex gap-4">
        <div className="w-48">
          <Input
            label={t('milestones.dueDate')}
            type="date"
            value={dueDate}
            onChange={(e) => setDueDate(e.target.value)}
          />
        </div>
        {milestone && (
          <div className="w-48">
            <Select label={t('workitems.form.status')} value={status} onChange={(e) => setStatus(e.target.value as 'open' | 'closed')}>
              <option value="open">{t('milestones.statusOpen')}</option>
              <option value="closed">{t('milestones.statusClosed')}</option>
            </Select>
          </div>
        )}
      </div>
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={isPending}>
          {isPending ? t('common.saving') : milestone ? t('common.save') : t('common.create')}
        </Button>
      </div>
    </form>
  )
}

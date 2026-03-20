import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  useWorkflows,
  useWorkflow,
  useCreateSystemWorkflow,
  useUpdateSystemWorkflow,
  useDeleteSystemWorkflow,
  useSystemStatuses,
} from '@/hooks/useWorkflows'
import { useSystemSetting, useSetSystemSetting } from '@/hooks/useSystemSettings'
import { useNotification } from '@/contexts/NotificationContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { Plus, Pencil, Trash2, ArrowRight, Check, Eye } from 'lucide-react'
import { Tooltip } from '@/components/ui/Tooltip'
import type { WorkflowStatus } from '@/api/workflows'
import type { AxiosError } from 'axios'

const CATEGORY_COLORS: Record<string, 'blue' | 'indigo' | 'green' | 'gray'> = {
  todo: 'blue',
  in_progress: 'indigo',
  done: 'green',
  cancelled: 'gray',
}

export function SystemWorkflowsPage() {
  const { t } = useTranslation()
  const { showNotification } = useNotification()
  const { data: workflows, isLoading } = useWorkflows()

  const [editorOpen, setEditorOpen] = useState(false)
  const [editingWorkflowId, setEditingWorkflowId] = useState<string | null>(null)
  const [viewingWorkflowId, setViewingWorkflowId] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null)
  const [savedId, setSavedId] = useState<string | null>(null)

  const { data: defaultTypeWorkflows } = useSystemSetting<Record<string, string>>('default_type_workflows')
  const setSystemSetting = useSetSystemSetting()

  const deleteMutation = useDeleteSystemWorkflow()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    )
  }

  function flashSaved(id: string) {
    setSavedId(id)
    setTimeout(() => setSavedId(null), 2000)
  }

  function handleDelete() {
    if (!deleteTarget) return
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => {
        setDeleteTarget(null)
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        showNotification(axiosErr.response?.data?.error?.message ?? t('workflows.deleteError'), 'error')
        setDeleteTarget(null)
      },
    })
  }

  function openEditor(workflowId?: string) {
    setEditingWorkflowId(workflowId ?? null)
    setEditorOpen(true)
  }

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('admin.workflows.title')}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('admin.workflows.description')}</p>
      </div>

      {/* Workflow Definitions */}
      <div>
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">
            {t('workflows.definitionsTitle')}
          </h3>
          <Button onClick={() => openEditor()} className="border border-transparent">
            <span className="sm:hidden">{t('workitems.newShort')}</span>
            <span className="hidden sm:inline">{t('workflows.create')}</span>
          </Button>
        </div>
      </div>

      {(!workflows || workflows.length === 0) ? (
        <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('admin.workflows.noWorkflows')}</p>
          <Button size="sm" variant="secondary" className="mt-3" onClick={() => openEditor()}>
            <Plus className="h-4 w-4 mr-1" />
            {t('workflows.createFirst')}
          </Button>
        </div>
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {workflows.map((wf) => (
            <div key={wf.id} className="p-4">
              <div className="flex items-center justify-between mb-2">
                <div>
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{wf.name}</span>
                  {wf.description && (
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{wf.description}</p>
                  )}
                </div>
                <div className="flex items-center gap-1">
                  {savedId === wf.id && <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />}
                  <Tooltip content={t('workflows.viewDetails')}>
                    <Button variant="ghost" size="sm" onClick={() => setViewingWorkflowId(wf.id)}>
                      <Eye className="h-3.5 w-3.5" />
                    </Button>
                  </Tooltip>
                  <Tooltip content={t('common.edit')}>
                    <Button variant="ghost" size="sm" onClick={() => openEditor(wf.id)}>
                      <Pencil className="h-3.5 w-3.5" />
                    </Button>
                  </Tooltip>
                  <Tooltip content={wf.is_default ? t('admin.workflows.cannotDeleteDefault') : t('common.delete')}>
                    <span>
                      <Button variant="ghost" size="sm" onClick={() => setDeleteTarget({ id: wf.id, name: wf.name })} disabled={wf.is_default}>
                        <Trash2 className={`h-3.5 w-3.5 ${wf.is_default ? 'text-gray-300 dark:text-gray-600' : 'text-red-500'}`} />
                      </Button>
                    </span>
                  </Tooltip>
                </div>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {wf.statuses.map((s) => (
                  <Badge key={s.name} color={CATEGORY_COLORS[s.category] ?? 'gray'}>
                    {t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name })}
                  </Badge>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Default Workflow Mapping */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('admin.workflows.defaultMappingTitle')}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('admin.workflows.defaultMappingDescription')}</p>
      </div>

      <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
        {['task', 'ticket', 'bug', 'feedback', 'epic'].map((itemType) => {
          const currentWfId = (defaultTypeWorkflows as Record<string, string> | undefined)?.[itemType] ?? ''
          return (
            <div key={itemType} className="flex items-center justify-between p-3">
              <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                {t(`workitems.types.${itemType}`)}
              </span>
              <div className="flex items-center gap-2">
                {savedId === `dtw:${itemType}` && (
                  <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                )}
                <select
                  className="min-w-0 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-60 disabled:cursor-not-allowed"
                  value={currentWfId}
                  onChange={(e) => {
                    const updated = { ...(defaultTypeWorkflows as Record<string, string> ?? {}), [itemType]: e.target.value }
                    setSystemSetting.mutate(
                      { key: 'default_type_workflows', value: updated },
                      {
                        onSuccess: () => flashSaved(`dtw:${itemType}`),
                        onError: (err) => {
                          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
                          showNotification(axiosErr.response?.data?.error?.message ?? t('admin.workflows.updateError'), 'error')
                        },
                      },
                    )
                  }}
                  disabled={setSystemSetting.isPending}
                >
                  {!currentWfId && <option value="">{t('admin.workflows.selectWorkflow')}</option>}
                  {workflows?.map((wf) => (
                    <option key={wf.id} value={wf.id}>{wf.name}</option>
                  ))}
                </select>
              </div>
            </div>
          )
        })}
      </div>

      {/* Workflow Editor Modal */}
      {editorOpen && (
        <SystemWorkflowEditorModal
          workflowId={editingWorkflowId}
          onClose={() => {
            setEditorOpen(false)
            setEditingWorkflowId(null)
          }}
          onSuccess={(id) => {
            flashSaved(id)
          }}
          onError={(msg) => showNotification(msg, 'error')}
        />
      )}

      {/* View Workflow Detail Modal */}
      {viewingWorkflowId && (
        <SystemWorkflowDetailModal
          workflowId={viewingWorkflowId}
          onClose={() => setViewingWorkflowId(null)}
        />
      )}

      {/* Delete confirmation modal */}
      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('workflows.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          {t('workflows.deleteConfirmBody', { name: deleteTarget?.name })}
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

// --- Workflow Detail Modal ---

function SystemWorkflowDetailModal({
  workflowId,
  onClose,
}: {
  workflowId: string
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { data: workflow, isLoading } = useWorkflow(workflowId)

  return (
    <Modal open onClose={onClose} title={workflow?.name ?? t('workflows.title')}>
      {isLoading || !workflow ? (
        <div className="flex justify-center py-8"><Spinner /></div>
      ) : (
        <div className="space-y-4">
          {workflow.description && (
            <p className="text-sm text-gray-600 dark:text-gray-300">{workflow.description}</p>
          )}

          <div>
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">{t('workflows.statuses')}</h4>
            <div className="flex flex-wrap gap-1.5">
              {workflow.statuses.map((s) => (
                <Badge key={s.name} color={CATEGORY_COLORS[s.category] ?? 'gray'}>
                  {t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name })}
                </Badge>
              ))}
            </div>
          </div>

          <div>
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">{t('workflows.transitions')}</h4>
            {workflow.transitions.length === 0 ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">{t('workflows.noTransitions')}</p>
            ) : (
              <div className="space-y-1">
                {workflow.transitions.map((tr) => {
                  const fromStatus = workflow.statuses.find((s) => s.name === tr.from_status)
                  const toStatus = workflow.statuses.find((s) => s.name === tr.to_status)
                  return (
                    <div key={tr.id} className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                      <Badge color={CATEGORY_COLORS[fromStatus?.category ?? ''] ?? 'gray'}>
                        {t(`workitems.statuses.${tr.from_status}`, { defaultValue: fromStatus?.display_name ?? tr.from_status })}
                      </Badge>
                      <ArrowRight className="h-3.5 w-3.5 text-gray-400 shrink-0" />
                      <Badge color={CATEGORY_COLORS[toStatus?.category ?? ''] ?? 'gray'}>
                        {t(`workitems.statuses.${tr.to_status}`, { defaultValue: toStatus?.display_name ?? tr.to_status })}
                      </Badge>
                      {tr.name && (
                        <span className="text-xs text-gray-400 ml-1">({tr.name})</span>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      )}
    </Modal>
  )
}

// --- Workflow Editor Modal ---

interface StatusDraft {
  name: string
  display_name: string
  category: 'todo' | 'in_progress' | 'done' | 'cancelled'
  position: number
}

interface TransitionDraft {
  from_status: string
  to_status: string
  name: string
}

function SystemWorkflowEditorModal({
  workflowId,
  onClose,
  onSuccess,
  onError,
}: {
  workflowId: string | null
  onClose: () => void
  onSuccess: (id: string) => void
  onError: (msg: string) => void
}) {
  const { t } = useTranslation()
  const { data: existingWorkflow } = useWorkflow(workflowId ?? '')
  const { data: availableStatuses } = useSystemStatuses()
  const createMutation = useCreateSystemWorkflow()
  const updateMutation = useUpdateSystemWorkflow()

  const isEdit = !!workflowId

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [statuses, setStatuses] = useState<StatusDraft[]>([])
  const [transitions, setTransitions] = useState<TransitionDraft[]>([])
  const [initialized, setInitialized] = useState(false)
  const [validationError, setValidationError] = useState('')

  // Initialize from existing workflow when editing
  if (isEdit && existingWorkflow && !initialized) {
    setName(existingWorkflow.name)
    setDescription(existingWorkflow.description ?? '')
    setStatuses(
      existingWorkflow.statuses.map((s) => ({
        name: s.name,
        display_name: s.display_name,
        category: s.category,
        position: s.position,
      }))
    )
    setTransitions(
      existingWorkflow.transitions.map((tr) => ({
        from_status: tr.from_status,
        to_status: tr.to_status,
        name: tr.name ?? '',
      }))
    )
    setInitialized(true)
  }

  const statusPool = availableStatuses ?? []

  function toggleStatus(status: WorkflowStatus) {
    const exists = statuses.find((s) => s.name === status.name)
    if (exists) {
      setStatuses(statuses.filter((s) => s.name !== status.name).map((s, i) => ({ ...s, position: i })))
      setTransitions(transitions.filter((tr) => tr.from_status !== status.name && tr.to_status !== status.name))
    } else {
      setStatuses([
        ...statuses,
        {
          name: status.name,
          display_name: status.display_name,
          category: status.category,
          position: statuses.length,
        },
      ])
    }
  }

  function addTransition() {
    if (statuses.length < 2) return
    setTransitions([...transitions, { from_status: statuses[0].name, to_status: statuses[1].name, name: '' }])
  }

  function removeTransition(idx: number) {
    setTransitions(transitions.filter((_, i) => i !== idx))
  }

  function updateTransition(idx: number, field: keyof TransitionDraft, value: string) {
    setTransitions(transitions.map((tr, i) => (i === idx ? { ...tr, [field]: value } : tr)))
  }

  function handleSave() {
    setValidationError('')

    if (!name.trim()) {
      setValidationError(t('workflows.nameRequired'))
      return
    }
    if (statuses.length === 0) {
      setValidationError(t('workflows.statusRequired'))
      return
    }
    const hasTodo = statuses.some((s) => s.category === 'todo')
    if (!hasTodo) {
      setValidationError(t('workflows.todoStatusRequired'))
      return
    }

    // Check for duplicate transitions
    const seenTransitions = new Set<string>()
    for (const tr of transitions) {
      const key = `${tr.from_status}->${tr.to_status}`
      if (seenTransitions.has(key)) {
        const fromLabel = t(`workitems.statuses.${tr.from_status}`, { defaultValue: tr.from_status })
        const toLabel = t(`workitems.statuses.${tr.to_status}`, { defaultValue: tr.to_status })
        setValidationError(t('workflows.duplicateTransition', { from: fromLabel, to: toLabel }))
        return
      }
      seenTransitions.add(key)
    }

    // Ensure position 0 is a todo status
    const sortedStatuses = [...statuses].sort((a, b) => a.position - b.position)
    if (sortedStatuses[0]?.category !== 'todo') {
      const todoIdx = sortedStatuses.findIndex((s) => s.category === 'todo')
      if (todoIdx > 0) {
        const [todoStatus] = sortedStatuses.splice(todoIdx, 1)
        sortedStatuses.unshift(todoStatus)
      }
    }
    const reindexedStatuses = sortedStatuses.map((s, i) => ({ ...s, position: i }))

    const input = {
      name: name.trim(),
      description: description.trim() || undefined,
      statuses: reindexedStatuses.map((s) => ({
        name: s.name,
        display_name: s.display_name,
        category: s.category,
        position: s.position,
        color: null,
      })),
      transitions: transitions.map((tr) => ({
        from_status: tr.from_status,
        to_status: tr.to_status,
        name: tr.name.trim() || null,
      })),
    }

    if (isEdit && workflowId) {
      updateMutation.mutate(
        { workflowId, input },
        {
          onSuccess: (data) => {
            onSuccess(data.id)
            onClose()
          },
          onError: (err) => {
            const axiosErr = err as AxiosError<{ error?: { message?: string } }>
            onError(axiosErr.response?.data?.error?.message ?? t('workflows.updateError'))
          },
        },
      )
    } else {
      createMutation.mutate(input, {
        onSuccess: (data) => {
          onSuccess(data.id)
          onClose()
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          onError(axiosErr.response?.data?.error?.message ?? t('workflows.createError'))
        },
      })
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Modal
      open
      onClose={onClose}
      title={isEdit ? t('workflows.editWorkflow') : t('workflows.createWorkflow')}
      className="!max-w-2xl overflow-x-hidden"
    >
      <div className="space-y-5">
        {validationError && (
          <p className="text-sm text-red-600 dark:text-red-400">{validationError}</p>
        )}

        {/* Name */}
        <Input
          label={t('workflows.workflowName')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('workflows.namePlaceholder')}
        />

        {/* Description */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('common.description')}
          </label>
          <textarea
            rows={2}
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('workflows.descriptionPlaceholder')}
          />
        </div>

        {/* Status Selection */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('workflows.selectStatuses')}
          </label>
          <div className="flex flex-wrap gap-2">
            {statusPool.map((status) => {
              const isSelected = statuses.some((s) => s.name === status.name)
              return (
                <button
                  key={status.name}
                  type="button"
                  onClick={() => toggleStatus(status)}
                  className={`inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium border transition-colors ${
                    isSelected
                      ? 'bg-indigo-50 border-indigo-300 text-indigo-700 dark:bg-indigo-900/30 dark:border-indigo-600 dark:text-indigo-300'
                      : 'bg-white border-gray-300 text-gray-600 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700'
                  }`}
                >
                  {isSelected && <span className="text-indigo-500">&#10003;</span>}
                  {t(`workitems.statuses.${status.name}`, { defaultValue: status.display_name })}
                  <span className="text-[10px] opacity-60">({status.category})</span>
                </button>
              )
            })}
          </div>

          {/* Selected statuses order */}
          {statuses.length > 0 && (
            <div className="mt-3">
              <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">{t('workflows.statusOrder')}</p>
              <div className="flex flex-wrap gap-1.5">
                {[...statuses].sort((a, b) => a.position - b.position).map((s, idx) => (
                  <Badge key={s.name} color={CATEGORY_COLORS[s.category] ?? 'gray'}>
                    {idx + 1}. {t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name })}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Transitions */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('workflows.transitions')}
            </label>
            <Button
              variant="ghost"
              size="sm"
              onClick={addTransition}
              disabled={statuses.length < 2}
            >
              <Plus className="h-3.5 w-3.5 mr-1" />
              {t('workflows.addTransition')}
            </Button>
          </div>

          {transitions.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">{t('workflows.noTransitionsYet')}</p>
          ) : (
            <div className="space-y-2">
              {transitions.map((tr, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <select
                    className="min-w-0 flex-1 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    value={tr.from_status}
                    onChange={(e) => updateTransition(idx, 'from_status', e.target.value)}
                  >
                    {statuses.map((s) => (
                      <option key={s.name} value={s.name}>{t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name })}</option>
                    ))}
                  </select>
                  <ArrowRight className="h-3.5 w-3.5 text-gray-400 shrink-0" />
                  <select
                    className="min-w-0 flex-1 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    value={tr.to_status}
                    onChange={(e) => updateTransition(idx, 'to_status', e.target.value)}
                  >
                    {statuses.map((s) => (
                      <option key={s.name} value={s.name}>{t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name })}</option>
                    ))}
                  </select>
                  <input
                    type="text"
                    className="min-w-0 w-16 sm:w-28 shrink rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    placeholder={t('workflows.transitionName')}
                    value={tr.name}
                    onChange={(e) => updateTransition(idx, 'name', e.target.value)}
                  />
                  <button
                    type="button"
                    className="text-gray-400 hover:text-red-500 p-1"
                    onClick={() => removeTransition(idx)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
          <Button variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSave} disabled={isPending}>
            {isPending ? t('common.saving') : isEdit ? t('common.save') : t('common.create')}
          </Button>
        </div>
      </div>
    </Modal>
  )
}

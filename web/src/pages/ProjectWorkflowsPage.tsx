import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useProjectWorkflows,
  useProjectWorkflowDetail,
  useCreateProjectWorkflow,
  useUpdateProjectWorkflow,
  useDeleteProjectWorkflow,
  useAvailableStatuses,
} from '@/hooks/useWorkflows'
import { useMembers, useProject, useUpdateProject, useTypeWorkflows, useUpdateTypeWorkflow } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { TimezoneSelect } from '@/components/ui/TimezoneSelect'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { Lock, Plus, Pencil, Trash2, ArrowRight, Check, Eye, Clock } from 'lucide-react'
import { Tooltip } from '@/components/ui/Tooltip'
import { SLAConfigModal } from '@/components/SLAConfigModal'
import type { WorkflowStatus } from '@/api/workflows'
import type { AxiosError } from 'axios'

const CATEGORY_COLORS: Record<string, 'blue' | 'indigo' | 'green' | 'gray'> = {
  todo: 'blue',
  in_progress: 'indigo',
  done: 'green',
  cancelled: 'gray',
}

export function ProjectWorkflowsPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const { user } = useAuth()
  const { data: members } = useMembers(projectKey ?? '')
  const { data: workflows, isLoading } = useProjectWorkflows(projectKey ?? '')
  const { data: project } = useProject(projectKey ?? '')
  const updateProjectMutation = useUpdateProject(projectKey ?? '')
  const { data: typeWorkflows } = useTypeWorkflows(projectKey ?? '')
  const updateTypeWorkflowMutation = useUpdateTypeWorkflow(projectKey ?? '')

  const [editorOpen, setEditorOpen] = useState(false)
  const [editingWorkflowId, setEditingWorkflowId] = useState<string | null>(null)
  const [viewingWorkflowId, setViewingWorkflowId] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null)
  const [error, setError] = useState('')
  const [savedId, setSavedId] = useState<string | null>(null)
  const [workflowError, setWorkflowError] = useState('')

  // SLA modal state
  const [slaModalType, setSlaModalType] = useState<string | null>(null)
  const [slaModalWorkflowId, setSlaModalWorkflowId] = useState<string | null>(null)

  // Business hours state
  const [bhDays, setBhDays] = useState<number[] | null>(null)
  const [bhStart, setBhStart] = useState<number | null>(null)
  const [bhEnd, setBhEnd] = useState<number | null>(null)
  const [bhTimezone, setBhTimezone] = useState<string | null>(null)
  const [bhError, setBhError] = useState('')

  // Inline checkmark indicators
  const [saved, setSaved] = useState<Record<string, boolean>>({})
  function showSaved(key: string) {
    setSaved((prev) => ({ ...prev, [key]: true }))
    setTimeout(() => setSaved((prev) => ({ ...prev, [key]: false })), 2000)
  }

  const deleteMutation = useDeleteProjectWorkflow(projectKey ?? '')

  // Determine permissions
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

  const systemWorkflows = workflows?.filter((w) => w.is_default || !w.project_id) ?? []
  const projectWorkflows = workflows?.filter((w) => !w.is_default && w.project_id) ?? []

  function flashSaved(id: string) {
    setSavedId(id)
    setTimeout(() => setSavedId(null), 2000)
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
        setError(axiosErr.response?.data?.error?.message ?? t('workflows.deleteError'))
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
      {/* Workflow Definitions */}
      <div>
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('workflows.definitionsTitle')}</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('workflows.definitionsDescription')}</p>
          </div>
          {canManage && (
            <Button size="sm" onClick={() => openEditor()}>
              <Plus className="h-4 w-4 mr-1" />
              {t('workflows.create')}
            </Button>
          )}
        </div>
      </div>

      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      {/* System Workflows */}
      {systemWorkflows.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2 flex items-center gap-1.5">
            <Lock className="h-3.5 w-3.5" />
            {t('workflows.systemWorkflows')}
          </h3>
          <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
            {systemWorkflows.map((wf) => (
              <div key={wf.id} className="p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{wf.name}</span>
                    <Badge color="gray">{t('common.system')}</Badge>
                  </div>
                  <Tooltip content={t('workflows.viewDetails')}>
                    <Button variant="ghost" size="sm" onClick={() => setViewingWorkflowId(wf.id)}>
                      <Eye className="h-3.5 w-3.5" />
                    </Button>
                  </Tooltip>
                </div>
                {wf.description && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">{wf.description}</p>
                )}
                <div className="flex flex-wrap gap-1.5">
                  {wf.statuses.map((s) => (
                    <Badge key={s.name} color={CATEGORY_COLORS[s.category] ?? 'gray'}>
                      {s.display_name}
                    </Badge>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Project Workflows */}
      <div>
        <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">
          {t('workflows.projectWorkflows')}
        </h3>
        {projectWorkflows.length === 0 ? (
          <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
            <p className="text-sm text-gray-500 dark:text-gray-400">{t('workflows.noProjectWorkflows')}</p>
            {canManage && (
              <Button size="sm" variant="secondary" className="mt-3" onClick={() => openEditor()}>
                <Plus className="h-4 w-4 mr-1" />
                {t('workflows.createFirst')}
              </Button>
            )}
          </div>
        ) : (
          <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
            {projectWorkflows.map((wf) => (
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
                    {canManage && (
                      <>
                        <Tooltip content={t('common.edit')}>
                          <Button variant="ghost" size="sm" onClick={() => openEditor(wf.id)}>
                            <Pencil className="h-3.5 w-3.5" />
                          </Button>
                        </Tooltip>
                        <Tooltip content={t('common.delete')}>
                          <Button variant="ghost" size="sm" onClick={() => setDeleteTarget({ id: wf.id, name: wf.name })}>
                            <Trash2 className="h-3.5 w-3.5 text-red-500" />
                          </Button>
                        </Tooltip>
                      </>
                    )}
                  </div>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {wf.statuses.map((s) => (
                    <Badge key={s.name} color={CATEGORY_COLORS[s.category] ?? 'gray'}>
                      {s.display_name}
                    </Badge>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Mapping */}
      {canManage && (
        <>
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('workflows.mappingTitle')}</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('workflows.mappingDescription')}</p>
          </div>

          {workflowError && <p className="text-sm text-red-600 dark:text-red-400">{workflowError}</p>}

          <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
            {['task', 'ticket', 'bug', 'feedback', 'epic'].map((itemType) => {
              const mapping = typeWorkflows?.find((tw) => tw.work_item_type === itemType)
              return (
                <div key={itemType} className="flex items-center justify-between p-3">
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    {t(`workitems.types.${itemType}`)}
                  </span>
                  <div className="flex items-center gap-2">
                    {saved[`wf:${itemType}`] && (
                      <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
                    <select
                      className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                      value={mapping?.workflow_id ?? ''}
                      onChange={(e) => {
                        setWorkflowError('')
                        updateTypeWorkflowMutation.mutate(
                          { workItemType: itemType, workflowId: e.target.value },
                          {
                            onSuccess: () => {
                              showSaved(`wf:${itemType}`)
                            },
                            onError: (err) => {
                              const axiosErr = err as AxiosError<{ error?: { message?: string } }>
                              setWorkflowError(axiosErr.response?.data?.error?.message ?? t('projects.settings.workflowUpdateError'))
                            },
                          },
                        )
                      }}
                      disabled={updateTypeWorkflowMutation.isPending}
                    >
                      {!mapping && <option value="">{t('projects.settings.selectWorkflow')}</option>}
                      {workflows?.map((wf) => (
                        <option key={wf.id} value={wf.id}>{wf.name}</option>
                      ))}
                    </select>
                    <Tooltip content={t('sla.configure')}>
                      <button
                        className="p-1 text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 disabled:opacity-50 disabled:cursor-not-allowed"
                        onClick={() => {
                          if (mapping?.workflow_id) {
                            setSlaModalType(itemType)
                            setSlaModalWorkflowId(mapping.workflow_id)
                          }
                        }}
                        disabled={!mapping?.workflow_id}
                      >
                        <Clock className="h-4 w-4" />
                      </button>
                    </Tooltip>
                  </div>
                </div>
              )
            })}
          </div>
        </>
      )}

      {/* Business Hours */}
      {canManage && project && (
        <>
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('workflows.businessHoursTitle')}</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('workflows.businessHoursDescription')}</p>
          </div>

          {bhError && <p className="text-sm text-red-600 dark:text-red-400">{bhError}</p>}

          <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-4">
            {(() => {
              const currentDays = bhDays ?? project.business_hours?.days ?? []
              const currentStart = bhStart ?? project.business_hours?.start_hour ?? 9
              const currentEnd = bhEnd ?? project.business_hours?.end_hour ?? 17
              const currentTz = bhTimezone ?? project.business_hours?.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone

              const dayLabels = [
                { value: 0, label: t('businessHours.sunday') },
                { value: 1, label: t('businessHours.monday') },
                { value: 2, label: t('businessHours.tuesday') },
                { value: 3, label: t('businessHours.wednesday') },
                { value: 4, label: t('businessHours.thursday') },
                { value: 5, label: t('businessHours.friday') },
                { value: 6, label: t('businessHours.saturday') },
              ]

              const hours = Array.from({ length: 24 }, (_, i) => i)

              return (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('businessHours.days')}
                    </label>
                    <div className="flex gap-2 flex-wrap">
                      {dayLabels.map((d) => (
                        <button
                          key={d.value}
                          type="button"
                          className={`px-3 py-1.5 text-sm rounded-md border transition-colors ${
                            currentDays.includes(d.value)
                              ? 'bg-indigo-100 dark:bg-indigo-900 border-indigo-300 dark:border-indigo-600 text-indigo-700 dark:text-indigo-200'
                              : 'bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
                          }`}
                          onClick={() => {
                            const newDays = currentDays.includes(d.value)
                              ? currentDays.filter((v: number) => v !== d.value)
                              : [...currentDays, d.value].sort()
                            setBhDays(newDays)
                          }}
                        >
                          {d.label}
                        </button>
                      ))}
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        {t('businessHours.startHour')}
                      </label>
                      <select
                        className="w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        value={currentStart}
                        onChange={(e) => setBhStart(Number(e.target.value))}
                      >
                        {hours.map((h) => (
                          <option key={h} value={h}>{String(h).padStart(2, '0')}:00</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        {t('businessHours.endHour')}
                      </label>
                      <select
                        className="w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        value={currentEnd}
                        onChange={(e) => setBhEnd(Number(e.target.value))}
                      >
                        {hours.map((h) => (
                          <option key={h} value={h}>{String(h).padStart(2, '0')}:00</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        {t('businessHours.timezone')}
                      </label>
                      <TimezoneSelect
                        value={currentTz}
                        onChange={(tz) => setBhTimezone(tz)}
                      />
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <Button
                      disabled={bhDays === null && bhStart === null && bhEnd === null && bhTimezone === null}
                      onClick={() => {
                        setBhError('')
                        updateProjectMutation.mutate(
                          {
                            business_hours: {
                              days: currentDays,
                              start_hour: currentStart,
                              end_hour: currentEnd,
                              timezone: currentTz,
                            },
                          },
                          {
                            onSuccess: () => {
                              setBhDays(null)
                              setBhStart(null)
                              setBhEnd(null)
                              setBhTimezone(null)
                              showSaved('businessHours')
                            },
                            onError: (err) => {
                              const axiosErr = err as AxiosError<{ error?: { message?: string } }>
                              setBhError(axiosErr.response?.data?.error?.message ?? t('projects.settings.updateError'))
                            },
                          }
                        )
                      }}
                    >
                      {t('common.save')}
                    </Button>
                    {project.business_hours && (
                      <Button
                        variant="secondary"
                        onClick={() => {
                          setBhError('')
                          updateProjectMutation.mutate(
                            { business_hours: null },
                            {
                              onSuccess: () => {
                                setBhDays(null)
                                setBhStart(null)
                                setBhEnd(null)
                                setBhTimezone(null)
                                showSaved('businessHours')
                              },
                              onError: (err) => {
                                const axiosErr = err as AxiosError<{ error?: { message?: string } }>
                                setBhError(axiosErr.response?.data?.error?.message ?? t('projects.settings.updateError'))
                              },
                            }
                          )
                        }}
                      >
                        {t('businessHours.clear')}
                      </Button>
                    )}
                    {saved.businessHours && (
                      <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
                  </div>
                </>
              )
            })()}
          </div>
        </>
      )}

      {/* SLA config modal */}
      {slaModalType && slaModalWorkflowId && (() => {
        const wf = workflows?.find((w) => w.id === slaModalWorkflowId)
        if (!wf) return null
        return (
          <SLAConfigModal
            open
            onClose={() => { setSlaModalType(null); setSlaModalWorkflowId(null) }}
            onSave={() => { showSaved(`wf:${slaModalType}`) }}
            projectKey={projectKey!}
            workItemType={slaModalType}
            workflow={wf}
            hasBusinessHours={!!project?.business_hours}
          />
        )
      })()}

      {/* Workflow Editor Modal */}
      {editorOpen && (
        <WorkflowEditorModal
          projectKey={projectKey ?? ''}
          workflowId={editingWorkflowId}
          onClose={() => {
            setEditorOpen(false)
            setEditingWorkflowId(null)
          }}
          onSuccess={(id) => {
            flashSaved(id)
          }}
          onError={setError}
        />
      )}

      {/* View Workflow Detail Modal */}
      {viewingWorkflowId && (
        <WorkflowDetailModal
          projectKey={projectKey ?? ''}
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

function WorkflowDetailModal({
  projectKey,
  workflowId,
  onClose,
}: {
  projectKey: string
  workflowId: string
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { data: workflow, isLoading } = useProjectWorkflowDetail(projectKey, workflowId)

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
                  {s.display_name}
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
                        {fromStatus?.display_name ?? tr.from_status}
                      </Badge>
                      <ArrowRight className="h-3.5 w-3.5 text-gray-400 shrink-0" />
                      <Badge color={CATEGORY_COLORS[toStatus?.category ?? ''] ?? 'gray'}>
                        {toStatus?.display_name ?? tr.to_status}
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

function WorkflowEditorModal({
  projectKey,
  workflowId,
  onClose,
  onSuccess,
  onError,
}: {
  projectKey: string
  workflowId: string | null
  onClose: () => void
  onSuccess: (msg: string) => void
  onError: (msg: string) => void
}) {
  const { t } = useTranslation()
  const { data: existingWorkflow } = useProjectWorkflowDetail(projectKey, workflowId ?? '')
  const { data: availableStatuses } = useAvailableStatuses(projectKey)
  const createMutation = useCreateProjectWorkflow(projectKey)
  const updateMutation = useUpdateProjectWorkflow(projectKey)

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

  // Group available statuses by category
  const statusPool = availableStatuses ?? []

  function toggleStatus(status: WorkflowStatus) {
    const exists = statuses.find((s) => s.name === status.name)
    if (exists) {
      setStatuses(statuses.filter((s) => s.name !== status.name).map((s, i) => ({ ...s, position: i })))
      // Remove transitions referencing this status
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

    // Ensure position 0 is a todo status
    const sortedStatuses = [...statuses].sort((a, b) => a.position - b.position)
    if (sortedStatuses[0]?.category !== 'todo') {
      // Re-order so first todo is at position 0
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
                  {status.display_name}
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
                    {idx + 1}. {s.display_name}
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
                    className="flex-1 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    value={tr.from_status}
                    onChange={(e) => updateTransition(idx, 'from_status', e.target.value)}
                  >
                    {statuses.map((s) => (
                      <option key={s.name} value={s.name}>{s.display_name}</option>
                    ))}
                  </select>
                  <ArrowRight className="h-3.5 w-3.5 text-gray-400 shrink-0" />
                  <select
                    className="flex-1 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    value={tr.to_status}
                    onChange={(e) => updateTransition(idx, 'to_status', e.target.value)}
                  >
                    {statuses.map((s) => (
                      <option key={s.name} value={s.name}>{s.display_name}</option>
                    ))}
                  </select>
                  <input
                    type="text"
                    className="w-28 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
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

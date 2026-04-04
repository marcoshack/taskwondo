import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useQueue, useUpdateQueue, useDeleteQueue } from '@/hooks/useQueues'
import { useQueueCategories, useCreateCategory, useUpdateCategory, useDeleteCategory, useQueueTeams, useAssignQueueTeam, useUnassignQueueTeam } from '@/hooks/useQueueCategories'
import { useTeams } from '@/hooks/useTeams'
import { useMembers } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Tabs } from '@/components/ui/Tabs'
import { Toggle } from '@/components/ui/Toggle'
import { ArrowLeft, Plus, Pencil, Trash2, Check, X } from 'lucide-react'
import type { QueueCategory, CreateCategoryInput, UpdateCategoryInput } from '@/api/queueCategories'
import type { Team } from '@/api/teams'
import { getLocalizedError } from '@/utils/apiError'

const QUEUE_TYPES = ['support', 'alerts', 'feedback', 'general'] as const
const PRIORITIES = ['critical', 'high', 'medium', 'low'] as const

export function QueueSettingsPage() {
  const { t } = useTranslation()
  const { projectKey, queueId } = useParams<{ projectKey: string; queueId: string }>()
  const navigate = useNavigate()
  const { p } = useNamespacePath()
  const { user } = useAuth()
  const { data: members } = useMembers(projectKey ?? '')
  const { data: queue, isLoading } = useQueue(projectKey ?? '', queueId ?? '')
  const deleteMutation = useDeleteQueue(projectKey ?? '')
  const [activeTab, setActiveTab] = useState('general')
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteError, setDeleteError] = useState('')

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

  if (!queue) {
    return (
      <div className="max-w-3xl">
        <p className="text-red-600 dark:text-red-400">{t('queues.notFound')}</p>
      </div>
    )
  }

  const tabs = [
    { key: 'general', label: t('queues.general') },
    { key: 'categories', label: t('queues.categories') },
    { key: 'teams', label: t('queues.teams') },
  ]

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to={p(`/projects/${projectKey}/queues`)}
          className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{queue.name}</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('queues.settings')}</p>
        </div>
      </div>

      <Tabs tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} />

      <div className="mt-4">
        {activeTab === 'general' && (
          <GeneralTab projectKey={projectKey ?? ''} queue={queue} canManage={canManage} />
        )}
        {activeTab === 'categories' && (
          <CategoriesTab projectKey={projectKey ?? ''} queueId={queueId ?? ''} canManage={canManage} />
        )}
        {activeTab === 'teams' && (
          <TeamsTab projectKey={projectKey ?? ''} queueId={queueId ?? ''} canManage={canManage} />
        )}
      </div>

      {/* Danger Zone */}
      {canManage && (
        <div className="border border-red-200 dark:border-red-800 rounded-lg p-4 mt-8">
          <h3 className="text-sm font-semibold text-red-600 dark:text-red-400 mb-2">{t('queues.dangerZone')}</h3>
          <p className="text-sm text-gray-600 dark:text-gray-300 mb-3">{t('queues.dangerZoneDescription')}</p>
          {deleteError && <p className="text-sm text-red-600 dark:text-red-400 mb-3">{deleteError}</p>}
          <Button variant="danger" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="h-3.5 w-3.5 mr-1" />
            {t('queues.deleteQueue')}
          </Button>
        </div>
      )}

      {/* Delete queue confirmation */}
      <Modal open={deleteOpen} onClose={() => setDeleteOpen(false)} title={t('queues.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="queues.deleteConfirmBody" values={{ name: queue.name }} components={{ bold: <strong /> }} />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setDeleteOpen(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="danger"
            disabled={deleteMutation.isPending}
            onClick={() => {
              setDeleteError('')
              deleteMutation.mutate(queue.id, {
                onSuccess: () => navigate(p(`/projects/${projectKey}/queues`)),
                onError: (err) => {
                  setDeleteError(getLocalizedError(err, t, 'queues.deleteError'))
                  setDeleteOpen(false)
                },
              })
            }}
          >
            {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

// --- General Tab ---

interface QueueData {
  id: string
  name: string
  description: string | null
  queue_type: string
  is_public: boolean
  default_priority: string
}

function GeneralTab({
  projectKey,
  queue,
  canManage,
}: {
  projectKey: string
  queue: QueueData
  canManage: boolean
}) {
  const { t } = useTranslation()
  const updateMutation = useUpdateQueue(projectKey)
  const [name, setName] = useState(queue.name)
  const [description, setDescription] = useState(queue.description ?? '')
  const [queueType, setQueueType] = useState(queue.queue_type)
  const [isPublic, setIsPublic] = useState(queue.is_public)
  const [defaultPriority, setDefaultPriority] = useState(queue.default_priority)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  function handleSave() {
    setError('')
    const input: Record<string, unknown> = {}
    if (name.trim() !== queue.name) input.name = name.trim()
    if (description !== (queue.description ?? '')) input.description = description || null
    if (queueType !== queue.queue_type) input.queue_type = queueType
    if (isPublic !== queue.is_public) input.is_public = isPublic
    if (defaultPriority !== queue.default_priority) input.default_priority = defaultPriority

    if (Object.keys(input).length === 0) return

    updateMutation.mutate(
      { queueId: queue.id, input },
      {
        onSuccess: () => {
          setSaved(true)
          setTimeout(() => setSaved(false), 2000)
        },
        onError: (err) => {
          setError(getLocalizedError(err, t, 'queues.updateError'))
        },
      },
    )
  }

  return (
    <div className="space-y-6">
      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      <Input
        label={t('queues.name')}
        value={name}
        onChange={(e) => setName(e.target.value)}
        disabled={!canManage}
      />

      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('common.description')}
        </label>
        <textarea
          rows={3}
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 disabled:opacity-50"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={!canManage}
        />
      </div>

      <Select
        label={t('queues.type')}
        value={queueType}
        onChange={(e) => setQueueType(e.target.value)}
        disabled={!canManage}
      >
        {QUEUE_TYPES.map((qt) => (
          <option key={qt} value={qt}>{t(`queues.types.${qt}`)}</option>
        ))}
      </Select>

      <Select
        label={t('queues.defaultPriority')}
        value={defaultPriority}
        onChange={(e) => setDefaultPriority(e.target.value)}
        disabled={!canManage}
      >
        {PRIORITIES.map((p) => (
          <option key={p} value={p}>{p.charAt(0).toUpperCase() + p.slice(1)}</option>
        ))}
      </Select>

      <div className="flex items-center gap-3">
        <Toggle enabled={isPublic} onChange={setIsPublic} disabled={!canManage} label={t('queues.isPublic')} />
        <span className="text-sm text-gray-700 dark:text-gray-300">{t('queues.isPublic')}</span>
      </div>

      {canManage && (
        <div className="flex items-center gap-3 pt-2">
          <Button onClick={handleSave} disabled={updateMutation.isPending}>
            {updateMutation.isPending ? t('common.saving') : t('common.save')}
          </Button>
          {saved && <Check className="h-5 w-5 text-green-500" />}
        </div>
      )}
    </div>
  )
}

// --- Categories Tab ---

function CategoriesTab({
  projectKey,
  queueId,
  canManage,
}: {
  projectKey: string
  queueId: string
  canManage: boolean
}) {
  const { t } = useTranslation()
  const { data: categories, isLoading } = useQueueCategories(projectKey, queueId)
  const createMutation = useCreateCategory(projectKey, queueId)
  const updateMutation = useUpdateCategory(projectKey, queueId)
  const deleteMutation = useDeleteCategory(projectKey, queueId)

  const [editorOpen, setEditorOpen] = useState(false)
  const [editingCategory, setEditingCategory] = useState<QueueCategory | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<QueueCategory | null>(null)
  const [error, setError] = useState('')
  const [savedId, setSavedId] = useState<string | null>(null)

  if (isLoading) {
    return <Spinner />
  }

  function flashSaved(id: string) {
    setSavedId(id)
    setTimeout(() => setSavedId(null), 2000)
  }

  function openEditor(category?: QueueCategory) {
    setEditingCategory(category ?? null)
    setEditorOpen(true)
  }

  function handleSave(input: CreateCategoryInput | UpdateCategoryInput) {
    setError('')
    if (editingCategory) {
      const id = editingCategory.id
      updateMutation.mutate(
        { categoryId: id, input: input as UpdateCategoryInput },
        {
          onSuccess: () => {
            flashSaved(id)
            setEditorOpen(false)
            setEditingCategory(null)
          },
          onError: (err) => {
            setError(getLocalizedError(err, t, 'queues.categories.updateError'))
          },
        },
      )
    } else {
      createMutation.mutate(input as CreateCategoryInput, {
        onSuccess: (data) => {
          flashSaved(data.id)
          setEditorOpen(false)
        },
        onError: (err) => {
          setError(getLocalizedError(err, t, 'queues.categories.createError'))
        },
      })
    }
  }

  function handleDelete() {
    if (!deleteTarget) return
    setError('')
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => setDeleteTarget(null),
      onError: (err) => {
        setError(getLocalizedError(err, t, 'queues.categories.deleteError'))
        setDeleteTarget(null)
      },
    })
  }

  const sortedCategories = [...(categories ?? [])].sort((a, b) => a.position - b.position)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('queues.categories')}</h3>
        {canManage && (
          <Button size="sm" variant="secondary" onClick={() => openEditor()}>
            <Plus className="h-4 w-4 mr-1" />
            {t('queues.categories.create')}
          </Button>
        )}
      </div>

      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      {sortedCategories.length === 0 ? (
        <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('queues.categories.noCategories')}</p>
        </div>
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {sortedCategories.map((cat) => (
            <div key={cat.id} className="p-3 flex items-center justify-between gap-2">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-xs text-gray-400 dark:text-gray-500 w-6 text-right shrink-0">#{cat.position}</span>
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{cat.name}</span>
                </div>
                {cat.description && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 ml-8">{cat.description}</p>
                )}
              </div>
              <div className="flex items-center gap-1 shrink-0">
                {savedId === cat.id && <Check className="h-4 w-4 text-green-500" />}
                {canManage && (
                  <>
                    <Button variant="ghost" size="sm" onClick={() => openEditor(cat)}>
                      <Pencil className="h-3.5 w-3.5" />
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(cat)}>
                      <Trash2 className="h-3.5 w-3.5 text-red-500" />
                    </Button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Editor modal */}
      <Modal
        open={editorOpen}
        onClose={() => { setEditorOpen(false); setEditingCategory(null) }}
        title={editingCategory ? t('queues.categories.editCategory') : t('queues.categories.createCategory')}
      >
        <CategoryForm
          category={editingCategory}
          nextPosition={sortedCategories.length + 1}
          onSubmit={handleSave}
          onCancel={() => { setEditorOpen(false); setEditingCategory(null) }}
          isPending={createMutation.isPending || updateMutation.isPending}
        />
      </Modal>

      {/* Delete confirmation */}
      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('queues.categories.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="queues.categories.deleteConfirmBody" values={{ name: deleteTarget?.name }} components={{ bold: <strong /> }} />
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

// --- Category Form ---

function CategoryForm({
  category,
  nextPosition,
  onSubmit,
  onCancel,
  isPending,
}: {
  category: QueueCategory | null
  nextPosition: number
  onSubmit: (input: CreateCategoryInput | UpdateCategoryInput) => void
  onCancel: () => void
  isPending: boolean
}) {
  const { t } = useTranslation()
  const [name, setName] = useState(category?.name ?? '')
  const [description, setDescription] = useState(category?.description ?? '')
  const [position, setPosition] = useState(String(category?.position ?? nextPosition))
  const [validationError, setValidationError] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setValidationError('')

    if (!name.trim()) {
      setValidationError(t('queues.categories.nameRequired'))
      return
    }

    const posNum = parseInt(position, 10)
    if (isNaN(posNum) || posNum < 0) {
      setValidationError(t('queues.categories.positionInvalid'))
      return
    }

    if (category) {
      const input: UpdateCategoryInput = {}
      if (name.trim() !== category.name) input.name = name.trim()
      if (description !== (category.description ?? '')) input.description = description || null
      if (posNum !== category.position) input.position = posNum
      onSubmit(input)
    } else {
      onSubmit({
        name: name.trim(),
        description: description || undefined,
        position: posNum,
      })
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {validationError && (
        <p className="text-sm text-red-600 dark:text-red-400">{validationError}</p>
      )}
      <Input
        label={t('queues.categories.name')}
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder={t('queues.categories.namePlaceholder')}
        required
        autoFocus
      />
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('common.description')}
        </label>
        <textarea
          rows={2}
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <div className="w-32">
        <Input
          label={t('queues.categories.position')}
          type="number"
          min={0}
          value={position}
          onChange={(e) => setPosition(e.target.value)}
        />
      </div>
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={isPending}>
          {isPending ? t('common.saving') : category ? t('common.save') : t('common.create')}
        </Button>
      </div>
    </form>
  )
}

// --- Teams Tab ---

function TeamsTab({
  projectKey,
  queueId,
  canManage,
}: {
  projectKey: string
  queueId: string
  canManage: boolean
}) {
  const { t } = useTranslation()
  const { data: queueTeams, isLoading: loadingQueueTeams } = useQueueTeams(projectKey, queueId)
  const { data: allTeams, isLoading: loadingAllTeams } = useTeams(projectKey)
  const assignMutation = useAssignQueueTeam(projectKey, queueId)
  const unassignMutation = useUnassignQueueTeam(projectKey, queueId)

  const [selectedTeamId, setSelectedTeamId] = useState('')
  const [error, setError] = useState('')
  const [savedId, setSavedId] = useState<string | null>(null)
  const [unassignTarget, setUnassignTarget] = useState<Team | null>(null)

  if (loadingQueueTeams || loadingAllTeams) {
    return <Spinner />
  }

  function flashSaved(id: string) {
    setSavedId(id)
    setTimeout(() => setSavedId(null), 2000)
  }

  const assignedIds = new Set(queueTeams?.map((t) => t.id) ?? [])
  const availableTeams = (allTeams ?? []).filter((t) => !assignedIds.has(t.id))

  function handleAssign() {
    if (!selectedTeamId) return
    setError('')
    assignMutation.mutate(selectedTeamId, {
      onSuccess: () => {
        flashSaved(selectedTeamId)
        setSelectedTeamId('')
      },
      onError: (err) => {
        setError(getLocalizedError(err, t, 'queues.teams.assignError'))
      },
    })
  }

  function handleUnassign() {
    if (!unassignTarget) return
    setError('')
    unassignMutation.mutate(unassignTarget.id, {
      onSuccess: () => setUnassignTarget(null),
      onError: (err) => {
        setError(getLocalizedError(err, t, 'queues.teams.unassignError'))
        setUnassignTarget(null)
      },
    })
  }

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('queues.teams')}</h3>

      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      {/* Assign team */}
      {canManage && availableTeams.length > 0 && (
        <div className="flex items-end gap-2">
          <div className="flex-1">
            <Select
              label={t('queues.teams.assign')}
              value={selectedTeamId}
              onChange={(e) => setSelectedTeamId(e.target.value)}
            >
              <option value="">{t('queues.teams.selectTeam')}</option>
              {availableTeams.map((team) => (
                <option key={team.id} value={team.id}>{team.name}</option>
              ))}
            </Select>
          </div>
          <Button
            onClick={handleAssign}
            disabled={!selectedTeamId || assignMutation.isPending}
          >
            {assignMutation.isPending ? t('common.saving') : t('common.add')}
          </Button>
        </div>
      )}

      {/* Assigned teams list */}
      {(!queueTeams || queueTeams.length === 0) ? (
        <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('queues.teams.noTeams')}</p>
        </div>
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {queueTeams.map((team) => (
            <div key={team.id} className="p-3 flex items-center justify-between gap-2">
              <div className="min-w-0 flex-1">
                <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{team.name}</span>
                {team.description && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{team.description}</p>
                )}
              </div>
              <div className="flex items-center gap-1 shrink-0">
                {savedId === team.id && <Check className="h-4 w-4 text-green-500" />}
                {canManage && (
                  <Button variant="ghost" size="sm" onClick={() => setUnassignTarget(team)}>
                    <X className="h-3.5 w-3.5 text-red-500" />
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Unassign confirmation */}
      <Modal open={!!unassignTarget} onClose={() => setUnassignTarget(null)} title={t('queues.teams.unassignConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="queues.teams.unassignConfirmBody" values={{ name: unassignTarget?.name }} components={{ bold: <strong /> }} />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setUnassignTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button variant="danger" disabled={unassignMutation.isPending} onClick={handleUnassign}>
            {unassignMutation.isPending ? t('common.deleting') : t('queues.teams.unassign')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

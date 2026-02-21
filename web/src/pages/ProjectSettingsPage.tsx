import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useProject, useUpdateProject, useDeleteProject, useMembers, useAddMember, useUpdateMemberRole, useRemoveMember, useTypeWorkflows, useUpdateTypeWorkflow } from '@/hooks/useProjects'
import { useProjectWorkflows } from '@/hooks/useWorkflows'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Tooltip } from '@/components/ui/Tooltip'
import { Check, Trash2 } from 'lucide-react'
import { UserSearchInput } from '@/components/UserSearchInput'
import type { AxiosError } from 'axios'
import type { UserSearchResult } from '@/api/auth'

const ROLE_OPTIONS = ['admin', 'member', 'viewer'] as const
const ROLE_BADGE_COLORS: Record<string, 'indigo' | 'blue' | 'green' | 'gray'> = {
  owner: 'indigo',
  admin: 'blue',
  member: 'green',
  viewer: 'gray',
}

export function ProjectSettingsPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const { data: project, isLoading } = useProject(projectKey ?? '')
  const updateMutation = useUpdateProject(projectKey ?? '')
  const deleteMutation = useDeleteProject(projectKey ?? '')
  const { data: members } = useMembers(projectKey ?? '')
  const addMemberMutation = useAddMember(projectKey ?? '')
  const updateRoleMutation = useUpdateMemberRole(projectKey ?? '')
  const removeMemberMutation = useRemoveMember(projectKey ?? '')
  const { data: typeWorkflows } = useTypeWorkflows(projectKey ?? '')
  const { data: allWorkflows } = useProjectWorkflows(projectKey ?? '')
  const updateTypeWorkflowMutation = useUpdateTypeWorkflow(projectKey ?? '')

  const [name, setName] = useState<string | null>(null)
  const [description, setDescription] = useState<string | null>(null)
  const [saveError, setSaveError] = useState('')

  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')

  // Members state
  const [selectedUser, setSelectedUser] = useState<UserSearchResult | null>(null)
  const [newMemberRole, setNewMemberRole] = useState('member')
  const [memberError, setMemberError] = useState('')
  const [removeTarget, setRemoveTarget] = useState<{ userId: string; name: string } | null>(null)
  const [workflowError, setWorkflowError] = useState('')
  const [complexityInput, setComplexityInput] = useState<string | null>(null)
  const [complexityError, setComplexityError] = useState('')

  // Inline checkmark indicators (keyed by section/item identifier)
  const [saved, setSaved] = useState<Record<string, boolean>>({})
  function showSaved(key: string) {
    setSaved((prev) => ({ ...prev, [key]: true }))
    setTimeout(() => setSaved((prev) => ({ ...prev, [key]: false })), 2000)
  }

  if (isLoading || !project) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    )
  }

  const currentName = name ?? project.name
  const currentDescription = description ?? project.description ?? ''

  const hasChanges =
    (name !== null && name !== project.name) ||
    (description !== null && description !== (project.description ?? ''))

  // Determine current user's role in this project
  const currentUserMember = members?.find((m) => m.user_id === user?.id)
  const currentUserRole = currentUserMember?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canManageMembers = currentUserRole === 'owner' || currentUserRole === 'admin' || user?.global_role === 'admin'
  const isOwner = currentUserRole === 'owner' || user?.global_role === 'admin'

  function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaveError('')

    if (!currentName.trim()) {
      setSaveError(t('projects.settings.nameRequired'))
      return
    }

    const input: Record<string, string | null> = {}
    if (name !== null && name !== project!.name) input.name = name.trim()
    if (description !== null && description !== (project!.description ?? '')) {
      input.description = description.trim() || null
    }

    if (Object.keys(input).length === 0) return

    updateMutation.mutate(input, {
      onSuccess: () => {
        setName(null)
        setDescription(null)
        showSaved('general')
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setSaveError(axiosErr.response?.data?.error?.message ?? t('projects.settings.updateError'))
      },
    })
  }

  function handleDelete() {
    deleteMutation.mutate(undefined, {
      onSuccess: () => {
        navigate('/projects', { replace: true })
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setSaveError(axiosErr.response?.data?.error?.message ?? t('projects.settings.deleteError'))
        setShowDeleteModal(false)
      },
    })
  }

  function handleAddMember() {
    if (!selectedUser) return
    setMemberError('')

    addMemberMutation.mutate(
      { user_id: selectedUser.id, role: newMemberRole },
      {
        onSuccess: () => {
          setSelectedUser(null)
          setNewMemberRole('member')
          showSaved('addMember')
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          setMemberError(axiosErr.response?.data?.error?.message ?? t('projects.settings.addMemberError'))
        },
      },
    )
  }

  function handleRoleChange(userId: string, role: string) {
    setMemberError('')

    updateRoleMutation.mutate(
      { userId, role },
      {
        onSuccess: () => {
          showSaved(`role:${userId}`)
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          setMemberError(axiosErr.response?.data?.error?.message ?? t('projects.settings.updateRoleError'))
        },
      },
    )
  }

  function handleRemoveMember() {
    if (!removeTarget) return
    setMemberError('')

    removeMemberMutation.mutate(removeTarget.userId, {
      onSuccess: () => {
        setRemoveTarget(null)
        if (removeTarget.userId === user?.id) {
          navigate('/projects', { replace: true })
        }
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setMemberError(axiosErr.response?.data?.error?.message ?? t('projects.settings.removeMemberError'))
        setRemoveTarget(null)
      },
    })
  }

  const memberIds = members?.map((m) => m.user_id) ?? []

  return (
    <div className="max-w-3xl space-y-8">
      {/* General section */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('projects.settings.general')}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.description')}</p>
      </div>

      <form onSubmit={handleSave} className="space-y-4">
        <Input label={t('projects.settings.projectKey')} value={project.key} disabled className="bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed" />

        <Input
          label={t('projects.settings.projectName')}
          value={currentName}
          onChange={(e) => setName(e.target.value)}
          required
        />

        <div>
          <label htmlFor="description" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('common.description')}
          </label>
          <textarea
            id="description"
            rows={12}
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            value={currentDescription}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>

        {saveError && <p className="text-sm text-red-600 dark:text-red-400">{saveError}</p>}

        <div className="flex items-center gap-2">
          <Button type="submit" disabled={!hasChanges || updateMutation.isPending}>
            {updateMutation.isPending ? t('common.saving') : t('projects.settings.saveChanges')}
          </Button>
          {saved.general && (
            <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
          )}
        </div>
      </form>

      {/* Members section */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('projects.settings.members')}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.membersDescription')}</p>
      </div>

      {memberError && <p className="text-sm text-red-600 dark:text-red-400">{memberError}</p>}

      {/* Add member form */}
      {canManageMembers && (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('projects.settings.addMember')}</h3>
          <div className="flex gap-2 items-end">
            {selectedUser ? (
              <div className="flex-1 flex items-center gap-2 rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm bg-white dark:bg-gray-800">
                <Avatar name={selectedUser.display_name} size="xs" />
                <span className="text-gray-900 dark:text-gray-100">{selectedUser.display_name}</span>
                <span className="text-gray-400 text-xs">{selectedUser.email}</span>
                <button
                  type="button"
                  className="ml-auto text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  onClick={() => setSelectedUser(null)}
                >
                  &times;
                </button>
              </div>
            ) : (
              <UserSearchInput
                excludeUserIds={memberIds}
                onSelect={setSelectedUser}
              />
            )}
            <select
              className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              value={newMemberRole}
              onChange={(e) => setNewMemberRole(e.target.value)}
            >
              {isOwner && <option value="owner">{t('projects.settings.roles.owner')}</option>}
              {ROLE_OPTIONS.map((role) => (
                <option key={role} value={role}>{t(`projects.settings.roles.${role}`)}</option>
              ))}
            </select>
            <Button
              disabled={!selectedUser || addMemberMutation.isPending}
              onClick={handleAddMember}
            >
              {addMemberMutation.isPending ? t('common.saving') : t('common.add')}
            </Button>
            {saved.addMember && (
              <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
            )}
          </div>
        </div>
      )}

      {/* Member list */}
      {members && members.length > 0 && (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {members.map((member) => {
            const isSelf = member.user_id === user?.id
            const memberIsOwner = member.role === 'owner'
            const ownerCount = members.filter((m) => m.role === 'owner').length
            const isLastOwner = memberIsOwner && ownerCount <= 1
            const canEditRole = canManageMembers && (!memberIsOwner || isOwner)

            return (
              <div key={member.user_id} className="flex items-center justify-between p-3">
                <div className="flex items-center gap-3 min-w-0">
                  <Avatar name={member.display_name} size="sm" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                      {member.display_name}
                      {isSelf && <span className="ml-1 text-xs text-gray-400">({t('common.you')})</span>}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{member.email}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  {canEditRole ? (
                    <>
                      {saved[`role:${member.user_id}`] && (
                        <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                      )}
                      <Tooltip content={isLastOwner ? t('projects.settings.lastOwnerTooltip') : undefined}>
                        <select
                          className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                          value={member.role}
                          onChange={(e) => handleRoleChange(member.user_id, e.target.value)}
                          disabled={updateRoleMutation.isPending || isLastOwner}
                        >
                          {isOwner && <option value="owner">{t('projects.settings.roles.owner')}</option>}
                          {ROLE_OPTIONS.map((role) => (
                            <option key={role} value={role}>{t(`projects.settings.roles.${role}`)}</option>
                          ))}
                        </select>
                      </Tooltip>
                      <Tooltip content={isLastOwner ? t('projects.settings.lastOwnerTooltip') : undefined}>
                        <button
                          className={`p-1 ${isLastOwner ? 'text-gray-300 dark:text-gray-600 cursor-not-allowed' : 'text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300'}`}
                          onClick={() => !isLastOwner && setRemoveTarget({ userId: member.user_id, name: member.display_name })}
                          disabled={isLastOwner}
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </Tooltip>
                    </>
                  ) : (
                    <Badge color={ROLE_BADGE_COLORS[member.role] ?? 'gray'}>
                      {t(`projects.settings.roles.${member.role}`)}
                    </Badge>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Workflows section */}
      {canManageMembers && (
        <>
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('projects.settings.workflows')}</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.workflowsDescription')}</p>
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
                      {allWorkflows?.map((wf) => (
                        <option key={wf.id} value={wf.id}>{wf.name}</option>
                      ))}
                    </select>
                  </div>
                </div>
              )
            })}
          </div>
        </>
      )}

      {/* Complexity section */}
      {canManageMembers && (
        <>
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('projects.settings.complexity')}</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.complexityDescription')}</p>
          </div>

          {complexityError && <p className="text-sm text-red-600 dark:text-red-400">{complexityError}</p>}

          <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3">
            <Input
              label={t('projects.settings.complexityValues')}
              value={complexityInput ?? project.allowed_complexity_values.join(', ')}
              onChange={(e) => setComplexityInput(e.target.value)}
              placeholder={t('projects.settings.complexityPlaceholder')}
            />
            <div className="flex items-center gap-2">
              <Button
                disabled={complexityInput === null || updateMutation.isPending}
                onClick={() => {
                  setComplexityError('')
                  const values = complexityInput
                    ? complexityInput.split(',').map((v) => v.trim()).filter(Boolean).map(Number)
                    : []
                  if (values.some((v) => isNaN(v) || v <= 0 || !Number.isInteger(v))) {
                    setComplexityError(t('projects.settings.complexityInvalid'))
                    return
                  }
                  updateMutation.mutate({ allowed_complexity_values: values }, {
                    onSuccess: () => {
                      setComplexityInput(null)
                      showSaved('complexity')
                    },
                    onError: (err) => {
                      const axiosErr = err as AxiosError<{ error?: { message?: string } }>
                      setComplexityError(axiosErr.response?.data?.error?.message ?? t('projects.settings.updateError'))
                    },
                  })
                }}
              >
                {t('common.save')}
              </Button>
              {project.allowed_complexity_values.length > 0 && (
                <Button
                  variant="secondary"
                  disabled={updateMutation.isPending}
                  onClick={() => {
                    setComplexityError('')
                    updateMutation.mutate({ allowed_complexity_values: [] }, {
                      onSuccess: () => {
                        setComplexityInput(null)
                        showSaved('complexity')
                      },
                      onError: (err) => {
                        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
                        setComplexityError(axiosErr.response?.data?.error?.message ?? t('projects.settings.updateError'))
                      },
                    })
                  }}
                >
                  {t('projects.settings.complexityClear')}
                </Button>
              )}
              {saved.complexity && (
                <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
              )}
            </div>
          </div>
        </>
      )}

      {/* Danger Zone */}
      <div className="border border-red-300 dark:border-red-800 rounded-lg mt-8">
        <div className="px-4 py-3 border-b border-red-300 dark:border-red-800">
          <h3 className="text-base font-semibold text-red-600">{t('projects.settings.dangerZone')}</h3>
        </div>
        <div className="p-4 flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('projects.settings.deleteThisProject')}</p>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {t('projects.settings.deleteWarning')}
            </p>
          </div>
          <Button variant="danger" size="sm" onClick={() => setShowDeleteModal(true)}>
            {t('projects.settings.deleteProject')}
          </Button>
        </div>
      </div>

      {/* Delete confirmation modal */}
      <Modal open={showDeleteModal} onClose={() => setShowDeleteModal(false)} title={t('projects.settings.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="projects.settings.deleteConfirmBody" values={{ projectKey: project.key }} components={{ bold: <strong /> }} />
        </p>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-2">
          <Trans i18nKey="projects.settings.deleteConfirmType" values={{ projectKey: project.key }} components={{ bold: <strong /> }} />
        </p>
        <Input
          value={deleteConfirmText}
          onChange={(e) => setDeleteConfirmText(e.target.value)}
          placeholder={project.key}
        />
        <div className="flex justify-end gap-2 mt-4">
          <Button variant="secondary" onClick={() => setShowDeleteModal(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="danger"
            disabled={deleteConfirmText !== project.key || deleteMutation.isPending}
            onClick={handleDelete}
          >
            {deleteMutation.isPending ? t('common.deleting') : t('projects.settings.deleteConfirmButton')}
          </Button>
        </div>
      </Modal>

      {/* Remove member confirmation modal */}
      <Modal open={!!removeTarget} onClose={() => setRemoveTarget(null)} title={t('projects.settings.removeMemberConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          {removeTarget?.userId === user?.id ? (
            t('projects.settings.removeSelfConfirmBody')
          ) : (
            <Trans
              i18nKey="projects.settings.removeMemberConfirmBody"
              values={{ name: removeTarget?.name }}
              components={{ bold: <strong /> }}
            />
          )}
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setRemoveTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="danger"
            disabled={removeMemberMutation.isPending}
            onClick={handleRemoveMember}
          >
            {removeMemberMutation.isPending ? t('common.deleting') : t('projects.settings.removeMemberConfirmButton')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

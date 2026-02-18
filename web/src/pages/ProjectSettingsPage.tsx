import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useProject, useUpdateProject, useDeleteProject, useMembers, useAddMember, useUpdateMemberRole, useRemoveMember } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
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

  const [name, setName] = useState<string | null>(null)
  const [description, setDescription] = useState<string | null>(null)
  const [saveError, setSaveError] = useState('')
  const [saveSuccess, setSaveSuccess] = useState(false)

  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')

  // Members state
  const [selectedUser, setSelectedUser] = useState<UserSearchResult | null>(null)
  const [newMemberRole, setNewMemberRole] = useState('member')
  const [memberError, setMemberError] = useState('')
  const [memberSuccess, setMemberSuccess] = useState('')
  const [removeTarget, setRemoveTarget] = useState<{ userId: string; name: string } | null>(null)

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
    setSaveSuccess(false)

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
        setSaveSuccess(true)
        setName(null)
        setDescription(null)
        setTimeout(() => setSaveSuccess(false), 3000)
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
    setMemberSuccess('')

    addMemberMutation.mutate(
      { user_id: selectedUser.id, role: newMemberRole },
      {
        onSuccess: () => {
          setMemberSuccess(t('projects.settings.memberAdded'))
          setSelectedUser(null)
          setNewMemberRole('member')
          setTimeout(() => setMemberSuccess(''), 3000)
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
    setMemberSuccess('')

    updateRoleMutation.mutate(
      { userId, role },
      {
        onSuccess: () => {
          setMemberSuccess(t('projects.settings.memberRoleUpdated'))
          setTimeout(() => setMemberSuccess(''), 3000)
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
    setMemberSuccess('')

    removeMemberMutation.mutate(removeTarget.userId, {
      onSuccess: () => {
        setMemberSuccess(t('projects.settings.memberRemoved'))
        setRemoveTarget(null)
        setTimeout(() => setMemberSuccess(''), 3000)
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
    <div className="max-w-2xl space-y-8">
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
        {saveSuccess && <p className="text-sm text-green-600 dark:text-green-400">{t('projects.settings.updateSuccess')}</p>}

        <div className="flex gap-2">
          <Button type="submit" disabled={!hasChanges || updateMutation.isPending}>
            {updateMutation.isPending ? t('common.saving') : t('projects.settings.saveChanges')}
          </Button>
        </div>
      </form>

      {/* Members section */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('projects.settings.members')}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.membersDescription')}</p>
      </div>

      {memberError && <p className="text-sm text-red-600 dark:text-red-400">{memberError}</p>}
      {memberSuccess && <p className="text-sm text-green-600 dark:text-green-400">{memberSuccess}</p>}

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
          </div>
        </div>
      )}

      {/* Member list */}
      {members && members.length > 0 && (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {members.map((member) => {
            const isSelf = member.user_id === user?.id
            const memberIsOwner = member.role === 'owner'

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
                  {canManageMembers && !memberIsOwner ? (
                    <select
                      className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                      value={member.role}
                      onChange={(e) => handleRoleChange(member.user_id, e.target.value)}
                      disabled={updateRoleMutation.isPending}
                    >
                      {isOwner && <option value="owner">{t('projects.settings.roles.owner')}</option>}
                      {ROLE_OPTIONS.map((role) => (
                        <option key={role} value={role}>{t(`projects.settings.roles.${role}`)}</option>
                      ))}
                    </select>
                  ) : (
                    <Badge color={ROLE_BADGE_COLORS[member.role] ?? 'gray'}>
                      {t(`projects.settings.roles.${member.role}`)}
                    </Badge>
                  )}
                  {canManageMembers && (!memberIsOwner || isOwner) && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setRemoveTarget({ userId: member.user_id, name: member.display_name })}
                    >
                      {t('common.remove')}
                    </Button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
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

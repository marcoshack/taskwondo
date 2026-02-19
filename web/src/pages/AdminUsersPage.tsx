import { useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useAdminUsers, useUpdateUser, useUserProjects, useAddUserToProject, useRemoveUserFromProject } from '@/hooks/useAdmin'
import { useProjects } from '@/hooks/useProjects'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import type { AdminUser } from '@/api/admin'
import type { AxiosError } from 'axios'

const ROLE_BADGE_COLORS: Record<string, 'indigo' | 'blue' | 'green' | 'gray'> = {
  owner: 'indigo',
  admin: 'blue',
  member: 'green',
  viewer: 'gray',
}

export function AdminUsersPage() {
  const { t } = useTranslation()
  const { user: currentUser } = useAuth()
  const { data: users, isLoading } = useAdminUsers()
  const updateUserMutation = useUpdateUser()

  const [expandedUserId, setExpandedUserId] = useState<string | null>(null)
  const [feedback, setFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(null)
  const [disableTarget, setDisableTarget] = useState<AdminUser | null>(null)

  function showFeedback(type: 'success' | 'error', message: string) {
    setFeedback({ type, message })
    if (type === 'success') setTimeout(() => setFeedback(null), 3000)
  }

  function handleRoleChange(userId: string, role: string) {
    updateUserMutation.mutate(
      { userId, input: { global_role: role } },
      {
        onSuccess: () => showFeedback('success', t('admin.users.roleUpdated')),
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          showFeedback('error', axiosErr.response?.data?.error?.message ?? t('admin.users.updateError'))
        },
      },
    )
  }

  function handleDisable() {
    if (!disableTarget) return
    updateUserMutation.mutate(
      { userId: disableTarget.id, input: { is_active: false } },
      {
        onSuccess: () => {
          showFeedback('success', t('admin.users.statusUpdated'))
          setDisableTarget(null)
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          showFeedback('error', axiosErr.response?.data?.error?.message ?? t('admin.users.updateError'))
          setDisableTarget(null)
        },
      },
    )
  }

  function handleEnable(userId: string) {
    updateUserMutation.mutate(
      { userId, input: { is_active: true } },
      {
        onSuccess: () => showFeedback('success', t('admin.users.statusUpdated')),
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          showFeedback('error', axiosErr.response?.data?.error?.message ?? t('admin.users.updateError'))
        },
      },
    )
  }

  function formatLastLogin(lastLogin?: string) {
    if (!lastLogin) return t('admin.users.never')
    const date = new Date(lastLogin)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
    if (diffDays === 0) return 'Today'
    if (diffDays === 1) return 'Yesterday'
    if (diffDays < 30) return `${diffDays}d ago`
    return date.toLocaleDateString()
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="max-w-4xl space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('admin.users.title')}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('admin.users.description')}</p>
      </div>

      {feedback && (
        <p className={`text-sm ${feedback.type === 'success' ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
          {feedback.message}
        </p>
      )}

      {!users || users.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">{t('admin.users.noUsers')}</p>
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {users.map((u) => {
            const isSelf = u.id === currentUser?.id
            const isExpanded = expandedUserId === u.id

            return (
              <div key={u.id}>
                <div
                  className="flex items-center justify-between p-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50"
                  onClick={() => setExpandedUserId(isExpanded ? null : u.id)}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <button className="text-gray-400 shrink-0">
                      {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    </button>
                    <Avatar name={u.display_name} size="sm" />
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                        {u.display_name}
                        {isSelf && <span className="ml-1 text-xs text-gray-400">({t('common.you')})</span>}
                      </p>
                      <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{u.email}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3 flex-shrink-0" onClick={(e) => e.stopPropagation()}>
                    {!isSelf ? (
                      <select
                        className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        value={u.global_role}
                        onChange={(e) => handleRoleChange(u.id, e.target.value)}
                        disabled={updateUserMutation.isPending}
                      >
                        <option value="admin">{t('admin.users.roles.admin')}</option>
                        <option value="user">{t('admin.users.roles.user')}</option>
                      </select>
                    ) : (
                      <Badge color={u.global_role === 'admin' ? 'blue' : 'gray'}>
                        {t(`admin.users.roles.${u.global_role}`)}
                      </Badge>
                    )}
                    {u.is_active ? (
                      !isSelf ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setDisableTarget(u)}
                        >
                          {t('admin.users.active')}
                        </Button>
                      ) : (
                        <Badge color="green">{t('admin.users.active')}</Badge>
                      )
                    ) : (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleEnable(u.id)}
                      >
                        <span className="text-red-600 dark:text-red-400">{t('admin.users.disabled')}</span>
                      </Button>
                    )}
                    <span className="text-xs text-gray-400 hidden sm:block w-16 text-right">
                      {formatLastLogin(u.last_login_at)}
                    </span>
                  </div>
                </div>

                {isExpanded && (
                  <UserProjectsPanel userId={u.id} userName={u.display_name} onFeedback={showFeedback} />
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Disable user confirmation modal */}
      <Modal open={!!disableTarget} onClose={() => setDisableTarget(null)} title={t('admin.users.disableConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans
            i18nKey="admin.users.disableConfirmBody"
            values={{ name: disableTarget?.display_name }}
            components={{ bold: <strong /> }}
          />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setDisableTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="danger"
            disabled={updateUserMutation.isPending}
            onClick={handleDisable}
          >
            {updateUserMutation.isPending ? t('common.saving') : t('admin.users.disableConfirmButton')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

function UserProjectsPanel({
  userId,
  userName,
  onFeedback,
}: {
  userId: string
  userName: string
  onFeedback: (type: 'success' | 'error', message: string) => void
}) {
  const { t } = useTranslation()
  const { data: userProjects, isLoading } = useUserProjects(userId)
  const { data: allProjects } = useProjects()
  const addMutation = useAddUserToProject(userId)
  const removeMutation = useRemoveUserFromProject(userId)

  const [selectedProjectId, setSelectedProjectId] = useState('')
  const [selectedRole, setSelectedRole] = useState('member')
  const [removeTarget, setRemoveTarget] = useState<{ projectId: string; projectName: string } | null>(null)

  // Filter out projects the user is already a member of
  const memberProjectIds = new Set(userProjects?.map((p) => p.project_id) ?? [])
  const availableProjects = allProjects?.filter((p) => !memberProjectIds.has(p.id)) ?? []

  function handleAdd() {
    if (!selectedProjectId) return
    addMutation.mutate(
      { project_id: selectedProjectId, role: selectedRole },
      {
        onSuccess: () => {
          onFeedback('success', t('admin.users.addedToProject'))
          setSelectedProjectId('')
          setSelectedRole('member')
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          const msg = axiosErr.response?.status === 409
            ? t('admin.users.alreadyMember')
            : axiosErr.response?.data?.error?.message ?? t('admin.users.addToProjectError')
          onFeedback('error', msg)
        },
      },
    )
  }

  function handleRemove() {
    if (!removeTarget) return
    removeMutation.mutate(removeTarget.projectId, {
      onSuccess: () => {
        onFeedback('success', t('admin.users.removedFromProject'))
        setRemoveTarget(null)
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        onFeedback('error', axiosErr.response?.data?.error?.message ?? t('admin.users.removeFromProjectError'))
        setRemoveTarget(null)
      },
    })
  }

  return (
    <div className="px-3 pb-3 pl-12">
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-3 space-y-3">
        <h4 className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
          {t('admin.users.projects')}
        </h4>

        {isLoading ? (
          <div className="flex justify-center py-2"><Spinner /></div>
        ) : !userProjects || userProjects.length === 0 ? (
          <p className="text-xs text-gray-400">{t('admin.users.noProjects')}</p>
        ) : (
          <div className="space-y-1">
            {userProjects.map((p) => (
              <div key={p.project_id} className="flex items-center justify-between py-1.5">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="inline-flex items-center rounded-md bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 px-2 py-0.5 text-xs font-bold shrink-0">
                    {p.project_key}
                  </span>
                  <span className="text-sm text-gray-900 dark:text-gray-100 truncate">{p.project_name}</span>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <Badge color={ROLE_BADGE_COLORS[p.role] ?? 'gray'}>
                    {t(`projects.settings.roles.${p.role}`)}
                  </Badge>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setRemoveTarget({ projectId: p.project_id, projectName: p.project_name })}
                  >
                    {t('common.remove')}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Add to project form */}
        {availableProjects.length > 0 && (
          <div className="flex gap-2 items-end pt-2 border-t border-gray-100 dark:border-gray-700">
            <select
              className="flex-1 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              value={selectedProjectId}
              onChange={(e) => setSelectedProjectId(e.target.value)}
            >
              <option value="">{t('admin.users.selectProject')}</option>
              {availableProjects.map((p) => (
                <option key={p.id} value={p.id}>{p.key} — {p.name}</option>
              ))}
            </select>
            <select
              className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              value={selectedRole}
              onChange={(e) => setSelectedRole(e.target.value)}
            >
              <option value="owner">{t('projects.settings.roles.owner')}</option>
              <option value="admin">{t('projects.settings.roles.admin')}</option>
              <option value="member">{t('projects.settings.roles.member')}</option>
              <option value="viewer">{t('projects.settings.roles.viewer')}</option>
            </select>
            <Button size="sm" disabled={!selectedProjectId || addMutation.isPending} onClick={handleAdd}>
              {addMutation.isPending ? t('common.saving') : t('common.add')}
            </Button>
          </div>
        )}
      </div>

      {/* Remove from project confirmation modal */}
      <Modal open={!!removeTarget} onClose={() => setRemoveTarget(null)} title={t('admin.users.removeFromProjectTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans
            i18nKey="admin.users.removeFromProjectBody"
            values={{ name: userName, project: removeTarget?.projectName }}
            components={{ bold: <strong /> }}
          />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setRemoveTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="danger"
            disabled={removeMutation.isPending}
            onClick={handleRemove}
          >
            {removeMutation.isPending ? t('common.deleting') : t('admin.users.removeFromProjectButton')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

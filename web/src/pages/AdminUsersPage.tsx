import { useMemo, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { Check, ChevronDown, ChevronRight, Trash2 } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useAdminUsers, useUpdateUser, useUserProjects, useAddUserToProject, useUpdateUserProjectRole, useRemoveUserFromProject } from '@/hooks/useAdmin'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useProjects } from '@/hooks/useProjects'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { MultiSelect } from '@/components/ui/MultiSelect'
import { Spinner } from '@/components/ui/Spinner'
import { Tooltip } from '@/components/ui/Tooltip'
import type { AdminUser } from '@/api/admin'
import type { AxiosError } from 'axios'

const PREFERENCE_KEY = 'admin_users_status_filter'

function parseStatusFilter(v: unknown): string[] {
  if (typeof v === 'string') {
    try { const arr = JSON.parse(v); if (Array.isArray(arr)) return arr } catch { /* ignore */ }
    if (v === 'active' || v === 'disabled') return [v]
  }
  return ['active']
}

export function AdminUsersPage() {
  const { t } = useTranslation()
  const { user: currentUser } = useAuth()
  const { data: users, isLoading } = useAdminUsers()
  const updateUserMutation = useUpdateUser()

  const { data: savedFilter } = usePreference<string>(PREFERENCE_KEY)
  const setPref = useSetPreference()
  const statusFilter = parseStatusFilter(savedFilter)

  function handleStatusFilterChange(selected: string[]) {
    setPref.mutate({ key: PREFERENCE_KEY, value: JSON.stringify(selected) })
  }

  const filteredUsers = useMemo(() => {
    if (!users) return []
    if (statusFilter.length === 0 || statusFilter.length === 2) return users
    if (statusFilter.includes('active')) return users.filter((u) => u.is_active)
    if (statusFilter.includes('disabled')) return users.filter((u) => !u.is_active)
    return users
  }, [users, statusFilter])

  const [expandedUserId, setExpandedUserId] = useState<string | null>(null)
  const [saved, setSaved] = useState<Record<string, boolean>>({})
  const [error, setError] = useState<string | null>(null)
  const [disableTarget, setDisableTarget] = useState<AdminUser | null>(null)

  function showSaved(key: string) {
    setSaved((prev) => ({ ...prev, [key]: true }))
    setTimeout(() => setSaved((prev) => ({ ...prev, [key]: false })), 2000)
  }

  function handleRoleChange(userId: string, role: string) {
    setError(null)
    updateUserMutation.mutate(
      { userId, input: { global_role: role } },
      {
        onSuccess: () => showSaved(`role:${userId}`),
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          setError(axiosErr.response?.data?.error?.message ?? t('admin.users.updateError'))
        },
      },
    )
  }

  function handleDisable() {
    if (!disableTarget) return
    setError(null)
    updateUserMutation.mutate(
      { userId: disableTarget.id, input: { is_active: false } },
      {
        onSuccess: () => {
          showSaved(`status:${disableTarget.id}`)
          setDisableTarget(null)
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          setError(axiosErr.response?.data?.error?.message ?? t('admin.users.updateError'))
          setDisableTarget(null)
        },
      },
    )
  }

  function handleEnable(userId: string) {
    setError(null)
    updateUserMutation.mutate(
      { userId, input: { is_active: true } },
      {
        onSuccess: () => showSaved(`status:${userId}`),
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          setError(axiosErr.response?.data?.error?.message ?? t('admin.users.updateError'))
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
    <div className="max-w-3xl space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('admin.users.title')}</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('admin.users.description')}</p>
        </div>
        <MultiSelect
          options={[
            { value: 'active', label: t('admin.users.filterActive') },
            { value: 'disabled', label: t('admin.users.filterDisabled') },
          ]}
          selected={statusFilter}
          onChange={handleStatusFilterChange}
          placeholder={t('admin.users.title')}
          className="w-44 shrink-0"
        />
      </div>

      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      )}

      {filteredUsers.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">{t('admin.users.noUsers')}</p>
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {filteredUsers.map((u) => {
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
                    {(saved[`role:${u.id}`] || saved[`status:${u.id}`]) && (
                      <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
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
                  <UserProjectsPanel userId={u.id} userName={u.display_name} />
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
}: {
  userId: string
  userName: string
}) {
  const { t } = useTranslation()
  const { data: userProjects, isLoading } = useUserProjects(userId)
  const { data: allProjects } = useProjects()
  const addMutation = useAddUserToProject(userId)
  const updateRoleMutation = useUpdateUserProjectRole(userId)
  const removeMutation = useRemoveUserFromProject(userId)

  const [selectedProjectId, setSelectedProjectId] = useState('')
  const [selectedRole, setSelectedRole] = useState('member')
  const [removeTarget, setRemoveTarget] = useState<{ projectId: string; projectName: string } | null>(null)
  const [saved, setSaved] = useState<Record<string, boolean>>({})
  const [error, setError] = useState<string | null>(null)

  function showSaved(key: string) {
    setSaved((prev) => ({ ...prev, [key]: true }))
    setTimeout(() => setSaved((prev) => ({ ...prev, [key]: false })), 2000)
  }

  // Filter out projects the user is already a member of
  const memberProjectIds = new Set(userProjects?.map((p) => p.project_id) ?? [])
  const availableProjects = allProjects?.filter((p) => !memberProjectIds.has(p.id)) ?? []

  function handleAdd() {
    if (!selectedProjectId) return
    setError(null)
    addMutation.mutate(
      { project_id: selectedProjectId, role: selectedRole },
      {
        onSuccess: () => {
          showSaved('addProject')
          setSelectedProjectId('')
          setSelectedRole('member')
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          const msg = axiosErr.response?.status === 409
            ? t('admin.users.alreadyMember')
            : axiosErr.response?.data?.error?.message ?? t('admin.users.addToProjectError')
          setError(msg)
        },
      },
    )
  }

  function handleRemove() {
    if (!removeTarget) return
    setError(null)
    removeMutation.mutate(removeTarget.projectId, {
      onSuccess: () => {
        setRemoveTarget(null)
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setError(axiosErr.response?.data?.error?.message ?? t('admin.users.removeFromProjectError'))
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

        {error && <p className="text-xs text-red-600 dark:text-red-400">{error}</p>}

        {isLoading ? (
          <div className="flex justify-center py-2"><Spinner /></div>
        ) : !userProjects || userProjects.length === 0 ? (
          <p className="text-xs text-gray-400">{t('admin.users.noProjects')}</p>
        ) : (
          <div className="space-y-1">
            {userProjects.map((p) => {
              const isLastOwner = p.role === 'owner' && p.owner_count <= 1
              return (
                <div key={p.project_id} className="flex items-center justify-between py-1.5">
                  <div className="flex items-center gap-2 min-w-0">
                    <ProjectKeyBadge>{p.project_key}</ProjectKeyBadge>
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{p.project_name}</span>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {saved[`role:${p.project_id}`] && (
                      <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
                    <Tooltip content={isLastOwner ? t('projects.settings.lastOwnerTooltip') : undefined}>
                      <select
                        className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                        value={p.role}
                        onChange={(e) => {
                          setError(null)
                          updateRoleMutation.mutate(
                            { projectId: p.project_id, role: e.target.value },
                            {
                              onSuccess: () => showSaved(`role:${p.project_id}`),
                              onError: (err) => {
                                const axiosErr = err as AxiosError<{ error?: { message?: string } }>
                                setError(axiosErr.response?.data?.error?.message ?? t('admin.users.updateError'))
                              },
                            },
                          )
                        }}
                        disabled={updateRoleMutation.isPending || isLastOwner}
                      >
                        <option value="owner">{t('projects.settings.roles.owner')}</option>
                        <option value="admin">{t('projects.settings.roles.admin')}</option>
                        <option value="member">{t('projects.settings.roles.member')}</option>
                        <option value="viewer">{t('projects.settings.roles.viewer')}</option>
                      </select>
                    </Tooltip>
                    <Tooltip content={isLastOwner ? t('projects.settings.lastOwnerTooltip') : undefined}>
                      <button
                        className={`p-1 ${isLastOwner ? 'text-gray-300 dark:text-gray-600 cursor-not-allowed' : 'text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300'}`}
                        onClick={() => !isLastOwner && setRemoveTarget({ projectId: p.project_id, projectName: p.project_name })}
                        disabled={isLastOwner}
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </Tooltip>
                  </div>
                </div>
              )
            })}
          </div>
        )}

        {/* Add to project form */}
        {availableProjects.length > 0 && (
          <div className="flex gap-2 items-center pt-2 border-t border-gray-100 dark:border-gray-700">
            <select
              className="min-w-0 flex-1 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1.5 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
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
            {saved.addProject && (
              <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
            )}
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

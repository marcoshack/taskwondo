import { useEffect, useMemo, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { Check, ChevronDown, ChevronRight, Copy, KeyRound, Plus, Save, Trash2 } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useAdminUsers, useUpdateUser, useCreateUser, useResetUserPassword, useUserProjects, useAddUserToProject, useUpdateUserProjectRole, useRemoveUserFromProject } from '@/hooks/useAdmin'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useProjects } from '@/hooks/useProjects'
import { useSystemSetting, useSetSystemSetting } from '@/hooks/useSystemSettings'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
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
  const createUserMutation = useCreateUser()
  const resetPasswordMutation = useResetUserPassword()

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
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [createEmail, setCreateEmail] = useState('')
  const [createDisplayName, setCreateDisplayName] = useState('')
  const [createError, setCreateError] = useState<string | null>(null)
  const [revealedPassword, setRevealedPassword] = useState<string | null>(null)
  const [revealedUserName, setRevealedUserName] = useState('')
  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null)
  const [copied, setCopied] = useState(false)

  // Global default project limit
  const { data: savedProjectLimit, isLoading: limitLoading } = useSystemSetting<number>('max_projects_per_user')
  const setSettingMutation = useSetSystemSetting()
  const [projectLimit, setProjectLimit] = useState('')
  const [limitSaved, setLimitSaved] = useState(false)

  useEffect(() => {
    if (savedProjectLimit !== undefined) {
      setProjectLimit(savedProjectLimit != null ? String(savedProjectLimit) : '5')
    }
  }, [savedProjectLimit])

  const limitDirty = projectLimit !== (savedProjectLimit != null ? String(savedProjectLimit) : '5')

  function handleLimitSave() {
    setLimitSaved(false)
    const value = parseInt(projectLimit, 10)
    if (isNaN(value) || value < 0) return
    setSettingMutation.mutate(
      { key: 'max_projects_per_user', value },
      { onSuccess: () => setLimitSaved(true) },
    )
  }

  // Per-user project limit
  const [userLimitInputs, setUserLimitInputs] = useState<Record<string, string>>({})

  function handleUserLimitChange(userId: string, value: string) {
    setUserLimitInputs((prev) => ({ ...prev, [userId]: value }))
  }

  function handleUserLimitSave(userId: string, currentMaxProjects: number | null | undefined) {
    const raw = userLimitInputs[userId]
    if (raw === undefined) return
    const trimmed = raw.trim()

    // Empty means "reset to default" (send -1)
    if (trimmed === '') {
      if (currentMaxProjects == null) return // already default, no-op
      updateUserMutation.mutate(
        { userId, input: { max_projects: -1 } },
        { onSuccess: () => showSaved(`limit:${userId}`) },
      )
      return
    }

    const value = parseInt(trimmed, 10)
    if (isNaN(value) || value < 0) return
    if (currentMaxProjects != null && value === currentMaxProjects) return // unchanged
    updateUserMutation.mutate(
      { userId, input: { max_projects: value } },
      { onSuccess: () => showSaved(`limit:${userId}`) },
    )
  }

  function getUserLimitDisplay(u: AdminUser): string {
    if (userLimitInputs[u.id] !== undefined) return userLimitInputs[u.id]
    if (u.max_projects != null) return String(u.max_projects)
    return ''
  }

  function isUserLimitDirty(u: AdminUser): boolean {
    const raw = userLimitInputs[u.id]
    if (raw === undefined) return false
    const current = u.max_projects != null ? String(u.max_projects) : ''
    return raw !== current
  }

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

  function handleCreateUser() {
    setCreateError(null)
    createUserMutation.mutate(
      { email: createEmail, display_name: createDisplayName },
      {
        onSuccess: (data) => {
          setCreateModalOpen(false)
          setCreateEmail('')
          setCreateDisplayName('')
          setRevealedPassword(data.temporary_password)
          setRevealedUserName(data.user.display_name)
        },
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string; code?: string } }>
          if (axiosErr.response?.status === 409) {
            setCreateError(t('admin.users.emailExists'))
          } else {
            setCreateError(axiosErr.response?.data?.error?.message ?? t('admin.users.createUserError'))
          }
        },
      },
    )
  }

  function handleResetPassword() {
    if (!resetTarget) return
    resetPasswordMutation.mutate(resetTarget.id, {
      onSuccess: (data) => {
        setRevealedPassword(data.temporary_password)
        setRevealedUserName(resetTarget.display_name)
        setResetTarget(null)
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setError(axiosErr.response?.data?.error?.message ?? t('admin.users.resetPasswordError'))
        setResetTarget(null)
      },
    })
  }

  function handleCopyPassword() {
    if (revealedPassword) {
      navigator.clipboard.writeText(revealedPassword)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
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
      <div className="space-y-3 sm:space-y-0 sm:flex sm:items-start sm:justify-between sm:gap-4">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('admin.users.title')}</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('admin.users.description')}</p>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          <MultiSelect
            options={[
              { value: 'active', label: t('admin.users.filterActive') },
              { value: 'disabled', label: t('admin.users.filterDisabled') },
            ]}
            selected={statusFilter}
            onChange={handleStatusFilterChange}
            placeholder={t('admin.users.title')}
            className="w-44"
          />
          <Button size="sm" onClick={() => setCreateModalOpen(true)}>
            <Plus className="h-4 w-4 mr-1" />
            {t('admin.users.createUser')}
          </Button>
        </div>
      </div>

      {/* Global default project limit */}
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-4">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-1">
          {t('admin.general.projectLimit.title')}
        </h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
          {t('admin.general.projectLimit.help')}
        </p>
        <div className="flex items-end gap-3">
          <Input
            label={t('admin.general.projectLimit.label')}
            type="number"
            min={0}
            value={projectLimit}
            onChange={(e) => { setProjectLimit(e.target.value); setLimitSaved(false) }}
            className="w-24"
            disabled={limitLoading}
          />
          <Button
            onClick={handleLimitSave}
            disabled={!limitDirty || setSettingMutation.isPending}
          >
            {setSettingMutation.isPending ? t('common.saving') : t('common.save')}
          </Button>
          {limitSaved && (
            <span className="text-sm text-green-600 dark:text-green-400 pb-2">
              {t('admin.general.projectLimit.saved')}
            </span>
          )}
        </div>
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
                  className="p-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50"
                  onClick={() => setExpandedUserId(isExpanded ? null : u.id)}
                >
                  {/* Row 1: identity */}
                  <div className="flex items-center gap-3 min-w-0">
                    <button className="text-gray-400 shrink-0">
                      {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    </button>
                    <Avatar name={u.display_name} avatarUrl={u.avatar_url} size="sm" />
                    <div className="min-w-0 flex-1">
                      {/* Desktop: name + email on separate lines */}
                      <div className="hidden sm:block">
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                          {u.display_name}
                          {isSelf && <span className="ml-1 text-xs text-gray-400">({t('common.you')})</span>}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{u.email}</p>
                      </div>
                      {/* Mobile: name + email inline, horizontally scrollable */}
                      <div className="sm:hidden overflow-x-auto scrollbar-none">
                        <p className="text-sm whitespace-nowrap">
                          <span className="font-medium text-gray-900 dark:text-gray-100">{u.display_name}</span>
                          {isSelf && <span className="ml-1 text-xs text-gray-400">({t('common.you')})</span>}
                          <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">{u.email}</span>
                        </p>
                      </div>
                    </div>
                    {/* Desktop-only: controls inline on row 1 */}
                    <div className="hidden sm:flex items-center gap-3 flex-shrink-0" onClick={(e) => e.stopPropagation()}>
                      {(saved[`role:${u.id}`] || saved[`status:${u.id}`] || saved[`limit:${u.id}`]) && (
                        <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                      )}
                      {u.global_role !== 'admin' && (
                        <Tooltip content={t('admin.users.maxProjectsHelp')}>
                          <span className="inline-flex items-center gap-1">
                            <input
                              type="number"
                              min={0}
                              className="w-[5.2rem] rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-1.5 py-1 text-xs text-center shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                              placeholder={t('admin.users.maxProjectsDefault')}
                              value={getUserLimitDisplay(u)}
                              onChange={(e) => handleUserLimitChange(u.id, e.target.value)}
                              onKeyDown={(e) => { if (e.key === 'Enter') handleUserLimitSave(u.id, u.max_projects) }}
                            />
                            <button
                              className={`px-1 py-1 rounded-md border ${isUserLimitDirty(u) ? 'border-indigo-400 text-indigo-600 hover:bg-indigo-50 dark:border-indigo-500 dark:text-indigo-400 dark:hover:bg-indigo-900/30' : 'border-gray-300 text-gray-300 cursor-default dark:border-gray-600 dark:text-gray-600'}`}
                              onClick={() => isUserLimitDirty(u) && handleUserLimitSave(u.id, u.max_projects)}
                              disabled={!isUserLimitDirty(u)}
                            >
                              <Save className="h-3.5 w-3.5" />
                            </button>
                          </span>
                        </Tooltip>
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
                      {!isSelf && (
                        <Tooltip content={t('admin.users.resetPasswordButton')}>
                          <button
                            className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            onClick={() => setResetTarget(u)}
                          >
                            <KeyRound className="h-4 w-4" />
                          </button>
                        </Tooltip>
                      )}
                      <span className="text-xs text-gray-400 w-16 text-right">
                        {formatLastLogin(u.last_login_at)}
                      </span>
                    </div>
                  </div>
                  {/* Row 2: mobile-only controls */}
                  <div className="flex sm:hidden items-center gap-2 mt-2 pl-11" onClick={(e) => e.stopPropagation()}>
                    {(saved[`role:${u.id}`] || saved[`status:${u.id}`] || saved[`limit:${u.id}`]) && (
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
                    {!isSelf && (
                      <button
                        className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                        onClick={() => setResetTarget(u)}
                      >
                        <KeyRound className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>

                {isExpanded && (
                  <UserProjectsPanel
                    userId={u.id}
                    userName={u.display_name}
                    mobileProjectLimit={u.global_role !== 'admin' ? {
                      value: getUserLimitDisplay(u),
                      dirty: isUserLimitDirty(u),
                      saved: !!saved[`limit:${u.id}`],
                      onChange: (v: string) => handleUserLimitChange(u.id, v),
                      onSave: () => handleUserLimitSave(u.id, u.max_projects),
                    } : undefined}
                  />
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

      {/* Create user modal */}
      <Modal
        open={createModalOpen}
        onClose={() => { setCreateModalOpen(false); setCreateEmail(''); setCreateDisplayName(''); setCreateError(null) }}
        title={t('admin.users.createUserTitle')}
      >
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          {t('admin.users.createUserDescription')}
        </p>
        <div className="space-y-3">
          <Input
            label={t('admin.users.displayName')}
            value={createDisplayName}
            onChange={(e) => setCreateDisplayName(e.target.value)}
            autoFocus
          />
          <Input
            label={t('admin.users.emailAddress')}
            type="email"
            value={createEmail}
            onChange={(e) => setCreateEmail(e.target.value)}
          />
          {createError && (
            <p className="text-sm text-red-600 dark:text-red-400">{createError}</p>
          )}
        </div>
        <div className="flex justify-end gap-2 mt-4">
          <Button variant="secondary" onClick={() => { setCreateModalOpen(false); setCreateEmail(''); setCreateDisplayName(''); setCreateError(null) }}>
            {t('common.cancel')}
          </Button>
          <Button
            disabled={!createEmail || !createDisplayName || createUserMutation.isPending}
            onClick={handleCreateUser}
          >
            {createUserMutation.isPending ? t('common.saving') : t('admin.users.createUserButton')}
          </Button>
        </div>
      </Modal>

      {/* Temporary password reveal modal */}
      <Modal
        open={!!revealedPassword}
        onClose={() => { setRevealedPassword(null); setCopied(false) }}
        title={t('admin.users.temporaryPasswordTitle')}
      >
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-3">
          {t('admin.users.temporaryPasswordDescription', { name: revealedUserName })}
        </p>
        <div className="flex items-center gap-2 bg-gray-100 dark:bg-gray-800 rounded-lg p-3 font-mono text-sm">
          <span className="flex-1 select-all text-gray-900 dark:text-gray-100">{revealedPassword}</span>
          <button
            className="p-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            onClick={handleCopyPassword}
          >
            {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
          </button>
        </div>
        <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">
          {t('admin.users.temporaryPasswordWarning')}
        </p>
        <div className="flex justify-end mt-4">
          <Button onClick={() => { setRevealedPassword(null); setCopied(false) }}>
            {t('common.close')}
          </Button>
        </div>
      </Modal>

      {/* Reset password confirmation modal */}
      <Modal open={!!resetTarget} onClose={() => setResetTarget(null)} title={t('admin.users.resetPasswordTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans
            i18nKey="admin.users.resetPasswordBody"
            values={{ name: resetTarget?.display_name }}
            components={{ bold: <strong /> }}
          />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setResetTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button
            disabled={resetPasswordMutation.isPending}
            onClick={handleResetPassword}
          >
            {resetPasswordMutation.isPending ? t('common.saving') : t('admin.users.resetPasswordButton')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

interface MobileProjectLimit {
  value: string
  dirty: boolean
  saved: boolean
  onChange: (v: string) => void
  onSave: () => void
}

function UserProjectsPanel({
  userId,
  userName,
  mobileProjectLimit,
}: {
  userId: string
  userName: string
  mobileProjectLimit?: MobileProjectLimit
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
    <div className="px-3 pb-3 pl-12 space-y-3">
      {mobileProjectLimit && (
        <div className="sm:hidden border border-gray-200 dark:border-gray-700 rounded-lg p-3">
          <label className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
            {t('admin.general.projectLimit.label')}
          </label>
          <div className="flex items-center gap-2 mt-2">
            <input
              type="number"
              min={0}
              className="w-20 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1 text-xs text-center shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              placeholder={t('admin.users.maxProjectsDefault')}
              value={mobileProjectLimit.value}
              onChange={(e) => mobileProjectLimit.onChange(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') mobileProjectLimit.onSave() }}
            />
            <button
              className={`px-1 py-1 rounded-md border ${mobileProjectLimit.dirty ? 'border-indigo-400 text-indigo-600 hover:bg-indigo-50 dark:border-indigo-500 dark:text-indigo-400 dark:hover:bg-indigo-900/30' : 'border-gray-300 text-gray-300 cursor-default dark:border-gray-600 dark:text-gray-600'}`}
              onClick={() => mobileProjectLimit.dirty && mobileProjectLimit.onSave()}
              disabled={!mobileProjectLimit.dirty}
            >
              <Save className="h-3.5 w-3.5" />
            </button>
            {mobileProjectLimit.saved && (
              <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
            )}
          </div>
        </div>
      )}
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

import { useMemo, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { Check, ChevronDown, ChevronRight, Copy, KeyRound, Save, Trash2 } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useAdminUsers, useUpdateUser, useCreateUser, useResetUserPassword, useUserProjects, useAddUserToProject, useUpdateUserProjectRole, useRemoveUserFromProject } from '@/hooks/useAdmin'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useProjects } from '@/hooks/useProjects'
import { usePublicSettings } from '@/hooks/useSystemSettings'
import { useDebounce } from '@/hooks/useDebounce'
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
import { isAxiosError } from 'axios'
import { getLocalizedError } from '@/utils/apiError'

const PREFERENCE_KEY = 'admin_users_status_filter'

function parseStatusFilter(v: unknown): string[] {
  if (typeof v === 'string') {
    try { const arr = JSON.parse(v); if (Array.isArray(arr)) return arr } catch { /* ignore */ }
    if (v === 'active' || v === 'disabled') return [v]
  }
  return ['active']
}

const USER_SORT_ACCESSORS: Record<string, (u: AdminUser) => string | number> = {
  name: (u) => u.display_name.toLowerCase(),
  maxProjects: (u) => u.max_projects ?? Number.POSITIVE_INFINITY,
  maxNamespaces: (u) => u.max_namespaces ?? Number.POSITIVE_INFINITY,
  role: (u) => u.global_role,
  status: (u) => (u.is_active ? 0 : 1),
  lastLogin: (u) => u.last_login_at ?? '',
  created: (u) => u.created_at,
}

function compareValues(a: string | number, b: string | number, order: 'asc' | 'desc'): number {
  let cmp: number
  if (typeof a === 'number' && typeof b === 'number') cmp = a - b
  else cmp = String(a).localeCompare(String(b))
  return order === 'asc' ? cmp : -cmp
}

function SortIndicator({ active, direction }: { active: boolean; direction?: 'asc' | 'desc' }) {
  if (!active) {
    return (
      <svg className="w-3 h-3 text-[var(--foreground-secondary)]" viewBox="0 0 10 14" fill="currentColor">
        <path d="M5 0L9 5H1L5 0Z" />
        <path d="M5 14L1 9H9L5 14Z" />
      </svg>
    )
  }
  if (direction === 'asc') {
    return (
      <svg className="w-3 h-3 text-[var(--primary)]" viewBox="0 0 10 7" fill="currentColor">
        <path d="M5 0L10 7H0L5 0Z" />
      </svg>
    )
  }
  return (
    <svg className="w-3 h-3 text-[var(--primary)]" viewBox="0 0 10 7" fill="currentColor">
      <path d="M5 7L0 0H10L5 7Z" />
    </svg>
  )
}

function SortableHeader({
  sortKey,
  active,
  order,
  onSort,
  className,
  children,
  align = 'left',
}: {
  sortKey: string
  active: boolean
  order: 'asc' | 'desc'
  onSort: (key: string) => void
  className?: string
  children: React.ReactNode
  align?: 'left' | 'center' | 'right'
}) {
  const justify = align === 'right' ? 'justify-end' : align === 'center' ? 'justify-center' : 'justify-start'
  return (
    <button
      type="button"
      onClick={() => onSort(sortKey)}
      className={`flex items-center gap-1 ${justify} text-xs font-medium uppercase tracking-wider text-[var(--foreground-muted)] hover:text-[var(--foreground)] cursor-pointer select-none ${className ?? ''}`}
    >
      <span>{children}</span>
      <SortIndicator active={active} direction={active ? order : undefined} />
    </button>
  )
}

function formatCreated(iso: string): string {
  const d = new Date(iso)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleDateString()
}

export function DirectoryUsersTab() {
  const { t } = useTranslation()
  const { user: currentUser } = useAuth()
  const { data: users, isLoading } = useAdminUsers()
  const updateUserMutation = useUpdateUser()
  const createUserMutation = useCreateUser()
  const resetPasswordMutation = useResetUserPassword()
  const { data: publicSettings } = usePublicSettings()
  const namespacesEnabled = publicSettings?.namespaces_enabled === true

  const { data: savedFilter } = usePreference<string>(PREFERENCE_KEY)
  const setPref = useSetPreference()
  const statusFilter = parseStatusFilter(savedFilter)

  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)
  const [sortBy, setSortBy] = useState<string | null>(null)
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')

  function handleStatusFilterChange(selected: string[]) {
    setPref.mutate({ key: PREFERENCE_KEY, value: JSON.stringify(selected) })
  }

  function handleSort(key: string) {
    if (sortBy === key) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortBy(key)
      setSortOrder('asc')
    }
  }

  const filteredUsers = useMemo(() => {
    if (!users) return []
    let result = users
    if (statusFilter.length > 0 && statusFilter.length < 2) {
      if (statusFilter.includes('active')) result = result.filter((u) => u.is_active)
      else if (statusFilter.includes('disabled')) result = result.filter((u) => !u.is_active)
    }
    const q = debouncedSearch.trim().toLowerCase()
    if (q) {
      result = result.filter((u) =>
        u.display_name.toLowerCase().includes(q) || u.email.toLowerCase().includes(q),
      )
    }
    if (sortBy) {
      const accessor = USER_SORT_ACCESSORS[sortBy]
      if (accessor) {
        result = [...result].sort((a, b) => compareValues(accessor(a), accessor(b), sortOrder))
      }
    }
    return result
  }, [users, statusFilter, debouncedSearch, sortBy, sortOrder])

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

  // Per-user project limit
  const [userLimitInputs, setUserLimitInputs] = useState<Record<string, string>>({})

  function handleUserLimitChange(userId: string, value: string) {
    setUserLimitInputs((prev) => ({ ...prev, [userId]: value }))
  }

  function handleUserLimitSave(userId: string, currentMaxProjects: number | null | undefined) {
    const raw = userLimitInputs[userId]
    if (raw === undefined) return
    const trimmed = raw.trim()

    if (trimmed === '') {
      if (currentMaxProjects == null) return
      updateUserMutation.mutate(
        { userId, input: { max_projects: -1 } },
        { onSuccess: () => showSaved(`limit:${userId}`) },
      )
      return
    }

    const value = parseInt(trimmed, 10)
    if (isNaN(value) || value < 0) return
    if (currentMaxProjects != null && value === currentMaxProjects) return
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

  // Per-user namespace limit
  const [userNsLimitInputs, setUserNsLimitInputs] = useState<Record<string, string>>({})

  function handleUserNsLimitChange(userId: string, value: string) {
    setUserNsLimitInputs((prev) => ({ ...prev, [userId]: value }))
  }

  function handleUserNsLimitSave(userId: string, currentMaxNamespaces: number | null | undefined) {
    const raw = userNsLimitInputs[userId]
    if (raw === undefined) return
    const trimmed = raw.trim()

    if (trimmed === '') {
      if (currentMaxNamespaces == null) return
      updateUserMutation.mutate(
        { userId, input: { max_namespaces: -1 } },
        { onSuccess: () => showSaved(`nslimit:${userId}`) },
      )
      return
    }

    const value = parseInt(trimmed, 10)
    if (isNaN(value) || value < 0) return
    if (currentMaxNamespaces != null && value === currentMaxNamespaces) return
    updateUserMutation.mutate(
      { userId, input: { max_namespaces: value } },
      { onSuccess: () => showSaved(`nslimit:${userId}`) },
    )
  }

  function getUserNsLimitDisplay(u: AdminUser): string {
    if (userNsLimitInputs[u.id] !== undefined) return userNsLimitInputs[u.id]
    if (u.max_namespaces != null) return String(u.max_namespaces)
    return ''
  }

  function isUserNsLimitDirty(u: AdminUser): boolean {
    const raw = userNsLimitInputs[u.id]
    if (raw === undefined) return false
    const current = u.max_namespaces != null ? String(u.max_namespaces) : ''
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
          setError(getLocalizedError(err, t, 'admin.users.updateError'))
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
          setError(getLocalizedError(err, t, 'admin.users.updateError'))
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
          setError(getLocalizedError(err, t, 'admin.users.updateError'))
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
          if (isAxiosError(err) && err.response?.status === 409) {
            setCreateError(t('admin.users.emailExists'))
          } else {
            setCreateError(getLocalizedError(err, t, 'admin.users.createUserError'))
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
        setError(getLocalizedError(err, t, 'admin.users.resetPasswordError'))
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
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Input
          placeholder={t('admin.directory.search.users')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-sm"
        />
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
          <Button onClick={() => setCreateModalOpen(true)} className="border border-transparent">
            {t('admin.users.createUser')}
          </Button>
        </div>
      </div>

      {error && (
        <p className="text-sm text-[var(--danger)]">{error}</p>
      )}

      {filteredUsers.length === 0 ? (
        <p className="text-sm text-[var(--foreground-secondary)]">{t('admin.users.noUsers')}</p>
      ) : (
        (() => {
          const dataCols = namespacesEnabled ? 6 : 5
          const gridStyle: React.CSSProperties = {
            gridTemplateColumns: `25% repeat(${dataCols}, minmax(0, 1fr)) 2rem`,
          }
          return (
        <div className="border border-[var(--border)] rounded-lg overflow-hidden">
          {/* Desktop column headers */}
          <div
            className="hidden sm:grid items-center gap-3 px-3 py-2 bg-[var(--surface-secondary)] border-b border-[var(--border)]"
            style={gridStyle}
          >
            <SortableHeader sortKey="name" active={sortBy === 'name'} order={sortOrder} onSort={handleSort}>
              {t('admin.directory.col.user')}
            </SortableHeader>
            <SortableHeader sortKey="maxProjects" active={sortBy === 'maxProjects'} order={sortOrder} onSort={handleSort} align="center">
              {t('admin.directory.col.maxProjects')}
            </SortableHeader>
            {namespacesEnabled && (
              <SortableHeader sortKey="maxNamespaces" active={sortBy === 'maxNamespaces'} order={sortOrder} onSort={handleSort} align="center">
                {t('admin.directory.col.maxNamespaces')}
              </SortableHeader>
            )}
            <SortableHeader sortKey="role" active={sortBy === 'role'} order={sortOrder} onSort={handleSort} align="center">
              {t('admin.directory.col.role')}
            </SortableHeader>
            <SortableHeader sortKey="status" active={sortBy === 'status'} order={sortOrder} onSort={handleSort} align="center">
              {t('admin.directory.col.status')}
            </SortableHeader>
            <SortableHeader sortKey="lastLogin" active={sortBy === 'lastLogin'} order={sortOrder} onSort={handleSort} align="right">
              {t('admin.directory.col.lastLogin')}
            </SortableHeader>
            <SortableHeader sortKey="created" active={sortBy === 'created'} order={sortOrder} onSort={handleSort} align="right">
              {t('admin.directory.col.created')}
            </SortableHeader>
            <span aria-hidden="true" />
          </div>
          <div className="divide-y divide-[var(--border)]">
          {filteredUsers.map((u) => {
            const isSelf = u.id === currentUser?.id
            const isExpanded = expandedUserId === u.id

            return (
              <div key={u.id}>
                {/* Desktop row */}
                <div
                  className="hidden sm:grid items-center gap-3 px-3 py-3 cursor-pointer hover:bg-[var(--surface-hover)]/50"
                  style={gridStyle}
                  onClick={() => setExpandedUserId(isExpanded ? null : u.id)}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <button className="text-[var(--foreground-muted)] shrink-0" type="button">
                      {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    </button>
                    <Avatar name={u.display_name} avatarUrl={u.avatar_url} size="sm" />
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-[var(--foreground)] truncate">
                        {u.display_name}
                        {isSelf && <span className="ml-1 text-xs text-[var(--foreground-muted)]">({t('common.you')})</span>}
                      </p>
                      <p className="text-xs text-[var(--foreground-secondary)] truncate">{u.email}</p>
                    </div>
                  </div>
                  <div className="flex items-center justify-center gap-1 min-w-0" onClick={(e) => e.stopPropagation()}>
                    {u.global_role !== 'admin' && (
                      <Tooltip content={t('admin.users.maxProjectsHelp')}>
                        <span className="inline-flex items-center gap-1">
                          <input
                            type="number"
                            min={0}
                            className="w-14 rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-1 py-1 text-xs text-center focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
                            placeholder={t('admin.users.maxProjectsDefault')}
                            value={getUserLimitDisplay(u)}
                            onChange={(e) => handleUserLimitChange(u.id, e.target.value)}
                            onKeyDown={(e) => { if (e.key === 'Enter') handleUserLimitSave(u.id, u.max_projects) }}
                          />
                          <button
                            className={`px-1 py-1 rounded-md border ${isUserLimitDirty(u) ? 'border-[var(--primary-border)] text-[var(--primary)] hover:bg-[var(--primary-muted)] dark:border-[var(--primary)] dark:text-[var(--primary)] ' : 'border-[var(--border)] text-[var(--foreground-muted)] cursor-default dark:border-[var(--border)] text-[var(--foreground-muted)]'}`}
                            onClick={() => isUserLimitDirty(u) && handleUserLimitSave(u.id, u.max_projects)}
                            disabled={!isUserLimitDirty(u)}
                          >
                            <Save className="h-3.5 w-3.5" />
                          </button>
                        </span>
                      </Tooltip>
                    )}
                    {saved[`limit:${u.id}`] && (
                      <Check className="h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
                  </div>
                  {namespacesEnabled && (
                    <div className="flex items-center justify-center gap-1 min-w-0" onClick={(e) => e.stopPropagation()}>
                      {u.global_role !== 'admin' && (
                        <Tooltip content={t('admin.users.maxNamespacesHelp')}>
                          <span className="inline-flex items-center gap-1">
                            <input
                              type="number"
                              min={0}
                              className="w-14 rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-1 py-1 text-xs text-center focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
                              placeholder={t('admin.users.maxNamespacesDefault')}
                              value={getUserNsLimitDisplay(u)}
                              onChange={(e) => handleUserNsLimitChange(u.id, e.target.value)}
                              onKeyDown={(e) => { if (e.key === 'Enter') handleUserNsLimitSave(u.id, u.max_namespaces) }}
                            />
                            <button
                              className={`px-1 py-1 rounded-md border ${isUserNsLimitDirty(u) ? 'border-[var(--primary-border)] text-[var(--primary)] hover:bg-[var(--primary-muted)] dark:border-[var(--primary)] dark:text-[var(--primary)] ' : 'border-[var(--border)] text-[var(--foreground-muted)] cursor-default dark:border-[var(--border)] text-[var(--foreground-muted)]'}`}
                              onClick={() => isUserNsLimitDirty(u) && handleUserNsLimitSave(u.id, u.max_namespaces)}
                              disabled={!isUserNsLimitDirty(u)}
                            >
                              <Save className="h-3.5 w-3.5" />
                            </button>
                          </span>
                        </Tooltip>
                      )}
                      {saved[`nslimit:${u.id}`] && (
                        <Check className="h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                      )}
                    </div>
                  )}
                  <div className="flex items-center justify-center gap-1 min-w-0" onClick={(e) => e.stopPropagation()}>
                    {!isSelf ? (
                      <select
                        className="rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-2 py-1 text-xs focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
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
                    {saved[`role:${u.id}`] && (
                      <Check className="h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
                  </div>
                  <div className="flex items-center justify-center gap-1 min-w-0" onClick={(e) => e.stopPropagation()}>
                    {u.is_active ? (
                      !isSelf ? (
                        <Button variant="ghost" size="sm" onClick={() => setDisableTarget(u)}>
                          {t('admin.users.active')}
                        </Button>
                      ) : (
                        <Badge color="green">{t('admin.users.active')}</Badge>
                      )
                    ) : (
                      <Button variant="ghost" size="sm" onClick={() => handleEnable(u.id)}>
                        <span className="text-[var(--danger)]">{t('admin.users.disabled')}</span>
                      </Button>
                    )}
                    {saved[`status:${u.id}`] && (
                      <Check className="h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
                  </div>
                  <div className="text-right text-xs text-[var(--foreground-muted)] truncate min-w-0">
                    {formatLastLogin(u.last_login_at)}
                  </div>
                  <div className="text-right text-xs text-[var(--foreground-muted)] truncate min-w-0">
                    {formatCreated(u.created_at)}
                  </div>
                  <div className="flex items-center justify-end" onClick={(e) => e.stopPropagation()}>
                    {!isSelf && (
                      <Tooltip content={t('admin.users.resetPasswordButton')}>
                        <button
                          className="p-1 text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)]"
                          onClick={() => setResetTarget(u)}
                        >
                          <KeyRound className="h-4 w-4" />
                        </button>
                      </Tooltip>
                    )}
                  </div>
                </div>

                {/* Mobile row */}
                <div
                  className="sm:hidden p-3 cursor-pointer hover:bg-[var(--surface-hover)]/50"
                  onClick={() => setExpandedUserId(isExpanded ? null : u.id)}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <button className="text-[var(--foreground-muted)] shrink-0" type="button">
                      {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    </button>
                    <Avatar name={u.display_name} avatarUrl={u.avatar_url} size="sm" />
                    <div className="min-w-0 flex-1 overflow-x-auto scrollbar-none">
                      <p className="text-sm whitespace-nowrap">
                        <span className="font-medium text-[var(--foreground)]">{u.display_name}</span>
                        {isSelf && <span className="ml-1 text-xs text-[var(--foreground-muted)]">({t('common.you')})</span>}
                        <span className="ml-2 text-xs text-[var(--foreground-secondary)]">{u.email}</span>
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 mt-2 pl-11" onClick={(e) => e.stopPropagation()}>
                    {(saved[`role:${u.id}`] || saved[`status:${u.id}`] || saved[`limit:${u.id}`]) && (
                      <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
                    {!isSelf ? (
                      <select
                        className="rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-2 py-1 text-xs focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
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
                        <Button variant="ghost" size="sm" onClick={() => setDisableTarget(u)}>
                          {t('admin.users.active')}
                        </Button>
                      ) : (
                        <Badge color="green">{t('admin.users.active')}</Badge>
                      )
                    ) : (
                      <Button variant="ghost" size="sm" onClick={() => handleEnable(u.id)}>
                        <span className="text-[var(--danger)]">{t('admin.users.disabled')}</span>
                      </Button>
                    )}
                    {!isSelf && (
                      <button
                        className="p-1 text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)]"
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
                    mobileNamespaceLimit={u.global_role !== 'admin' && namespacesEnabled ? {
                      value: getUserNsLimitDisplay(u),
                      dirty: isUserNsLimitDirty(u),
                      saved: !!saved[`nslimit:${u.id}`],
                      onChange: (v: string) => handleUserNsLimitChange(u.id, v),
                      onSave: () => handleUserNsLimitSave(u.id, u.max_namespaces),
                    } : undefined}
                  />
                )}
              </div>
            )
          })}
          </div>
        </div>
          )
        })()
      )}

      {/* Disable user confirmation modal */}
      <Modal open={!!disableTarget} onClose={() => setDisableTarget(null)} title={t('admin.users.disableConfirmTitle')}>
        <p className="text-sm text-[var(--foreground-secondary)] mb-4">
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
        <p className="text-sm text-[var(--foreground-secondary)] mb-4">
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
            <p className="text-sm text-[var(--danger)]">{createError}</p>
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
        <p className="text-sm text-[var(--foreground-secondary)] mb-3">
          {t('admin.users.temporaryPasswordDescription', { name: revealedUserName })}
        </p>
        <div className="flex items-center gap-2 bg-[var(--surface-secondary)] rounded-lg p-3 font-mono text-sm">
          <span className="flex-1 select-all text-[var(--foreground)]">{revealedPassword}</span>
          <button
            className="p-1 text-[var(--foreground-muted)] hover:text-[var(--foreground)]"
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
        <p className="text-sm text-[var(--foreground-secondary)] mb-4">
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
  mobileNamespaceLimit,
}: {
  userId: string
  userName: string
  mobileProjectLimit?: MobileProjectLimit
  mobileNamespaceLimit?: MobileProjectLimit
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

  const memberProjectIds = new Set(userProjects?.map((p) => p.project_id) ?? [])
  const availableProjects = allProjects?.filter((p) => !memberProjectIds.has(p.id)) ?? []
  const selectedProject = availableProjects.find((p) => p.id === selectedProjectId)
  const addRoles = selectedProject?.available_roles ?? []

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
          if (isAxiosError(err) && err.response?.status === 409) {
            setError(t('admin.users.alreadyMember'))
          } else {
            setError(getLocalizedError(err, t, 'admin.users.addToProjectError'))
          }
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
        setError(getLocalizedError(err, t, 'admin.users.removeFromProjectError'))
        setRemoveTarget(null)
      },
    })
  }

  return (
    <div className="px-3 pb-3 pl-12 space-y-3">
      {mobileProjectLimit && (
        <div className="sm:hidden border border-[var(--border)] rounded-lg p-3">
          <label className="text-xs font-semibold text-[var(--foreground-secondary)] uppercase tracking-wider">
            {t('admin.general.projectLimit.label')}
          </label>
          <div className="flex items-center gap-2 mt-2">
            <input
              type="number"
              min={0}
              className="w-14 rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-1 py-1 text-xs text-center focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
              placeholder={t('admin.users.maxProjectsDefault')}
              value={mobileProjectLimit.value}
              onChange={(e) => mobileProjectLimit.onChange(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') mobileProjectLimit.onSave() }}
            />
            <button
              className={`px-1 py-1 rounded-md border ${mobileProjectLimit.dirty ? 'border-[var(--primary-border)] text-[var(--primary)] hover:bg-[var(--primary-muted)] dark:border-[var(--primary)] dark:text-[var(--primary)] ' : 'border-[var(--border)] text-[var(--foreground-muted)] cursor-default dark:border-[var(--border)] text-[var(--foreground-muted)]'}`}
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
      {mobileNamespaceLimit && (
        <div className="sm:hidden border border-[var(--border)] rounded-lg p-3">
          <label className="text-xs font-semibold text-[var(--foreground-secondary)] uppercase tracking-wider">
            {t('admin.general.namespaceLimit.label')}
          </label>
          <div className="flex items-center gap-2 mt-2">
            <input
              type="number"
              min={0}
              className="w-14 rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-1 py-1 text-xs text-center focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
              placeholder={t('admin.users.maxNamespacesDefault')}
              value={mobileNamespaceLimit.value}
              onChange={(e) => mobileNamespaceLimit.onChange(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') mobileNamespaceLimit.onSave() }}
            />
            <button
              className={`px-1 py-1 rounded-md border ${mobileNamespaceLimit.dirty ? 'border-[var(--primary-border)] text-[var(--primary)] hover:bg-[var(--primary-muted)] dark:border-[var(--primary)] dark:text-[var(--primary)] ' : 'border-[var(--border)] text-[var(--foreground-muted)] cursor-default dark:border-[var(--border)] text-[var(--foreground-muted)]'}`}
              onClick={() => mobileNamespaceLimit.dirty && mobileNamespaceLimit.onSave()}
              disabled={!mobileNamespaceLimit.dirty}
            >
              <Save className="h-3.5 w-3.5" />
            </button>
            {mobileNamespaceLimit.saved && (
              <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
            )}
          </div>
        </div>
      )}
      <div className="border border-[var(--border)] rounded-lg p-3 space-y-3">
        <h4 className="text-xs font-semibold text-[var(--foreground-secondary)] uppercase tracking-wider">
          {t('admin.users.projects')}
        </h4>

        {error && <p className="text-xs text-[var(--danger)]">{error}</p>}

        {isLoading ? (
          <div className="flex justify-center py-2"><Spinner /></div>
        ) : !userProjects || userProjects.length === 0 ? (
          <p className="text-xs text-[var(--foreground-muted)]">{t('admin.users.noProjects')}</p>
        ) : (
          <div className="space-y-1">
            {userProjects.map((p) => {
              const isLastOwner = p.role === 'owner' && p.owner_count <= 1
              return (
                <div key={p.project_id} className="flex items-center justify-between py-1.5">
                  <div className="flex items-center gap-2 min-w-0">
                    <ProjectKeyBadge>{p.project_key}</ProjectKeyBadge>
                    <span className="text-sm font-medium text-[var(--foreground)] truncate">{p.project_name}</span>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {saved[`role:${p.project_id}`] && (
                      <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                    )}
                    <Tooltip content={isLastOwner ? t('projects.settings.lastOwnerTooltip') : undefined}>
                      <select
                        className="rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-2 py-1 text-xs focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] disabled:opacity-50 disabled:cursor-not-allowed"
                        value={p.role}
                        onChange={(e) => {
                          setError(null)
                          updateRoleMutation.mutate(
                            { projectId: p.project_id, role: e.target.value },
                            {
                              onSuccess: () => showSaved(`role:${p.project_id}`),
                              onError: (err) => {
                                setError(getLocalizedError(err, t, 'admin.users.updateError'))
                              },
                            },
                          )
                        }}
                        disabled={updateRoleMutation.isPending || isLastOwner}
                      >
                        {p.available_roles.map((role) => (
                          <option key={role} value={role}>{t(`projects.settings.roles.${role}`)}</option>
                        ))}
                      </select>
                    </Tooltip>
                    <Tooltip content={isLastOwner ? t('projects.settings.lastOwnerTooltip') : undefined}>
                      <button
                        className={`p-1 ${isLastOwner ? 'text-[var(--foreground-secondary)] cursor-not-allowed' : 'text-[var(--danger)] hover:text-red-700 text-[var(--danger)] dark:hover:text-red-300'}`}
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
          <div className="flex gap-2 items-center pt-2 border-t border-[var(--border)]">
            <select
              className="min-w-0 flex-1 rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-2 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
              value={selectedProjectId}
              onChange={(e) => setSelectedProjectId(e.target.value)}
            >
              <option value="">{t('admin.users.selectProject')}</option>
              {availableProjects.map((p) => (
                <option key={p.id} value={p.id}>{p.key} — {p.name}</option>
              ))}
            </select>
            <select
              className="rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-2 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
              value={selectedRole}
              onChange={(e) => setSelectedRole(e.target.value)}
            >
              {addRoles.map((role) => (
                <option key={role} value={role}>{t(`projects.settings.roles.${role}`)}</option>
              ))}
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
        <p className="text-sm text-[var(--foreground-secondary)] mb-4">
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

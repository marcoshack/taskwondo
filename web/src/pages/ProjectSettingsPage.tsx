import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { createPortal } from 'react-dom'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { useProject, useUpdateProject, useDeleteProject, useMembers, useAddMember, useUpdateMemberRole, useRemoveMember, useInvites, useCreateInvite, useDeleteInvite } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Tooltip } from '@/components/ui/Tooltip'
import { Tabs } from '@/components/ui/Tabs'
import { Check, Trash2, Copy, Link, AlertTriangle, ArrowRightLeft, ChevronDown, Info, Mail } from 'lucide-react'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useMigrateProject } from '@/hooks/useNamespaces'
import { clearLastProjectKey } from '@/hooks/useLastProjectKey'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { UserSearchInput } from '@/components/UserSearchInput'
import { MentionSearchModal } from '@/components/ui/MentionSearchModal'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import { getLocalizedError } from '@/utils/apiError'
import { TeamsPage } from './TeamsPage'
import type { UserSearchResult } from '@/api/users'

function TruncatedName({ name, className }: { name: string; className?: string }) {
  const ref = useRef<HTMLSpanElement>(null)
  const [isTruncated, setIsTruncated] = useState(false)

  const checkTruncation = useCallback(() => {
    const el = ref.current
    if (el) setIsTruncated(el.scrollWidth > el.clientWidth)
  }, [])

  useEffect(() => {
    checkTruncation()
  }, [name, checkTruncation])

  const span = (
    <span ref={ref} className={className} onMouseEnter={checkTruncation}>
      {name}
    </span>
  )

  if (isTruncated) {
    return <Tooltip content={name}>{span}</Tooltip>
  }
  return span
}

function formatRelativeTime(date: Date): string {
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  if (diffMs <= 0) return ''
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 60) return `${diffMin}m`
  const diffHrs = Math.floor(diffMin / 60)
  if (diffHrs < 24) return `${diffHrs}h`
  const diffDays = Math.floor(diffHrs / 24)
  return `${diffDays}d`
}

const ROLE_BADGE_COLORS: Record<string, 'indigo' | 'blue' | 'green' | 'gray' | 'yellow'> = {
  owner: 'indigo',
  admin: 'blue',
  member: 'green',
  viewer: 'gray',
  customer: 'yellow',
}

// --- Custom role dropdown with info icon and descriptions ---

function RoleSelect({
  value,
  onChange,
  roles,
  isOwner,
  disabled,
  size = 'md',
}: {
  value: string
  onChange: (role: string) => void
  roles: string[]
  isOwner: boolean
  disabled?: boolean
  size?: 'sm' | 'md'
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      const target = e.target as Node
      if (
        ref.current && !ref.current.contains(target) &&
        menuRef.current && !menuRef.current.contains(target)
      ) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  const allRoles = isOwner ? ['owner', ...roles] : roles

  const sizeClasses = size === 'sm'
    ? 'px-2 py-1 text-xs'
    : 'px-3 py-2 text-sm'

  const getMenuPosition = () => {
    const btn = buttonRef.current?.getBoundingClientRect()
    const menu = menuRef.current
    if (!btn) return { top: 0, left: 0 }
    const menuHeight = menu?.offsetHeight ?? 320
    const spaceBelow = window.innerHeight - btn.bottom - 4
    const openAbove = spaceBelow < menuHeight && btn.top > spaceBelow
    return {
      top: openAbove ? btn.top - menuHeight - 4 : btn.bottom + 4,
      left: Math.min(btn.right - 288, window.innerWidth - 296),
    }
  }

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => !disabled && setOpen(!open)}
        disabled={disabled}
        className={`flex items-center gap-1.5 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap ${sizeClasses}`}
        ref={buttonRef}
      >
        <span>{t(`projects.settings.roles.${value}`)}</span>
        <Info className="h-3 w-3 text-gray-400 shrink-0" />
        <ChevronDown className="h-3 w-3 text-gray-400 shrink-0" />
      </button>
      {open && createPortal(
        <RoleMenu
          menuRef={menuRef}
          getPosition={getMenuPosition}
          allRoles={allRoles}
          value={value}
          onChange={onChange}
          setOpen={setOpen}
          t={t}
        />,
        document.body,
      )}
    </div>
  )
}

function RoleMenu({
  menuRef,
  getPosition,
  allRoles,
  value,
  onChange,
  setOpen,
  t,
}: {
  menuRef: React.RefObject<HTMLDivElement | null>
  getPosition: () => { top: number; left: number }
  allRoles: string[]
  value: string
  onChange: (role: string) => void
  setOpen: (open: boolean) => void
  t: (key: string) => string
}) {
  const [pos, setPos] = useState({ top: 0, left: 0 })

  useEffect(() => {
    // Defer position calc to after the menu is rendered and has a measured height
    requestAnimationFrame(() => setPos(getPosition()))
  }, [getPosition])

  return (
    <div
      ref={menuRef}
      className="fixed z-[9999] w-72 rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-lg py-1 max-h-80 overflow-y-auto"
      style={pos}
    >
      {allRoles.map((role) => (
        <button
          key={role}
          type="button"
          onClick={() => { onChange(role); setOpen(false) }}
          className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2 ${
            role === value
              ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
              : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
          }`}
        >
          <div className="flex-1 min-w-0">
            <div className="font-medium">{t(`projects.settings.roles.${role}`)}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400">{t(`projects.settings.roles.${role}.description`)}</div>
          </div>
          <Info className="h-3.5 w-3.5 text-gray-400 shrink-0" />
        </button>
      ))}
    </div>
  )
}

export function ProjectSettingsPage() {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const { projectKey } = useParams<{ projectKey: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { user } = useAuth()
  const { data: project, isLoading } = useProject(projectKey ?? '')
  const updateMutation = useUpdateProject(projectKey ?? '')
  const deleteMutation = useDeleteProject(projectKey ?? '')
  const { data: members, totalCount: membersTotalCount } = useMembers(projectKey ?? '')
  const addMemberMutation = useAddMember(projectKey ?? '')
  const updateRoleMutation = useUpdateMemberRole(projectKey ?? '')
  const removeMemberMutation = useRemoveMember(projectKey ?? '')
  const { data: invites } = useInvites(projectKey ?? '')
  const createInviteMutation = useCreateInvite(projectKey ?? '')
  const deleteInviteMutation = useDeleteInvite(projectKey ?? '')
  const assignableRoles = (project?.available_roles ?? []).filter((r: string) => r !== 'owner')
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
  const [emailInput, setEmailInput] = useState('')
  const [showEmailInviteModal, setShowEmailInviteModal] = useState(false)
  const [emailInviteRole, setEmailInviteRole] = useState('member')
  const [complexityInput, setComplexityInput] = useState<string | null>(null)
  const [complexityError, setComplexityError] = useState('')

  // Invite state
  const [inviteRole, setInviteRole] = useState('member')
  const [inviteExpiration, setInviteExpiration] = useState('')
  const [inviteMaxUses, setInviteMaxUses] = useState('')
  const [inviteError, setInviteError] = useState('')
  const [revokeTarget, setRevokeTarget] = useState<{ id: string; code: string } | null>(null)

  // Namespace transfer state
  const { namespaces, activeNamespace, showSwitcher: hasMultipleNamespaces } = useNamespaceContext()
  const migrateMutation = useMigrateProject()
  const [showTransferModal, setShowTransferModal] = useState(false)
  const [transferTarget, setTransferTarget] = useState('')
  const [transferConfirm, setTransferConfirm] = useState('')
  const [transferDropdownOpen, setTransferDropdownOpen] = useState(false)
  const [transferError, setTransferError] = useState('')
  const transferDropdownRef = useRef<HTMLDivElement>(null)
  const transferButtonRef = useRef<HTMLButtonElement>(null)
  const transferMenuRef = useRef<HTMLDivElement>(null)

  // Close transfer dropdown on outside click
  useEffect(() => {
    if (!transferDropdownOpen) return
    function handleClick(e: MouseEvent) {
      const target = e.target as Node
      if (
        transferDropdownRef.current && !transferDropdownRef.current.contains(target) &&
        transferMenuRef.current && !transferMenuRef.current.contains(target)
      ) {
        setTransferDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [transferDropdownOpen])

  const descRef = useRef<HTMLTextAreaElement>(null)
  const descMention = useMentionAutocomplete({
    value: description ?? project?.description ?? '',
    onValueChange: setDescription,
    textareaRef: descRef,
  })

  // Inline checkmark indicators (keyed by section/item identifier)
  const [saved, setSaved] = useState<Record<string, boolean>>({})
  function showSaved(key: string) {
    setSaved((prev) => ({ ...prev, [key]: true }))
    setTimeout(() => setSaved((prev) => ({ ...prev, [key]: false })), 2000)
  }

  // Tab state from URL search params
  const activeTab = searchParams.get('tab') || 'general'
  function setActiveTab(tab: string) {
    const params = new URLSearchParams(searchParams)
    if (tab === 'general') {
      params.delete('tab')
    } else {
      params.set('tab', tab)
    }
    setSearchParams(params, { replace: true })
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

  const tabs = [
    { key: 'general', label: t('projects.settings.tab.general') },
    { key: 'users', label: t('projects.settings.tab.users') },
    { key: 'teams', label: t('projects.settings.tab.teams') },
    ...(canManageMembers ? [
      { key: 'invites', label: t('projects.settings.tab.invites') },
      { key: 'workItems', label: t('projects.settings.tab.workItems') },
    ] : []),
  ]

  // Fallback to General if the current tab is not available (e.g. permission-gated tabs before data loads)
  const effectiveTab = tabs.some((tb) => tb.key === activeTab) ? activeTab : 'general'

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
        setSaveError(getLocalizedError(err, t, 'projects.settings.updateError'))
      },
    })
  }

  function handleDelete() {
    deleteMutation.mutate(undefined, {
      onSuccess: () => {
        clearLastProjectKey()
        navigate(p('/projects'), { replace: true })
      },
      onError: (err) => {
        setSaveError(getLocalizedError(err, t, 'projects.settings.deleteError'))
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
          setMemberError(getLocalizedError(err, t, 'projects.settings.addMemberError'))
        },
      },
    )
  }

  function handleEmailInvite() {
    if (!emailInput.trim()) return
    setMemberError('')
    setShowEmailInviteModal(false)

    createInviteMutation.mutate(
      { role: emailInviteRole, email: emailInput.trim() },
      {
        onSuccess: (result) => {
          setEmailInput('')
          setEmailInviteRole('member')
          if (result.direct_add) {
            showSaved('addMember')
          } else {
            showSaved('emailInvite')
          }
        },
        onError: (err) => {
          setMemberError(getLocalizedError(err, t, 'projects.settings.emailInviteError'))
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
          setMemberError(getLocalizedError(err, t, 'projects.settings.updateRoleError'))
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
          navigate(p('/projects'), { replace: true })
        }
      },
      onError: (err) => {
        setMemberError(getLocalizedError(err, t, 'projects.settings.removeMemberError'))
        setRemoveTarget(null)
      },
    })
  }

  const memberIds = members?.map((m) => m.user_id) ?? []

  const pendingEmailInvites = useMemo(
    () =>
      invites?.filter(
        (inv) =>
          inv.invitee_email &&
          (inv.max_uses === 0 || inv.use_count < inv.max_uses) &&
          (!inv.expires_at || new Date(inv.expires_at) >= new Date()),
      ) ?? [],
    [invites],
  )

  return (
    <div className="max-w-3xl space-y-6">
      {/* Page Header */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{t('projects.settings.title')}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.subtitle')}</p>
      </div>

      <Tabs tabs={tabs} activeTab={effectiveTab} onTabChange={setActiveTab} />

      {/* ───── General tab ───── */}
      {effectiveTab === 'general' && (
        <div className="space-y-8">
          <form onSubmit={handleSave} className="space-y-4">
            <div className="flex gap-4">
              <div className="w-[30%]">
                <Input label={t('projects.settings.projectKey')} value={project.key} disabled className="bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed" />
              </div>
              <div className="w-[70%]">
                <Input
                  label={t('projects.settings.projectName')}
                  value={currentName}
                  onChange={(e) => setName(e.target.value)}
                  required
                  disabled={!canManageMembers}
                />
              </div>
            </div>

            <div>
              <label htmlFor="description" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('common.description')}
              </label>
              <textarea
                ref={descRef}
                id="description"
                rows={12}
                className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 disabled:opacity-60 disabled:cursor-not-allowed"
                value={currentDescription}
                onChange={(e) => setDescription(e.target.value)}
                onKeyDown={descMention.onMentionKeyDown}
                disabled={!canManageMembers}
              />
              <MentionSearchModal
                open={descMention.mentionModalOpen}
                position={descMention.dropdownPosition}
                onClose={descMention.onMentionClose}
                onSelect={descMention.onMentionSelect}
              />
            </div>

            {saveError && <p className="text-sm text-red-600 dark:text-red-400">{saveError}</p>}

            {canManageMembers && (
              <div className="flex items-center gap-2">
                <Button type="submit" disabled={!hasChanges || updateMutation.isPending}>
                  {updateMutation.isPending ? t('common.saving') : t('projects.settings.saveChanges')}
                </Button>
                {saved.general && (
                  <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                )}
              </div>
            )}
          </form>

          {/* Danger Zone */}
          {canManageMembers && (
            <div className="border border-red-300 dark:border-red-800 rounded-lg">
              <div className="px-4 py-3 border-b border-red-300 dark:border-red-800">
                <h3 className="text-base font-semibold text-red-600">{t('projects.settings.dangerZone')}</h3>
              </div>
              {hasMultipleNamespaces && (
                <div className="p-4 flex items-center justify-between border-b border-red-200 dark:border-red-900">
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('projects.settings.transferNamespace')}</p>
                    <p className="text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.transferNamespaceDescription')}</p>
                  </div>
                  <Button variant="danger" className="min-w-[7.5rem] h-10" onClick={() => { setTransferTarget(''); setTransferConfirm(''); setTransferError(''); setShowTransferModal(true) }}>
                    <ArrowRightLeft className="h-4 w-4 mr-1" />
                    {t('projects.settings.transfer')}
                  </Button>
                </div>
              )}
              <div className="p-4 flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('projects.settings.deleteThisProject')}</p>
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    {t('projects.settings.deleteWarning')}
                  </p>
                </div>
                <Button variant="danger" className="min-w-[7.5rem] h-10" onClick={() => setShowDeleteModal(true)}>
                  {t('projects.settings.deleteProject')}
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ───── Users tab ───── */}
      {effectiveTab === 'users' && (
        <div className="space-y-6">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.membersDescription')}</p>

          {memberError && <p className="text-sm text-red-600 dark:text-red-400">{memberError}</p>}

          {/* Add user form */}
          {canManageMembers && (
            <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3">
              <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('projects.settings.addUser')}</h3>
              <div className="flex gap-2 items-end">
                {selectedUser ? (
                  <div className="flex-1 min-w-0 flex items-center gap-2 rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm bg-white dark:bg-gray-800">
                    <span className="shrink-0"><Avatar name={selectedUser.display_name} avatarUrl={selectedUser.avatar_url} size="xs" /></span>
                    <span className="truncate text-gray-900 dark:text-gray-100">{selectedUser.display_name}</span>
                    <span className="truncate text-gray-400 text-xs hidden sm:inline">{selectedUser.email}</span>
                    <button
                      type="button"
                      className="shrink-0 ml-auto text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
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
                <RoleSelect
                  value={newMemberRole}
                  onChange={setNewMemberRole}
                  roles={assignableRoles}
                  isOwner={isOwner}
                />
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
              <div className="flex items-center gap-2 text-xs text-gray-400 dark:text-gray-500">
                <div className="flex-1 border-t border-gray-200 dark:border-gray-700" />
                <span>{t('common.or')}</span>
                <div className="flex-1 border-t border-gray-200 dark:border-gray-700" />
              </div>
              <div className="flex gap-2 items-end">
                <div className="flex-1 min-w-0">
                  <Input
                    placeholder={t('projects.settings.emailInvitePlaceholder')}
                    value={emailInput}
                    onChange={(e) => setEmailInput(e.target.value)}
                    type="email"
                  />
                </div>
                <RoleSelect
                  value={emailInviteRole}
                  onChange={setEmailInviteRole}
                  roles={assignableRoles}
                  isOwner={false}
                />
                <Button
                  disabled={!emailInput.trim() || createInviteMutation.isPending}
                  onClick={() => setShowEmailInviteModal(true)}
                >
                  {createInviteMutation.isPending ? t('common.saving') : t('projects.settings.emailInviteButton')}
                </Button>
                {saved.emailInvite && (
                  <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                )}
              </div>
            </div>
          )}

          {/* Member list */}
          {((members && members.length > 0) || pendingEmailInvites.length > 0) && (
            <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
              {members?.map((member) => {
                const isSelf = member.user_id === user?.id
                const memberIsOwner = member.role === 'owner'
                const ownerCount = members!.filter((m) => m.role === 'owner').length
                const isLastOwner = memberIsOwner && ownerCount <= 1
                const canEditRole = canManageMembers && (!memberIsOwner || isOwner)

                return (
                  <div key={member.user_id} className="flex flex-wrap items-center justify-between gap-2 p-3">
                    <div className="flex items-center gap-3 min-w-0">
                      <Avatar name={member.display_name} avatarUrl={member.avatar_url} size="sm" />
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                          {member.display_name}
                          {isSelf && <span className="ml-1 text-xs text-gray-400">({t('common.you')})</span>}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{member.email}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0 ml-auto">
                      {canEditRole ? (
                        <>
                          {saved[`role:${member.user_id}`] && (
                            <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                          )}
                          <Tooltip content={isLastOwner ? t('projects.settings.lastOwnerTooltip') : undefined}>
                            <span>
                              <RoleSelect
                                value={member.role}
                                onChange={(role) => handleRoleChange(member.user_id, role)}
                                roles={assignableRoles}
                                isOwner={isOwner}
                                disabled={updateRoleMutation.isPending || isLastOwner}
                                size="sm"
                              />
                            </span>
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
              {members && membersTotalCount != null && membersTotalCount > members.length && (
                <div className="py-2 text-sm text-gray-500 dark:text-gray-400 pl-[60px]">
                  {t('projects.settings.hiddenMembers', { count: membersTotalCount - members.length })}
                </div>
              )}
              {pendingEmailInvites.map((invite) => (
                <div key={invite.id} className="flex flex-wrap items-center justify-between gap-2 p-3">
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="h-8 w-8 rounded-full bg-gray-100 dark:bg-gray-700 flex items-center justify-center flex-shrink-0">
                      <Mail className="h-4 w-4 text-gray-400 dark:text-gray-500" />
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm text-gray-500 dark:text-gray-400 truncate">
                        {invite.invitee_email}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0 ml-auto">
                    <Badge color="yellow">{t('projects.settings.invitedBadge')}</Badge>
                    {canManageMembers ? (
                      <>
                        <Badge color={ROLE_BADGE_COLORS[invite.role] ?? 'gray'}>
                          {t(`projects.settings.roles.${invite.role}`)}
                        </Badge>
                        <button
                          className="p-1 text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                          onClick={() => setRevokeTarget({ id: invite.id, code: invite.code })}
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </>
                    ) : (
                      <Badge color={ROLE_BADGE_COLORS[invite.role] ?? 'gray'}>
                        {t(`projects.settings.roles.${invite.role}`)}
                      </Badge>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ───── Teams tab ───── */}
      {effectiveTab === 'teams' && <TeamsPage />}

      {/* ───── Invites tab ───── */}
      {effectiveTab === 'invites' && (
        <div className="space-y-6">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.invitesDescription')}</p>

          {inviteError && <p className="text-sm text-red-600 dark:text-red-400">{inviteError}</p>}

          {/* Create invite form */}
          <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3">
            <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('projects.settings.createInvite')}</h3>
            <div className="flex gap-2 items-end flex-wrap">
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{t('projects.settings.inviteRole')}</label>
                <select
                  className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  value={inviteRole}
                  onChange={(e) => {
                    const role = e.target.value
                    setInviteRole(role)
                    if (role === 'admin') {
                      setInviteMaxUses('1')
                      setInviteExpiration('1d')
                    }
                  }}
                >
                  {assignableRoles.map((role) => (
                    <option key={role} value={role}>{t(`projects.settings.roles.${role}`)}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{t('projects.settings.inviteExpiration')}</label>
                <select
                  className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  value={inviteExpiration}
                  onChange={(e) => setInviteExpiration(e.target.value)}
                >
                  <option value="">{t('projects.settings.inviteExpiresNever')}</option>
                  <option value="1h">{t('projects.settings.inviteExpires1h')}</option>
                  <option value="1d">{t('projects.settings.inviteExpires1d')}</option>
                  <option value="7d">{t('projects.settings.inviteExpires7d')}</option>
                  <option value="30d">{t('projects.settings.inviteExpires30d')}</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{t('projects.settings.inviteMaxUses')}</label>
                <input
                  type="number"
                  min="0"
                  className="w-24 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  value={inviteMaxUses}
                  onChange={(e) => setInviteMaxUses(e.target.value)}
                  placeholder={t('projects.settings.inviteMaxUsesPlaceholder')}
                />
              </div>
              <Button
                disabled={createInviteMutation.isPending}
                onClick={() => {
                  setInviteError('')
                  createInviteMutation.mutate(
                    {
                      role: inviteRole,
                      expires_in: inviteExpiration || undefined,
                      max_uses: inviteMaxUses ? parseInt(inviteMaxUses, 10) : undefined,
                    },
                    {
                      onSuccess: () => {
                        setInviteMaxUses('')
                        showSaved('createInvite')
                      },
                      onError: () => {
                        setInviteError(t('projects.settings.inviteCreateError'))
                      },
                    },
                  )
                }}
              >
                <Link className="h-4 w-4 mr-1" />
                {createInviteMutation.isPending ? t('common.creating') : t('projects.settings.createInvite')}
              </Button>
              {saved.createInvite && (
                <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
              )}
            </div>
            {inviteRole === 'admin' && (
              <div className="flex items-start gap-2 rounded-md bg-amber-50 dark:bg-amber-900/30 border border-amber-200 dark:border-amber-700 p-3 text-sm text-amber-800 dark:text-amber-200">
                <AlertTriangle className="h-4 w-4 flex-shrink-0 mt-0.5" />
                <span>{t('projects.settings.inviteAdminWarning')}</span>
              </div>
            )}
          </div>

          {/* Invite list */}
          {invites && invites.length > 0 ? (
            <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
              {invites.map((invite) => {
                const isExpired = invite.expires_at && new Date(invite.expires_at) < new Date()

                return (
                  <div key={invite.id} className="p-3 space-y-2">
                    <div className="flex flex-wrap items-center gap-3">
                      <TruncatedName
                        name={invite.created_by_name}
                        className="text-xs font-semibold text-gray-900 dark:text-gray-100 whitespace-nowrap truncate w-[80px] flex-shrink-0 inline-block"
                      />
                      <code className="text-xs bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded text-gray-700 dark:text-gray-300">
                        {invite.code}
                      </code>
                      <Badge color={ROLE_BADGE_COLORS[invite.role] ?? 'gray'}>
                        {t(`projects.settings.roles.${invite.role}`)}
                      </Badge>
                      {invite.invitee_email && (
                        <span className="text-xs text-gray-500 dark:text-gray-400 truncate max-w-[160px]" title={invite.invitee_email}>
                          {invite.invitee_email}
                        </span>
                      )}
                      {isExpired && (
                        <Badge color="gray">{t('projects.settings.inviteExpired')}</Badge>
                      )}
                      <div className="ml-auto flex items-center gap-3 flex-shrink-0">
                        <span className="hidden md:inline text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
                          {invite.max_uses > 0
                            ? t('projects.settings.inviteUsage', { count: invite.use_count, max: invite.max_uses })
                            : t('projects.settings.inviteUsageUnlimited', { count: invite.use_count })}
                        </span>
                        <span className="hidden md:inline text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
                          {invite.expires_at
                            ? (isExpired
                              ? t('projects.settings.inviteExpired')
                              : t('projects.settings.inviteExpiresIn', {
                                  time: formatRelativeTime(new Date(invite.expires_at)),
                                }))
                            : t('projects.settings.inviteNeverExpires')}
                        </span>
                        <span className="w-4 h-4 flex-shrink-0">
                          {saved[`copy:${invite.id}`] && (
                            <Check className="h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                          )}
                        </span>
                        <Tooltip content={t('projects.settings.inviteCopy')}>
                          <button
                            className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            onClick={() => {
                              navigator.clipboard.writeText(invite.url)
                              showSaved(`copy:${invite.id}`)
                            }}
                          >
                            <Copy className="h-4 w-4" />
                          </button>
                        </Tooltip>
                        <button
                          className="p-1 text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                          onClick={() => setRevokeTarget({ id: invite.id, code: invite.code })}
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                    <div className="flex items-center gap-3 md:hidden text-xs text-gray-500 dark:text-gray-400">
                      <span>
                        {invite.max_uses > 0
                          ? t('projects.settings.inviteUsage', { count: invite.use_count, max: invite.max_uses })
                          : t('projects.settings.inviteUsageUnlimited', { count: invite.use_count })}
                      </span>
                      <span>
                        {invite.expires_at
                          ? (isExpired
                            ? t('projects.settings.inviteExpired')
                            : t('projects.settings.inviteExpiresIn', {
                                time: formatRelativeTime(new Date(invite.expires_at)),
                              }))
                          : t('projects.settings.inviteNeverExpires')}
                      </span>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <p className="text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.noInvites')}</p>
          )}
        </div>
      )}

      {/* ───── Work Items tab ───── */}
      {effectiveTab === 'workItems' && (
        <div className="space-y-6">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.complexityDescription')}</p>

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
                      setComplexityError(getLocalizedError(err, t, 'projects.settings.updateError'))
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
                        setComplexityError(getLocalizedError(err, t, 'projects.settings.updateError'))
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
        </div>
      )}

      {/* ───── Modals ───── */}

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
          onChange={(e) => setDeleteConfirmText(e.target.value.toUpperCase())}
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

      {/* Revoke invite confirmation modal */}
      <Modal open={!!revokeTarget} onClose={() => setRevokeTarget(null)} title={t('projects.settings.deleteInviteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          {t('projects.settings.deleteInviteConfirmBody')}
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setRevokeTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="danger"
            disabled={deleteInviteMutation.isPending}
            onClick={() => {
              if (!revokeTarget) return
              setInviteError('')
              deleteInviteMutation.mutate(revokeTarget.id, {
                onSuccess: () => setRevokeTarget(null),
                onError: () => {
                  setInviteError(t('projects.settings.inviteDeleteError'))
                  setRevokeTarget(null)
                },
              })
            }}
          >
            {deleteInviteMutation.isPending ? t('common.deleting') : t('common.delete')}
          </Button>
        </div>
      </Modal>

      {/* Email invite confirmation modal */}
      <Modal open={showEmailInviteModal} onClose={() => setShowEmailInviteModal(false)} title={t('projects.settings.emailInviteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans
            i18nKey="projects.settings.emailInviteConfirmBody"
            values={{ email: emailInput, role: t(`projects.settings.roles.${emailInviteRole}`) }}
            components={{ bold: <strong /> }}
          />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setShowEmailInviteModal(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            disabled={createInviteMutation.isPending}
            onClick={handleEmailInvite}
          >
            {createInviteMutation.isPending ? t('common.saving') : t('projects.settings.emailInviteSend')}
          </Button>
        </div>
      </Modal>

      {/* Namespace transfer modal */}
      <Modal open={showTransferModal} onClose={() => setShowTransferModal(false)} title={t('projects.settings.transferConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="projects.settings.transferConfirmBody" values={{ projectKey: project.key }} components={{ bold: <strong /> }} />
        </p>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-1">
          <Trans i18nKey="projects.settings.transferConfirmType" values={{ projectKey: project.key }} components={{ bold: <strong /> }} />
        </p>
        <input
          type="text"
          value={transferConfirm}
          onChange={(e) => setTransferConfirm(e.target.value.toUpperCase())}
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 mb-4"
          placeholder={project.key}
        />
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('projects.settings.transferTargetNamespace')}</label>
        <div className="mb-4" ref={transferDropdownRef}>
          <button
            ref={transferButtonRef}
            type="button"
            onClick={() => setTransferDropdownOpen(!transferDropdownOpen)}
            className="flex items-center gap-2 w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          >
            {(() => {
              const selected = namespaces.find((ns) => ns.slug === transferTarget)
              return selected ? (
                <>
                  <NamespaceIcon
                    icon={selected.icon}
                    color={selected.color}
                    className="h-4 w-4 shrink-0"
                  />
                  <span className="flex-1 text-left truncate">
                    {selected.display_name}
                  </span>
                  <span className="text-xs text-gray-400 dark:text-gray-500 shrink-0">{selected.slug}</span>
                </>
              ) : (
                <span className="flex-1 text-left text-gray-400 dark:text-gray-500">{t('projects.settings.selectNamespace')}</span>
              )
            })()}
            <ChevronDown className="h-4 w-4 shrink-0 text-gray-400" />
          </button>
          {transferDropdownOpen && createPortal(
            <div
              ref={transferMenuRef}
              className="fixed z-[9999] rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-lg py-1 max-h-60 overflow-y-auto"
              style={{
                top: (transferButtonRef.current?.getBoundingClientRect().bottom ?? 0) + 4,
                left: transferButtonRef.current?.getBoundingClientRect().left ?? 0,
                width: transferButtonRef.current?.getBoundingClientRect().width ?? 'auto',
              }}
            >
              {namespaces
                .filter((ns) => ns.slug !== activeNamespace?.slug)
                .map((ns) => (
                  <button
                    key={ns.slug}
                    type="button"
                    onClick={() => {
                      setTransferTarget(ns.slug)
                      setTransferDropdownOpen(false)
                    }}
                    className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2.5 ${
                      ns.slug === transferTarget
                        ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                        : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
                    }`}
                  >
                    <NamespaceIcon icon={ns.icon} color={ns.color} className="h-4 w-4 shrink-0" />
                    <div className="min-w-0 flex-1">
                      <div className="font-medium truncate">{ns.display_name}</div>
                      {!ns.is_default && <div className="text-xs text-gray-400 dark:text-gray-500">{ns.slug}</div>}
                    </div>
                  </button>
                ))}
            </div>,
            document.body,
          )}
        </div>
        {transferError && <p className="text-sm text-red-600 dark:text-red-400 mb-4">{transferError}</p>}
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setShowTransferModal(false)}>{t('common.cancel')}</Button>
          <Button
            variant="danger"
            disabled={!transferTarget || transferConfirm !== project.key || migrateMutation.isPending}
            onClick={() => {
              if (!transferTarget || !activeNamespace || !projectKey) return
              setTransferError('')
              migrateMutation.mutate(
                { fromSlug: activeNamespace.slug, projectKey, targetSlug: transferTarget },
                {
                  onSuccess: () => {
                    setShowTransferModal(false)
                    clearLastProjectKey()
                    navigate(p('/projects'), { replace: true })
                  },
                  onError: (err) => {
                    setTransferError(getLocalizedError(err, t, 'projects.settings.transferError'))
                  },
                },
              )
            }}
          >
            {migrateMutation.isPending ? t('common.saving') : t('projects.settings.transferConfirmButton')}
          </Button>
        </div>
      </Modal>

    </div>
  )
}

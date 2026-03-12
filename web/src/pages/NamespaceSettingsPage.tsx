import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { toUrlSegment, fromUrlSegment } from '@/hooks/useNamespacePath'
import { Check, Trash2 } from 'lucide-react'
import { NamespaceIcon, NAMESPACE_ICONS, NAMESPACE_COLORS, getColorClasses, getIconComponent } from '@/components/NamespaceIcon'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useNamespace, useUpdateNamespace, useDeleteNamespace, useNamespaceMembers, useAddNamespaceMember, useUpdateNamespaceMemberRole, useRemoveNamespaceMember } from '@/hooks/useNamespaces'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { AppSidebar } from '@/components/AppSidebar'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Tooltip } from '@/components/ui/Tooltip'
import { UserSearchInput } from '@/components/UserSearchInput'
import type { AxiosError } from 'axios'
import type { UserSearchResult } from '@/api/users'

const ROLE_OPTIONS = ['admin', 'member'] as const
const ROLE_BADGE_COLORS: Record<string, 'indigo' | 'blue' | 'green' | 'gray'> = {
  owner: 'indigo',
  admin: 'blue',
  member: 'green',
}

export function NamespaceSettingsPage() {
  const { t } = useTranslation()
  const { namespace: urlNamespace } = useParams<{ namespace: string }>()
  const slug = fromUrlSegment(urlNamespace ?? 'd')
  const navigate = useNavigate()
  const { user } = useAuth()
  const { collapsed } = useSidebar('app')
  const { containerClass } = useLayout()
  const { setActiveNamespace, namespaces } = useNamespaceContext()

  const { data: namespace, isLoading } = useNamespace(slug ?? '')
  const updateMutation = useUpdateNamespace(slug ?? '')
  const deleteMutation = useDeleteNamespace()
  const { data: members } = useNamespaceMembers(slug ?? '')
  const addMemberMutation = useAddNamespaceMember(slug ?? '')
  const updateRoleMutation = useUpdateNamespaceMemberRole(slug ?? '')
  const removeMemberMutation = useRemoveNamespaceMember(slug ?? '')

  const [displayName, setDisplayName] = useState<string | null>(null)
  const [slugInput, setSlugInput] = useState<string | null>(null)
  const [iconInput, setIconInput] = useState<string | null>(null)
  const [colorInput, setColorInput] = useState<string | null>(null)
  const [saveError, setSaveError] = useState('')
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')

  // Members state
  const [selectedUser, setSelectedUser] = useState<UserSearchResult | null>(null)
  const [newMemberRole, setNewMemberRole] = useState('member')
  const [memberError, setMemberError] = useState('')
  const [removeTarget, setRemoveTarget] = useState<{ userId: string; name: string } | null>(null)

  const [saved, setSaved] = useState<Record<string, boolean>>({})
  function showSaved(key: string) {
    setSaved((prev) => ({ ...prev, [key]: true }))
    setTimeout(() => setSaved((prev) => ({ ...prev, [key]: false })), 2000)
  }

  if (isLoading || !namespace) {
    return (
      <div className={`${containerClass(true)} py-6`}>
        <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
          <AppSidebar />
          <div className="flex-1 min-w-0 flex items-center justify-center py-24">
            <Spinner size="lg" />
          </div>
        </div>
      </div>
    )
  }

  const currentDisplayName = displayName ?? namespace.display_name
  const currentSlug = slugInput ?? namespace.slug
  const currentIcon = iconInput ?? namespace.icon
  const currentColor = colorInput ?? namespace.color
  const hasChanges =
    (displayName !== null && displayName !== namespace.display_name) ||
    (slugInput !== null && slugInput !== namespace.slug) ||
    (iconInput !== null && iconInput !== namespace.icon) ||
    (colorInput !== null && colorInput !== namespace.color)

  // Determine current user's role
  const currentUserMember = members?.find((m) => m.user_id === user?.id)
  const currentUserRole = currentUserMember?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canManage = currentUserRole === 'owner' || currentUserRole === 'admin' || user?.global_role === 'admin'
  const isOwner = currentUserRole === 'owner' || user?.global_role === 'admin'

  function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaveError('')

    const input: Record<string, string> = {}
    if (displayName !== null && displayName !== namespace!.display_name) input.display_name = displayName.trim()
    if (slugInput !== null && slugInput !== namespace!.slug) input.slug = slugInput.trim()
    if (iconInput !== null && iconInput !== namespace!.icon) input.icon = iconInput
    if (colorInput !== null && colorInput !== namespace!.color) input.color = colorInput
    if (Object.keys(input).length === 0) return

    updateMutation.mutate(input, {
      onSuccess: (updated) => {
        setDisplayName(null)
        setSlugInput(null)
        setIconInput(null)
        setColorInput(null)
        showSaved('general')
        if (updated.slug !== slug) {
          navigate(`/${toUrlSegment(updated.slug)}/settings`, { replace: true })
        }
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setSaveError(axiosErr.response?.data?.error?.message ?? t('namespaces.updateError'))
      },
    })
  }

  function handleDelete() {
    deleteMutation.mutate(slug!, {
      onSuccess: () => {
        // Switch to default namespace (setActiveNamespace navigates to projects)
        const defaultNs = namespaces.find((ns) => ns.is_default)
        if (defaultNs) setActiveNamespace(defaultNs.slug)
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setSaveError(axiosErr.response?.data?.error?.message ?? t('namespaces.deleteError'))
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
          setMemberError(axiosErr.response?.data?.error?.message ?? t('namespaces.addMemberError'))
        },
      },
    )
  }

  function handleRoleChange(userId: string, role: string) {
    setMemberError('')
    updateRoleMutation.mutate(
      { userId, role },
      {
        onSuccess: () => showSaved(`role:${userId}`),
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          setMemberError(axiosErr.response?.data?.error?.message ?? t('namespaces.updateRoleError'))
        },
      },
    )
  }

  function handleRemoveMember() {
    if (!removeTarget) return
    setMemberError('')
    removeMemberMutation.mutate(removeTarget.userId, {
      onSuccess: () => setRemoveTarget(null),
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setMemberError(axiosErr.response?.data?.error?.message ?? t('namespaces.removeMemberError'))
        setRemoveTarget(null)
      },
    })
  }

  const memberIds = members?.map((m) => m.user_id) ?? []

  return (
    <div className={`${containerClass(true)} py-6`}>
      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <AppSidebar />
        <div className="flex-1 min-w-0 max-w-3xl space-y-8">
          {/* General section */}
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('namespaces.settingsTitle')}</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('namespaces.settingsDescription')}</p>
          </div>

          <form onSubmit={handleSave} className="space-y-4">
            <Input
              label={t('namespaces.displayName')}
              value={currentDisplayName}
              onChange={(e) => setDisplayName(e.target.value)}
              required
              disabled={!canManage}
            />
            <Input
              label={t('namespaces.slug')}
              value={currentSlug}
              onChange={(e) => setSlugInput(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
              required
              disabled={!canManage}
            />
            <p className="text-xs text-gray-400 dark:text-gray-500 -mt-3">{t('namespaces.slugHint')}</p>

            {/* Icon & Color pickers + Preview */}
            <div className="flex flex-col sm:flex-row gap-4 sm:gap-6 sm:items-start">
              <div className="flex-1 min-w-0 space-y-4">
                {/* Icon picker */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{t('namespaces.icon')}</label>
                  <div className="overflow-x-auto pb-1 sm:overflow-x-visible sm:pb-0">
                    <div className="grid grid-rows-3 grid-flow-col auto-cols-min gap-2 sm:grid-rows-none sm:grid-flow-row sm:grid-cols-10 sm:gap-1.5">
                      {NAMESPACE_ICONS.map((iconName) => {
                        const Icon = getIconComponent(iconName)
                        const selected = currentIcon === iconName
                        return (
                          <button
                            key={iconName}
                            type="button"
                            disabled={!canManage}
                            onClick={() => setIconInput(iconName)}
                            className={`p-2 rounded-lg border-2 transition-all ${
                              selected
                                ? `border-current ${getColorClasses(currentColor).text} bg-gray-100 dark:bg-gray-700`
                                : 'border-transparent text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800'
                            } disabled:opacity-50 disabled:cursor-not-allowed`}
                          >
                            <Icon className="h-5 w-5" />
                          </button>
                        )
                      })}
                    </div>
                  </div>
                </div>

                {/* Color picker */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{t('namespaces.color')}</label>
                  <div className="flex flex-wrap gap-1.5">
                    {NAMESPACE_COLORS.map((c) => {
                      const selected = currentColor === c
                      return (
                        <button
                          key={c}
                          type="button"
                          disabled={!canManage}
                          onClick={() => setColorInput(c)}
                          className={`w-8 h-8 rounded-full transition-all ${getColorClasses(c).bg} ${
                            selected ? 'ring-2 ring-offset-2 ring-offset-white dark:ring-offset-gray-900 ' + getColorClasses(c).ring : 'opacity-60 hover:opacity-100'
                          } disabled:cursor-not-allowed`}
                          aria-label={c}
                        />
                      )
                    })}
                  </div>
                </div>
              </div>

              {/* Preview */}
              <div className="shrink-0">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{t('namespaces.preview')}</label>
                <div className="flex flex-col items-center gap-3 rounded-lg border border-gray-200 dark:border-gray-700 px-6 py-5 bg-white dark:bg-gray-800">
                  <NamespaceIcon icon={currentIcon} color={currentColor} className="h-10 w-10" />
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{currentDisplayName}</span>
                </div>
              </div>
            </div>

            {saveError && <p className="text-sm text-red-600 dark:text-red-400">{saveError}</p>}

            {canManage && (
              <div className="flex items-center gap-2">
                <Button type="submit" disabled={!hasChanges || updateMutation.isPending}>
                  {updateMutation.isPending ? t('common.saving') : t('common.save')}
                </Button>
                {saved.general && (
                  <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
                )}
              </div>
            )}
          </form>

          {/* Members section */}
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('namespaces.members')}</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('namespaces.membersDescription')}</p>
          </div>

          {memberError && <p className="text-sm text-red-600 dark:text-red-400">{memberError}</p>}

          {canManage && (
            <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3">
              <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('namespaces.addMember')}</h3>
              <div className="flex gap-2 items-end">
                {selectedUser ? (
                  <div className="flex-1 min-w-0 flex items-center gap-2 rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm bg-white dark:bg-gray-800">
                    <span className="shrink-0"><Avatar name={selectedUser.display_name} avatarUrl={selectedUser.avatar_url} size="xs" /></span>
                    <span className="truncate text-gray-900 dark:text-gray-100">{selectedUser.display_name}</span>
                    <button type="button" className="shrink-0 ml-auto text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" onClick={() => setSelectedUser(null)}>&times;</button>
                  </div>
                ) : (
                  <UserSearchInput excludeUserIds={memberIds} onSelect={setSelectedUser} />
                )}
                <select
                  className="shrink-0 rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  value={newMemberRole}
                  onChange={(e) => setNewMemberRole(e.target.value)}
                >
                  {isOwner && <option value="owner">{t('namespaces.roles.owner')}</option>}
                  {ROLE_OPTIONS.map((role) => (
                    <option key={role} value={role}>{t(`namespaces.roles.${role}`)}</option>
                  ))}
                </select>
                <Button disabled={!selectedUser || addMemberMutation.isPending} onClick={handleAddMember}>
                  {addMemberMutation.isPending ? t('common.saving') : t('common.add')}
                </Button>
                {saved.addMember && <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />}
              </div>
            </div>
          )}

          {members && members.length > 0 && (
            <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
              {members.map((member) => {
                const isSelf = member.user_id === user?.id
                const memberIsOwner = member.role === 'owner'
                const ownerCount = members.filter((m) => m.role === 'owner').length
                const isLastOwner = memberIsOwner && ownerCount <= 1
                const canEditRole = canManage && (!memberIsOwner || isOwner)

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
                          {saved[`role:${member.user_id}`] && <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />}
                          <Tooltip content={isLastOwner ? t('namespaces.lastOwnerTooltip') : undefined}>
                            <select
                              className="rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-2 py-1 text-xs shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                              value={member.role}
                              onChange={(e) => handleRoleChange(member.user_id, e.target.value)}
                              disabled={updateRoleMutation.isPending || isLastOwner}
                            >
                              {isOwner && <option value="owner">{t('namespaces.roles.owner')}</option>}
                              {ROLE_OPTIONS.map((role) => (
                                <option key={role} value={role}>{t(`namespaces.roles.${role}`)}</option>
                              ))}
                            </select>
                          </Tooltip>
                          <Tooltip content={isLastOwner ? t('namespaces.lastOwnerTooltip') : undefined}>
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
                          {t(`namespaces.roles.${member.role}`)}
                        </Badge>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}

          {/* Danger Zone */}
          {isOwner && (
            <div className="border border-red-300 dark:border-red-800 rounded-lg mt-8">
              <div className="px-4 py-3 border-b border-red-300 dark:border-red-800">
                <h3 className="text-base font-semibold text-red-600">{t('namespaces.dangerZone')}</h3>
              </div>
              <div className="p-4 flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('namespaces.deleteTitle')}</p>
                  <p className="text-sm text-gray-500 dark:text-gray-400">{t('namespaces.deleteWarning')}</p>
                </div>
                <Button variant="danger" size="sm" onClick={() => setShowDeleteModal(true)}>
                  {t('namespaces.delete')}
                </Button>
              </div>
            </div>
          )}

          {/* Delete confirmation modal */}
          <Modal open={showDeleteModal} onClose={() => setShowDeleteModal(false)} title={t('namespaces.deleteConfirmTitle')}>
            <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
              <Trans i18nKey="namespaces.deleteConfirmBody" values={{ name: namespace.display_name }} components={{ bold: <strong /> }} />
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-300 mb-2">
              <Trans i18nKey="namespaces.deleteConfirmType" values={{ slug: namespace.slug }} components={{ bold: <strong /> }} />
            </p>
            <Input
              value={deleteConfirmText}
              onChange={(e) => setDeleteConfirmText(e.target.value)}
              placeholder={namespace.slug}
            />
            <div className="flex justify-end gap-2 mt-4">
              <Button variant="secondary" onClick={() => setShowDeleteModal(false)}>{t('common.cancel')}</Button>
              <Button
                variant="danger"
                disabled={deleteConfirmText !== namespace.slug || deleteMutation.isPending}
                onClick={handleDelete}
              >
                {deleteMutation.isPending ? t('common.deleting') : t('namespaces.deleteConfirmButton')}
              </Button>
            </div>
          </Modal>

          {/* Remove member confirmation modal */}
          <Modal open={!!removeTarget} onClose={() => setRemoveTarget(null)} title={t('namespaces.removeMemberConfirmTitle')}>
            <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
              <Trans i18nKey="namespaces.removeMemberConfirmBody" values={{ name: removeTarget?.name }} components={{ bold: <strong /> }} />
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={() => setRemoveTarget(null)}>{t('common.cancel')}</Button>
              <Button variant="danger" disabled={removeMemberMutation.isPending} onClick={handleRemoveMember}>
                {removeMemberMutation.isPending ? t('common.deleting') : t('common.remove')}
              </Button>
            </div>
          </Modal>
        </div>
      </div>
    </div>
  )
}

import { useState, useRef, useEffect } from 'react'
import { useParams, Link, useNavigate, useSearchParams } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import {
  useTeam,
  useUpdateTeam,
  useDeleteTeam,
  useTeamMembers,
  useAddTeamMember,
  useRemoveTeamMember,
} from '@/hooks/useTeams'
import { useMembers } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Tabs } from '@/components/ui/Tabs'
import { Avatar } from '@/components/ui/Avatar'
import { ArrowLeft, UserPlus, Trash2, Check } from 'lucide-react'
import { getLocalizedError } from '@/utils/apiError'
import type { UpdateTeamInput } from '@/api/teams'
import { OncallTab } from '@/components/OncallTab'

export function TeamDetailPage() {
  const { t } = useTranslation()
  const { projectKey, teamId } = useParams<{ projectKey: string; teamId: string }>()
  const { p } = useNamespacePath()
  const { user } = useAuth()
  const { data: members } = useMembers(projectKey ?? '')
  const { data: team, isLoading } = useTeam(projectKey ?? '', teamId ?? '')
  const [searchParams, setSearchParams] = useSearchParams()
  const activeTab = searchParams.get('tab') || 'members'
  function setActiveTab(tab: string) {
    const params = new URLSearchParams(searchParams)
    if (tab === 'members') {
      params.delete('tab')
    } else {
      params.set('tab', tab)
    }
    setSearchParams(params, { replace: true })
  }

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

  if (!team) {
    return (
      <div className="max-w-3xl">
        <p className="text-red-600 dark:text-red-400">{t('teams.notFound')}</p>
      </div>
    )
  }

  const tabs = [
    { key: 'members', label: t('teams.members') },
    { key: 'oncall', label: t('teams.oncall.title') },
    { key: 'settings', label: t('teams.settings') },
  ]

  return (
    <div className="space-y-6">
      <div className="max-w-3xl">
        <Link
          to={p(`/projects/${projectKey}/settings?tab=teams`)}
          className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          {t('sidebar.settings')} / {t('teams.title')}
        </Link>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mt-1">{team.name}</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">{t('teams.detail')}</p>
      </div>

      <Tabs tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} />

      <div className="mt-4">
        {activeTab === 'oncall' ? (
          <OncallTab projectKey={projectKey ?? ''} teamId={teamId ?? ''} canManage={canManage} />
        ) : (
          <div className="max-w-3xl">
            {activeTab === 'members' && (
              <MembersTab projectKey={projectKey ?? ''} teamId={teamId ?? ''} canManage={canManage} />
            )}
            {activeTab === 'settings' && (
              <SettingsTab projectKey={projectKey ?? ''} team={team} canManage={canManage} />
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// --- Members Tab ---

function MembersTab({
  projectKey,
  teamId,
  canManage,
}: {
  projectKey: string
  teamId: string
  canManage: boolean
}) {
  const { t } = useTranslation()
  const { data: teamMembers, isLoading } = useTeamMembers(projectKey, teamId)
  const { data: projectMembers } = useMembers(projectKey)
  const addMutation = useAddTeamMember(projectKey, teamId)
  const removeMutation = useRemoveTeamMember(projectKey, teamId)
  const [search, setSearch] = useState('')
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const [savedUserId, setSavedUserId] = useState<string | null>(null)
  const [removeTarget, setRemoveTarget] = useState<{ userId: string; displayName: string } | null>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  function flashSaved(userId: string) {
    setSavedUserId(userId)
    setTimeout(() => setSavedUserId(null), 2000)
  }

  const teamMemberIds = new Set(teamMembers?.map((m) => m.user_id) ?? [])
  const availableMembers = (projectMembers ?? []).filter(
    (pm) => !teamMemberIds.has(pm.user_id) && pm.role !== 'viewer' && pm.role !== 'customer',
  )
  const filteredAvailable = availableMembers.filter((pm) => {
    if (!search) return true
    const q = search.toLowerCase()
    return pm.display_name.toLowerCase().includes(q) || pm.email.toLowerCase().includes(q)
  })

  function handleAdd(userId: string) {
    addMutation.mutate(userId, {
      onSuccess: () => {
        flashSaved(userId)
        setDropdownOpen(false)
        setSearch('')
      },
    })
  }

  function handleRemove() {
    if (!removeTarget) return
    removeMutation.mutate(removeTarget.userId, {
      onSuccess: () => {
        setRemoveTarget(null)
      },
    })
  }

  if (isLoading) {
    return <Spinner />
  }

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('teams.members')}</h3>

      {canManage && (
        <div className="flex items-center gap-2" ref={dropdownRef}>
          <div className="relative flex-1">
            <Input
              placeholder={t('teams.searchMembersPlaceholder')}
              value={search}
              onChange={(e) => { setSearch(e.target.value); setDropdownOpen(true) }}
              onFocus={() => setDropdownOpen(true)}
              className="h-10"
            />
            {dropdownOpen && search && (
              <div className="absolute z-10 mt-1 w-full border border-gray-200 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 shadow-lg">
                <ul className="max-h-48 overflow-auto py-1">
                  {filteredAvailable.map((pm) => (
                    <li key={pm.user_id}>
                      <button
                        type="button"
                        className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2"
                        onClick={() => handleAdd(pm.user_id)}
                        disabled={addMutation.isPending}
                      >
                        <Avatar name={pm.display_name} avatarUrl={pm.avatar_url} size="xs" />
                        <div className="min-w-0 flex-1">
                          <div className="font-medium text-gray-900 dark:text-gray-100">{pm.display_name}</div>
                          <div className="text-xs text-gray-400">{pm.email}</div>
                        </div>
                      </button>
                    </li>
                  ))}
                  {filteredAvailable.length === 0 && (
                    <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">{t('common.noResults')}</li>
                  )}
                </ul>
              </div>
            )}
          </div>
          <Button
            className="h-10 shrink-0"
            onClick={() => { if (filteredAvailable.length === 1) handleAdd(filteredAvailable[0].user_id) }}
            disabled={addMutation.isPending || filteredAvailable.length !== 1}
          >
            <UserPlus className="h-4 w-4 sm:mr-1.5" />
            <span className="hidden sm:inline">{t('teams.addMember')}</span>
          </Button>
        </div>
      )}

      {teamMembers && teamMembers.length > 0 ? (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {teamMembers.map((member) => (
            <div key={member.user_id} className="p-3 flex items-center justify-between gap-2">
              <div className="flex items-center gap-3 min-w-0">
                {savedUserId === member.user_id && <Check className="h-4 w-4 text-green-500 shrink-0" />}
                <Avatar name={member.display_name} avatarUrl={member.avatar_url ?? undefined} size="sm" />
                <div className="min-w-0">
                  <div className="text-sm font-medium text-gray-900 dark:text-gray-100">{member.display_name}</div>
                  <div className="text-xs text-gray-400 dark:text-gray-500">{member.email}</div>
                </div>
              </div>
              {canManage && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setRemoveTarget({ userId: member.user_id, displayName: member.display_name })}
                >
                  <Trash2 className="h-3.5 w-3.5 text-red-500" />
                </Button>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('teams.noMembers')}</p>
        </div>
      )}

      {/* Remove member confirmation */}
      <Modal open={!!removeTarget} onClose={() => setRemoveTarget(null)} title={t('teams.removeMemberConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="teams.removeMemberConfirmBody" values={{ name: removeTarget?.displayName }} components={{ bold: <strong /> }} />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setRemoveTarget(null)}>
            {t('common.cancel')}
          </Button>
          <Button variant="danger" disabled={removeMutation.isPending} onClick={handleRemove}>
            {removeMutation.isPending ? t('common.deleting') : t('common.remove')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

// --- Settings Tab ---

function SettingsTab({
  projectKey,
  team,
  canManage,
}: {
  projectKey: string
  team: { id: string; name: string; description: string | null }
  canManage: boolean
}) {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const navigate = useNavigate()
  const updateMutation = useUpdateTeam(projectKey)
  const deleteMutation = useDeleteTeam(projectKey)
  const [name, setName] = useState(team.name)
  const [description, setDescription] = useState(team.description ?? '')
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  function handleSave() {
    setError('')
    const input: UpdateTeamInput = {}
    if (name.trim() !== team.name) input.name = name.trim()
    if (description !== (team.description ?? '')) input.description = description || null

    if (Object.keys(input).length === 0) return

    updateMutation.mutate(
      { teamId: team.id, input },
      {
        onSuccess: () => {
          setSaved(true)
          setTimeout(() => setSaved(false), 2000)
        },
        onError: (err) => {
          setError(getLocalizedError(err, t, 'teams.updateError'))
        },
      },
    )
  }

  function handleDelete() {
    setError('')
    deleteMutation.mutate(team.id, {
      onSuccess: () => {
        navigate(p(`/projects/${projectKey}/settings?tab=teams`))
      },
      onError: (err) => {
        setError(getLocalizedError(err, t, 'teams.deleteError'))
        setDeleteOpen(false)
      },
    })
  }

  return (
    <div className="space-y-6">
      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      <Input
        label={t('teams.name')}
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

      {canManage && (
        <div className="flex items-center gap-3 pt-2">
          <Button onClick={handleSave} disabled={updateMutation.isPending}>
            {updateMutation.isPending ? t('common.saving') : t('common.save')}
          </Button>
          {saved && <Check className="h-5 w-5 text-green-500" />}
        </div>
      )}

      {/* Danger Zone */}
      {canManage && (
        <div className="border border-red-200 dark:border-red-800 rounded-lg p-4 mt-8">
          <h3 className="text-sm font-semibold text-red-600 dark:text-red-400 mb-2">{t('teams.dangerZone')}</h3>
          <p className="text-sm text-gray-600 dark:text-gray-300 mb-3">{t('teams.dangerZoneDescription')}</p>
          <Button variant="danger" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="h-3.5 w-3.5 mr-1" />
            {t('teams.deleteTeam')}
          </Button>
        </div>
      )}

      {/* Delete confirmation */}
      <Modal open={deleteOpen} onClose={() => setDeleteOpen(false)} title={t('teams.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="teams.deleteConfirmBody" values={{ name: team.name }} components={{ bold: <strong /> }} />
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setDeleteOpen(false)}>
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

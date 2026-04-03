import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import {
  useTeams,
  useCreateTeam,
  useUpdateTeam,
  useDeleteTeam,
  useTeamMembers,
  useAddTeamMember,
  useRemoveTeamMember,
} from '@/hooks/useTeams'
import { useMembers } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Avatar } from '@/components/ui/Avatar'
import { Plus, Pencil, Trash2, ChevronDown, ChevronRight, Check, UserPlus, X } from 'lucide-react'
import type { Team, CreateTeamInput, UpdateTeamInput } from '@/api/teams'
import { getLocalizedError } from '@/utils/apiError'

export function TeamsPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const { user } = useAuth()
  const { data: members } = useMembers(projectKey ?? '')
  const { data: teams, isLoading } = useTeams(projectKey ?? '')

  const createMutation = useCreateTeam(projectKey ?? '')
  const updateMutation = useUpdateTeam(projectKey ?? '')
  const deleteMutation = useDeleteTeam(projectKey ?? '')

  const [editorOpen, setEditorOpen] = useState(false)
  const [editingTeam, setEditingTeam] = useState<Team | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Team | null>(null)
  const [expandedTeamId, setExpandedTeamId] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [savedId, setSavedId] = useState<string | null>(null)

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

  function openEditor(team?: Team) {
    setEditingTeam(team ?? null)
    setEditorOpen(true)
  }

  function flashSaved(id: string) {
    setSavedId(id)
    setTimeout(() => setSavedId(null), 2000)
  }

  function handleSave(input: CreateTeamInput | UpdateTeamInput) {
    setError('')
    if (editingTeam) {
      const id = editingTeam.id
      updateMutation.mutate(
        { teamId: id, input: input as UpdateTeamInput },
        {
          onSuccess: () => {
            flashSaved(id)
            setEditorOpen(false)
            setEditingTeam(null)
          },
          onError: (err) => {
            setError(getLocalizedError(err, t, 'teams.updateError'))
          },
        },
      )
    } else {
      createMutation.mutate(input as CreateTeamInput, {
        onSuccess: (data) => {
          flashSaved(data.id)
          setEditorOpen(false)
        },
        onError: (err) => {
          setError(getLocalizedError(err, t, 'teams.createError'))
        },
      })
    }
  }

  function handleDelete() {
    if (!deleteTarget) return
    setError('')
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => {
        setDeleteTarget(null)
      },
      onError: (err) => {
        setError(getLocalizedError(err, t, 'teams.deleteError'))
        setDeleteTarget(null)
      },
    })
  }

  function toggleExpanded(teamId: string) {
    setExpandedTeamId((prev) => (prev === teamId ? null : teamId))
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('teams.title')}</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('teams.description')}</p>
        </div>
        {canManage && (
          <Button onClick={() => openEditor()} className="border border-transparent">
            <Plus className="h-4 w-4 mr-1" />
            {t('teams.create')}
          </Button>
        )}
      </div>

      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      {(!teams || teams.length === 0) ? (
        <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('teams.noTeams')}</p>
          {canManage && (
            <Button size="sm" variant="secondary" className="mt-3" onClick={() => openEditor()}>
              <Plus className="h-4 w-4 mr-1" />
              {t('teams.createFirst')}
            </Button>
          )}
        </div>
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
          {teams.map((team) => (
            <div key={team.id}>
              <div className="p-4">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <button
                        className="flex items-center gap-1 text-base font-semibold text-gray-900 dark:text-gray-100 hover:text-indigo-600 dark:hover:text-indigo-400"
                        onClick={() => toggleExpanded(team.id)}
                      >
                        {expandedTeamId === team.id ? (
                          <ChevronDown className="h-4 w-4 shrink-0" />
                        ) : (
                          <ChevronRight className="h-4 w-4 shrink-0" />
                        )}
                        {team.name}
                      </button>
                    </div>
                    {team.description && (
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 ml-5">{team.description}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    {savedId === team.id && <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />}
                    {canManage && (
                      <>
                        <Button variant="ghost" size="sm" onClick={() => openEditor(team)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(team)}>
                          <Trash2 className="h-3.5 w-3.5 text-red-500" />
                        </Button>
                      </>
                    )}
                  </div>
                </div>
              </div>
              {expandedTeamId === team.id && (
                <TeamMembersPanel
                  projectKey={projectKey ?? ''}
                  teamId={team.id}
                  canManage={canManage}
                />
              )}
            </div>
          ))}
        </div>
      )}

      {/* Editor modal */}
      <Modal
        open={editorOpen}
        onClose={() => { setEditorOpen(false); setEditingTeam(null) }}
        title={editingTeam ? t('teams.editTeam') : t('teams.createTeam')}
      >
        <TeamForm
          team={editingTeam}
          onSubmit={handleSave}
          onCancel={() => { setEditorOpen(false); setEditingTeam(null) }}
          isPending={createMutation.isPending || updateMutation.isPending}
        />
      </Modal>

      {/* Delete confirmation */}
      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('teams.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="teams.deleteConfirmBody" values={{ name: deleteTarget?.name }} components={{ bold: <strong /> }} />
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

// --- Team Form ---

function TeamForm({
  team,
  onSubmit,
  onCancel,
  isPending,
}: {
  team: Team | null
  onSubmit: (input: CreateTeamInput | UpdateTeamInput) => void
  onCancel: () => void
  isPending: boolean
}) {
  const { t } = useTranslation()
  const [name, setName] = useState(team?.name ?? '')
  const [description, setDescription] = useState(team?.description ?? '')
  const [validationError, setValidationError] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setValidationError('')

    if (!name.trim()) {
      setValidationError(t('teams.nameRequired'))
      return
    }

    if (team) {
      const input: UpdateTeamInput = {}
      if (name.trim() !== team.name) input.name = name.trim()
      if (description !== (team.description ?? '')) input.description = description || null
      onSubmit(input)
    } else {
      onSubmit({
        name: name.trim(),
        description: description || undefined,
      })
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {validationError && (
        <p className="text-sm text-red-600 dark:text-red-400">{validationError}</p>
      )}
      <Input
        label={t('teams.name')}
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder={t('teams.namePlaceholder')}
        required
        autoFocus
      />
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('common.description')}
        </label>
        <textarea
          rows={3}
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={isPending}>
          {isPending ? t('common.saving') : team ? t('common.save') : t('common.create')}
        </Button>
      </div>
    </form>
  )
}

// --- Team Members Panel ---

function TeamMembersPanel({
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
  const [addOpen, setAddOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [savedUserId, setSavedUserId] = useState<string | null>(null)
  const [removeTarget, setRemoveTarget] = useState<{ userId: string; displayName: string } | null>(null)

  function flashSaved(userId: string) {
    setSavedUserId(userId)
    setTimeout(() => setSavedUserId(null), 2000)
  }

  const teamMemberIds = new Set(teamMembers?.map((m) => m.user_id) ?? [])
  const availableMembers = (projectMembers ?? []).filter(
    (pm) => !teamMemberIds.has(pm.user_id) && pm.role !== 'viewer',
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
        setAddOpen(false)
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

  return (
    <div className="border-t border-gray-100 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-4 py-3">
      <div className="flex items-center justify-between mb-2">
        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('teams.members')}</h4>
        {canManage && (
          <Button
            size="sm"
            variant="secondary"
            onClick={() => setAddOpen(!addOpen)}
          >
            <UserPlus className="h-3.5 w-3.5 mr-1" />
            {t('teams.addMember')}
          </Button>
        )}
      </div>

      {/* Add member dropdown */}
      {addOpen && (
        <div className="mb-3 border border-gray-200 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800">
          <div className="p-2">
            <Input
              placeholder={t('teams.searchMembersPlaceholder')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              autoFocus
            />
          </div>
          <ul className="max-h-40 overflow-auto">
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
          <div className="border-t border-gray-100 dark:border-gray-700 p-2 flex justify-end">
            <Button size="sm" variant="ghost" onClick={() => { setAddOpen(false); setSearch('') }}>
              <X className="h-3.5 w-3.5 mr-1" />
              {t('common.close')}
            </Button>
          </div>
        </div>
      )}

      {isLoading ? (
        <Spinner />
      ) : teamMembers && teamMembers.length > 0 ? (
        <ul className="space-y-1">
          {teamMembers.map((member) => (
            <li key={member.user_id} className="flex items-center justify-between py-1.5 px-1">
              <div className="flex items-center gap-2 min-w-0">
                {savedUserId === member.user_id && <Check className="h-4 w-4 text-green-500 shrink-0" />}
                <Avatar name={member.display_name} avatarUrl={member.avatar_url ?? undefined} size="xs" />
                <div className="min-w-0">
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{member.display_name}</span>
                  <span className="text-xs text-gray-400 dark:text-gray-500 ml-2">{member.email}</span>
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
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-gray-400 dark:text-gray-500">{t('teams.noMembers')}</p>
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

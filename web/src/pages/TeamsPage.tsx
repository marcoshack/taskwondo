import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import {
  useTeams,
  useCreateTeam,
  useDeleteTeam,
  useTeamMembers,
} from '@/hooks/useTeams'
import { useMembers } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { Plus, Trash2, Check, Settings } from 'lucide-react'
import type { Team, CreateTeamInput } from '@/api/teams'
import { getLocalizedError } from '@/utils/apiError'

export function TeamsPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const { p } = useNamespacePath()
  const { user } = useAuth()
  const { data: members } = useMembers(projectKey ?? '')
  const { data: teams, isLoading } = useTeams(projectKey ?? '')

  const createMutation = useCreateTeam(projectKey ?? '')
  const deleteMutation = useDeleteTeam(projectKey ?? '')

  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Team | null>(null)
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

  function flashSaved(id: string) {
    setSavedId(id)
    setTimeout(() => setSavedId(null), 2000)
  }

  function handleCreate(input: CreateTeamInput) {
    setError('')
    createMutation.mutate(input, {
      onSuccess: (data) => {
        flashSaved(data.id)
        setCreateOpen(false)
      },
      onError: (err) => {
        setError(getLocalizedError(err, t, 'teams.createError'))
      },
    })
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

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-[var(--foreground)]">{t('teams.title')}</h2>
          <p className="mt-1 text-sm text-[var(--foreground-secondary)]">{t('teams.description')}</p>
        </div>
        {canManage && (
          <Button onClick={() => setCreateOpen(true)} className="border border-transparent">
            <Plus className="h-4 w-4 sm:mr-1" />
            <span className="hidden sm:inline">{t('teams.create')}</span>
          </Button>
        )}
      </div>

      {error && <p className="text-sm text-[var(--danger)]">{error}</p>}

      {(!teams || teams.length === 0) ? (
        <div className="border border-dashed border-[var(--border)] rounded-lg p-6 text-center">
          <p className="text-sm text-[var(--foreground-secondary)]">{t('teams.noTeams')}</p>
          {canManage && (
            <Button size="sm" variant="secondary" className="mt-3" onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4 mr-1" />
              {t('teams.createFirst')}
            </Button>
          )}
        </div>
      ) : (
        <div className="border border-[var(--border)] rounded-lg divide-y divide-[var(--border)]">
          {teams.map((team) => (
            <TeamCard
              key={team.id}
              team={team}
              projectKey={projectKey ?? ''}
              pathFn={p}
              canManage={canManage}
              savedId={savedId}
              onDelete={setDeleteTarget}
            />
          ))}
        </div>
      )}

      {/* Create modal */}
      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={t('teams.createTeam')}
      >
        <TeamCreateForm
          onSubmit={handleCreate}
          onCancel={() => setCreateOpen(false)}
          isPending={createMutation.isPending}
        />
      </Modal>

      {/* Delete confirmation */}
      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('teams.deleteConfirmTitle')}>
        <p className="text-sm text-[var(--foreground-secondary)] mb-4">
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

// --- Team Card ---

function TeamCard({
  team,
  projectKey,
  pathFn,
  canManage,
  savedId,
  onDelete,
}: {
  team: Team
  projectKey: string
  pathFn: (path: string) => string
  canManage: boolean
  savedId: string | null
  onDelete: (team: Team) => void
}) {
  const { t } = useTranslation()
  const { data: teamMembers } = useTeamMembers(projectKey, team.id)
  const memberCount = teamMembers?.length ?? 0

  return (
    <div className="p-4">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <Link
              to={pathFn(`/projects/${projectKey}/teams/${team.id}`)}
              className="text-base font-semibold text-[var(--foreground)] hover:text-[var(--primary)] dark:hover:text-[var(--primary)]"
            >
              {team.name}
            </Link>
            <Badge color="gray">
              {t('teams.memberCount', { count: memberCount })}
            </Badge>
          </div>
          {team.description && (
            <p className="text-xs text-[var(--foreground-secondary)] mt-1">{team.description}</p>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {savedId === team.id && <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />}
          <Link to={pathFn(`/projects/${projectKey}/teams/${team.id}`)}>
            <Button variant="ghost" size="sm">
              <Settings className="h-3.5 w-3.5" />
            </Button>
          </Link>
          {canManage && (
            <Button variant="ghost" size="sm" onClick={() => onDelete(team)}>
              <Trash2 className="h-3.5 w-3.5 text-[var(--danger)]" />
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}

// --- Team Create Form ---

function TeamCreateForm({
  onSubmit,
  onCancel,
  isPending,
}: {
  onSubmit: (input: CreateTeamInput) => void
  onCancel: () => void
  isPending: boolean
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [validationError, setValidationError] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setValidationError('')

    if (!name.trim()) {
      setValidationError(t('teams.nameRequired'))
      return
    }

    onSubmit({
      name: name.trim(),
      description: description || undefined,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {validationError && (
        <p className="text-sm text-[var(--danger)]">{validationError}</p>
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
        <label className="block text-sm font-medium text-[var(--foreground)] mb-1">
          {t('common.description')}
        </label>
        <textarea
          rows={3}
          className="block w-full rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)]"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={isPending}>
          {isPending ? t('common.creating') : t('common.create')}
        </Button>
      </div>
    </form>
  )
}

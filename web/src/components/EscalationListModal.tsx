import { useState, useEffect, useRef, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { X, Plus, Trash2, Users } from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Avatar } from '@/components/ui/Avatar'
import { useCreateEscalationList, useUpdateEscalationList, useEscalationListDetail } from '@/hooks/useEscalation'
import type { ProjectMember } from '@/api/projects'
import type { Team } from '@/api/teams'

interface LevelDraft {
  threshold_pct: string
  users: { id: string; display_name: string; email: string; avatar_url?: string }[]
  teams: { id: string; name: string }[]
}

interface Props {
  open: boolean
  onClose: () => void
  onSave?: () => void
  projectKey: string
  editingId?: string | null
  members: ProjectMember[]
  teams: Team[]
}

export function EscalationListModal({ open, onClose, onSave, projectKey, editingId, members, teams }: Props) {
  const { t } = useTranslation()
  const createMutation = useCreateEscalationList(projectKey)
  const updateMutation = useUpdateEscalationList(projectKey)
  const { data: existing } = useEscalationListDetail(projectKey, editingId ?? '')

  const [name, setName] = useState('')
  const [levels, setLevels] = useState<LevelDraft[]>([])
  const [error, setError] = useState('')
  const [initialized, setInitialized] = useState(false)

  const isEdit = !!editingId

  // Initialize from existing when editing
  useEffect(() => {
    if (isEdit && existing && !initialized) {
      setName(existing.name)
      setLevels(
        existing.levels.map((lv) => ({
          threshold_pct: String(lv.threshold_pct),
          users: lv.users.map((u) => ({
            id: u.id,
            display_name: u.display_name,
            email: u.email,
            avatar_url: u.avatar_url,
          })),
          teams: (lv.teams ?? []).map((t) => ({
            id: t.id,
            name: t.name,
          })),
        }))
      )
      setInitialized(true)
    }
  }, [isEdit, existing, initialized])

  // Reset when modal opens/closes
  useEffect(() => {
    if (!open) {
      setName('')
      setLevels([])
      setError('')
      setInitialized(false)
    }
  }, [open])

  function addLevel() {
    setLevels([...levels, { threshold_pct: '', users: [], teams: [] }])
  }

  function removeLevel(index: number) {
    setLevels(levels.filter((_, i) => i !== index))
  }

  function updateLevelThreshold(index: number, value: string) {
    setLevels(levels.map((lv, i) => (i === index ? { ...lv, threshold_pct: value } : lv)))
  }

  function sortLevelsByThreshold() {
    setLevels((prev) =>
      [...prev].sort(
        (a, b) => (Number(a.threshold_pct) || Infinity) - (Number(b.threshold_pct) || Infinity)
      )
    )
  }

  function addUserToLevel(index: number, member: ProjectMember) {
    setLevels(
      levels.map((lv, i) => {
        if (i !== index) return lv
        if (lv.users.some((u) => u.id === member.user_id)) return lv
        return {
          ...lv,
          users: [...lv.users, { id: member.user_id, display_name: member.display_name, email: member.email, avatar_url: member.avatar_url ?? undefined }],
        }
      })
    )
  }

  function removeUserFromLevel(levelIndex: number, userId: string) {
    setLevels(
      levels.map((lv, i) => {
        if (i !== levelIndex) return lv
        return { ...lv, users: lv.users.filter((u) => u.id !== userId) }
      })
    )
  }

  function addTeamToLevel(index: number, team: Team) {
    setLevels(
      levels.map((lv, i) => {
        if (i !== index) return lv
        if (lv.teams.some((t) => t.id === team.id)) return lv
        return {
          ...lv,
          teams: [...lv.teams, { id: team.id, name: team.name }],
        }
      })
    )
  }

  function removeTeamFromLevel(levelIndex: number, teamId: string) {
    setLevels(
      levels.map((lv, i) => {
        if (i !== levelIndex) return lv
        return { ...lv, teams: lv.teams.filter((t) => t.id !== teamId) }
      })
    )
  }

  function validate(): boolean {
    setError('')

    if (!name.trim()) {
      setError(t('escalation.nameRequired'))
      return false
    }

    if (levels.length === 0) {
      setError(t('escalation.levelsRequired'))
      return false
    }

    // Check thresholds
    const thresholds = new Set<number>()
    for (const lv of levels) {
      const pct = Number(lv.threshold_pct)
      if (!pct || pct <= 0) {
        setError(t('escalation.thresholdRequired'))
        return false
      }
      if (thresholds.has(pct)) {
        setError(t('escalation.duplicateThreshold'))
        return false
      }
      thresholds.add(pct)
    }

    // Check users or teams
    for (const lv of levels) {
      if (lv.users.length === 0 && lv.teams.length === 0) {
        setError(t('escalation.recipientsRequired'))
        return false
      }
    }

    return true
  }

  function handleSave() {
    if (!validate()) return

    const input = {
      name: name.trim(),
      levels: levels.map((lv) => ({
        threshold_pct: Number(lv.threshold_pct),
        user_ids: lv.users.map((u) => u.id),
        team_ids: lv.teams.map((t) => t.id),
      })),
    }

    if (isEdit && editingId) {
      updateMutation.mutate(
        { escalationListId: editingId, input },
        {
          onSuccess: () => {
            onSave?.()
            onClose()
          },
          onError: () => {
            setError(t('escalation.saveError'))
          },
        }
      )
    } else {
      createMutation.mutate(input, {
        onSuccess: () => {
          onSave?.()
          onClose()
        },
        onError: () => {
          setError(t('escalation.saveError'))
        },
      })
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? t('escalation.edit') : t('escalation.create')}
      className="!max-w-2xl"
    >
      <div className="space-y-5">
        {error && <p className="text-sm text-[var(--danger)]">{error}</p>}

        {/* Name */}
        <Input
          label={t('escalation.name')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('escalation.namePlaceholder')}
        />

        {/* Levels */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="block text-sm font-medium text-[var(--foreground)]">
              {t('escalation.levels')}
            </label>
            {levels.length > 0 && (
              <Button variant="ghost" size="sm" onClick={addLevel}>
                <Plus className="h-3.5 w-3.5 mr-1" />
                {t('escalation.addLevel')}
              </Button>
            )}
          </div>

          {levels.length === 0 ? (
            <div className="border border-dashed border-[var(--border)] rounded-lg p-6 text-center">
              <p className="text-sm text-[var(--foreground-secondary)] mb-3">{t('escalation.noLevels')}</p>
              <Button variant="secondary" size="sm" onClick={addLevel}>
                <Plus className="h-3.5 w-3.5 mr-1" />
                {t('escalation.addLevel')}
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              {levels.map((level, index) => (
                <div
                  key={index}
                  className="border border-[var(--border)] rounded-lg p-4 space-y-3"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3 flex-1">
                      <div>
                        <label className="block text-xs font-medium text-[var(--foreground-secondary)] mb-1">
                          {t('escalation.threshold')}
                        </label>
                        <div className="flex items-center gap-1.5">
                          <input
                            type="number"
                            min="1"
                            className="block w-24 rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)] [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                            value={level.threshold_pct}
                            onChange={(e) => updateLevelThreshold(index, e.target.value)}
                            onBlur={sortLevelsByThreshold}
                            placeholder={t('escalation.thresholdPlaceholder')}
                          />
                          <span className="text-xs text-[var(--foreground-secondary)] shrink-0">
                            {t('escalation.thresholdHelp')}
                          </span>
                        </div>
                      </div>
                    </div>
                    <button
                      type="button"
                      className="text-[var(--foreground-muted)] hover:text-[var(--danger)] p-1"
                      onClick={() => removeLevel(index)}
                      title={t('escalation.removeLevel')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>

                  {/* Users and Teams */}
                  <div>
                    <label className="block text-xs font-medium text-[var(--foreground-secondary)] mb-1">
                      {t('escalation.notifyUsersAndTeams')}
                    </label>

                    {/* Selected chips */}
                    {(level.teams.length > 0 || level.users.length > 0) && (
                      <div className="flex flex-wrap gap-1.5 mb-2">
                        {/* Team chips (green) */}
                        {level.teams.map((team) => (
                          <span
                            key={`team-${team.id}`}
                            className="inline-flex items-center gap-1 rounded-full bg-green-50 dark:bg-green-900/30 px-2.5 py-1 text-xs font-medium text-green-700 dark:text-green-300 border border-green-200 dark:border-green-700"
                          >
                            <Users className="h-3 w-3" />
                            <span>{team.name}</span>
                            <button
                              type="button"
                              className="ml-0.5 hover:text-[var(--danger)]"
                              onClick={() => removeTeamFromLevel(index, team.id)}
                            >
                              <X className="h-3 w-3" />
                            </button>
                          </span>
                        ))}
                        {/* User chips (indigo) */}
                        {level.users.map((user) => (
                          <span
                            key={`user-${user.id}`}
                            className="inline-flex items-center gap-1 rounded-full bg-[var(--primary-muted)] px-2.5 py-1 text-xs font-medium text-[var(--primary)] border border-[var(--primary-border)] dark:border-[var(--primary-border)]"
                          >
                            <Avatar name={user.display_name} avatarUrl={user.avatar_url} size="xs" />
                            <span>{user.display_name}</span>
                            <button
                              type="button"
                              className="ml-0.5 hover:text-[var(--danger)]"
                              onClick={() => removeUserFromLevel(index, user.id)}
                            >
                              <X className="h-3 w-3" />
                            </button>
                          </span>
                        ))}
                      </div>
                    )}

                    {/* Picker */}
                    <MemberTeamPicker
                      members={members}
                      teams={teams}
                      excludeUserIds={level.users.map((u) => u.id)}
                      excludeTeamIds={level.teams.map((t) => t.id)}
                      onSelectMember={(member) => addUserToLevel(index, member)}
                      onSelectTeam={(team) => addTeamToLevel(index, team)}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 pt-2 border-t border-[var(--border)]">
          <Button variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSave} disabled={isPending}>
            {isPending ? t('common.saving') : isEdit ? t('common.save') : t('common.create')}
          </Button>
        </div>
      </div>
    </Modal>
  )
}

function MemberTeamPicker({
  members,
  teams,
  excludeUserIds,
  excludeTeamIds,
  onSelectMember,
  onSelectTeam,
}: {
  members: ProjectMember[]
  teams: Team[]
  excludeUserIds: string[]
  excludeTeamIds: string[]
  onSelectMember: (member: ProjectMember) => void
  onSelectTeam: (team: Team) => void
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({})
  const containerRef = useRef<HTMLDivElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const q = search.toLowerCase()

  const availableTeams = teams.filter((t) => {
    if (excludeTeamIds.includes(t.id)) return false
    if (!search) return true
    return t.name.toLowerCase().includes(q)
  })

  const availableMembers = members.filter((m) => {
    if (excludeUserIds.includes(m.user_id)) return false
    if (!search) return true
    return m.display_name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q)
  })

  const hasResults = availableTeams.length > 0 || availableMembers.length > 0

  const updatePosition = useCallback(() => {
    if (!inputRef.current) return
    const rect = inputRef.current.getBoundingClientRect()
    setDropdownStyle({
      position: 'fixed',
      top: rect.bottom + 4,
      left: rect.left,
      width: rect.width,
      zIndex: 60,
    })
  }, [])

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      const target = e.target as Node
      if (containerRef.current?.contains(target) || dropdownRef.current?.contains(target)) return
      setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  function handleSelectMember(member: ProjectMember) {
    onSelectMember(member)
    setSearch('')
    setOpen(false)
  }

  function handleSelectTeam(team: Team) {
    onSelectTeam(team)
    setSearch('')
    setOpen(false)
  }

  return (
    <div ref={containerRef} className="relative">
      <input
        ref={inputRef}
        className="block w-full rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)]"
        placeholder={t('escalation.searchPlaceholder')}
        value={search}
        onChange={(e) => { setSearch(e.target.value); updatePosition(); setOpen(true) }}
        onFocus={() => { updatePosition(); setOpen(true) }}
      />

      {open && createPortal(
        <div
          ref={dropdownRef}
          style={dropdownStyle}
          className="bg-[var(--surface)] border border-[var(--border)] rounded-md shadow-lg"
        >
          <ul className="max-h-48 overflow-auto">
            {!hasResults ? (
              <li className="px-3 py-2 text-sm text-[var(--foreground-muted)]">
                {t('projects.settings.noUsersFound')}
              </li>
            ) : (
              <>
                {/* Teams section */}
                {availableTeams.length > 0 && (
                  <>
                    <li className="px-3 py-1 text-xs font-semibold text-[var(--foreground-secondary)] uppercase tracking-wider bg-[var(--surface-secondary)]/50">
                      {t('escalation.teamsSection')}
                    </li>
                    {availableTeams.map((team) => (
                      <li key={`team-${team.id}`}>
                        <button
                          type="button"
                          className="w-full text-left px-3 py-2 text-sm hover:bg-[var(--surface-hover)] text-[var(--foreground)] flex items-center gap-2"
                          onClick={() => handleSelectTeam(team)}
                        >
                          <span className="shrink-0 h-5 w-5 rounded-full bg-green-100 dark:bg-green-900/40 flex items-center justify-center">
                            <Users className="h-3 w-3 text-green-600 dark:text-green-400" />
                          </span>
                          <div className="min-w-0">
                            <div className="font-medium truncate">{team.name}</div>
                          </div>
                        </button>
                      </li>
                    ))}
                  </>
                )}
                {/* Users section */}
                {availableMembers.length > 0 && (
                  <>
                    <li className="px-3 py-1 text-xs font-semibold text-[var(--foreground-secondary)] uppercase tracking-wider bg-[var(--surface-secondary)]/50">
                      {t('escalation.usersSection')}
                    </li>
                    {availableMembers.map((member) => (
                      <li key={member.user_id}>
                        <button
                          type="button"
                          className="w-full text-left px-3 py-2 text-sm hover:bg-[var(--surface-hover)] text-[var(--foreground)] flex items-center gap-2"
                          onClick={() => handleSelectMember(member)}
                        >
                          <span className="shrink-0"><Avatar name={member.display_name} avatarUrl={member.avatar_url} size="xs" /></span>
                          <div className="min-w-0">
                            <div className="font-medium truncate">{member.display_name}</div>
                            <div className="text-xs text-[var(--foreground-muted)] truncate">{member.email}</div>
                          </div>
                        </button>
                      </li>
                    ))}
                  </>
                )}
              </>
            )}
          </ul>
        </div>,
        document.body
      )}
    </div>
  )
}

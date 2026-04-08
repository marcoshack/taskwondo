import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  useOncallRotation,
  useOncallHistory,
  useCreateOncallRotation,
  useUpdateOncallRotation,
  useDeleteOncallRotation,
  useOncallOverrides,
  useCreateOncallOverride,
  useUpdateOncallOverride,
  useDeleteOncallOverride,
} from '@/hooks/useOncall'
import { useTeamMembers } from '@/hooks/useTeams'
import { useMembers } from '@/hooks/useProjects'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Tooltip } from '@/components/ui/Tooltip'
import { UserPicker } from '@/components/ui/UserPicker'
import { Clock, Plus, Pencil, Trash2, Shuffle, GripVertical, ShieldAlert } from 'lucide-react'
import type { OncallRotationWithMembers, OncallOverride, CreateOncallRotationInput, UpdateOncallRotationInput, CreateOncallOverrideInput, UpdateOncallOverrideInput } from '@/api/oncall'
import type { TeamMemberWithUser } from '@/api/teams'
import { OncallCalendar } from '@/components/OncallCalendar'
import { getLocalizedError } from '@/utils/apiError'

const TIMEZONES = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Asia/Tokyo',
  'Asia/Shanghai',
  'Australia/Sydney',
  'Pacific/Auckland',
]

export function OncallTab({
  projectKey,
  teamId,
  canManage,
}: {
  projectKey: string
  teamId: string
  canManage: boolean
}) {
  const { t } = useTranslation()
  const { data: rotationData, isLoading, error: queryError } = useOncallRotation(projectKey, teamId)
  const deleteMutation = useDeleteOncallRotation(projectKey, teamId)

  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [overrideOpen, setOverrideOpen] = useState(false)
  const [error, setError] = useState('')

  // 404 means no rotation configured — that's expected
  const is404 = (queryError as { response?: { status?: number } })?.response?.status === 404
  const hasRotation = !!rotationData && !is404
  const noRotation = !isLoading && (!rotationData || is404)

  function handleDelete() {
    setError('')
    deleteMutation.mutate(undefined, {
      onError: (err) => {
        setError(getLocalizedError(err, t, 'teams.oncall.deleteError'))
      },
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    )
  }

  if (noRotation) {
    return (
      <div className="max-w-3xl space-y-4">
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-12 text-center">
          <Clock className="h-12 w-12 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
          <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{t('teams.oncall.noRotation')}</p>
          {canManage && (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4 mr-1" />
              {t('teams.oncall.setUp')}
            </Button>
          )}
        </div>

        <CreateRotationModal
          open={createOpen}
          onClose={() => setCreateOpen(false)}
          projectKey={projectKey}
          teamId={teamId}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      {/* Two-column layout: 70% calendar/history | 30% details */}
      {hasRotation && (
        <div className="flex flex-col lg:flex-row gap-6">
          {/* Left column — calendar + history */}
          <div className="lg:w-[70%] space-y-6 min-w-0">
            <OncallCalendar
              members={rotationData.members}
              projectKey={projectKey}
              teamId={teamId}
            />
            <HistoryLog projectKey={projectKey} teamId={teamId} />
          </div>

          {/* Right column — details sidebar */}
          <div className="lg:w-[30%] space-y-4">
            {canManage && (
              <div className="flex items-center gap-2">
                <Button className="flex-1" onClick={() => setEditOpen(true)}>
                  <Pencil className="h-4 w-4 mr-1.5" />
                  {t('teams.oncall.editRotation')}
                </Button>
                <Button variant="secondary" className="flex-1" onClick={() => setOverrideOpen(true)}>
                  <ShieldAlert className="h-4 w-4 mr-1.5" />
                  {t('teams.oncall.override.add')}
                </Button>
              </div>
            )}
            <RotationDetailsCard data={rotationData} />
            <RotationMembersPanel data={rotationData} />
            <OverridePanel projectKey={projectKey} teamId={teamId} canManage={canManage} />
          </div>
        </div>
      )}

      {/* Edit modal (includes delete) */}
      {hasRotation && (
        <EditRotationModal
          open={editOpen}
          onClose={() => setEditOpen(false)}
          projectKey={projectKey}
          teamId={teamId}
          data={rotationData}
          onDelete={handleDelete}
          isDeleting={deleteMutation.isPending}
        />
      )}

      {/* Create override modal */}
      {hasRotation && (
        <CreateOverrideModal
          open={overrideOpen}
          onClose={() => setOverrideOpen(false)}
          projectKey={projectKey}
          teamId={teamId}
        />
      )}
    </div>
  )
}

// --- Rotation Details Card (sidebar) ---

function RotationDetailsCard({ data }: { data: OncallRotationWithMembers }) {
  const { t } = useTranslation()

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3">
      <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wider">
        {t('teams.oncall.title')}
      </p>
      <div className="space-y-1 text-xs text-gray-500 dark:text-gray-400">
        <div className="flex justify-between">
          <span>{t('teams.oncall.periodDays')}</span>
          <span className="text-gray-700 dark:text-gray-300">{t('teams.oncall.period', { days: data.period_days })}</span>
        </div>
        <div className="flex justify-between">
          <span>{t('teams.oncall.timezone')}</span>
          <span className="text-gray-700 dark:text-gray-300">{data.timezone}</span>
        </div>
        {data.next_rotation_at && (
          <div className="flex justify-between">
            <span>{t('teams.oncall.nextRotationLabel')}</span>
            <span className="text-gray-700 dark:text-gray-300">
              {new Date(data.next_rotation_at).toLocaleDateString()}{' '}
              {(() => { const match = data.rotation_time?.match(/T(\d{2}:\d{2})/); return match ? match[1] : data.rotation_time?.slice(0, 5) })()}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}

// --- Rotation Members Panel (sidebar) ---

function RotationMembersPanel({ data }: { data: OncallRotationWithMembers }) {
  const { t } = useTranslation()
  const sortedMembers = [...data.members].sort((a, b) => a.position - b.position)

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3">
      <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wider">
        {t('teams.oncall.participants')}
      </p>
      <div className="space-y-2">
        {sortedMembers.map((member, idx) => (
          <div key={member.user_id} className="flex items-center gap-2">
            <span className="text-xs text-gray-400 w-4 text-right shrink-0">{idx + 1}</span>
            <Avatar name={member.display_name} avatarUrl={member.avatar_url} size="xs" />
            <span className="text-sm text-gray-900 dark:text-gray-100">{member.display_name}</span>
            {member.user_id === data.current_user_id && !data.active_override && (
              <Badge color="green">{t('teams.oncall.active')}</Badge>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

// --- Create Rotation Modal ---

function CreateRotationModal({
  open,
  onClose,
  projectKey,
  teamId,
}: {
  open: boolean
  onClose: () => void
  projectKey: string
  teamId: string
}) {
  const { t } = useTranslation()
  const createMutation = useCreateOncallRotation(projectKey, teamId)
  const { data: teamMembers } = useTeamMembers(projectKey, teamId)
  const [error, setError] = useState('')

  function handleSubmit(input: CreateOncallRotationInput) {
    setError('')
    createMutation.mutate(input, {
      onSuccess: () => onClose(),
      onError: (err) => {
        setError(getLocalizedError(err, t, 'teams.oncall.createError'))
      },
    })
  }

  return (
    <Modal open={open} onClose={onClose} title={t('teams.oncall.createRotation')}>
      {error && <p className="text-sm text-red-600 dark:text-red-400 mb-3">{error}</p>}
      <RotationForm
        teamMembers={teamMembers ?? []}
        onSubmit={handleSubmit}
        onCancel={onClose}
        isPending={createMutation.isPending}
      />
    </Modal>
  )
}

// --- Edit Rotation Modal ---

function EditRotationModal({
  open,
  onClose,
  projectKey,
  teamId,
  data,
  onDelete,
  isDeleting,
}: {
  open: boolean
  onClose: () => void
  projectKey: string
  teamId: string
  data: OncallRotationWithMembers
  onDelete: () => void
  isDeleting: boolean
}) {
  const { t } = useTranslation()
  const updateMutation = useUpdateOncallRotation(projectKey, teamId)
  const { data: teamMembers } = useTeamMembers(projectKey, teamId)
  const [error, setError] = useState('')
  const [deleteOpen, setDeleteOpen] = useState(false)

  function handleSubmit(input: CreateOncallRotationInput) {
    setError('')
    const updateInput: UpdateOncallRotationInput = {
      period_days: input.period_days,
      rotation_time: input.rotation_time,
      timezone: input.timezone,
      start_date: input.start_date,
      member_ids: input.member_ids,
    }
    updateMutation.mutate(updateInput, {
      onSuccess: () => onClose(),
      onError: (err) => {
        setError(getLocalizedError(err, t, 'teams.oncall.updateError'))
      },
    })
  }

  return (
    <>
      <Modal open={open} onClose={onClose} title={t('teams.oncall.editRotation')}>
        {error && <p className="text-sm text-red-600 dark:text-red-400 mb-3">{error}</p>}
        <RotationForm
          teamMembers={teamMembers ?? []}
          initial={data}
          onSubmit={handleSubmit}
          onCancel={onClose}
          isPending={updateMutation.isPending}
          onDelete={() => setDeleteOpen(true)}
        />
      </Modal>

      <Modal open={deleteOpen} onClose={() => setDeleteOpen(false)} title={t('teams.oncall.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">{t('teams.oncall.deleteConfirmBody')}</p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setDeleteOpen(false)}>
            {t('common.cancel')}
          </Button>
          <Button variant="danger" disabled={isDeleting} onClick={() => { onDelete(); setDeleteOpen(false); onClose() }}>
            {isDeleting ? t('common.deleting') : t('common.delete')}
          </Button>
        </div>
      </Modal>
    </>
  )
}

// --- Rotation Form ---

function RotationForm({
  teamMembers,
  initial,
  onSubmit,
  onCancel,
  isPending,
  onDelete,
}: {
  teamMembers: TeamMemberWithUser[]
  initial?: OncallRotationWithMembers
  onSubmit: (input: CreateOncallRotationInput) => void
  onCancel: () => void
  isPending: boolean
  onDelete?: () => void
}) {
  const { t } = useTranslation()

  // Initialize selected member IDs (ordered)
  const initialMemberIds = initial
    ? [...initial.members].sort((a, b) => a.position - b.position).map((m) => m.user_id)
    : []

  const [selectedIds, setSelectedIds] = useState<string[]>(initialMemberIds)
  const [periodDays, setPeriodDays] = useState(String(initial?.period_days ?? 7))
  const [rotationTime, setRotationTime] = useState(() => {
    if (!initial?.rotation_time) return '12:00'
    // API returns TIME as "0000-01-01T12:00:00Z"; extract HH:MM
    const match = initial.rotation_time.match(/T(\d{2}:\d{2})/)
    return match ? match[1] : initial.rotation_time.slice(0, 5)
  })
  const [timezone, setTimezone] = useState(initial?.timezone ?? 'UTC')
  const [startDate, setStartDate] = useState(() => {
    if (!initial?.start_date) return new Date().toISOString().slice(0, 10)
    // API may return "2026-04-06T00:00:00Z" — extract YYYY-MM-DD
    const m = initial.start_date.match(/^(\d{4}-\d{2}-\d{2})/)
    return m ? m[1] : initial.start_date
  })
  const [validationError, setValidationError] = useState('')
  const [draggedIdx, setDraggedIdx] = useState<number | null>(null)

  function toggleMember(userId: string) {
    setSelectedIds((prev) =>
      prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId],
    )
  }

  function randomizeOrder() {
    setSelectedIds((prev) => {
      const shuffled = [...prev]
      for (let i = shuffled.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1))
        ;[shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]]
      }
      return shuffled
    })
  }

  // Native drag-and-drop for reordering
  function handleDragStart(idx: number) {
    setDraggedIdx(idx)
  }

  function handleDragOver(e: React.DragEvent, idx: number) {
    e.preventDefault()
    if (draggedIdx === null || draggedIdx === idx) return
    setSelectedIds((prev) => {
      const updated = [...prev]
      const [moved] = updated.splice(draggedIdx, 1)
      updated.splice(idx, 0, moved)
      return updated
    })
    setDraggedIdx(idx)
  }

  function handleDragEnd() {
    setDraggedIdx(null)
  }

  const parsedDays = parseInt(periodDays, 10)
  const isValidPeriod = !isNaN(parsedDays) && parsedDays >= 1
  const isValidTime = !!rotationTime
  const isValidMembers = selectedIds.length >= 2
  const isValidStartDate = !!startDate
  const canSubmit = isValidMembers && isValidPeriod && isValidTime && isValidStartDate && !isPending

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setValidationError('')

    if (!isValidMembers) {
      setValidationError(t('teams.oncall.minMembers'))
      return
    }
    if (!isValidPeriod) {
      setValidationError(t('teams.oncall.invalidPeriod'))
      return
    }
    if (!isValidTime) {
      setValidationError(t('teams.oncall.invalidTime'))
      return
    }
    if (!isValidStartDate) {
      setValidationError(t('teams.oncall.invalidStartDate'))
      return
    }

    onSubmit({
      period_days: parsedDays,
      rotation_time: rotationTime + ':00',
      timezone,
      start_date: startDate,
      member_ids: selectedIds,
    })
  }

  const memberMap = new Map(teamMembers.map((m) => [m.user_id, m]))

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {validationError && (
        <p className="text-sm text-red-600 dark:text-red-400">{validationError}</p>
      )}

      {/* Participants selection */}
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('teams.oncall.participants')}
        </label>
        <div className="border border-gray-200 dark:border-gray-600 rounded-md max-h-40 overflow-auto">
          {teamMembers.map((member) => (
            <label
              key={member.user_id}
              className="flex items-center gap-2 px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
            >
              <input
                type="checkbox"
                checked={selectedIds.includes(member.user_id)}
                onChange={() => toggleMember(member.user_id)}
                className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
              />
              <Avatar name={member.display_name} avatarUrl={member.avatar_url ?? undefined} size="xs" />
              <span className="text-sm text-gray-900 dark:text-gray-100">{member.display_name}</span>
            </label>
          ))}
          {teamMembers.length === 0 && (
            <p className="px-3 py-2 text-sm text-gray-400">{t('teams.noMembers')}</p>
          )}
        </div>
      </div>

      {/* Order */}
      {selectedIds.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('teams.oncall.order')}
            </label>
            <Button type="button" variant="ghost" size="sm" onClick={randomizeOrder}>
              <Shuffle className="h-3.5 w-3.5 mr-1" />
              {t('teams.oncall.randomize')}
            </Button>
          </div>
          <div className="border border-gray-200 dark:border-gray-600 rounded-md">
            {selectedIds.map((userId, idx) => {
              const member = memberMap.get(userId)
              if (!member) return null
              return (
                <div
                  key={userId}
                  draggable
                  onDragStart={() => handleDragStart(idx)}
                  onDragOver={(e) => handleDragOver(e, idx)}
                  onDragEnd={handleDragEnd}
                  className={`flex items-center gap-2 px-3 py-2 cursor-grab active:cursor-grabbing ${
                    draggedIdx === idx ? 'bg-indigo-50 dark:bg-indigo-900/20' : 'hover:bg-gray-50 dark:hover:bg-gray-700'
                  } ${idx > 0 ? 'border-t border-gray-100 dark:border-gray-700' : ''}`}
                >
                  <GripVertical className="h-3.5 w-3.5 text-gray-400 shrink-0" />
                  <span className="text-xs text-gray-400 w-5 text-right shrink-0">{idx + 1}</span>
                  <Avatar name={member.display_name} avatarUrl={member.avatar_url ?? undefined} size="xs" />
                  <span className="text-sm text-gray-900 dark:text-gray-100">{member.display_name}</span>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Period, rotation time, start date */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <Input
          label={t('teams.oncall.periodDays')}
          type="number"
          min={1}
          value={periodDays}
          onChange={(e) => setPeriodDays(e.target.value)}
        />
        <Input
          label={t('teams.oncall.rotationTime')}
          type="time"
          value={rotationTime}
          onChange={(e) => setRotationTime(e.target.value)}
        />
        <Input
          label={t('teams.oncall.startDate')}
          type="date"
          value={startDate}
          onChange={(e) => setStartDate(e.target.value)}
        />
      </div>

      {/* Timezone */}
      <Select
        label={t('teams.oncall.timezone')}
        value={timezone}
        onChange={(e) => setTimezone(e.target.value)}
      >
        {TIMEZONES.map((tz) => (
          <option key={tz} value={tz}>{tz}</option>
        ))}
      </Select>

      <div className="flex items-center pt-2">
        {onDelete && (
          <Button type="button" variant="secondary" onClick={onDelete} className="mr-auto px-2.5">
            <Trash2 className="h-4 w-4 text-red-500" />
          </Button>
        )}
        <div className="flex gap-3 ml-auto">
          <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
          <Tooltip content={
            !canSubmit && !isPending ? [
              !isValidMembers ? t('teams.oncall.minMembers') : '',
              !isValidPeriod ? t('teams.oncall.invalidPeriod') : '',
              !isValidTime ? t('teams.oncall.invalidTime') : '',
              !isValidStartDate ? t('teams.oncall.invalidStartDate') : '',
            ].filter(Boolean).join(' ') || undefined : undefined
          }>
            <Button type="submit" disabled={!canSubmit}>
              {isPending ? t('common.saving') : initial ? t('common.save') : t('common.create')}
            </Button>
          </Tooltip>
        </div>
      </div>
    </form>
  )
}

// --- Override Panel (sidebar) ---

function OverridePanel({
  projectKey,
  teamId,
  canManage,
}: {
  projectKey: string
  teamId: string
  canManage: boolean
}) {
  const { t } = useTranslation()
  const { data: overrides } = useOncallOverrides(projectKey, teamId)
  const deleteMutation = useDeleteOncallOverride(projectKey, teamId)
  const [deleteTarget, setDeleteTarget] = useState<OncallOverride | null>(null)
  const [editTarget, setEditTarget] = useState<OncallOverride | null>(null)
  const [error, setError] = useState('')

  const now = new Date()

  function handleDelete() {
    if (!deleteTarget) return
    setError('')
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => setDeleteTarget(null),
      onError: (err) => {
        setError(getLocalizedError(err, t, 'teams.oncall.override.deleteError'))
        setDeleteTarget(null)
      },
    })
  }

  return (
    <>
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3">
        <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wider">
          {t('teams.oncall.override.title')}
        </p>
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        {overrides && overrides.length > 0 ? (
          <div className="space-y-2">
            {overrides.map((override) => {
              const isActive = new Date(override.start_at) <= now && new Date(override.end_at) > now
              return (
                <div key={override.id} className="space-y-1">
                  {/* First row: name + badge + actions */}
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {override.override_user_name}
                    </span>
                    <Badge color={isActive ? 'yellow' : 'blue'}>
                      {isActive ? t('teams.oncall.override.active') : t('teams.oncall.override.upcoming')}
                    </Badge>
                    {canManage && (
                      <div className="flex items-center gap-0.5 ml-auto">
                        <Button variant="ghost" size="sm" onClick={() => setEditTarget(override)}>
                          <Pencil className="h-3 w-3" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(override)}>
                          <Trash2 className="h-3 w-3 text-red-500" />
                        </Button>
                      </div>
                    )}
                  </div>
                  {/* Second row: time range */}
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {new Date(override.start_at).toLocaleString()} — {new Date(override.end_at).toLocaleString()}
                  </div>
                  {override.reason && (
                    <div className="text-xs text-gray-400 dark:text-gray-500">{override.reason}</div>
                  )}
                </div>
              )
            })}
          </div>
        ) : (
          <p className="text-xs text-gray-400 dark:text-gray-500">{t('teams.oncall.override.empty')}</p>
        )}
      </div>

      {/* Edit override modal */}
      {editTarget && (
        <EditOverrideModal
          open={!!editTarget}
          onClose={() => setEditTarget(null)}
          projectKey={projectKey}
          teamId={teamId}
          override={editTarget}
        />
      )}

      {/* Delete confirmation */}
      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('teams.oncall.override.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">{t('teams.oncall.override.deleteConfirmBody')}</p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setDeleteTarget(null)}>{t('common.cancel')}</Button>
          <Button variant="danger" disabled={deleteMutation.isPending} onClick={handleDelete}>
            {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
          </Button>
        </div>
      </Modal>
    </>
  )
}

// --- Create Override Modal ---

function CreateOverrideModal({
  open,
  onClose,
  projectKey,
  teamId,
}: {
  open: boolean
  onClose: () => void
  projectKey: string
  teamId: string
}) {
  const { t } = useTranslation()
  const createMutation = useCreateOncallOverride(projectKey, teamId)
  const { data: projectMembers } = useMembers(projectKey)
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null)
  const [startAt, setStartAt] = useState('')
  const [endAt, setEndAt] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')

  // Filter out customers and viewers
  const eligibleMembers = (projectMembers ?? []).filter(
    (m) => m.role !== 'customer' && m.role !== 'viewer',
  )

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!selectedUserId) return
    if (!startAt || !endAt) return
    if (new Date(endAt) <= new Date(startAt)) {
      setError(t('teams.oncall.override.invalidTimeRange'))
      return
    }

    const input: CreateOncallOverrideInput = {
      override_user_id: selectedUserId,
      start_at: new Date(startAt).toISOString(),
      end_at: new Date(endAt).toISOString(),
      reason: reason || undefined,
    }

    createMutation.mutate(input, {
      onSuccess: () => {
        setSelectedUserId(null)
        setStartAt('')
        setEndAt('')
        setReason('')
        onClose()
      },
      onError: (err) => {
        setError(getLocalizedError(err, t, 'teams.oncall.override.createError'))
      },
    })
  }

  const canSubmit = !!selectedUserId && !!startAt && !!endAt && !createMutation.isPending

  return (
    <Modal open={open} onClose={onClose} title={t('teams.oncall.override.createTitle')}>
      {error && <p className="text-sm text-red-600 dark:text-red-400 mb-3">{error}</p>}
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('teams.oncall.override.coveringMember')}
          </label>
          <UserPicker
            members={eligibleMembers}
            value={selectedUserId}
            onChange={(id) => setSelectedUserId(id)}
            placeholder={t('teams.oncall.override.selectMember')}
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Input
            label={t('teams.oncall.override.startAt')}
            type="datetime-local"
            value={startAt}
            onChange={(e) => setStartAt(e.target.value)}
          />
          <Input
            label={t('teams.oncall.override.endAt')}
            type="datetime-local"
            value={endAt}
            onChange={(e) => setEndAt(e.target.value)}
          />
        </div>

        <Input
          label={t('teams.oncall.override.reason')}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder={t('teams.oncall.override.reasonPlaceholder')}
        />

        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="submit" disabled={!canSubmit}>
            {createMutation.isPending ? t('common.saving') : t('common.create')}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

// --- Edit Override Modal ---

function EditOverrideModal({
  open,
  onClose,
  projectKey,
  teamId,
  override,
}: {
  open: boolean
  onClose: () => void
  projectKey: string
  teamId: string
  override: OncallOverride
}) {
  const { t } = useTranslation()
  const updateMutation = useUpdateOncallOverride(projectKey, teamId)
  const { data: projectMembers } = useMembers(projectKey)
  const [selectedUserId, setSelectedUserId] = useState<string | null>(override.override_user_id)
  const [startAt, setStartAt] = useState(() => toDatetimeLocal(override.start_at))
  const [endAt, setEndAt] = useState(() => toDatetimeLocal(override.end_at))
  const [reason, setReason] = useState(override.reason ?? '')
  const [error, setError] = useState('')

  const eligibleMembers = (projectMembers ?? []).filter(
    (m) => m.role !== 'customer' && m.role !== 'viewer',
  )

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!selectedUserId || !startAt || !endAt) return
    if (new Date(endAt) <= new Date(startAt)) {
      setError(t('teams.oncall.override.invalidTimeRange'))
      return
    }

    const input: UpdateOncallOverrideInput = {
      override_user_id: selectedUserId,
      start_at: new Date(startAt).toISOString(),
      end_at: new Date(endAt).toISOString(),
      reason: reason || undefined,
    }

    updateMutation.mutate(
      { overrideId: override.id, input },
      {
        onSuccess: () => onClose(),
        onError: (err) => {
          setError(getLocalizedError(err, t, 'teams.oncall.override.updateError'))
        },
      },
    )
  }

  const canSubmit = !!selectedUserId && !!startAt && !!endAt && !updateMutation.isPending

  return (
    <Modal open={open} onClose={onClose} title={t('teams.oncall.override.editTitle')}>
      {error && <p className="text-sm text-red-600 dark:text-red-400 mb-3">{error}</p>}
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('teams.oncall.override.coveringMember')}
          </label>
          <UserPicker
            members={eligibleMembers}
            value={selectedUserId}
            onChange={(id) => setSelectedUserId(id)}
            placeholder={t('teams.oncall.override.selectMember')}
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Input
            label={t('teams.oncall.override.startAt')}
            type="datetime-local"
            value={startAt}
            onChange={(e) => setStartAt(e.target.value)}
          />
          <Input
            label={t('teams.oncall.override.endAt')}
            type="datetime-local"
            value={endAt}
            onChange={(e) => setEndAt(e.target.value)}
          />
        </div>

        <Input
          label={t('teams.oncall.override.reason')}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder={t('teams.oncall.override.reasonPlaceholder')}
        />

        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="submit" disabled={!canSubmit}>
            {updateMutation.isPending ? t('common.saving') : t('common.save')}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

function toDatetimeLocal(iso: string): string {
  const d = new Date(iso)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${y}-${m}-${day}T${h}:${min}`
}

// --- History Log ---

function HistoryLog({
  projectKey,
  teamId,
}: {
  projectKey: string
  teamId: string
}) {
  const { t } = useTranslation()
  const [limit] = useState(10)
  const [offset, setOffset] = useState(0)
  const { data: history, isLoading } = useOncallHistory(projectKey, teamId, limit, offset)

  if (isLoading) {
    return <Spinner />
  }

  if (!history || history.length === 0) {
    return null
  }

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('teams.oncall.history')}</h3>
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
        {history.map((entry) => (
          <div key={entry.id} className="p-3 flex items-center gap-3">
            <Avatar name={entry.display_name} avatarUrl={entry.avatar_url} size="xs" />
            <div className="min-w-0 flex-1">
              <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{entry.display_name}</span>
              <div className="text-xs text-gray-500 dark:text-gray-400">
                {new Date(entry.started_at).toLocaleDateString()} - {entry.ended_at ? new Date(entry.ended_at).toLocaleDateString() : t('common.current')}
              </div>
            </div>
          </div>
        ))}
      </div>
      {history.length === limit && (
        <div className="flex justify-center">
          <Button variant="ghost" size="sm" onClick={() => setOffset((prev) => prev + limit)}>
            {t('common.loadMore')}
          </Button>
        </div>
      )}
    </div>
  )
}

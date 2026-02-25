import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useTimeEntries, useCreateTimeEntry, useUpdateTimeEntry, useDeleteTimeEntry } from '@/hooks/useWorkItems'
import { useAuth } from '@/contexts/AuthContext'
import { useMembers } from '@/hooks/useProjects'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Avatar } from '@/components/ui/Avatar'
import { Spinner } from '@/components/ui/Spinner'

interface TimeEntryListProps {
  projectKey: string
  itemNumber: number
  sortOrder?: 'asc' | 'desc'
  readOnly?: boolean
}

function formatDuration(totalSeconds: number): string {
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  if (h === 0) return `${m}m`
  if (m === 0) return `${h}h`
  return `${h}h ${m}m`
}

function parseDurationString(input: string): number | null {
  const regex = /^(?:(\d+)h)?\s*(?:(\d+)m)?$/i
  const match = input.match(regex)
  if (!match) {
    // Try plain number as hours
    const num = parseFloat(input)
    if (!isNaN(num) && num > 0) return Math.round(num * 3600)
    return null
  }
  const h = match[1] ? parseInt(match[1], 10) : 0
  const m = match[2] ? parseInt(match[2], 10) : 0
  if (h === 0 && m === 0) return null
  return h * 3600 + m * 60
}

export function TimeEntryList({ projectKey, itemNumber, sortOrder = 'desc', readOnly = false }: TimeEntryListProps) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { data: timeData, isLoading } = useTimeEntries(projectKey, itemNumber)
  const { data: members } = useMembers(projectKey)
  const createMutation = useCreateTimeEntry(projectKey, itemNumber)
  const updateMutation = useUpdateTimeEntry(projectKey, itemNumber)
  const deleteMutation = useDeleteTimeEntry(projectKey, itemNumber)

  const [duration, setDuration] = useState('')
  const [startedAt, setStartedAt] = useState(() => new Date().toISOString().slice(0, 10))
  const [description, setDescription] = useState('')

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editDuration, setEditDuration] = useState('')
  const [editStartedAt, setEditStartedAt] = useState('')
  const [editDescription, setEditDescription] = useState('')

  const [deletingId, setDeletingId] = useState<string | null>(null)

  const entries = timeData?.entries ?? []
  const sorted = sortOrder === 'desc' ? [...entries].reverse() : entries

  function authorName(authorId: string): string {
    const member = members?.find((m) => m.user_id === authorId)
    return member?.display_name ?? t('common.unknown')
  }

  function handleCreate() {
    const durationSeconds = parseDurationString(duration)
    if (!durationSeconds) return
    createMutation.mutate(
      {
        started_at: new Date(startedAt).toISOString(),
        duration_seconds: durationSeconds,
        description: description.trim() || undefined,
      },
      {
        onSuccess: () => {
          setDuration('')
          setDescription('')
          setStartedAt(new Date().toISOString().slice(0, 10))
        },
      },
    )
  }

  function startEdit(entry: { id: string; started_at: string; duration_seconds: number; description?: string | null }) {
    setEditingId(entry.id)
    setEditDuration(formatDuration(entry.duration_seconds))
    setEditStartedAt(new Date(entry.started_at).toISOString().slice(0, 10))
    setEditDescription(entry.description ?? '')
  }

  function handleUpdate() {
    if (!editingId) return
    const durationSeconds = parseDurationString(editDuration)
    if (!durationSeconds) return
    updateMutation.mutate(
      {
        entryId: editingId,
        input: {
          started_at: new Date(editStartedAt).toISOString(),
          duration_seconds: durationSeconds,
          description: editDescription.trim() || null,
        },
      },
      { onSuccess: () => setEditingId(null) },
    )
  }

  if (isLoading) return <Spinner size="sm" />

  return (
    <div className="space-y-4">
      {!readOnly && (
        <div className="pb-3 border-b border-gray-100 dark:border-gray-700 space-y-2">
          <div className="flex gap-2 items-center">
            <Input
              type="text"
              className="w-24 shrink-0"
              value={duration}
              onChange={(e) => setDuration(e.target.value)}
              placeholder={t('timeTracking.durationPlaceholder')}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && parseDurationString(duration)) {
                  e.preventDefault()
                  handleCreate()
                }
              }}
            />
            <Input
              type="date"
              className="shrink-0"
              value={startedAt}
              onChange={(e) => setStartedAt(e.target.value)}
            />
            <Input
              className="hidden sm:block flex-1 min-w-0"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t('timeTracking.descriptionPlaceholder')}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && parseDurationString(duration)) {
                  e.preventDefault()
                  handleCreate()
                }
              }}
            />
            <Button
              className="py-2 text-sm shrink-0"
              onClick={handleCreate}
              disabled={!parseDurationString(duration) || createMutation.isPending}
            >
              {createMutation.isPending ? t('timeTracking.logging') : t('timeTracking.logTime')}
            </Button>
          </div>
          <Input
            className="sm:hidden"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('timeTracking.descriptionPlaceholder')}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && parseDurationString(duration)) {
                e.preventDefault()
                handleCreate()
              }
            }}
          />
        </div>
      )}

      {sorted.length === 0 && (
        <p className="text-sm text-gray-400 dark:text-gray-500 italic">{t('timeTracking.noEntries')}</p>
      )}

      {sorted.map((entry) => (
        <div
          key={entry.id}
          className="group/entry border-b border-gray-100 dark:border-gray-700 pb-3"
        >
          <div className="flex items-center gap-2 mb-1">
            <Avatar name={authorName(entry.user_id)} size="xs" />
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{authorName(entry.user_id)}</span>
            <span className="text-sm font-semibold text-indigo-600 dark:text-indigo-400">
              {formatDuration(entry.duration_seconds)}
            </span>
            <span className="text-xs text-gray-400 dark:text-gray-500">
              {new Date(entry.started_at).toLocaleDateString()}
            </span>
            {user && entry.user_id === user.id && !readOnly && editingId !== entry.id && (
              <>
                <button
                  className="group/edit relative inline-flex items-center justify-center w-7 h-7 rounded-md text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:text-gray-500 dark:hover:text-gray-300 dark:hover:bg-gray-700 transition-colors opacity-0 group-hover/entry:opacity-100"
                  onClick={() => startEdit(entry)}
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M11.5 2.5a1.5 1.5 0 012.121 2.121L6.5 11.743l-2.5.757.757-2.5L11.5 2.5z" />
                  </svg>
                  <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 px-2 py-1 text-xs text-white bg-gray-900 dark:bg-gray-700 rounded whitespace-nowrap opacity-0 group-hover/edit:opacity-100 transition-opacity">
                    {t('common.edit')}
                  </span>
                </button>
                <button
                  className="group/del relative inline-flex items-center justify-center w-7 h-7 rounded-md text-red-400 hover:text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/30 transition-colors opacity-0 group-hover/entry:opacity-100"
                  onClick={() => setDeletingId(entry.id)}
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M3 4.5h10M6.5 4.5V3a1 1 0 011-1h1a1 1 0 011 1v1.5M5 4.5v8a1 1 0 001 1h4a1 1 0 001-1v-8" />
                  </svg>
                  <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 px-2 py-1 text-xs text-white bg-gray-900 dark:bg-gray-700 rounded whitespace-nowrap opacity-0 group-hover/del:opacity-100 transition-opacity">
                    {t('common.delete')}
                  </span>
                </button>
              </>
            )}
          </div>
          {entry.description && editingId !== entry.id && (
            <p className="text-sm text-gray-600 dark:text-gray-400 pl-8">{entry.description}</p>
          )}
          {editingId === entry.id && (
            <div className="mt-2 space-y-2">
              <div className="flex gap-2 items-center">
                <Input
                  type="text"
                  className="w-24 shrink-0"
                  value={editDuration}
                  onChange={(e) => setEditDuration(e.target.value)}
                  placeholder={t('timeTracking.durationPlaceholder')}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') { e.preventDefault(); handleUpdate() }
                    if (e.key === 'Escape') setEditingId(null)
                  }}
                />
                <Input
                  type="date"
                  className="shrink-0"
                  value={editStartedAt}
                  onChange={(e) => setEditStartedAt(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Escape') setEditingId(null)
                  }}
                />
                <Input
                  className="hidden sm:block flex-1 min-w-0"
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                  placeholder={t('timeTracking.descriptionPlaceholder')}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') { e.preventDefault(); handleUpdate() }
                    if (e.key === 'Escape') setEditingId(null)
                  }}
                />
                <Button
                  className="py-2 text-sm shrink-0 w-20"
                  onClick={handleUpdate}
                  disabled={!parseDurationString(editDuration) || updateMutation.isPending}
                >
                  {updateMutation.isPending ? t('common.saving') : t('common.save')}
                </Button>
                <Button
                  className="py-2 text-sm shrink-0 w-20"
                  variant="ghost"
                  onClick={() => setEditingId(null)}
                >
                  {t('common.cancel')}
                </Button>
              </div>
              <Input
                className="sm:hidden"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                placeholder={t('timeTracking.descriptionPlaceholder')}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') { e.preventDefault(); handleUpdate() }
                  if (e.key === 'Escape') setEditingId(null)
                }}
              />
            </div>
          )}
        </div>
      ))}

      <Modal open={!!deletingId} onClose={() => setDeletingId(null)} title={t('timeTracking.deleteTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          {t('timeTracking.deleteConfirm')}
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={() => setDeletingId(null)}>{t('common.cancel')}</Button>
          <Button
            variant="danger"
            onClick={() => {
              if (deletingId) {
                deleteMutation.mutate(deletingId, {
                  onSuccess: () => setDeletingId(null),
                })
              }
            }}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}

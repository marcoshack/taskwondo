import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Bell, X } from 'lucide-react'
import { useWatchers, useAddWatcher, useRemoveWatcher, useToggleWatch } from '@/hooks/useWorkItems'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'
import { Avatar } from '@/components/ui/Avatar'
import { useAuth } from '@/contexts/AuthContext'
import type { Watcher, ViewerWatcherResponse } from '@/api/workitems'
import type { ProjectMember } from '@/api/projects'

interface WatcherListProps {
  projectKey: string
  itemNumber: number
  members: ProjectMember[]
  currentUserRole: string | null
}

function isViewerResponse(data: unknown): data is ViewerWatcherResponse {
  return data !== null && typeof data === 'object' && 'other_count' in (data as Record<string, unknown>)
}

export function WatcherList({ projectKey, itemNumber, members, currentUserRole }: WatcherListProps) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { data: watcherData, isLoading } = useWatchers(projectKey, itemNumber)
  const addMutation = useAddWatcher(projectKey, itemNumber)
  const removeMutation = useRemoveWatcher(projectKey, itemNumber)
  const toggleMutation = useToggleWatch(projectKey, itemNumber)

  const isViewer = currentUserRole === 'viewer'
  const isAdminOrOwner = currentUserRole === 'owner' || currentUserRole === 'admin' || user?.global_role === 'admin'

  if (isLoading) return <Spinner size="sm" />

  // Viewer: restricted view
  if (isViewer && isViewerResponse(watcherData)) {
    const isWatching = !!watcherData.me
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-2 pb-3 border-b border-gray-100 dark:border-gray-700">
          <Button
            className="py-2 text-sm"
            onClick={() => toggleMutation.mutate()}
            disabled={toggleMutation.isPending}
          >
            <Bell className="h-4 w-4 mr-1.5" />
            {isWatching ? t('watchers.unwatch') : t('watchers.watch')}
          </Button>
        </div>
        {isWatching && (
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {t('watchers.youAreWatching')}
          </p>
        )}
        {watcherData.other_count > 0 && (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {isWatching
              ? t('watchers.otherWatchers', { count: watcherData.other_count })
              : t('watchers.totalWatchers', { count: watcherData.other_count })}
          </p>
        )}
        {!isWatching && watcherData.other_count === 0 && (
          <p className="text-sm text-gray-400">{t('watchers.noWatchers')}</p>
        )}
      </div>
    )
  }

  // Full list for members/admins/owners
  const watchers = (Array.isArray(watcherData) ? watcherData : []) as Watcher[]
  const isCurrentUserWatching = watchers.some((w) => w.user_id === user?.id)

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center gap-2 pb-3 border-b border-gray-100 dark:border-gray-700">
        <Button
          className="py-2 text-sm"
          onClick={() => toggleMutation.mutate()}
          disabled={toggleMutation.isPending}
        >
          <Bell className={`h-4 w-4 mr-1.5 ${isCurrentUserWatching ? 'fill-current' : ''}`} />
          {isCurrentUserWatching ? t('watchers.unwatch') : t('watchers.watch')}
        </Button>
        {!isViewer && (
          <AddWatcherForm
            projectKey={projectKey}
            itemNumber={itemNumber}
            members={members}
            existingWatcherIds={watchers.map((w) => w.user_id)}
            onAdd={(userId) => addMutation.mutate(userId)}
            isPending={addMutation.isPending}
          />
        )}
      </div>

      {watchers.length === 0 ? (
        <p className="text-sm text-gray-400">{t('watchers.noWatchers')}</p>
      ) : (
        <div className="space-y-1">
          {watchers.map((w) => {
            const isSelf = w.user_id === user?.id
            const canRemove = isSelf || isAdminOrOwner
            return (
              <div key={w.id} className="group/watcher flex items-center justify-between text-sm py-1.5">
                <div className="flex items-center gap-2 min-w-0">
                  <Avatar name={w.display_name} avatarUrl={w.avatar_url} size="sm" />
                  <span className="text-gray-900 dark:text-gray-100 truncate">{w.display_name}</span>
                  <span className="text-gray-400 dark:text-gray-500 text-xs truncate">{w.email}</span>
                  {isSelf && (
                    <span className="text-xs text-indigo-500 dark:text-indigo-400 font-medium shrink-0">
                      ({t('common.you')})
                    </span>
                  )}
                </div>
                {canRemove && (
                  <button
                    className="inline-flex items-center justify-center w-7 h-7 rounded-md text-red-400 hover:text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/30 transition-colors sm:opacity-0 sm:group-hover/watcher:opacity-100 shrink-0 ml-2"
                    onClick={() => removeMutation.mutate(w.user_id)}
                    aria-label={t('common.remove')}
                  >
                    <X className="w-4 h-4" />
                  </button>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

// --- Inline add-watcher form ---

interface AddWatcherFormProps {
  projectKey: string
  itemNumber: number
  members: ProjectMember[]
  existingWatcherIds: string[]
  onAdd: (userId: string) => void
  isPending: boolean
}

function AddWatcherForm({ members, existingWatcherIds, onAdd, isPending }: AddWatcherFormProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Include viewers (they can be added as watchers)
  const available = members.filter((m) => !existingWatcherIds.includes(m.user_id))
  const filtered = available.filter((m) => {
    if (!search) return true
    const q = search.toLowerCase()
    return m.display_name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q)
  })

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <div ref={ref} className="relative sm:flex-1">
      <Button
        variant="secondary"
        className="py-2 text-sm"
        onClick={() => { setOpen(!open); setTimeout(() => inputRef.current?.focus(), 0) }}
        disabled={isPending || available.length === 0}
      >
        {t('watchers.addWatcher')}
      </Button>

      {open && (
        <div className="absolute z-20 mt-1 w-64 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg">
          <div className="p-2">
            <input
              ref={inputRef}
              className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-500"
              placeholder={t('userPicker.searchMembers')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <ul className="max-h-48 overflow-auto">
            {filtered.map((m) => (
              <li key={m.user_id}>
                <button
                  type="button"
                  className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100"
                  onClick={() => {
                    onAdd(m.user_id)
                    setOpen(false)
                    setSearch('')
                  }}
                >
                  <div className="flex items-center gap-2">
                    <Avatar name={m.display_name} avatarUrl={m.avatar_url} size="xs" />
                    <div>
                      <div className="font-medium">{m.display_name}</div>
                      <div className="text-xs text-gray-400">{m.email}</div>
                    </div>
                  </div>
                </button>
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">{t('userPicker.noMembersFound')}</li>
            )}
          </ul>
        </div>
      )}
    </div>
  )
}

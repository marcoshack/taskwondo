import { useState, useMemo, useCallback, useRef } from 'react'
import { Routes, Route, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ChevronUp, ChevronDown, X, Search, BrushCleaning, Inbox, Check, Rss, Bookmark, Settings, User, History } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { Input } from '@/components/ui/Input'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { Tooltip } from '@/components/ui/Tooltip'
import { SLAIndicator } from '@/components/SLAIndicator'
import { AppSidebar } from '@/components/AppSidebar'
import { useSidebar } from '@/contexts/SidebarContext'
import { useInboxItems, useRemoveFromInbox, useReorderInboxItem, useClearCompletedInbox } from '@/hooks/useInbox'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useDebounce } from '@/hooks/useDebounce'
import { listInboxItems, type InboxItem } from '@/api/inbox'
import { formatRelativeTime } from '@/utils/duration'

const categoryColors = {
  todo: 'gray',
  in_progress: 'blue',
  done: 'green',
  cancelled: 'red',
} as const

function InboxStatusBadge({ status, category }: { status: string; category: string }) {
  const { t } = useTranslation()
  const color = categoryColors[category as keyof typeof categoryColors] ?? 'gray'
  const key = `workitems.statuses.${status}`
  const translated = t(key)
  return <Badge color={color}>{translated === key ? status : translated}</Badge>
}

// --- Table Row ---

interface InboxRowProps {
  item: InboxItem
  isCompleted: boolean
  isFirst: boolean
  isLast: boolean
  onRemove: (id: string) => void
  onMoveUp: (item: InboxItem) => void
  onMoveDown: (item: InboxItem) => void
  onClick: (item: InboxItem) => void
  removedId: string | null
  reorderedId: string | null
  autoRemove: boolean
}

function getDescriptionPreview(description: string): string {
  const line = description.split('\n').find(l => l.trim() !== '')
  if (!line) return ''
  return line.trim().replace(/^#+\s+/, '').replace(/[*_~`[\]]/g, '')
}

function InboxRow({ item, isCompleted, isFirst, isLast, onRemove, onMoveUp, onMoveDown, onClick, removedId, reorderedId, autoRemove }: InboxRowProps) {
  const { t } = useTranslation()
  const isRemoving = removedId === item.id

  return (
    <tr
      className={`group border-b border-gray-200 dark:border-gray-700
        ${reorderedId === item.id ? 'animate-[inbox-highlight_1s_ease-in-out]' : ''}
        ${isRemoving && autoRemove ? 'transition-all duration-300 opacity-0 -translate-y-2' : ''}
        hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer`}
      onClick={() => onClick(item)}
    >
      {/* Reorder arrows */}
      <td className={`w-10 px-1 py-3 ${isCompleted && !isRemoving ? 'opacity-40' : ''}`} onClick={(e) => e.stopPropagation()}>
        <div className="flex flex-col items-center -my-1">
          <button
            onClick={() => onMoveUp(item)}
            disabled={isFirst}
            className={`p-0.5 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-700 ${isFirst ? 'invisible' : ''}`}
            aria-label={t('inbox.moveUp')}
          >
            <ChevronUp className="h-4 w-4" />
          </button>
          <button
            onClick={() => onMoveDown(item)}
            disabled={isLast}
            className={`p-0.5 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-700 ${isLast ? 'invisible' : ''}`}
            aria-label={t('inbox.moveDown')}
          >
            <ChevronDown className="h-4 w-4" />
          </button>
        </div>
      </td>
      {/* Display ID */}
      <td className={`px-3 py-3 text-sm font-mono whitespace-nowrap ${isCompleted && !isRemoving ? 'text-gray-400 dark:text-gray-500' : 'text-gray-500 dark:text-gray-400'}`}>
        {item.display_id}
      </td>
      {/* Type */}
      <td className={`px-3 py-3 whitespace-nowrap ${isCompleted && !isRemoving ? 'opacity-40' : ''}`}>
        <TypeBadge type={item.type} />
      </td>
      {/* Title */}
      <td className={`px-3 py-3 text-sm truncate ${item.description && !isCompleted ? 'text-gray-400 dark:text-gray-500' : ''}`}>
        <span className={isCompleted && !isRemoving ? 'line-through text-gray-400 dark:text-gray-500' : 'text-gray-900 dark:text-gray-100'}>
          {item.title}
        </span>
        {item.description && !isCompleted && (
          <span className="text-xs"> – {getDescriptionPreview(item.description)}</span>
        )}
      </td>
      {/* Status — always shows color for visibility */}
      <td className="px-3 py-3 whitespace-nowrap">
        <InboxStatusBadge status={item.status} category={item.status_category} />
      </td>
      {/* Priority */}
      <td className={`px-3 py-3 whitespace-nowrap ${isCompleted && !isRemoving ? 'opacity-40' : ''}`}>
        <PriorityBadge priority={item.priority} />
      </td>
      {/* SLA */}
      <td className={`px-3 py-3 whitespace-nowrap overflow-hidden ${isCompleted && !isRemoving ? 'opacity-40' : ''}`}>
        <SLAIndicator sla={item.sla} />
      </td>
      {/* Updated */}
      <td className={`px-3 py-3 whitespace-nowrap text-sm text-right ${isCompleted && !isRemoving ? 'text-gray-300 dark:text-gray-600' : 'text-gray-500 dark:text-gray-400'}`}>
        {new Date(item.updated_at).toLocaleDateString()}
      </td>
      {/* Remove button */}
      <td className="w-10 px-2 py-3" onClick={(e) => e.stopPropagation()}>
        {isRemoving ? (
          <Check className="h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
        ) : (
          <button
            onClick={() => onRemove(item.id)}
            className="sm:opacity-0 sm:group-hover:opacity-100 text-gray-400 hover:text-red-500 transition-opacity"
            aria-label={t('inbox.removeFromInbox')}
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </td>
    </tr>
  )
}

// --- Mobile Card ---

interface InboxCardProps {
  item: InboxItem
  isCompleted: boolean
  isFirst: boolean
  isLast: boolean
  onRemove: (id: string) => void
  onMoveUp: (item: InboxItem) => void
  onMoveDown: (item: InboxItem) => void
  onClick: (item: InboxItem) => void
  removedId: string | null
  reorderedId: string | null
  autoRemove: boolean
}

function InboxCard({ item, isCompleted, isFirst, isLast, onRemove, onMoveUp, onMoveDown, onClick, removedId, reorderedId, autoRemove }: InboxCardProps) {
  const { t } = useTranslation()
  const isRemoving = removedId === item.id
  const dimmed = isCompleted && !isRemoving
  const assigneeName = item.assignee_display_name || t('userPicker.unassigned')

  return (
    <div
      className={`flex items-center gap-2 p-3 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-sm hover:border-indigo-300 dark:hover:border-indigo-600 transition-colors
        ${reorderedId === item.id ? 'animate-[inbox-highlight_1s_ease-in-out]' : ''}
        ${isRemoving && autoRemove ? 'transition-all duration-300 opacity-0 -translate-y-2' : ''}`}
    >
      {/* Reorder arrows — vertically centered */}
      <div className={`flex flex-col items-center flex-shrink-0 ${dimmed ? 'opacity-40' : ''}`}>
        <button
          onClick={() => onMoveUp(item)}
          disabled={isFirst}
          className={`p-0.5 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-700 ${isFirst ? 'invisible' : ''}`}
          aria-label={t('inbox.moveUp')}
        >
          <ChevronUp className="h-4 w-4" />
        </button>
        <button
          onClick={() => onMoveDown(item)}
          disabled={isLast}
          className={`p-0.5 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-700 ${isLast ? 'invisible' : ''}`}
          aria-label={t('inbox.moveDown')}
        >
          <ChevronDown className="h-4 w-4" />
        </button>
      </div>
      {/* Card content */}
      <button
        className="flex-1 text-left min-w-0"
        onClick={() => onClick(item)}
      >
        {/* Line 1: Display ID + badges */}
        <div className="flex items-center gap-2 flex-wrap">
          <span className={`font-mono text-sm font-semibold ${dimmed ? 'text-gray-400 dark:text-gray-500' : 'text-gray-700 dark:text-gray-300'}`}>{item.display_id}</span>
          <span className={`inline-flex ${dimmed ? 'opacity-40' : ''}`}><TypeBadge type={item.type} /></span>
          <InboxStatusBadge status={item.status} category={item.status_category} />
          <span className={`inline-flex ${dimmed ? 'opacity-40' : ''}`}><PriorityBadge priority={item.priority} /></span>
          {!dimmed && item.sla && <span className="ml-auto"><SLAIndicator sla={item.sla} compact /></span>}
        </div>
        {/* Line 2: Assignee, Updated, SLA (when no compact SLA above) */}
        <div className={`flex items-center gap-3 mt-1.5 text-xs ${dimmed ? 'text-gray-300 dark:text-gray-600' : 'text-gray-400 dark:text-gray-500'}`}>
          <span className="inline-flex items-center gap-1">
            <User className="h-3 w-3" />
            <span className="truncate max-w-[8rem]">{assigneeName}</span>
          </span>
          <span className="inline-flex items-center gap-1">
            <History className="h-3 w-3" />
            {formatRelativeTime(item.updated_at)}
          </span>
        </div>
        {/* Line 3: Title */}
        <p className={`mt-1.5 text-sm font-medium truncate ${dimmed ? 'line-through text-gray-400 dark:text-gray-500' : 'text-gray-900 dark:text-gray-100'}`}>
          {item.title}
        </p>
        {/* Line 4: Description (first line) */}
        {item.description && (
          <p className={`mt-0.5 text-xs truncate ${dimmed ? 'text-gray-300 dark:text-gray-600' : 'text-gray-500 dark:text-gray-400'}`}>
            {item.description}
          </p>
        )}
      </button>
      {/* Remove button */}
      <button
        onClick={() => onRemove(item.id)}
        className="text-gray-400 hover:text-red-500 flex-shrink-0"
        aria-label={t('inbox.removeFromInbox')}
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}

// --- Inbox List Page (main inbox content) ---

function InboxListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [searchInput, setSearchInput] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)
  const debouncedSearch = useDebounce(searchInput, 300)

  useKeyboardShortcut({ key: '/' }, () => searchRef.current?.focus())

  const { data: autoRemovePref } = usePreference<boolean>('inbox_auto_remove')
  const autoRemove = autoRemovePref ?? true
  const setPreferenceMutation = useSetPreference()

  const { data, isLoading } = useInboxItems({
    search: debouncedSearch || undefined,
    include_completed: true,
  })

  const [settingsOpen, setSettingsOpen] = useState(false)

  const [loadedPages, setLoadedPages] = useState<InboxItem[][]>([])
  const [prevSearch, setPrevSearch] = useState(debouncedSearch)
  if (debouncedSearch !== prevSearch) {
    setPrevSearch(debouncedSearch)
    setLoadedPages([])
  }

  const allLoadedItems = useMemo(() => {
    if (!data) return []
    if (loadedPages.length === 0) return data.items
    return [...data.items, ...loadedPages.flat()]
  }, [data, loadedPages])

  const completedItems = useMemo(() =>
    allLoadedItems.filter((i) => i.status_category === 'done' || i.status_category === 'cancelled'),
    [allLoadedItems],
  )

  const allItems = useMemo(() =>
    autoRemove ? allLoadedItems.filter((i) => i.status_category !== 'done' && i.status_category !== 'cancelled') : allLoadedItems,
    [allLoadedItems, autoRemove],
  )

  const hasMore = loadedPages.length > 0
    ? loadedPages[loadedPages.length - 1].length > 0 // approximate
    : (data?.has_more ?? false)

  const removeMutation = useRemoveFromInbox()
  const reorderMutation = useReorderInboxItem()
  const clearCompletedMutation = useClearCompletedInbox()

  const [removedId, setRemovedId] = useState<string | null>(null)
  const [reorderedId, setReorderedId] = useState<string | null>(null)

  const handleRemove = useCallback((inboxItemId: string) => {
    setRemovedId(inboxItemId)
    setTimeout(() => {
      removeMutation.mutate(inboxItemId)
      setRemovedId(null)
    }, autoRemove ? 300 : 0)
  }, [removeMutation, autoRemove])

  const handleClick = useCallback((item: InboxItem) => {
    navigate(`/projects/${item.project_key}/items/${item.display_id.split('-')[1]}`, { state: { from: 'inbox' } })
  }, [navigate])

  // Reorder via arrow buttons — place item directly before/after neighbor
  const handleMoveUp = useCallback((item: InboxItem) => {
    const index = allItems.findIndex((i) => i.id === item.id)
    if (index <= 0) return
    const prev = allItems[index - 1]
    reorderMutation.mutate({ inboxItemId: item.id, position: prev.position - 1 }, {
      onSuccess: () => {
        setReorderedId(item.id)
        setTimeout(() => setReorderedId(null), 1000)
      },
    })
  }, [allItems, reorderMutation])

  const handleMoveDown = useCallback((item: InboxItem) => {
    const index = allItems.findIndex((i) => i.id === item.id)
    if (index < 0 || index >= allItems.length - 1) return
    const next = allItems[index + 1]
    reorderMutation.mutate({ inboxItemId: item.id, position: next.position + 1 }, {
      onSuccess: () => {
        setReorderedId(item.id)
        setTimeout(() => setReorderedId(null), 1000)
      },
    })
  }, [allItems, reorderMutation])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    )
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('inbox.title')}</h1>
          {data && (
            <span className="text-sm text-gray-500 dark:text-gray-400">
              ({data.total})
            </span>
          )}
        </div>
        {/* Desktop: auto-hide toggle */}
        <Tooltip content={t('inbox.autoRemoveDescription')}>
          <label className="hidden sm:flex items-center gap-2 cursor-pointer select-none">
            <span className="text-sm text-gray-600 dark:text-gray-400">{t('inbox.autoRemove')}</span>
            <button
              type="button"
              role="switch"
              aria-checked={autoRemove}
              onClick={() => setPreferenceMutation.mutate({ key: 'inbox_auto_remove', value: !autoRemove })}
              className={`relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors ${
                autoRemove ? 'bg-indigo-600' : 'bg-gray-200 dark:bg-gray-600'
              }`}
            >
              <span className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition-transform ${
                autoRemove ? 'translate-x-4' : 'translate-x-0'
              }`} />
            </button>
          </label>
        </Tooltip>
      </div>

      {/* Mobile settings modal */}
      <Modal open={settingsOpen} onClose={() => setSettingsOpen(false)} title={t('inbox.settings')}>
        <div className="py-2">
          <label className="flex items-center justify-between cursor-pointer select-none">
            <span className="text-sm text-gray-700 dark:text-gray-300">{t('inbox.autoRemove')}</span>
            <button
              type="button"
              role="switch"
              aria-checked={autoRemove}
              onClick={() => setPreferenceMutation.mutate({ key: 'inbox_auto_remove', value: !autoRemove })}
              className={`relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors ${
                autoRemove ? 'bg-indigo-600' : 'bg-gray-200 dark:bg-gray-600'
              }`}
            >
              <span className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition-transform ${
                autoRemove ? 'translate-x-4' : 'translate-x-0'
              }`} />
            </button>
          </label>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t('inbox.autoRemoveDescription')}</p>
        </div>
      </Modal>

      {/* Search + icons */}
      <div className="flex items-center gap-2 mb-4">
        <div className="relative flex-1 min-w-0">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            ref={searchRef}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder={t('inbox.searchPlaceholder')}
            className="pl-10"
            onKeyDown={(e) => { if (e.key === 'Escape') searchRef.current?.blur() }}
          />
        </div>
        {/* Clear completed */}
        <Tooltip content={completedItems.length > 0 ? `${t('inbox.clearCompleted')} (${completedItems.length})` : t('inbox.noCompletedItems')}>
          <button
            onClick={() => clearCompletedMutation.mutate()}
            disabled={clearCompletedMutation.isPending || completedItems.length === 0}
            className={`relative p-2 rounded-lg border border-gray-300 dark:border-gray-600 transition-colors ${
              completedItems.length === 0
                ? 'opacity-40 cursor-not-allowed text-gray-400 dark:text-gray-500'
                : 'text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-300'
            }`}
            aria-label={t('inbox.clearCompleted')}
          >
            <BrushCleaning className="h-5 w-5" />
            {completedItems.length > 0 && (
              <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
                {completedItems.length}
              </span>
            )}
          </button>
        </Tooltip>
        {/* Mobile: settings button */}
        <button
          onClick={() => setSettingsOpen(true)}
          className="sm:hidden p-2 rounded-lg border border-gray-300 dark:border-gray-600 text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-300 transition-colors"
          aria-label={t('inbox.settings')}
        >
          <Settings className="h-5 w-5" />
        </button>
      </div>

      {allItems.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-gray-500 dark:text-gray-400">
          <Inbox className="h-16 w-16 mb-4 opacity-30" />
          <p className="text-lg font-medium">{t('inbox.empty')}</p>
          <p className="text-sm mt-1">{t('inbox.emptyHint')}</p>
        </div>
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden sm:block overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
            <table className="w-full table-fixed">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="w-10 px-1 py-3"></th>
                  <th className="w-24 px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">{t('workitems.table.id')}</th>
                  <th className="w-20 px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">{t('workitems.table.type')}</th>
                  <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">{t('workitems.table.title')}</th>
                  <th className="w-28 px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">{t('workitems.table.status')}</th>
                  <th className="w-24 px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">{t('workitems.table.priority')}</th>
                  <th className="w-[110px] px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">{t('sla.columnHeader')}</th>
                  <th className="w-[130px] px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">{t('workitems.table.updated')}</th>
                  <th className="w-10 px-2 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {allItems.map((item, index) => (
                  <InboxRow
                    key={item.id}
                    item={item}
                    isCompleted={item.status_category === 'done' || item.status_category === 'cancelled'}
                    isFirst={index === 0}
                    isLast={index === allItems.length - 1}
                    onRemove={handleRemove}
                    onMoveUp={handleMoveUp}
                    onMoveDown={handleMoveDown}
                    onClick={handleClick}
                    removedId={removedId}
                    reorderedId={reorderedId}
                    autoRemove={autoRemove}
                  />
                ))}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="sm:hidden space-y-2">
            {allItems.map((item, index) => (
              <InboxCard
                key={item.id}
                item={item}
                isCompleted={item.status_category === 'done' || item.status_category === 'cancelled'}
                isFirst={index === 0}
                isLast={index === allItems.length - 1}
                onRemove={handleRemove}
                onMoveUp={handleMoveUp}
                onMoveDown={handleMoveDown}
                onClick={handleClick}
                removedId={removedId}
                reorderedId={reorderedId}
                autoRemove={autoRemove}
              />
            ))}
          </div>
        </>
      )}

      {/* Load more */}
      {hasMore && (
        <div className="flex justify-center pt-4">
          <Button
            variant="secondary"
            onClick={async () => {
              const lastItem = allItems[allItems.length - 1]
              if (!lastItem) return
              const next = await listInboxItems({
                search: debouncedSearch || undefined,
                include_completed: true,
                cursor: lastItem.id,
              })
              setLoadedPages((prev) => [...prev, next.items])
            }}
          >
            {t('common.loadMore')}
          </Button>
        </div>
      )}
    </div>
  )
}

// --- Feed Page (placeholder) ---

function FeedPage() {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Rss className="h-12 w-12 mb-4 opacity-30" />
      <p className="text-lg font-medium">{t('user.feedComingSoon')}</p>
    </div>
  )
}

// --- Watchlist Page (placeholder) ---

function WatchlistPage() {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <Bookmark className="h-12 w-12 mb-4 opacity-30" />
      <p className="text-lg font-medium">{t('user.watchlistComingSoon')}</p>
    </div>
  )
}

// --- Layout Page ---

export default function UserPage() {
  const { collapsed } = useSidebar('app')

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 h-full flex flex-col">
      <div className={`flex flex-1 min-h-0 transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <AppSidebar />
        <div className="flex-1 min-w-0">
          <Routes>
            <Route path="inbox" element={<InboxListPage />} />
            <Route path="feed" element={<FeedPage />} />
            <Route path="watchlist" element={<WatchlistPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

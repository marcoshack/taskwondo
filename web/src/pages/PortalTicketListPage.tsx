import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePortalTickets, usePortalQueues, useCreatePortalTicket } from '@/hooks/usePortal'
import { useDebounce } from '@/hooks/useDebounce'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { RefreshButton, type RefreshInterval } from '@/components/ui/RefreshButton'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { Plus, Search, CalendarPlus, History, Settings } from 'lucide-react'

const PRIORITIES = ['low', 'medium', 'high', 'critical'] as const

function getDescriptionPreview(description: string): string {
  const line = description.split('\n').find(l => l.trim() !== '')
  if (!line) return ''
  return line.trim().replace(/^#+\s+/, '').replace(/[*_~`[\]]/g, '')
}

export function PortalTicketListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { namespace = 'default', projectKey = '' } = useParams<{ namespace: string; projectKey: string }>()
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)

  const { data: hideCompletedPref } = usePreference<boolean>('portal_hide_completed')
  const { data: refreshIntervalPref } = usePreference<number>('portal_refresh_interval')
  const setPreferenceMutation = useSetPreference()
  const hideCompleted = hideCompletedPref ?? false
  const [refreshInterval, setRefreshInterval] = useState<RefreshInterval>(0)
  const [settingsOpen, setSettingsOpen] = useState(false)

  useEffect(() => {
    if (refreshIntervalPref != null) setRefreshInterval(refreshIntervalPref as RefreshInterval)
  }, [refreshIntervalPref])

  const { data, isLoading, refetch, isFetching } = usePortalTickets(namespace, projectKey, {
    search: debouncedSearch || undefined,
    hide_completed: hideCompleted || undefined,
  }, refreshInterval)
  const { data: queues } = usePortalQueues(namespace, projectKey)
  const createMutation = useCreatePortalTicket(namespace, projectKey)
  const publicQueue = queues?.[0]
  const hasPublicQueue = !!publicQueue

  const [createOpen, setCreateOpen] = useState(false)
  const [categoryId, setCategoryId] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState('medium')
  const [createError, setCreateError] = useState('')

  function resetForm() {
    setTitle('')
    setDescription('')
    setCategoryId('')
    setPriority('medium')
    setCreateError('')
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setCreateError('')
    if (!title.trim()) {
      setCreateError(t('portal.errorTitleRequired'))
      return
    }
    try {
      const ticket = await createMutation.mutateAsync({
        title: title.trim(),
        description: description.trim() || undefined,
        priority,
        category_id: categoryId || undefined,
      })
      setCreateOpen(false)
      resetForm()
      navigate(`${ticket.item_number}`)
    } catch {
      setCreateError(t('portal.errorCreateFailed'))
    }
  }

  const tickets = data?.items ?? []

  return (
    <div>
      {/* Mobile header */}
      <div className="flex sm:hidden items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-[var(--foreground)]">
          {t('portal.myTickets')}
        </h1>
        {hasPublicQueue && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4 mr-1" />
            {t('portal.createTicket')}
          </Button>
        )}
      </div>

      <Modal open={createOpen} onClose={() => { setCreateOpen(false); resetForm() }} title={t('portal.createTicket')}>
        <form onSubmit={handleCreate} className="space-y-4">
          {publicQueue && publicQueue.categories.length > 0 && (
            <Select label={t('portal.category')} value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
              <option value="">{t('portal.selectCategory')}</option>
              {publicQueue.categories.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </Select>
          )}
          <Input
            label={`${t('portal.ticketTitle')} *`}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={t('portal.ticketTitle')}
          />
          <div>
            <label className="block text-sm font-medium text-[var(--foreground)] mb-1">
              {t('portal.ticketDescription')}
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              className="block w-full rounded-md border border-[var(--border)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)] bg-[var(--surface)] text-[var(--foreground)] placeholder-[var(--foreground-muted)]"
              placeholder={t('portal.ticketDescription')}
            />
          </div>
          <Select label={t('portal.ticketPriority')} value={priority} onChange={(e) => setPriority(e.target.value)}>
            {PRIORITIES.map((p) => (
              <option key={p} value={p}>{t(`workitems.priorities.${p}`)}</option>
            ))}
          </Select>
          {createError && (
            <p className="text-sm text-[var(--danger)]">{createError}</p>
          )}
          <div className="flex justify-end gap-3 pt-2">
            <Button type="button" variant="secondary" onClick={() => { setCreateOpen(false); resetForm() }}>
              {t('portal.cancel')}
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? t('common.creating') : t('portal.submit')}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Desktop header + toolbar (single row) */}
      <div className="hidden sm:flex items-center gap-3 mb-4">
        <h1 className="text-xl font-semibold text-[var(--foreground)] whitespace-nowrap">
          {t('portal.myTickets')}
        </h1>
        <div className="relative flex-1 mx-3">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--foreground-muted)]" />
          <Input
            placeholder={t('common.search')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <label className="flex items-center gap-2 text-sm text-[var(--foreground-secondary)] cursor-pointer select-none whitespace-nowrap">
          {t('portal.hideCompleted')}
          <button
            type="button"
            role="switch"
            aria-checked={hideCompleted}
            onClick={() => setPreferenceMutation.mutate({ key: 'portal_hide_completed', value: !hideCompleted })}
            className={`relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors ${
              hideCompleted ? 'bg-[var(--primary)]' : 'bg-[var(--surface-tertiary)] dark:bg-[var(--foreground-secondary)]'
            }`}
          >
            <span className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition-transform ${
              hideCompleted ? 'translate-x-4' : 'translate-x-0'
            }`} />
          </button>
        </label>
        <RefreshButton
          interval={refreshInterval}
          onIntervalChange={(val) => {
            setRefreshInterval(val)
            setPreferenceMutation.mutate({ key: 'portal_refresh_interval', value: val })
          }}
          onRefresh={() => refetch()}
          isRefreshing={isFetching}
        />
        {hasPublicQueue && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4 mr-1" />
            {t('portal.createTicket')}
          </Button>
        )}
      </div>

      {/* Mobile toolbar */}
      <div className="flex sm:hidden items-stretch gap-2 mb-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--foreground-muted)]" />
          <Input
            placeholder={t('common.search')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <button
          onClick={() => setSettingsOpen(true)}
          className="shrink-0 px-2.5 flex items-center rounded-md border border-[var(--border)] text-[var(--foreground-secondary)] hover:bg-[var(--surface-hover)]"
          aria-label={t('workitems.settings.title')}
        >
          <Settings className="h-5 w-5" />
        </button>
        <RefreshButton
          interval={refreshInterval}
          onIntervalChange={(val) => {
            setRefreshInterval(val)
            setPreferenceMutation.mutate({ key: 'portal_refresh_interval', value: val })
          }}
          onRefresh={() => refetch()}
          isRefreshing={isFetching}
        />
      </div>

      {/* Mobile settings modal */}
      <Modal open={settingsOpen} onClose={() => setSettingsOpen(false)} title={t('workitems.settings.title')} position="top" containerClassName="!pt-[10.5rem]">
        <label className="flex items-center justify-between cursor-pointer">
          <span className="text-sm font-medium text-[var(--foreground)]">{t('portal.hideCompleted')}</span>
          <button
            type="button"
            role="switch"
            aria-checked={hideCompleted}
            onClick={() => setPreferenceMutation.mutate({ key: 'portal_hide_completed', value: !hideCompleted })}
            className={`relative inline-flex h-6 w-11 shrink-0 rounded-full border-2 border-transparent transition-colors ${
              hideCompleted ? 'bg-[var(--primary)]' : 'bg-[var(--surface-tertiary)] dark:bg-[var(--foreground-secondary)]'
            }`}
          >
            <span className={`pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform ${
              hideCompleted ? 'translate-x-5' : 'translate-x-0'
            }`} />
          </button>
        </label>
      </Modal>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Spinner size="lg" />
        </div>
      ) : tickets.length === 0 ? (
        <p className="text-sm text-[var(--foreground-secondary)] py-8 text-center">
          {t('portal.noTickets')}
        </p>
      ) : (
        <div className="space-y-2">
          {tickets.map((ticket) => {
            const isCompleted = !!ticket.resolved_at
            return (
              <Link
                key={ticket.id}
                to={`${ticket.item_number}`}
                className="block rounded-lg border border-[var(--border)] bg-[var(--surface)] p-4 hover:border-[var(--primary-border)] dark:hover:border-[var(--primary-border)] transition-colors"
              >
                {/* Desktop: single row */}
                <div className="hidden sm:flex sm:items-center sm:gap-4">
                  <span className={`shrink-0 font-mono text-sm font-semibold ${isCompleted ? 'text-[var(--foreground-muted)]' : 'text-[var(--foreground)]'}`}>
                    {ticket.display_id}
                  </span>
                  <p className={`flex-1 min-w-0 truncate text-sm ${!isCompleted && ticket.description ? 'text-[var(--foreground-muted)]' : ''}`}>
                    <span className={isCompleted ? 'line-through text-[var(--foreground-muted)]' : 'font-medium text-[var(--foreground)]'}>
                      {ticket.title}
                    </span>
                    {ticket.description && !isCompleted && (
                      <span className="font-normal text-xs"> – {getDescriptionPreview(ticket.description)}</span>
                    )}
                  </p>
                  <span className="shrink-0 inline-flex"><StatusBadge status={ticket.status} /></span>
                  <span className={`shrink-0 inline-flex ${isCompleted ? 'opacity-40' : ''}`}><PriorityBadge priority={ticket.priority} /></span>
                  <div className={`shrink-0 flex items-center gap-3 text-xs ${isCompleted ? 'text-[var(--foreground-secondary)]' : 'text-[var(--foreground-muted)]'}`}>
                    <span className="inline-flex items-center gap-1" title={new Date(ticket.created_at).toLocaleString()}>
                      <CalendarPlus className="h-3.5 w-3.5" />
                      {new Date(ticket.created_at).toLocaleDateString()}
                    </span>
                    <span className="inline-flex items-center gap-1" title={new Date(ticket.updated_at).toLocaleString()}>
                      <History className="h-3.5 w-3.5" />
                      {new Date(ticket.updated_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>

                {/* Mobile: stacked layout */}
                <div className="sm:hidden">
                  {/* Row 1: Display ID + badges */}
                  <ScrollableRow contentClassName="gap-2" gradientFrom="from-white dark:from-gray-800">
                    <span className={`shrink-0 font-mono text-sm font-semibold ${isCompleted ? 'text-[var(--foreground-muted)]' : 'text-[var(--foreground)]'}`}>{ticket.display_id}</span>
                    <span className="shrink-0 inline-flex"><StatusBadge status={ticket.status} /></span>
                    <span className={`shrink-0 inline-flex ${isCompleted ? 'opacity-40' : ''}`}><PriorityBadge priority={ticket.priority} /></span>
                  </ScrollableRow>
                  {/* Row 2: Dates */}
                  <ScrollableRow className="mt-1.5" contentClassName={`gap-4 text-xs ${isCompleted ? 'text-[var(--foreground-secondary)]' : 'text-[var(--foreground-muted)]'}`} gradientFrom="from-white dark:from-gray-800">
                    <span className="inline-flex items-center gap-1 shrink-0">
                      <CalendarPlus className="h-3.5 w-3.5" />
                      {new Date(ticket.created_at).toLocaleString()}
                    </span>
                    <span className="inline-flex items-center gap-1 shrink-0">
                      <History className="h-3.5 w-3.5" />
                      {new Date(ticket.updated_at).toLocaleString()}
                    </span>
                  </ScrollableRow>
                  {/* Row 3: Title */}
                  <p className={`mt-1.5 text-base font-medium truncate ${isCompleted ? 'line-through text-[var(--foreground-muted)]' : 'text-[var(--foreground)]'}`}>
                    {ticket.title}
                  </p>
                  {/* Row 4: Description preview */}
                  {ticket.description && (
                    <p className={`mt-0.5 text-xs truncate ${isCompleted ? 'text-[var(--foreground-secondary)]' : 'text-[var(--foreground-secondary)]'}`}>
                      {ticket.description}
                    </p>
                  )}
                </div>
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}

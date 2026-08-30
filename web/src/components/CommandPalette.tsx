import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useNamespacePath, toUrlSegment } from '@/hooks/useNamespacePath'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { useAuth } from '@/contexts/AuthContext'
import {
  Search,
  FileText,
  FolderKanban,
  MessageSquare,
  Milestone,
  Layers,
  Paperclip,
  Users,
  Loader2,
  SearchX,
  AlertCircle,
  FlaskConical,
} from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { useSearch } from '@/hooks/useSearch'
import { useDebounce } from '@/hooks/useDebounce'
import { usePreference } from '@/hooks/usePreferences'
import { useLastProjectKey } from '@/hooks/useLastProjectKey'
import { useNavigationCatalog } from '@/hooks/useNavigationCatalog'
import {
  composePalette,
  entityRowId,
  isPaletteToggleEvent,
  moveSelection,
  navRowId,
  rowIndex,
} from '@/utils/paletteResults'
import type { PaletteEntitySection, PaletteRow } from '@/utils/paletteResults'
import type { NavigationEntry } from '@/utils/navigationCatalog'
import type { SearchResult } from '@/api/search'

function entityIcon(type: string) {
  switch (type) {
    case 'project':
      return FolderKanban
    case 'work_item':
      return FileText
    case 'milestone':
      return Milestone
    case 'queue':
      return Layers
    case 'team':
      return Users
    case 'comment':
      return MessageSquare
    case 'attachment':
      return Paperclip
    default:
      return FileText
  }
}

/** Extract a clean display line from the indexed content. */
function parseSnippet(result: SearchResult): { type?: string; text: string } {
  const s = result.snippet
  // Work items: "[task] Title\n\nDescription..."
  if (result.entity_type === 'work_item') {
    const m = s.match(/^\[(\w+)]\s*(.*)/)
    if (m) return { type: m[1], text: m[2].split('\n')[0] }
  }
  // Comments: "Comment:\n\nBody..."
  if (result.entity_type === 'comment') {
    const body = s.replace(/^Comment:\s*\n*/, '')
    return { text: body.split('\n')[0] }
  }
  // Attachments: "Attachment filename\nComment: ..."
  if (result.entity_type === 'attachment') {
    const line = s.replace(/^Attachment\s+/, '')
    return { text: line.split('\n')[0] }
  }
  // Everything else: first line
  return { text: s.split('\n')[0] }
}

export interface CommandPaletteProps {
  open: boolean
  onClose: () => void
  /**
   * Project to scope entity search to and list sections for. Defaults to the
   * last project the user opened — the same source `AppSidebar` falls back to.
   */
  projectKey?: string | null
}

/**
 * The Cmd/Ctrl+K command palette: everywhere the user can go on top, entity
 * search below, one selection running across both.
 *
 * Navigation comes from the in-memory catalog and matches from the first
 * character; entity hits keep the debounce and two-character floor of the
 * search modal this replaces, and are scoped to the active project (except
 * `project` hits, which the backend deliberately keeps global so a project row
 * is how you jump elsewhere).
 */
export function CommandPalette({ open, onClose, projectKey }: CommandPaletteProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { guardedNavigate } = useNavigationGuard()
  const { segment } = useNamespacePath()
  const { setActiveNamespace, showSwitcher } = useNamespaceContext()
  const { user } = useAuth()

  const lastProjectKey = useLastProjectKey()
  const activeProjectKey = projectKey ?? lastProjectKey ?? undefined

  // Set of project keys where the authenticated user holds the "customer"
  // role — results in these projects must route to the portal (support) view,
  // NOT the regular /items/ URL which they can't access. Global admins never
  // have portal_projects populated, so they always route to /items/.
  const customerProjectKeys = useMemo(
    () => new Set((user?.portal_projects ?? []).map((p) => p.project_key)),
    [user?.portal_projects],
  )

  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [limit, setLimit] = useState(20)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)

  const { data: strikethroughPref } = usePreference<boolean>('strikethrough_completed')
  const strikethroughEnabled = strikethroughPref ?? true

  const catalog = useNavigationCatalog(activeProjectKey)
  // Namespace rows are meaningless on an instance with namespaces turned off,
  // the same condition AppSidebar hides its switcher under.
  const navEntries = useMemo(
    () => (showSwitcher ? catalog : catalog.filter((e) => e.group !== 'namespace')),
    [catalog, showSwitcher],
  )

  // 400 ms, unchanged from the search modal this replaces.
  const debouncedQuery = useDebounce(query, 400)
  const {
    ftsResults,
    semanticResults,
    semanticAvailable,
    semanticStatus,
    semanticError,
    isLoading,
  } = useSearch({ query: debouncedQuery, limit, project: activeProjectKey })

  const { navSections, ftsSections, semanticSections, rows } = useMemo(
    () => composePalette({ navEntries, query, ftsResults, semanticResults }),
    [navEntries, query, ftsResults, semanticResults],
  )

  const hasFtsResults = ftsSections.length > 0
  const hasSemanticResults = semanticSections.length > 0
  const ftsCount = useMemo(
    () => ftsSections.reduce((n, s) => n + s.results.length, 0),
    [ftsSections],
  )
  // Show entity section headers when semantic is available and there's
  // something to separate.
  const showSectionHeaders =
    semanticAvailable && (hasFtsResults || hasSemanticResults || semanticStatus === 'pending')

  // Entity search only runs past the two-character floor; below it the palette
  // is navigation-only and says so.
  const belowEntityFloor = query.length > 0 && query.length < 2
  const showInitialLoading = isLoading && !hasFtsResults && !hasSemanticResults

  // Focus and select input when the palette opens (preserve previous query).
  useEffect(() => {
    if (open) {
      setSelectedIndex(0)
      setTimeout(() => {
        inputRef.current?.focus()
        inputRef.current?.select()
      }, 0)
    }
  }, [open])

  // Reset the selection whenever what's on screen changes — navigation reacts
  // to every keystroke, entity results to each settled request.
  useEffect(() => {
    setSelectedIndex(0)
  }, [query, ftsResults, semanticResults])

  const scrollSelectedIntoView = useCallback((index: number) => {
    const container = listRef.current
    if (!container) return
    const items = container.querySelectorAll('[data-palette-item]')
    const item = items[index] as HTMLElement | undefined
    item?.scrollIntoView({ block: 'nearest' })
  }, [])

  const openNavigationEntry = useCallback(
    (entry: NavigationEntry) => {
      onClose()
      if (entry.target.kind === 'namespace') {
        // Switching namespace is context state, not a URL; the context itself
        // navigates to that namespace's projects page.
        setActiveNamespace(entry.target.slug)
      } else {
        guardedNavigate(entry.target.to)
      }
    },
    [onClose, setActiveNamespace, guardedNavigate],
  )

  const openResult = useCallback(
    (result: SearchResult) => {
      onClose()
      const key = result.project_key
      const num = result.item_number

      // Use the result's namespace if available, otherwise fall back to current
      const nsSegment = result.namespace_slug ? toUrlSegment(result.namespace_slug) : segment
      const prefix = (path: string) => `/${nsSegment}${path.startsWith('/') ? path : `/${path}`}`

      // If the user is a customer in the result's project, the regular
      // /items/:num route is blocked by ExcludeCustomer on the backend and
      // the customer AppShell only renders /support/:num. Route accordingly.
      const isCustomerProject = !!key && customerProjectKeys.has(key)

      switch (result.entity_type) {
        case 'work_item':
          if (key && num != null) {
            navigate(prefix(isCustomerProject
              ? `/projects/${key}/support/${num}`
              : `/projects/${key}/items/${num}`))
          }
          break
        case 'comment':
          if (key && num != null) {
            // PortalTicketDetailPage manages tabs via state, not query params,
            // so for customer projects we simply open the ticket.
            navigate(prefix(isCustomerProject
              ? `/projects/${key}/support/${num}`
              : `/projects/${key}/items/${num}?tab=comments&highlight=${result.entity_id}`))
          }
          break
        case 'attachment':
          if (key && num != null) {
            navigate(prefix(isCustomerProject
              ? `/projects/${key}/support/${num}`
              : `/projects/${key}/items/${num}?tab=attachments&highlight=${result.entity_id}`))
          }
          break
        case 'project':
          if (key) navigate(prefix(isCustomerProject ? `/projects/${key}/support` : `/projects/${key}`))
          break
        case 'milestone':
          // Customers have no access to milestones; fall back to support list.
          if (key) navigate(prefix(isCustomerProject ? `/projects/${key}/support` : `/projects/${key}/milestones`))
          break
        case 'queue':
          // Customers have no access to queues; fall back to support list.
          if (key) navigate(prefix(isCustomerProject ? `/projects/${key}/support` : `/projects/${key}/queues`))
          break
        case 'team':
          // Customers have no access to teams; fall back to support list.
          if (key) navigate(prefix(isCustomerProject ? `/projects/${key}/support` : `/projects/${key}/teams/${result.entity_id}`))
          break
      }
    },
    [navigate, onClose, segment, customerProjectKeys],
  )

  const openRow = useCallback(
    (row: PaletteRow) => {
      if (row.kind === 'nav') openNavigationEntry(row.entry)
      else openResult(row.result)
    },
    [openNavigationEntry, openResult],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // A second Cmd/Ctrl+K closes the palette. It has to be handled here:
      // KeyboardShortcutContext skips the global handler while a modal is open
      // and again for INPUT targets, so it never sees this press.
      if (isPaletteToggleEvent(e)) {
        e.preventDefault()
        onClose()
        return
      }
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault()
        const next = moveSelection(selectedIndex, e.key === 'ArrowDown' ? 1 : -1, rows.length)
        setSelectedIndex(next)
        scrollSelectedIntoView(next)
        return
      }
      if (e.key === 'Enter') {
        const row = rows[selectedIndex]
        if (row) {
          e.preventDefault()
          openRow(row)
        }
      }
      // Escape is deliberately not handled — it falls through to Modal.
    },
    [selectedIndex, rows, openRow, scrollSelectedIntoView, onClose],
  )

  /** Heading for a navigation group; the project group names the project. */
  const navGroupLabel = (group: NavigationEntry['group']) =>
    group === 'project'
      ? t('palette.group.project', { project: activeProjectKey ?? '' })
      : t(`palette.group.${group}`)

  const rowClass = (index: number) =>
    `w-full text-left flex items-center gap-3 px-3 py-2 rounded-md text-sm ${
      index === selectedIndex
        ? 'bg-[var(--primary-muted)]'
        : 'hover:bg-[var(--surface-hover)]'
    }`

  const sectionHeading = (text: string) => (
    <span className="text-xs font-semibold uppercase tracking-wider text-[var(--foreground-muted)]">
      {text}
    </span>
  )

  const renderEntitySections = (sections: PaletteEntitySection[], showScores: boolean) =>
    sections.map((section) => {
      const Icon = entityIcon(section.entityType)
      return (
        <div key={`${showScores ? 'semantic' : 'fts'}-${section.entityType}`} className="mb-2 last:mb-0">
          <div className="flex items-center gap-2 px-3 py-1.5">
            <Icon className="h-3.5 w-3.5 text-[var(--foreground-muted)]" />
            {sectionHeading(t(`search.entityType.${section.entityType}`))}
          </div>
          {section.results.map((result) => {
            const id = entityRowId(showScores ? 'semantic' : 'fts', result)
            const index = rowIndex(rows, id)
            const parsed = parseSnippet(result)
            const ItemIcon = entityIcon(result.entity_type)
            const isCompleted = strikethroughEnabled && result.entity_type === 'work_item' &&
              (result.status_category === 'done' || result.status_category === 'cancelled')
            return (
              <button
                key={id}
                data-palette-item
                data-search-item
                onClick={() => openResult(result)}
                onMouseEnter={() => setSelectedIndex(index)}
                className={rowClass(index)}
              >
                {parsed.type && <TypeBadge type={parsed.type} className={isCompleted ? 'opacity-40' : ''} />}
                {result.entity_type === 'work_item' && result.project_key && result.item_number != null && (
                  <span className={`text-xs shrink-0 ${
                    isCompleted ? 'text-[var(--foreground-muted)]' : 'text-[var(--foreground-secondary)]'
                  }`}>
                    {result.project_key}-{result.item_number}
                  </span>
                )}
                {(result.entity_type === 'comment' || result.entity_type === 'attachment') && (
                  <ItemIcon className="h-3.5 w-3.5 text-[var(--foreground-muted)] shrink-0" />
                )}
                <span className={`truncate flex-1 ${
                  isCompleted
                    ? 'line-through text-[var(--foreground-muted)]'
                    : 'text-[var(--foreground)]'
                }`}>
                  {parsed.text}
                </span>
                {showScores && result.score > 0 && (
                  <span className="shrink-0 flex items-center gap-1.5">
                    <div className="w-12 h-1.5 bg-[var(--surface-tertiary)] rounded-full overflow-hidden">
                      <div
                        className="h-full bg-[var(--primary-muted)] rounded-full"
                        style={{ width: `${Math.round(result.score * 100)}%` }}
                      />
                    </div>
                    <span className="text-xs text-[var(--foreground-muted)] w-8 text-right">
                      {Math.round(result.score * 100)}%
                    </span>
                  </span>
                )}
              </button>
            )
          })}
        </div>
      )
    })

  return (
    <Modal open={open} onClose={onClose} position="top" className="!max-w-2xl">
      <div className="flex items-center gap-2 mb-3">
        <Search className="h-5 w-5 text-[var(--foreground-muted)] shrink-0" />
        <input
          ref={inputRef}
          type="text"
          placeholder={t('palette.placeholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          className="block w-full text-sm bg-transparent border-none outline-none text-[var(--foreground)] placeholder-gray-400 placeholder-[var(--foreground-muted)]"
        />
        {isLoading && (
          <Loader2 className="h-4 w-4 text-[var(--foreground-muted)] animate-spin shrink-0" />
        )}
      </div>

      <div className="border-t border-[var(--border)]" />

      <div ref={listRef} className="max-h-[60vh] overflow-y-auto py-2">
        {/* Navigation — always first, matched live from the first character. */}
        {navSections.map((section) => (
          <div key={section.group} className="mb-2 last:mb-0">
            <div className="flex items-center gap-2 px-3 py-1.5">
              {sectionHeading(navGroupLabel(section.group))}
            </div>
            {section.entries.map((entry) => {
              const id = navRowId(entry)
              const index = rowIndex(rows, id)
              const Icon = entry.icon
              return (
                <button
                  key={id}
                  data-palette-item
                  data-nav-item
                  onClick={() => openNavigationEntry(entry)}
                  onMouseEnter={() => setSelectedIndex(index)}
                  className={rowClass(index)}
                >
                  {entry.namespace ? (
                    <NamespaceIcon
                      icon={entry.namespace.icon}
                      color={entry.namespace.color}
                      className="h-4 w-4 shrink-0"
                    />
                  ) : (
                    <Icon className="h-4 w-4 text-[var(--foreground-muted)] shrink-0" />
                  )}
                  <span className="truncate flex-1 text-[var(--foreground)]">
                    {entry.label}
                  </span>
                </button>
              )
            })}
          </div>
        ))}

        {/* Entity results — exact matches, then semantic. */}
        {hasFtsResults && (
          <div className="mt-1">
            {showSectionHeaders && (
              <div className="px-3 py-1.5 mb-1">
                {sectionHeading(t('search.perfectMatches'))}
              </div>
            )}
            {renderEntitySections(ftsSections, false)}
          </div>
        )}

        {semanticAvailable && (
          <>
            {showSectionHeaders && hasSemanticResults && (
              <div className="flex items-center gap-2 px-3 py-1.5 mt-3 mb-1 border-t border-[var(--border)]/50 pt-3">
                {sectionHeading(t('search.relatedResults'))}
                <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium bg-[var(--surface-tertiary)] text-[var(--foreground-secondary)] bg-[var(--surface-secondary)]/50 text-[var(--foreground-muted)]">
                  <FlaskConical className="h-3 w-3" />
                  {t('common.experimental')}
                </span>
              </div>
            )}
            {hasSemanticResults && renderEntitySections(semanticSections, true)}

            {semanticStatus === 'pending' && (
              <div className="flex items-center gap-2 px-3 py-3 mt-2 border-t border-[var(--border)]/50">
                <Loader2 className="h-3.5 w-3.5 text-[var(--foreground-muted)] animate-spin" />
                <span className="text-xs text-[var(--foreground-muted)]">
                  {t('search.semanticLoading')}
                </span>
              </div>
            )}

            {semanticError && (
              <div className="flex items-center gap-2 px-3 py-3 mt-2 border-t border-[var(--border)]/50">
                <AlertCircle className="h-3.5 w-3.5 text-[var(--foreground-muted)] shrink-0" />
                <span className="text-xs text-[var(--foreground-muted)]">
                  {t('search.semanticError')}
                </span>
              </div>
            )}
          </>
        )}

        {/* Nothing at all: either still fetching, or genuinely no match. */}
        {rows.length === 0 && (
          showInitialLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 text-[var(--foreground-muted)] animate-spin" />
            </div>
          ) : (
            <div className="flex flex-col items-center py-8 gap-2">
              <SearchX className="h-8 w-8 text-[var(--foreground-secondary)]" />
              <p className="text-sm text-[var(--foreground-secondary)]">
                {t('search.noResults')}
              </p>
            </div>
          )
        )}

        {/* One character in: navigation already answers, entity search doesn't. */}
        {belowEntityFloor && (
          <p className="px-3 py-2 text-xs text-[var(--foreground-muted)]">
            {t('search.hint')}
          </p>
        )}

        {ftsCount >= limit && (
          <button
            onClick={() => setLimit((prev) => prev + 20)}
            className="w-full text-center py-2 text-sm text-[var(--primary)] hover:bg-[var(--surface-hover)] rounded-md"
          >
            {t('search.showMore')}
          </button>
        )}
      </div>

      <div className="border-t border-[var(--border)] pt-2 mt-1 hidden [@media(hover:hover)_and_(pointer:fine)]:flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-[var(--foreground-muted)]">
          <kbd className="inline-flex items-center justify-center min-w-[1.25rem] px-1 py-0.5 font-mono text-[10px] text-[var(--foreground-secondary)] bg-[var(--surface-secondary)] border border-[var(--border)] rounded">
            ↑↓
          </kbd>
          <span>{t('search.navigate')}</span>
          <kbd className="inline-flex items-center justify-center min-w-[1.25rem] px-1 py-0.5 font-mono text-[10px] text-[var(--foreground-secondary)] bg-[var(--surface-secondary)] border border-[var(--border)] rounded">
            ↵
          </kbd>
          <span>{t('search.open')}</span>
          <kbd className="inline-flex items-center justify-center min-w-[1.25rem] px-1 py-0.5 font-mono text-[10px] text-[var(--foreground-secondary)] bg-[var(--surface-secondary)] border border-[var(--border)] rounded">
            esc
          </kbd>
          <span>{t('search.dismiss')}</span>
        </div>
      </div>
    </Modal>
  )
}

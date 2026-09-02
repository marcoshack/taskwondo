import { Outlet, useNavigate, useMatch } from 'react-router-dom'

import { useTranslation } from 'react-i18next'
import { useNamespacePath, toUrlSegment } from '@/hooks/useNamespacePath'
import { Settings, UserCog, Menu, HelpCircle, Inbox, LogOut, Search, Home, Plus } from 'lucide-react'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { useProject, useAllProjects } from '@/hooks/useProjects'
import { useLastProjectKey, clearLastProjectKey } from '@/hooks/useLastProjectKey'
import { Avatar } from '@/components/ui/Avatar'
import { Modal } from '@/components/ui/Modal'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { Spinner } from '@/components/ui/Spinner'
import { useState, useRef, useEffect, useLayoutEffect, useCallback } from 'react'
import { useKeyboardShortcutContext } from '@/contexts/KeyboardShortcutContext'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { KeyboardShortcutsModal } from '@/components/KeyboardShortcutsModal'
import { CommandPalette } from '@/components/CommandPalette'
import { WelcomeModal } from '@/components/WelcomeModal'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useBrand } from '@/contexts/BrandContext'
import { useLayout } from '@/contexts/LayoutContext'
import { useInboxCount } from '@/hooks/useInbox'
import { PoweredByFooter } from '@/components/PoweredByFooter'
import { AppSidebar } from '@/components/AppSidebar'
import { CreateNamespaceModal } from '@/components/CreateNamespaceModal'
import { projectSwitchSuffix } from '@/utils/projectSection'

export function AppShell() {
  const { t } = useTranslation()
  const { brandName } = useBrand()
  const { containerClass } = useLayout()
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { guardedNavigate } = useNavigationGuard()
  const { toggleMobileOpen } = useSidebar()
  const { namespaces, activeNamespace, setActiveNamespace, showSwitcher } = useNamespaceContext()
  const [menuOpen, setMenuOpen] = useState(false)
  const [switcherOpen, setSwitcherOpen] = useState(false)
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [welcomeOpen, setWelcomeOpen] = useState(false)
  const [nsDropdownOpen, setNsDropdownOpen] = useState(false)
  const [nsCreateOpen, setNsCreateOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const nsRef = useRef<HTMLDivElement>(null)

  const { data: welcomeDismissed, isSuccess: welcomeLoaded, isError: welcomeNotFound } = usePreference<boolean>('welcome_dismissed')
  const { mutate: savePref } = useSetPreference()

  // Show welcome modal on first load if not dismissed
  const [welcomeAutoShown, setWelcomeAutoShown] = useState(false)
  useEffect(() => {
    if (!welcomeAutoShown && (welcomeLoaded || welcomeNotFound)) {
      setWelcomeAutoShown(true)
      if (welcomeDismissed !== true) {
        setWelcomeOpen(true)
      }
    }
  }, [welcomeLoaded, welcomeNotFound, welcomeAutoShown, welcomeDismissed])

  const { data: inboxCount } = useInboxCount()

  // Hide inbox/watchlist/feed when user is a customer in ALL their projects
  const portalCount = user?.portal_projects?.length ?? 0
  const totalCount = user?.total_project_count ?? 0
  const isCustomerOnly = portalCount > 0 && portalCount === totalCount && user?.global_role !== 'admin'

  const { p } = useNamespacePath()
  const projectMatch = useMatch('/:namespace/projects/:projectKey/*')
  const adminMatch = useMatch('/admin/*')
  const preferencesMatch = useMatch('/preferences/*')
  const routeProjectKey = projectMatch?.params.projectKey
  // Section to reopen in the target project when switching projects (TF-412).
  const projectSection = projectSwitchSuffix(projectMatch?.params['*'])
  const lastProjectKey = useLastProjectKey() ?? undefined
  const activeProjectKey = routeProjectKey ?? lastProjectKey

  // Customer projects are blocked by the ExcludeCustomer middleware on /api/v1/projects/:key,
  // so resolving them via useProject would return 403 and cause the clear-on-error effect
  // below to wipe the remembered project. Short-circuit those via portal_projects instead.
  const customerActiveProject = activeProjectKey
    ? (user?.portal_projects ?? []).find((pp) => pp.project_key === activeProjectKey)
    : undefined
  const { data: fetchedActiveProject, error: activeProjectError } = useProject(
    customerActiveProject ? '' : (activeProjectKey ?? '')
  )
  const activeProject = customerActiveProject
    ? { key: customerActiveProject.project_key, name: customerActiveProject.project_name }
    : fetchedActiveProject

  // If the remembered project no longer exists (e.g. deleted or DB reset), clear
  // the stored key so the sidebar and top bar don't show a stale reference.
  // Skip this for customer projects — we didn't fetch them, so there's no error to act on.
  useEffect(() => {
    if (routeProjectKey || !lastProjectKey || customerActiveProject || !activeProjectError) return
    const status = (activeProjectError as { response?: { status?: number } })?.response?.status
    if (status === 404 || status === 403) {
      clearLastProjectKey()
    }
  }, [routeProjectKey, lastProjectKey, customerActiveProject, activeProjectError])

  useEffect(() => {
    if (!menuOpen && !nsDropdownOpen) return
    const handler = (e: MouseEvent) => {
      if (menuOpen && menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
      if (nsDropdownOpen && nsRef.current && !nsRef.current.contains(e.target as Node)) {
        setNsDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen, nsDropdownOpen])

  // Sequential combos: g-i (inbox), g-o (project items)
  // useLayoutEffect ensures combos are registered before paint, so keyboard
  // shortcuts are available as soon as the UI is visible (prevents flaky E2E tests).
  // The project switcher no longer has a combo — it opens from the nav badge or
  // from the command palette.
  const { registerSequentialCombo } = useKeyboardShortcutContext()
  useLayoutEffect(() => {
    return registerSequentialCombo({
      id: 'go-to-inbox',
      keys: ['g', 'i'],
      callback: () => guardedNavigate('/user/inbox'),
    })
  }, [registerSequentialCombo])
  useLayoutEffect(() => {
    if (!activeProjectKey) return
    return registerSequentialCombo({
      id: 'go-to-items',
      keys: ['g', 'o'],
      callback: () => guardedNavigate(p(`/projects/${activeProjectKey}/items`)),
    })
    // `p` must be in deps: it changes only when the user switches namespaces
    // (see useNamespacePath), and without it the callback closes over the
    // previous namespace's segment and routes to the wrong URL (TF-345).
  }, [activeProjectKey, p, guardedNavigate, registerSequentialCombo])

  // Cmd+K on macOS, Ctrl+K elsewhere — `ctrlKey` in useKeyboardShortcut matches
  // either. Closing on a second press is handled inside the palette's input,
  // because the global layer is muted while a modal is open.
  useKeyboardShortcut({ key: 'k', ctrlKey: true }, () => setSearchOpen(true))

  useKeyboardShortcut({ key: '?' }, () => setShortcutsOpen(true))
  useKeyboardShortcut({ key: ',', ctrlKey: true }, () => guardedNavigate('/preferences'))

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen flex flex-col bg-[var(--background-secondary)]">
      <nav className="bg-[var(--surface)] border-b border-[var(--border)]">
        <div className={containerClass(true)}>
          <div className="flex justify-between h-14 relative">
            <div className="flex items-center gap-6 min-w-0">
              {/* Desktop: always show brand name */}
              <button onClick={() => guardedNavigate(p('/projects'))} className="hidden sm:block text-lg font-bold text-[var(--primary)] shrink-0">
                {brandName}
              </button>
              {/* Mobile: home icon + project key when any project active, brand when none */}
              {activeProject ? (
                <div className="flex sm:hidden items-center gap-2 min-w-0">
                  <button
                    onClick={() => guardedNavigate(p('/projects'))}
                    className="p-1.5 rounded-[var(--radius)] text-[var(--primary)] hover:bg-[var(--hover)] shrink-0 transition-colors"
                    aria-label={t('nav.home')}
                  >
                    <Home className="h-5 w-5" />
                  </button>
                  <button
                    onClick={() => setSwitcherOpen(true)}
                    className="hover:opacity-80 transition-opacity shrink-0"
                  >
                    <ProjectKeyBadge size="nav-mobile">{activeProject.key}</ProjectKeyBadge>
                  </button>
                </div>
              ) : (
                <button onClick={() => guardedNavigate(p('/projects'))} className="sm:hidden text-lg font-bold text-[var(--primary)] shrink-0">
                  {brandName}
                </button>
              )}
              {adminMatch ? (
                <div className="hidden sm:flex items-center gap-2.5 min-w-0">
                  <Settings className="h-5 w-5 text-[var(--foreground-secondary)] shrink-0" />
                  <span className="text-base font-semibold text-[var(--foreground)]">{t('admin.title')}</span>
                </div>
              ) : preferencesMatch ? (
                <div className="hidden sm:flex items-center gap-2.5 min-w-0">
                  <UserCog className="h-5 w-5 text-[var(--foreground-secondary)] shrink-0" />
                  <span className="text-base font-semibold text-[var(--foreground)] truncate">{t('preferences.navTitle')}</span>
                </div>
              ) : activeProject ? (
                <button
                  onClick={() => setSwitcherOpen(true)}
                  className="hidden sm:flex items-center gap-2.5 hover:opacity-80 transition-opacity min-w-0"
                  data-testid="project-switcher-badge"
                >
                  <ProjectKeyBadge size="nav">{activeProject.key}</ProjectKeyBadge>
                  <span className="text-base font-semibold text-[var(--foreground)] truncate">
                    {activeProject.name}
                  </span>
                </button>
              ) : null}
            </div>
            <div className="relative flex items-center gap-1.5 sm:gap-2 shrink-0" ref={menuRef}>
              <button
                onClick={() => setSearchOpen(true)}
                className="p-2 rounded-[var(--radius)] text-[var(--foreground-secondary)] hover:bg-[var(--hover)] transition-colors"
                aria-label={t('nav.search')}
              >
                <Search className="h-5 w-5" />
              </button>
              {!isCustomerOnly && (
                <button
                  onClick={() => guardedNavigate('/user/inbox')}
                  className="relative p-2 rounded-[var(--radius)] text-[var(--foreground-secondary)] hover:bg-[var(--hover)] transition-colors"
                  aria-label={t('inbox.title')}
                >
                  <Inbox className="h-5 w-5" />
                  {inboxCount != null && inboxCount > 0 && (
                    <span className="absolute -top-0.5 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-[var(--primary)] px-1 text-[10px] font-bold text-[var(--primary-foreground)]">
                      {inboxCount > 99 ? '99+' : inboxCount}
                    </span>
                  )}
                </button>
              )}
              {/* Namespace switcher — icon-only dropdown */}
              {showSwitcher && (
                <div className="relative" ref={nsRef}>
                  <button
                    onClick={() => setNsDropdownOpen(!nsDropdownOpen)}
                    className="p-2 rounded-[var(--radius)] hover:bg-[var(--hover)] transition-colors"
                    aria-label={t('namespaces.switchNamespace')}
                    data-testid="namespace-switcher"
                  >
                    <NamespaceIcon
                      icon={activeNamespace?.icon ?? 'globe'}
                      color={activeNamespace?.color ?? 'slate'}
                      className="h-5 w-5"
                    />
                  </button>
                  {nsDropdownOpen && (
                    <div className="absolute right-0 top-full mt-1 w-64 bg-[var(--surface)] rounded-[var(--radius-md)] shadow-[var(--shadow-lg)] border border-[var(--border)] py-1 z-50">
                      <div className="px-3 py-1.5 text-xs font-medium text-[var(--foreground-muted)] uppercase tracking-wider">
                        {t('namespaces.title')}
                      </div>
                      {namespaces.map((ns) => (
                        <button
                          key={ns.slug}
                          onClick={() => {
                            setNsDropdownOpen(false)
                            if (ns.slug !== activeNamespace?.slug) {
                              setActiveNamespace(ns.slug)
                            }
                          }}
                          className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2.5 transition-colors ${
                            ns.slug === activeNamespace?.slug
                              ? 'bg-[var(--primary-muted)] text-[var(--primary)]'
                              : 'text-[var(--foreground)] hover:bg-[var(--hover)]'
                          }`}
                        >
                          <NamespaceIcon icon={ns.icon} color={ns.color} className="h-4 w-4 shrink-0" />
                          <div className="min-w-0 flex-1">
                            <div className="font-medium truncate">{ns.display_name}</div>
                            {!ns.is_default && <div className="text-xs text-[var(--foreground-muted)]">{ns.slug}</div>}
                          </div>
                          {ns.slug === activeNamespace?.slug && (
                            <span className="text-xs text-[var(--primary)] shrink-0">{t('common.current')}</span>
                          )}
                          {!ns.is_default && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                setNsDropdownOpen(false)
                                guardedNavigate(`/${toUrlSegment(ns.slug)}/settings`)
                              }}
                              className="p-1 rounded text-[var(--foreground-muted)] hover:text-[var(--foreground)] shrink-0 transition-colors"
                              aria-label={t('namespaces.settings')}
                            >
                              <Settings className="h-3.5 w-3.5" />
                            </button>
                          )}
                        </button>
                      ))}
                      <div className="border-t border-[var(--border)] mt-1 pt-1">
                        <button
                          onClick={() => {
                            setNsDropdownOpen(false)
                            setNsCreateOpen(true)
                          }}
                          className="w-full text-left px-3 py-2 text-sm text-[var(--foreground-secondary)] hover:bg-[var(--hover)] flex items-center gap-2.5 transition-colors"
                        >
                          <Plus className="h-4 w-4" />
                          {t('namespaces.createNew')}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
              <button
                onClick={toggleMobileOpen}
                className="sm:hidden p-2 rounded-[var(--radius)] text-[var(--foreground-secondary)] hover:bg-[var(--hover)] transition-colors"
                aria-label={t('sidebar.menu')}
              >
                <Menu className="h-5 w-5" />
              </button>
              <div className="hidden sm:block w-px h-5 bg-[var(--border)] mx-1" />
              <button
                onClick={() => setMenuOpen(!menuOpen)}
                className="flex items-center gap-2 text-sm text-[var(--foreground)] hover:text-[var(--foreground)] transition-colors"
              >
                <Avatar name={user?.display_name ?? ''} avatarUrl={user?.avatar_url} size="sm" />
                <span className="hidden sm:block">{user?.display_name}</span>
              </button>
              {menuOpen && (
                <div className="absolute right-0 top-full mt-1 w-48 bg-[var(--surface)] rounded-[var(--radius-md)] shadow-[var(--shadow-lg)] border border-[var(--border)] py-1 z-50">
                  <div className="px-4 py-2 text-xs text-[var(--foreground-secondary)] border-b border-[var(--border)]">
                    {user?.email}
                  </div>
                  <button
                    onClick={() => { setMenuOpen(false); guardedNavigate('/preferences') }}
                    className="w-full text-left px-4 py-2 text-sm text-[var(--foreground)] hover:bg-[var(--hover)] flex items-center gap-2 transition-colors"
                  >
                    <UserCog className="h-4 w-4 text-[var(--foreground-muted)]" />
                    {t('nav.preferences')}
                  </button>
                  {user?.global_role === 'admin' && (
                    <button
                      onClick={() => { setMenuOpen(false); guardedNavigate('/admin') }}
                      className="w-full text-left px-4 py-2 text-sm text-[var(--foreground)] hover:bg-[var(--hover)] flex items-center gap-2 transition-colors"
                    >
                      <Settings className="h-4 w-4 text-[var(--foreground-muted)]" />
                      {t('nav.systemSettings')}
                    </button>
                  )}
                  <button
                    onClick={() => { setMenuOpen(false); setWelcomeOpen(true) }}
                    className="w-full text-left px-4 py-2 text-sm text-[var(--foreground)] hover:bg-[var(--hover)] flex items-center gap-2 transition-colors"
                  >
                    <HelpCircle className="h-4 w-4 text-[var(--foreground-muted)]" />
                    {t('nav.help')}
                  </button>
                  <div className="border-t border-[var(--border)]" />
                  <button
                    onClick={handleLogout}
                    className="w-full text-left px-4 py-2 text-sm text-[var(--foreground)] hover:bg-[var(--hover)] flex items-center gap-2 transition-colors"
                  >
                    <LogOut className="h-4 w-4 text-[var(--foreground-muted)]" />
                    {t('nav.signOut')}
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </nav>
      <AppSidebar mobileOnly projectKey={activeProjectKey} />
      <main className="flex-1">
        <Outlet />
      </main>
      <PoweredByFooter />
      <ProjectSwitcherModal
        open={switcherOpen}
        onClose={() => setSwitcherOpen(false)}
        activeProjectKey={activeProjectKey}
        activeNamespaceSlug={activeNamespace?.slug}
        onSelect={(key, nsSlug) => {
          setSwitcherOpen(false)
          const segment = toUrlSegment(nsSlug || activeNamespace?.slug || 'default')
          guardedNavigate(`/${segment}/projects/${key}${projectSection}`)
        }}
      />
      <CommandPalette open={searchOpen} onClose={() => setSearchOpen(false)} projectKey={activeProjectKey} />
      <KeyboardShortcutsModal open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} />
      <WelcomeModal
        open={welcomeOpen}
        onClose={() => setWelcomeOpen(false)}
        onDismiss={() => savePref({ key: 'welcome_dismissed', value: true })}
        alreadyDismissed={welcomeDismissed === true}
      />
      <CreateNamespaceModal
        open={nsCreateOpen}
        onClose={() => setNsCreateOpen(false)}
        onCreated={(ns) => {
          setNsCreateOpen(false)
          setActiveNamespace(ns.slug)
          navigate(`/${toUrlSegment(ns.slug)}/settings`)
        }}
      />
    </div>
  )
}

function ProjectSwitcherModal({
  open,
  onClose,
  activeProjectKey,
  activeNamespaceSlug,
  onSelect,
}: {
  open: boolean
  onClose: () => void
  activeProjectKey?: string
  activeNamespaceSlug?: string
  onSelect: (key: string, nsSlug?: string) => void
}) {
  const { t } = useTranslation()
  const { showSwitcher: showNamespaces } = useNamespaceContext()
  const { data: projects, isLoading } = useAllProjects()
  const { data: showAllNsPref, isSuccess: prefLoaded } = usePreference<boolean>('project_switcher_all_namespaces')
  const { mutate: savePref } = useSetPreference()
  const [showAllLocal, setShowAllLocal] = useState<boolean | null>(null)
  const showAllNamespaces = showAllLocal ?? (!prefLoaded || showAllNsPref !== false)
  const [search, setSearch] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  useEffect(() => {
    if (open) {
      setSearch('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  const filtered = (projects ?? []).filter((p) => {
    // Filter by namespace when toggle is off
    if (showNamespaces && !showAllNamespaces && p.namespace_slug !== activeNamespaceSlug) return false
    if (!search) return true
    const q = search.toLowerCase()
    return p.key.toLowerCase().includes(q) || p.name.toLowerCase().includes(q) || (p.namespace_slug ?? '').toLowerCase().includes(q)
  })

  // Reset selection when search or namespace filter changes
  useEffect(() => {
    setSelectedIndex(0)
  }, [search, showAllNamespaces])

  // Scroll selected item into view
  const scrollSelectedIntoView = useCallback((index: number) => {
    const list = listRef.current
    if (!list) return
    const item = list.children[index] as HTMLElement | undefined
    item?.scrollIntoView({ block: 'nearest' })
  }, [])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      const next = Math.min(selectedIndex + 1, filtered.length - 1)
      setSelectedIndex(next)
      scrollSelectedIntoView(next)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      const prev = Math.max(selectedIndex - 1, 0)
      setSelectedIndex(prev)
      scrollSelectedIntoView(prev)
    } else if (e.key === 'Enter' && filtered.length > 0) {
      e.preventDefault()
      const item = filtered[selectedIndex]
      onSelect(item.key, item.namespace_slug)
    }
  }, [selectedIndex, filtered, onSelect, scrollSelectedIntoView])

  const isCurrent = (p: { key: string; namespace_slug?: string }) =>
    p.key === activeProjectKey && (!showNamespaces || p.namespace_slug === activeNamespaceSlug)

  const handleToggleAllNamespaces = () => {
    const newValue = !showAllNamespaces
    setShowAllLocal(newValue)
    savePref({ key: 'project_switcher_all_namespaces', value: newValue })
  }

  return (
    <Modal open={open} onClose={onClose} title={t('projects.switcher.title')} position="top">
      <input
        ref={inputRef}
        type="text"
        placeholder={t('projects.switcher.search')}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        onKeyDown={handleKeyDown}
        className="block w-full rounded-md border border-[var(--border)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)] bg-[var(--surface)] text-[var(--foreground)] placeholder-[var(--foreground-muted)] mb-3"
      />
      {isLoading ? (
        <div className="flex justify-center py-6"><Spinner /></div>
      ) : filtered.length === 0 ? (
        <p className="text-sm text-[var(--foreground-secondary)] py-4 text-center">{t('projects.noProjectsFound')}</p>
      ) : (
        <ul ref={listRef} className="max-h-64 overflow-y-auto -mx-2">
          {filtered.map((p, i) => (
            <li key={p.id}>
              <button
                onClick={() => onSelect(p.key, p.namespace_slug)}
                onMouseEnter={() => setSelectedIndex(i)}
                className={`w-full text-left flex items-center gap-3 px-3 py-2.5 rounded-md text-sm ${
                  i === selectedIndex
                    ? 'bg-[var(--primary-muted)]'
                    : 'hover:bg-[var(--surface-hover)]'
                }`}
              >
                <ProjectKeyBadge>{p.key}</ProjectKeyBadge>
                <span className="text-[var(--foreground)] font-medium truncate">{p.name}</span>
                <span className="ml-auto flex items-center gap-2 shrink-0">
                  {isCurrent(p) && (
                    <span className="text-xs text-[var(--primary)]">{t('common.current')}</span>
                  )}
                  {showNamespaces && p.namespace_slug && (
                    <span className="flex items-center gap-1 text-[0.7rem] text-[var(--foreground-muted)]">
                      <span>{p.namespace_slug}</span>
                      <NamespaceIcon icon={p.namespace_icon ?? 'building2'} color={p.namespace_color ?? 'slate'} className="h-3 w-3" />
                    </span>
                  )}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {showNamespaces && (
        <div className="flex justify-end mt-2 pt-2 border-t border-[var(--border)]">
          <label className="flex items-center gap-2 text-xs text-[var(--foreground-secondary)] cursor-pointer select-none" data-testid="all-namespaces-toggle">
            <input
              type="checkbox"
              checked={showAllNamespaces}
              onChange={handleToggleAllNamespaces}
              className="rounded border-[var(--border)] text-[var(--primary)] focus:ring-[var(--focus-ring)] h-3.5 w-3.5"
            />
            {t('projects.switcher.showAllNamespaces')}
          </label>
        </div>
      )}
    </Modal>
  )
}

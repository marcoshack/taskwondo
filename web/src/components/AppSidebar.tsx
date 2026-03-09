import { NavLink, useLocation } from 'react-router-dom'
import { useEffect, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { useNamespacePath, toUrlSegment } from '@/hooks/useNamespacePath'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { CreateNamespaceModal } from '@/components/CreateNamespaceModal'
import { useInboxCount } from '@/hooks/useInbox'
import {
  Inbox,
  Rss,
  Bookmark,
  FolderKanban,
  LayoutDashboard,
  ClipboardList,
  SquareStack,
  Target,
  Route,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  Plus,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

interface AppSidebarProps {
  projectKey?: string
  /** Render only the mobile overlay (used in AppShell for global availability) */
  mobileOnly?: boolean
}

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  end: boolean
  badge?: number
}

const LAST_PROJECT_KEY = 'taskwondo_last_project_key'

export function AppSidebar({ projectKey, mobileOnly }: AppSidebarProps) {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const { collapsed, toggleCollapsed, mobileOpen, closeMobile } = useSidebar('app')
  const { guardRef, guardedNavigate } = useNavigationGuard()
  const location = useLocation()
  const { namespaces, activeNamespace, setActiveNamespace, showSwitcher } = useNamespaceContext()
  const { data: inboxCount } = useInboxCount()
  const [nsDropdownOpen, setNsDropdownOpen] = useState(false)
  const [nsCreateOpen, setNsCreateOpen] = useState(false)
  const nsRef = useRef<HTMLDivElement>(null)

  // Close namespace dropdown on click outside
  useEffect(() => {
    if (!nsDropdownOpen) return
    const handler = (e: MouseEvent) => {
      if (nsRef.current && !nsRef.current.contains(e.target as Node)) {
        setNsDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [nsDropdownOpen])

  // Remember the last active project so sidebar persists on /projects and /user pages
  const [lastProjectKey, setLastProjectKey] = useState<string | undefined>(
    () => projectKey ?? localStorage.getItem(LAST_PROJECT_KEY) ?? undefined,
  )

  useEffect(() => {
    if (projectKey) {
      setLastProjectKey(projectKey)
      localStorage.setItem(LAST_PROJECT_KEY, projectKey)
    }
  }, [projectKey])

  const activeProjectKey = projectKey ?? lastProjectKey

  // Close mobile sidebar and namespace dropdown on route change
  useEffect(() => {
    closeMobile()
    setNsDropdownOpen(false)
  }, [location.pathname, closeMobile])

  const userNavItems: NavItem[] = [
    { to: '/user/inbox', label: t('user.sidebar.inbox'), icon: Inbox, end: true, badge: inboxCount && inboxCount > 0 ? inboxCount : undefined },
    { to: '/user/feed', label: t('user.sidebar.feed'), icon: Rss, end: false },
    { to: '/user/watchlist', label: t('user.sidebar.watchlist'), icon: Bookmark, end: false },
  ]

  const projectBase = activeProjectKey ? p(`/projects/${activeProjectKey}`) : ''

  const projectNavItems: NavItem[] = activeProjectKey ? [
    { to: `${projectBase}/`, label: t('sidebar.overview'), icon: LayoutDashboard, end: true },
    { to: `${projectBase}/items`, label: t('sidebar.items'), icon: ClipboardList, end: false },
    { to: `${projectBase}/queues`, label: t('sidebar.queues'), icon: SquareStack, end: false },
    { to: `${projectBase}/milestones`, label: t('sidebar.milestones'), icon: Target, end: false },
    { to: `${projectBase}/workflows`, label: t('sidebar.workflows'), icon: Route, end: false },
    { to: `${projectBase}/settings`, label: t('sidebar.settings'), icon: Settings, end: false },
  ] : []

  function renderNavItem(item: NavItem, showLabels: boolean) {
    return (
      <li key={item.to}>
        <NavLink
          to={item.to}
          end={item.end}
          onClick={(e) => {
            if (guardRef.current?.()) {
              e.preventDefault()
              guardedNavigate(item.to)
            }
          }}
          className={({ isActive }) =>
            `group/nav relative flex items-center gap-3 rounded-md text-sm font-medium transition-colors min-w-0 ${
              !showLabels ? 'justify-center px-0 py-2' : 'px-3 py-2'
            } ${
              isActive
                ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
            }`
          }
        >
          <span className="relative shrink-0">
            <item.icon className="h-5 w-5" />
            {!showLabels && item.badge != null && (
              <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-indigo-600 px-1 text-[10px] font-bold text-white">
                {item.badge > 99 ? '99+' : item.badge}
              </span>
            )}
          </span>
          {showLabels && (
            <>
              <span className="flex-1 truncate">{item.label}</span>
              {item.badge != null && (
                <span className="ml-auto flex h-5 min-w-5 items-center justify-center rounded-full bg-indigo-100 dark:bg-indigo-900/40 px-1.5 text-xs font-medium text-indigo-700 dark:text-indigo-300">
                  {item.badge > 99 ? '99+' : item.badge}
                </span>
              )}
            </>
          )}
          {!showLabels && (
            <span className="pointer-events-none absolute left-full ml-2 rounded bg-gray-900 px-2 py-1 text-xs whitespace-nowrap text-white opacity-0 transition-opacity group-hover/nav:opacity-100 dark:bg-gray-700 z-50">
              {item.label}
            </span>
          )}
        </NavLink>
      </li>
    )
  }

  function renderProjectsLink(showLabels: boolean) {
    // When a project is active: expanded shows icon + Projects + badge right-aligned; collapsed shows full-width badge
    if (activeProjectKey) {
      if (!showLabels) {
        // Collapsed: full-width project key badge
        return (
          <li>
            <NavLink
              to={p('/projects')}
              end
              onClick={(e) => {
                if (guardRef.current?.()) {
                  e.preventDefault()
                  guardedNavigate(p('/projects'))
                }
              }}
              className={({ isActive }) =>
                `group/nav relative flex items-center justify-center rounded-md text-sm font-bold transition-colors min-w-0 px-0 py-2 ${
                  isActive
                    ? 'bg-indigo-200 text-indigo-800 dark:bg-indigo-800/50 dark:text-indigo-200'
                    : 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/40 dark:text-indigo-300 dark:hover:bg-indigo-800/50'
                }`
              }
            >
              {activeProjectKey}
              <span className="pointer-events-none absolute left-full ml-2 rounded bg-gray-900 px-2 py-1 text-xs whitespace-nowrap text-white opacity-0 transition-opacity group-hover/nav:opacity-100 dark:bg-gray-700 z-50">
                {t('sidebar.projects')}
              </span>
            </NavLink>
          </li>
        )
      }

      // Expanded: icon + Projects text + badge on the right
      return (
        <li>
          <NavLink
            to={p('/projects')}
            end
            onClick={(e) => {
              if (guardRef.current?.()) {
                e.preventDefault()
                guardedNavigate(p('/projects'))
              }
            }}
            className={({ isActive }) =>
              `group/nav relative flex items-center gap-3 rounded-md text-sm font-medium transition-colors min-w-0 px-3 py-2 ${
                isActive
                  ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                  : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
              }`
            }
          >
            <FolderKanban className="h-5 w-5 shrink-0" />
            <span className="flex-1 truncate">{t('sidebar.projects')}</span>
            <span className="ml-auto inline-flex items-center justify-center rounded-md bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 text-xs font-bold px-1.5 py-1 shrink-0">
              {activeProjectKey}
            </span>
          </NavLink>
        </li>
      )
    }

    return (
      <li>
        <NavLink
          to={p('/projects')}
          end
          onClick={(e) => {
            if (guardRef.current?.()) {
              e.preventDefault()
              guardedNavigate(p('/projects'))
            }
          }}
          className={({ isActive }) =>
            `group/nav relative flex items-center gap-3 rounded-md text-sm font-medium transition-colors min-w-0 ${
              !showLabels ? 'justify-center px-0 py-2' : 'px-3 py-2'
            } ${
              isActive
                ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
            }`
          }
        >
          <FolderKanban className="h-5 w-5 shrink-0" />
          {showLabels && <span className="truncate">{t('sidebar.projects')}</span>}
          {!showLabels && (
            <span className="pointer-events-none absolute left-full ml-2 rounded bg-gray-900 px-2 py-1 text-xs whitespace-nowrap text-white opacity-0 transition-opacity group-hover/nav:opacity-100 dark:bg-gray-700 z-50">
              {t('sidebar.projects')}
            </span>
          )}
        </NavLink>
      </li>
    )
  }

  function renderNamespaceBanner(showLabels: boolean) {
    if (!showSwitcher || !activeNamespace) return null

    return (
      <div className="relative mb-1" ref={nsRef}>
        <button
          onClick={() => setNsDropdownOpen(!nsDropdownOpen)}
          className={`group/ns w-full flex items-center gap-2.5 rounded-md text-sm font-semibold transition-colors ${
            showLabels ? 'px-3 py-2' : 'justify-center py-2'
          } text-gray-900 dark:text-gray-100 hover:bg-gray-100 dark:hover:bg-gray-800`}
          aria-label={t('namespaces.switchNamespace')}
        >
          <NamespaceIcon icon={activeNamespace.icon} color={activeNamespace.color} className="h-5 w-5 shrink-0" />
          {showLabels && (
            <span className="flex-1 truncate text-left">{activeNamespace.display_name}</span>
          )}
          {!showLabels && (
            <span className="pointer-events-none absolute left-full ml-2 rounded bg-gray-900 px-2 py-1 text-xs whitespace-nowrap text-white opacity-0 transition-opacity group-hover/ns:opacity-100 dark:bg-gray-700 z-50">
              {activeNamespace.display_name}
            </span>
          )}
        </button>
        {nsDropdownOpen && (
          <div className={`absolute z-50 w-64 bg-white/40 dark:bg-gray-800/40 backdrop-blur-sm rounded-md shadow-lg border border-gray-200 dark:border-gray-600 py-1 ${
            showLabels ? 'left-0 top-full mt-1' : 'left-full top-0 ml-2'
          }`}>
            <div className="px-3 py-1.5 text-xs font-medium text-gray-400 dark:text-gray-500 uppercase tracking-wider">
              {t('namespaces.title')}
            </div>
            {namespaces.map((ns) => (
              <button
                key={ns.slug}
                onClick={() => {
                  setNsDropdownOpen(false)
                  if (ns.slug !== activeNamespace.slug) {
                    setActiveNamespace(ns.slug)
                  }
                }}
                className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2.5 ${
                  ns.slug === activeNamespace.slug
                    ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                    : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
                }`}
              >
                <NamespaceIcon icon={ns.icon} color={ns.color} className="h-4 w-4 shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="font-medium truncate">{ns.display_name}</div>
                  {!ns.is_default && <div className="text-xs text-gray-400 dark:text-gray-500">{ns.slug}</div>}
                </div>
                {ns.slug === activeNamespace.slug && (
                  <span className="text-xs text-indigo-600 dark:text-indigo-400 shrink-0">{t('common.current')}</span>
                )}
                {!ns.is_default && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      setNsDropdownOpen(false)
                      guardedNavigate(`/${toUrlSegment(ns.slug)}/settings`)
                    }}
                    className="p-1 rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 shrink-0"
                    aria-label={t('namespaces.settings')}
                  >
                    <Settings className="h-3.5 w-3.5" />
                  </button>
                )}
              </button>
            ))}
            <div className="border-t border-gray-100 dark:border-gray-700 mt-1 pt-1">
              <button
                onClick={() => {
                  setNsDropdownOpen(false)
                  setNsCreateOpen(true)
                }}
                className="w-full text-left px-3 py-2 text-sm text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2.5"
              >
                <Plus className="h-4 w-4" />
                {t('namespaces.createNew')}
              </button>
            </div>
          </div>
        )}
      </div>
    )
  }

  function renderContent(showLabels: boolean) {
    return (
      <>
        {/* User section */}
        <ul className="space-y-1">
          {userNavItems.map((item) => renderNavItem(item, showLabels))}
        </ul>

        {/* Separator */}
        <div className="border-t border-gray-200 dark:border-gray-700 my-2" />

        {/* Namespace banner */}
        {renderNamespaceBanner(showLabels)}

        {/* Projects section */}
        <ul className="space-y-1">
          {renderProjectsLink(showLabels)}
        </ul>

        {/* Active project context */}
        {activeProjectKey && projectNavItems.length > 0 && (
          <ul className="space-y-1 mt-1">
            {projectNavItems.map((item) => renderNavItem(item, showLabels))}
          </ul>
        )}
      </>
    )
  }

  const createModal = (
    <CreateNamespaceModal
      open={nsCreateOpen}
      onClose={() => setNsCreateOpen(false)}
      onCreated={(ns) => {
        setNsCreateOpen(false)
        setActiveNamespace(ns.slug)
        guardedNavigate(`/${toUrlSegment(ns.slug)}/settings`)
      }}
    />
  )

  if (mobileOnly) {
    // Render only the mobile dropdown overlay (used in AppShell for global availability)
    return (
      <>
        {mobileOpen && (
          <div className="fixed inset-0 z-40 sm:hidden" onClick={closeMobile}>
            <nav
              className="absolute right-4 top-14 w-52 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 p-2"
              onClick={(e) => e.stopPropagation()}
            >
              {renderContent(true)}
            </nav>
          </div>
        )}
        {createModal}
      </>
    )
  }

  return (
    <>
      {/* Desktop sidebar */}
      <nav
        className={`hidden sm:block shrink-0 transition-all duration-200 ${
          collapsed ? 'w-14' : 'w-48'
        }`}
      >
        {renderContent(!collapsed)}

        <div
          className={`mt-4 border-t border-gray-200 pt-4 dark:border-gray-700 ${
            collapsed ? 'flex justify-center' : ''
          }`}
        >
          <button
            onClick={toggleCollapsed}
            className={`group/toggle relative flex items-center gap-3 rounded-md text-sm font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-300 min-w-0 ${
              collapsed ? 'justify-center px-0 py-2 w-full' : 'px-3 py-2 w-full'
            }`}
            aria-label={collapsed ? t('sidebar.expand') : t('sidebar.collapse')}
          >
            {collapsed ? (
              <PanelLeftOpen className="h-5 w-5 shrink-0" />
            ) : (
              <>
                <PanelLeftClose className="h-5 w-5 shrink-0" />
                <span className="truncate">{t('sidebar.collapse')}</span>
              </>
            )}
            {collapsed && (
              <span className="pointer-events-none absolute left-full ml-2 rounded bg-gray-900 px-2 py-1 text-xs whitespace-nowrap text-white opacity-0 transition-opacity group-hover/toggle:opacity-100 dark:bg-gray-700 z-50">
                {t('sidebar.expand')}
              </span>
            )}
          </button>
        </div>
      </nav>
      {createModal}
    </>
  )
}

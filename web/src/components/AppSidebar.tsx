import { NavLink, useLocation } from 'react-router-dom'
import { useEffect, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { useNamespacePath, toUrlSegment } from '@/hooks/useNamespacePath'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useLastProjectKey, setLastProjectKey } from '@/hooks/useLastProjectKey'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { CreateNamespaceModal } from '@/components/CreateNamespaceModal'
import { useInboxCount } from '@/hooks/useInbox'
import {
  isCustomerOnly as isCustomerOnlyUser,
  isCustomerProject as isCustomerInProject,
  projectNavItems,
  userNavItems,
} from '@/utils/sidebarNav'
import type { SidebarNavItem } from '@/utils/sidebarNav'
import {
  FolderKanban,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  Plus,
} from 'lucide-react'

interface AppSidebarProps {
  projectKey?: string
  /** True when the active project is a customer-role project (show only Support nav) */
  customerProject?: boolean
  /** Render only the mobile overlay (used in AppShell for global availability) */
  mobileOnly?: boolean
}

interface NavItem extends SidebarNavItem {
  badge?: number
}

export function AppSidebar({ projectKey, customerProject, mobileOnly }: AppSidebarProps) {
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

  // Remember the last active project so sidebar persists on /projects and /user pages.
  // Backed by useSyncExternalStore in useLastProjectKey so all components stay in sync
  // when the stored key is cleared (e.g. the remembered project no longer exists).
  const storedLastProjectKey = useLastProjectKey()

  useEffect(() => {
    if (projectKey && projectKey !== storedLastProjectKey) {
      setLastProjectKey(projectKey)
    }
  }, [projectKey, storedLastProjectKey])

  const activeProjectKey = projectKey ?? storedLastProjectKey ?? undefined
  const { user } = useAuth()

  // Detect customer project: explicit prop or derived from portal_projects for the remembered project
  const isCustomerProject = !!customerProject
    || (!projectKey && isCustomerInProject(user, activeProjectKey))

  // Hide inbox/watchlist/feed when user is a customer in ALL their projects
  const isCustomerOnly = isCustomerOnlyUser(user)

  // Close mobile sidebar and namespace dropdown on route change
  useEffect(() => {
    closeMobile()
    setNsDropdownOpen(false)
  }, [location.pathname, closeMobile])

  // Nav definitions are shared with the command palette's navigation catalog —
  // see @/utils/sidebarNav. Only the inbox badge is sidebar-specific.
  const userNav: NavItem[] = userNavItems(t).map((item) =>
    item.to === '/user/inbox'
      ? { ...item, badge: inboxCount && inboxCount > 0 ? inboxCount : undefined }
      : item,
  )

  const projectBase = activeProjectKey ? p(`/projects/${activeProjectKey}`) : ''

  const projectNav: NavItem[] = activeProjectKey
    ? projectNavItems(t, projectBase, isCustomerProject)
    : []

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
            `group/nav relative flex items-center gap-3 rounded-[var(--radius)] text-sm font-medium transition-colors min-w-0 ${
              !showLabels ? 'justify-center px-0 py-2' : 'px-3 py-2'
            } ${
              isActive
                ? 'bg-[var(--primary-muted)] text-[var(--primary)]'
                : 'text-[var(--foreground)] hover:bg-[var(--hover)]'
            }`
          }
        >
          <span className="relative shrink-0">
            <item.icon className="h-5 w-5" />
            {!showLabels && item.badge != null && (
              <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-[var(--primary)] px-1 text-[10px] font-bold text-[var(--primary-foreground)]">
                {item.badge > 99 ? '99+' : item.badge}
              </span>
            )}
          </span>
          {showLabels && (
            <>
              <span className="flex-1 truncate">{item.label}</span>
              {item.badge != null && (
                <span className="ml-auto flex h-5 min-w-5 items-center justify-center rounded-full bg-[var(--primary-muted)] px-1.5 text-xs font-medium text-[var(--primary)]">
                  {item.badge > 99 ? '99+' : item.badge}
                </span>
              )}
            </>
          )}
          {!showLabels && (
            <span className="pointer-events-none absolute left-full ml-2 rounded bg-[var(--foreground)] px-2 py-1 text-xs whitespace-nowrap text-[var(--background)] opacity-0 transition-opacity group-hover/nav:opacity-100 z-50">
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
                `group/nav relative flex items-center justify-center rounded-[var(--radius)] text-sm font-bold transition-colors min-w-0 px-0 py-2 ${
                  isActive
                    ? 'bg-[var(--primary-muted)] text-[var(--primary)]'
                    : 'bg-[var(--primary-muted)]/50 text-[var(--primary)] hover:bg-[var(--primary-muted)]'
                }`
              }
            >
              {activeProjectKey}
              <span className="pointer-events-none absolute left-full ml-2 rounded bg-[var(--foreground)] px-2 py-1 text-xs whitespace-nowrap text-[var(--background)] opacity-0 transition-opacity group-hover/nav:opacity-100 z-50">
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
              `group/nav relative flex items-center gap-3 rounded-[var(--radius)] text-sm font-medium transition-colors min-w-0 px-3 py-2 ${
                isActive
                  ? 'bg-[var(--primary-muted)] text-[var(--primary)]'
                  : 'text-[var(--foreground)] hover:bg-[var(--hover)]'
              }`
            }
          >
            <FolderKanban className="h-5 w-5 shrink-0" />
            <span className="flex-1 truncate">{t('sidebar.projects')}</span>
            <span className="ml-auto inline-flex items-center justify-center rounded-[var(--radius-sm)] bg-[var(--primary-muted)] text-[var(--primary)] text-xs font-bold px-1.5 py-1 shrink-0">
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
            `group/nav relative flex items-center gap-3 rounded-[var(--radius)] text-sm font-medium transition-colors min-w-0 ${
              !showLabels ? 'justify-center px-0 py-2' : 'px-3 py-2'
            } ${
              isActive
                ? 'bg-[var(--primary-muted)] text-[var(--primary)]'
                : 'text-[var(--foreground)] hover:bg-[var(--hover)]'
            }`
          }
        >
          <FolderKanban className="h-5 w-5 shrink-0" />
          {showLabels && <span className="truncate">{t('sidebar.projects')}</span>}
          {!showLabels && (
            <span className="pointer-events-none absolute left-full ml-2 rounded bg-[var(--foreground)] px-2 py-1 text-xs whitespace-nowrap text-[var(--background)] opacity-0 transition-opacity group-hover/nav:opacity-100 z-50">
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
          className={`group/ns w-full flex items-center gap-2.5 rounded-[var(--radius)] text-sm font-semibold transition-colors ${
            showLabels ? 'px-3 py-2' : 'justify-center py-2'
          } text-[var(--foreground)] hover:bg-[var(--hover)]`}
          aria-label={t('namespaces.switchNamespace')}
        >
          <NamespaceIcon icon={activeNamespace.icon} color={activeNamespace.color} className="h-5 w-5 shrink-0" />
          {showLabels && (
            <span className="flex-1 truncate text-left">{activeNamespace.display_name}</span>
          )}
          {!showLabels && (
            <span className="pointer-events-none absolute left-full ml-2 rounded bg-[var(--foreground)] px-2 py-1 text-xs whitespace-nowrap text-[var(--background)] opacity-0 transition-opacity group-hover/ns:opacity-100 z-50">
              {activeNamespace.display_name}
            </span>
          )}
        </button>
        {nsDropdownOpen && (
          <div className={`absolute z-50 w-64 bg-[var(--surface)] rounded-[var(--radius-md)] shadow-[var(--shadow-lg)] border border-[var(--border)] py-1 ${
            showLabels ? 'left-0 top-full mt-1' : 'left-full top-0 ml-2'
          }`}>
            <div className="px-3 py-1.5 text-xs font-medium text-[var(--foreground-muted)] uppercase tracking-wider">
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
                className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2.5 transition-colors ${
                  ns.slug === activeNamespace.slug
                    ? 'bg-[var(--primary-muted)] text-[var(--primary)]'
                    : 'text-[var(--foreground)] hover:bg-[var(--hover)]'
                }`}
              >
                <NamespaceIcon icon={ns.icon} color={ns.color} className="h-4 w-4 shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="font-medium truncate">{ns.display_name}</div>
                  {!ns.is_default && <div className="text-xs text-[var(--foreground-muted)]">{ns.slug}</div>}
                </div>
                {ns.slug === activeNamespace.slug && (
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
    )
  }

  function renderContent(showLabels: boolean) {
    return (
      <>
        {/* User section — hidden when user is customer in all projects */}
        {!isCustomerOnly && (
          <>
            <ul className="space-y-0.5">
              {userNav.map((item) => renderNavItem(item, showLabels))}
            </ul>

            {/* Separator */}
            <div className="border-t border-[var(--border)] my-2" />
          </>
        )}

        {/* Namespace banner */}
        {renderNamespaceBanner(showLabels)}

        {/* Projects section */}
        <ul className="space-y-0.5">
          {renderProjectsLink(showLabels)}
        </ul>

        {/* Active project context */}
        {activeProjectKey && projectNav.length > 0 && (
          <>
            <div className="border-t border-[var(--border)] my-2" />
            <ul className="space-y-0.5">
              {projectNav.map((item) => renderNavItem(item, showLabels))}
            </ul>
          </>
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
              className="absolute right-4 top-14 w-52 bg-[var(--surface)] rounded-[var(--radius-lg)] shadow-[var(--shadow-lg)] border border-[var(--border)] p-2"
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
          collapsed ? 'w-14' : 'w-56'
        }`}
      >
        {renderContent(!collapsed)}

        <div
          className={`mt-4 border-t border-[var(--border)] pt-4 ${
            collapsed ? 'flex justify-center' : ''
          }`}
        >
          <button
            onClick={toggleCollapsed}
            className={`group/toggle relative flex items-center gap-3 rounded-[var(--radius)] text-sm font-medium text-[var(--foreground-secondary)] transition-colors hover:bg-[var(--hover)] hover:text-[var(--foreground)] min-w-0 ${
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
              <span className="pointer-events-none absolute left-full ml-2 rounded bg-[var(--foreground)] px-2 py-1 text-xs whitespace-nowrap text-[var(--background)] opacity-0 transition-opacity group-hover/toggle:opacity-100 z-50">
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

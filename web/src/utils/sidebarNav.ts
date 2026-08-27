/**
 * Shared sidebar navigation definitions.
 *
 * `AppSidebar`, `PreferencesSidebar` and `SystemSettingsSidebar` all render
 * from these builders, and so does the command palette's navigation catalog
 * (`@/utils/navigationCatalog`). Adding a page to a sidebar is therefore a
 * single edit here — the palette picks it up for free.
 *
 * Every builder takes `t` as a parameter instead of holding translated strings
 * at module scope (see AGENTS.md: display strings must not live in module-level
 * arrays, because `t` is only available at render time).
 *
 * This module must stay free of `@/api/client` — it is imported by units that
 * run under Vitest without a DOM.
 */
import {
  Inbox,
  Rss,
  Bookmark,
  Headphones,
  LayoutDashboard,
  ClipboardList,
  SquareStack,
  Target,
  Route,
  Settings,
  User,
  Palette,
  Bell,
  Lock,
  Users,
  Key,
  Plug,
  ToggleRight,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

/**
 * Minimal shape of i18next's `t`. Keeping it structural lets the pure units be
 * tested with a plain function and no i18n bootstrap.
 */
export type Translate = (key: string, params?: Record<string, unknown>) => string

export interface SidebarNavItem {
  /** Absolute path to navigate to */
  to: string
  /** Translated label */
  label: string
  icon: LucideIcon
  /** `end` prop for NavLink active matching */
  end: boolean
}

/** The subset of the authenticated user these gates depend on. */
export interface NavUser {
  global_role?: string
  portal_projects?: { project_key: string }[]
  total_project_count?: number
}

/** True when the user is a Taskwondo system administrator. */
export function isSystemAdmin(user: NavUser | null | undefined): boolean {
  return user?.global_role === 'admin'
}

/**
 * True when the user is a customer in *every* project they can see, in which
 * case Inbox/Watchlist/Feed are hidden entirely.
 */
export function isCustomerOnly(user: NavUser | null | undefined): boolean {
  const portalCount = user?.portal_projects?.length ?? 0
  const totalCount = user?.total_project_count ?? 0
  return portalCount > 0 && portalCount === totalCount && !isSystemAdmin(user)
}

/**
 * True when the user's role in `projectKey` is customer, in which case Support
 * is the only project section they may open.
 */
export function isCustomerProject(user: NavUser | null | undefined, projectKey: string | undefined): boolean {
  if (!projectKey) return false
  return !isSystemAdmin(user) && (user?.portal_projects ?? []).some((pp) => pp.project_key === projectKey)
}

/** Personal pages: Inbox, Watchlist, Feed. Hidden when `isCustomerOnly`. */
export function userNavItems(t: Translate): SidebarNavItem[] {
  return [
    { to: '/user/inbox', label: t('user.sidebar.inbox'), icon: Inbox, end: true },
    { to: '/user/watchlist', label: t('user.sidebar.watchlist'), icon: Bookmark, end: false },
    { to: '/user/feed', label: t('user.sidebar.feed'), icon: Rss, end: false },
  ]
}

/**
 * Sections of the active project. `base` is the namespace-prefixed project root
 * (e.g. `/d/projects/TF`). Customers get Support and nothing else.
 */
export function projectNavItems(t: Translate, base: string, customerProject: boolean): SidebarNavItem[] {
  if (customerProject) {
    return [{ to: `${base}/support`, label: t('sidebar.support'), icon: Headphones, end: false }]
  }
  return [
    { to: `${base}/`, label: t('sidebar.overview'), icon: LayoutDashboard, end: true },
    { to: `${base}/items`, label: t('sidebar.items'), icon: ClipboardList, end: false },
    { to: `${base}/milestones`, label: t('sidebar.milestones'), icon: Target, end: false },
    { to: `${base}/queues`, label: t('sidebar.queues'), icon: SquareStack, end: false },
    { to: `${base}/workflows`, label: t('sidebar.workflows'), icon: Route, end: false },
    { to: `${base}/settings`, label: t('sidebar.settings'), icon: Settings, end: false },
  ]
}

/** User Preferences pages. Available to every signed-in user. */
export function preferencesNavItems(t: Translate): SidebarNavItem[] {
  const base = '/preferences'
  return [
    { to: `${base}/profile`, label: t('preferences.sidebar.profile'), icon: User, end: false },
    { to: `${base}/general`, label: t('preferences.sidebar.general'), icon: Settings, end: false },
    { to: `${base}/appearance`, label: t('preferences.sidebar.appearance'), icon: Palette, end: false },
    { to: `${base}/notifications`, label: t('preferences.sidebar.notifications'), icon: Bell, end: false },
    { to: `${base}/authentication`, label: t('preferences.sidebar.authentication'), icon: Lock, end: false },
  ]
}

/** System Settings sections. System admins only — see `isSystemAdmin`. */
export function systemSettingsNavItems(t: Translate): SidebarNavItem[] {
  const base = '/admin'
  return [
    { to: `${base}/general`, label: t('admin.sidebar.general'), icon: Settings, end: false },
    { to: `${base}/directory`, label: t('admin.sidebar.directory'), icon: Users, end: false },
    { to: `${base}/workflows`, label: t('admin.sidebar.workflows'), icon: Route, end: false },
    { to: `${base}/authentication`, label: t('admin.sidebar.authentication'), icon: Lock, end: false },
    { to: `${base}/api-keys`, label: t('admin.sidebar.apiKeys'), icon: Key, end: false },
    { to: `${base}/integrations`, label: t('admin.sidebar.integrations'), icon: Plug, end: false },
    { to: `${base}/features`, label: t('admin.sidebar.features'), icon: ToggleRight, end: false },
  ]
}

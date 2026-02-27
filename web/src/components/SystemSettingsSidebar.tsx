import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import {
  Settings,
  Users,
  Route,
  Plug,
  KeyRound,
  ToggleRight,
  PanelLeftClose,
  PanelLeftOpen,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export function SystemSettingsSidebar() {
  const { t } = useTranslation()
  const { collapsed, toggleCollapsed } = useSidebar('settings')
  const { guardRef, guardedNavigate } = useNavigationGuard()
  const base = '/admin'

  const navItems: { to: string; label: string; icon: LucideIcon; end: boolean }[] = [
    { to: 'general', label: t('admin.sidebar.general'), icon: Settings, end: false },
    { to: 'users', label: t('admin.sidebar.users'), icon: Users, end: false },
    { to: 'workflows', label: t('admin.sidebar.workflows'), icon: Route, end: false },
    { to: 'integrations', label: t('admin.sidebar.integrations'), icon: Plug, end: false },
    { to: 'authentication', label: t('admin.sidebar.authentication'), icon: KeyRound, end: false },
    { to: 'features', label: t('admin.sidebar.features'), icon: ToggleRight, end: false },
  ]

  function renderNavItems(showLabels: boolean) {
    return (
      <ul className="space-y-1">
        {navItems.map((item) => (
          <li key={item.to}>
            <NavLink
              to={`${base}/${item.to}`}
              end={item.end}
              onClick={(e) => {
                if (guardRef.current?.()) {
                  e.preventDefault()
                  guardedNavigate(`${base}/${item.to}`)
                }
              }}
              className={({ isActive }) =>
                `group/nav relative flex items-center gap-3 rounded-md text-sm font-medium transition-colors ${
                  !showLabels ? 'justify-center px-0 py-2' : 'px-3 py-2'
                } ${
                  isActive
                    ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                    : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
                }`
              }
            >
              <item.icon className="h-5 w-5 shrink-0" />
              {showLabels && <span>{item.label}</span>}
              {!showLabels && (
                <span className="pointer-events-none absolute left-full ml-2 rounded bg-gray-900 px-2 py-1 text-xs whitespace-nowrap text-white opacity-0 transition-opacity group-hover/nav:opacity-100 dark:bg-gray-700 z-50">
                  {item.label}
                </span>
              )}
            </NavLink>
          </li>
        ))}
      </ul>
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
        {renderNavItems(!collapsed)}

        <div
          className={`mt-4 border-t border-gray-200 pt-4 dark:border-gray-700 ${
            collapsed ? 'flex justify-center' : ''
          }`}
        >
          <button
            onClick={toggleCollapsed}
            className={`group/toggle relative flex items-center gap-3 rounded-md text-sm font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-300 ${
              collapsed ? 'justify-center px-0 py-2 w-full' : 'px-3 py-2 w-full'
            }`}
            aria-label={collapsed ? t('sidebar.expand') : t('sidebar.collapse')}
          >
            {collapsed ? (
              <PanelLeftOpen className="h-5 w-5 shrink-0" />
            ) : (
              <>
                <PanelLeftClose className="h-5 w-5 shrink-0" />
                <span>{t('sidebar.collapse')}</span>
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

    </>
  )
}

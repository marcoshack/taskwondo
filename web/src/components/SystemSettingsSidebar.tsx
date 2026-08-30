import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { systemSettingsNavItems } from '@/utils/sidebarNav'
import { PanelLeftClose, PanelLeftOpen } from 'lucide-react'

export function SystemSettingsSidebar() {
  const { t } = useTranslation()
  const { collapsed, toggleCollapsed } = useSidebar('settings')
  const { guardRef, guardedNavigate } = useNavigationGuard()
  // Shared with the command palette's navigation catalog — see @/utils/sidebarNav
  const navItems = systemSettingsNavItems(t)

  function renderNavItems(showLabels: boolean) {
    return (
      <ul className="space-y-1">
        {navItems.map((item) => (
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
                `group/nav relative flex items-center gap-3 rounded-md text-sm font-medium transition-colors ${
                  !showLabels ? 'justify-center px-0 py-2' : 'px-3 py-2'
                } ${
                  isActive
                    ? 'bg-[var(--primary-muted)] text-[var(--primary)]  dark:text-[var(--primary)]'
                    : 'text-[var(--foreground)] hover:bg-[var(--surface-tertiary)] text-[var(--foreground)] dark:hover:bg-[var(--surface)]'
                }`
              }
            >
              <item.icon className="h-5 w-5 shrink-0" />
              {showLabels && <span>{item.label}</span>}
              {!showLabels && (
                <span className="pointer-events-none absolute left-full ml-2 rounded bg-[var(--background)] px-2 py-1 text-xs whitespace-nowrap text-white opacity-0 transition-opacity group-hover/nav:opacity-100 bg-[var(--surface-secondary)] z-50">
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
          className={`mt-4 border-t border-[var(--border)] pt-4 dark:border-[var(--border)] ${
            collapsed ? 'flex justify-center' : ''
          }`}
        >
          <button
            onClick={toggleCollapsed}
            className={`group/toggle relative flex items-center gap-3 rounded-md text-sm font-medium text-[var(--foreground-secondary)] transition-colors hover:bg-[var(--surface-tertiary)] hover:text-[var(--foreground)] text-[var(--foreground-muted)] dark:hover:bg-[var(--surface)] dark:hover:text-[var(--foreground-muted)] ${
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
              <span className="pointer-events-none absolute left-full ml-2 rounded bg-[var(--background)] px-2 py-1 text-xs whitespace-nowrap text-white opacity-0 transition-opacity group-hover/toggle:opacity-100 bg-[var(--surface-secondary)] z-50">
                {t('sidebar.expand')}
              </span>
            )}
          </button>
        </div>
      </nav>

    </>
  )
}

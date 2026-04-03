import { NavLink, Outlet, useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { User, Palette, ArrowLeft } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export function PortalPreferencesPage() {
  const { t } = useTranslation()
  const { namespace, projectKey } = useParams<{ namespace: string; projectKey: string }>()
  const base = `/portal/${namespace}/projects/${projectKey}/preferences`

  const navItems: { to: string; label: string; icon: LucideIcon }[] = [
    { to: 'profile', label: t('preferences.sidebar.profile'), icon: User },
    { to: 'appearance', label: t('preferences.sidebar.appearance'), icon: Palette },
  ]

  return (
    <div>
      <Link
        to={`/portal/${namespace}/projects/${projectKey}/tickets`}
        className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        {t('portal.backToTickets')}
      </Link>

      {/* Mobile top nav — above the flex row */}
      <nav className="flex sm:hidden mb-4 overflow-x-auto scrollbar-none -mx-4 px-4">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={`${base}/${item.to}`}
            className={({ isActive }) =>
              `flex shrink-0 flex-col items-center gap-1 py-3 px-4 text-xs font-medium transition-colors ${
                isActive
                  ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                  : 'text-gray-500 hover:bg-gray-50 dark:text-gray-400 dark:hover:bg-gray-800'
              }`
            }
          >
            <item.icon className="h-5 w-5" />
            <span className="whitespace-nowrap">{item.label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="flex gap-8">
        {/* Desktop sidebar */}
        <nav className="hidden sm:block shrink-0 w-48 pt-1">
          <ul className="space-y-1">
            {navItems.map((item) => (
              <li key={item.to}>
                <NavLink
                  to={`${base}/${item.to}`}
                  className={({ isActive }) =>
                    `flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                        : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
                    }`
                  }
                >
                  <item.icon className="h-5 w-5 shrink-0" />
                  <span>{item.label}</span>
                </NavLink>
              </li>
            ))}
          </ul>
          <div className="border-t border-gray-200 dark:border-gray-700 mt-4 pt-4">
            <Link
              to={`/portal/${namespace}/projects/${projectKey}/tickets`}
              className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-300 transition-colors"
            >
              <ArrowLeft className="h-5 w-5 shrink-0" />
              <span>{t('portal.backToTickets')}</span>
            </Link>
          </div>
        </nav>
        <div className="flex-1 min-w-0 max-w-3xl">
          <Outlet />
        </div>
      </div>
    </div>
  )
}

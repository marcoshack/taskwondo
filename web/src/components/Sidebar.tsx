import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

interface SidebarProps {
  projectKey: string
}

export function Sidebar({ projectKey }: SidebarProps) {
  const { t } = useTranslation()
  const base = `/projects/${projectKey}`

  const navItems = [
    { to: '', label: t('sidebar.overview'), end: true },
    { to: 'items', label: t('sidebar.items'), end: false },
    { to: 'queues', label: t('sidebar.queues'), end: false },
    { to: 'milestones', label: t('sidebar.milestones'), end: false },
    { to: 'settings', label: t('sidebar.settings'), end: false },
  ]

  return (
    <nav className="w-48 shrink-0">
      <ul className="space-y-1">
        {navItems.map((item) => (
          <li key={item.to}>
            <NavLink
              to={`${base}/${item.to}`}
              end={item.end}
              className={({ isActive }) =>
                `block px-3 py-2 rounded-md text-sm font-medium ${
                  isActive
                    ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                    : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
                }`
              }
            >
              {item.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}

import { NavLink, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Settings, Users, Route as RouteIcon, Plug } from 'lucide-react'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { SystemSettingsSidebar } from '@/components/SystemSettingsSidebar'
import { AdminUsersPage } from './AdminUsersPage'
import { SystemGeneralPage } from './SystemGeneralPage'
import { SystemWorkflowsPage } from './SystemWorkflowsPage'
import { SystemIntegrationsPage } from './SystemIntegrationsPage'

export function SystemSettingsPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar('settings')
  const { guardRef, guardedNavigate } = useNavigationGuard()

  const navItems = [
    { to: 'general', label: t('admin.sidebar.general'), icon: Settings },
    { to: 'users', label: t('admin.sidebar.users'), icon: Users },
    { to: 'workflows', label: t('admin.sidebar.workflows'), icon: RouteIcon },
    { to: 'integrations', label: t('admin.sidebar.integrations'), icon: Plug },
  ]

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      {/* Mobile top bar with navigation icons */}
      <nav className="flex sm:hidden mb-4 overflow-hidden">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={`/admin/${item.to}`}
            onClick={(e) => {
              if (guardRef.current?.()) {
                e.preventDefault()
                guardedNavigate(`/admin/${item.to}`)
              }
            }}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center gap-1 py-3 text-xs font-medium transition-colors ${
                isActive
                  ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                  : 'text-gray-500 hover:bg-gray-50 dark:text-gray-400 dark:hover:bg-gray-800'
              }`
            }
          >
            <item.icon className="h-5 w-5" />
            <span className="truncate max-w-full px-1">{item.label}</span>
          </NavLink>
        ))}
      </nav>

      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <SystemSettingsSidebar />
        <div className="flex-1 min-w-0">
          <Routes>
            <Route index element={<Navigate to="general" replace />} />
            <Route path="general" element={<SystemGeneralPage />} />
            <Route path="users" element={<AdminUsersPage />} />
            <Route path="workflows" element={<SystemWorkflowsPage />} />
            <Route path="integrations" element={<SystemIntegrationsPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

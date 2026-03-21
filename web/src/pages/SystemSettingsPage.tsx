import { NavLink, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Settings, Users, Route as RouteIcon, Plug, Lock, ToggleRight, Key, FolderOpen } from 'lucide-react'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { SystemSettingsSidebar } from '@/components/SystemSettingsSidebar'
import { AdminUsersPage } from './AdminUsersPage'
import { SystemGeneralPage } from './SystemGeneralPage'
import { SystemWorkflowsPage } from './SystemWorkflowsPage'
import { SystemIntegrationsPage } from './SystemIntegrationsPage'
import { SystemAuthenticationPage } from './SystemAuthenticationPage'
import { SystemFeaturesPage } from './SystemFeaturesPage'
import { SystemAPIKeysPage } from './SystemAPIKeysPage'
import { SystemProjectsPage } from './SystemProjectsPage'

export function SystemSettingsPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar('settings')
  const { containerClass } = useLayout()
  const { guardRef, guardedNavigate } = useNavigationGuard()

  const navItems = [
    { to: 'general', label: t('admin.sidebar.general'), icon: Settings },
    { to: 'users', label: t('admin.sidebar.users'), icon: Users },
    { to: 'project-overview', label: t('admin.sidebar.projects'), icon: FolderOpen },
    { to: 'workflows', label: t('admin.sidebar.workflows'), icon: RouteIcon },
    { to: 'integrations', label: t('admin.sidebar.integrations'), icon: Plug },
    { to: 'authentication', label: t('admin.sidebar.authentication'), icon: Lock },
    { to: 'api-keys', label: t('admin.sidebar.apiKeys'), icon: Key },
    { to: 'features', label: t('admin.sidebar.features'), icon: ToggleRight },
  ]

  return (
    <div className={`${containerClass(true)} py-6`}>
      {/* Mobile top bar with navigation icons */}
      <nav className="flex sm:hidden mb-4 overflow-x-auto scrollbar-none -mx-4 px-4">
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

      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <SystemSettingsSidebar />
        <div className="flex-1 min-w-0">
          <Routes>
            <Route index element={<Navigate to="general" replace />} />
            <Route path="general" element={<SystemGeneralPage />} />
            <Route path="users" element={<AdminUsersPage />} />
            <Route path="project-overview" element={<SystemProjectsPage />} />
            <Route path="workflows" element={<SystemWorkflowsPage />} />
            <Route path="integrations" element={<SystemIntegrationsPage />} />
            <Route path="authentication" element={<SystemAuthenticationPage />} />
            <Route path="features" element={<SystemFeaturesPage />} />
            <Route path="api-keys" element={<SystemAPIKeysPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

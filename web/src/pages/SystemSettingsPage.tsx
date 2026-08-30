import { NavLink, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Settings, Users, Route as RouteIcon, Plug, Lock, ToggleRight, Key } from 'lucide-react'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { SystemSettingsSidebar } from '@/components/SystemSettingsSidebar'
import { SystemGeneralPage } from './SystemGeneralPage'
import { SystemWorkflowsPage } from './SystemWorkflowsPage'
import { SystemIntegrationsPage } from './SystemIntegrationsPage'
import { SystemAuthenticationPage } from './SystemAuthenticationPage'
import { SystemFeaturesPage } from './SystemFeaturesPage'
import { SystemAPIKeysPage } from './SystemAPIKeysPage'
import { SystemDirectoryPage } from './SystemDirectoryPage'

export function SystemSettingsPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar('settings')
  const { containerClass } = useLayout()
  const { guardRef, guardedNavigate } = useNavigationGuard()

  const navItems = [
    { to: 'general', label: t('admin.sidebar.general'), icon: Settings },
    { to: 'directory', label: t('admin.sidebar.directory'), icon: Users },
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
                  ? 'bg-[var(--primary-muted)] text-[var(--primary)]  dark:text-[var(--primary)]'
                  : 'text-[var(--foreground-secondary)] hover:bg-[var(--surface-secondary)] text-[var(--foreground-muted)] dark:hover:bg-[var(--surface)]'
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
            <Route path="directory" element={<SystemDirectoryPage />} />
            {/* Backward-compat redirects from the previous routes */}
            <Route path="users" element={<Navigate to="/admin/directory?tab=users" replace />} />
            <Route path="project-overview" element={<Navigate to="/admin/directory?tab=projects" replace />} />
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

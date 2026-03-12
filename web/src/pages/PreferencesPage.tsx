import { NavLink, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Palette, Key, Bell, User, Settings } from 'lucide-react'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { PreferencesSidebar } from '@/components/PreferencesSidebar'
import { ProfilePage } from './ProfilePage'
import { GeneralPage } from './GeneralPage'
import { AppearancePage } from './AppearancePage'
import { NotificationsPage } from './NotificationsPage'
import { APIKeysPage } from './APIKeysPage'

export function PreferencesPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar('settings')
  const { containerClass } = useLayout()

  const navItems = [
    { to: 'profile', label: t('preferences.sidebar.profile'), icon: User },
    { to: 'general', label: t('preferences.sidebar.general'), icon: Settings },
    { to: 'appearance', label: t('preferences.sidebar.appearance'), icon: Palette },
    { to: 'notifications', label: t('preferences.sidebar.notifications'), icon: Bell },
    { to: 'api-keys', label: t('preferences.sidebar.apiKeys'), icon: Key },
  ]

  return (
    <div className={`${containerClass(true)} py-6 overflow-x-hidden`}>
      {/* Mobile top bar with navigation icons */}
      <nav className="flex sm:hidden mb-4 overflow-x-auto scrollbar-none -mx-4 px-4">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={`/preferences/${item.to}`}
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
        <PreferencesSidebar />
        <div className="flex-1 min-w-0 max-w-3xl">
          <Routes>
            <Route index element={<Navigate to="profile" replace />} />
            <Route path="profile" element={<ProfilePage />} />
            <Route path="general" element={<GeneralPage />} />
            <Route path="appearance" element={<AppearancePage />} />
            <Route path="notifications" element={<NotificationsPage />} />
            <Route path="api-keys" element={<APIKeysPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

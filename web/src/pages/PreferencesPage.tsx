import { NavLink, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Palette, Key } from 'lucide-react'
import { useSidebar } from '@/contexts/SidebarContext'
import { PreferencesSidebar } from '@/components/PreferencesSidebar'
import { AppearancePage } from './AppearancePage'
import { APIKeysPage } from './APIKeysPage'

export function PreferencesPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar('settings')

  const navItems = [
    { to: 'appearance', label: t('preferences.sidebar.appearance'), icon: Palette },
    { to: 'api-keys', label: t('preferences.sidebar.apiKeys'), icon: Key },
  ]

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      {/* Mobile top bar with navigation icons */}
      <nav className="flex sm:hidden mb-4 overflow-hidden">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={`/preferences/${item.to}`}
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
        <PreferencesSidebar />
        <div className="flex-1 min-w-0 max-w-3xl">
          <Routes>
            <Route index element={<Navigate to="appearance" replace />} />
            <Route path="appearance" element={<AppearancePage />} />
            <Route path="api-keys" element={<APIKeysPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

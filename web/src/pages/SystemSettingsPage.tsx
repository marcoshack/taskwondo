import { Routes, Route, Navigate } from 'react-router-dom'
import { useSidebar } from '@/contexts/SidebarContext'
import { SystemSettingsSidebar } from '@/components/SystemSettingsSidebar'
import { AdminUsersPage } from './AdminUsersPage'

export function SystemSettingsPage() {
  const { collapsed } = useSidebar()

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <SystemSettingsSidebar />
        <div className="flex-1 min-w-0">
          <Routes>
            <Route index element={<Navigate to="users" replace />} />
            <Route path="users" element={<AdminUsersPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

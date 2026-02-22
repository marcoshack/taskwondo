import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { Spinner } from '@/components/ui/Spinner'

export function ProtectedRoute() {
  const { user, isLoading, forcePasswordChange } = useAuth()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  if (forcePasswordChange) {
    return <Navigate to="/change-password" replace />
  }

  return <Outlet />
}

import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { Spinner } from '@/components/ui/Spinner'

export function ProtectedRoute() {
  const { user, isLoading, forcePasswordChange } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!user) {
    const currentPath = location.pathname + location.search + location.hash
    const loginUrl = currentPath && currentPath !== '/' ? `/login?next=${encodeURIComponent(currentPath)}` : '/login'
    return <Navigate to={loginUrl} replace />
  }

  if (forcePasswordChange) {
    return <Navigate to="/change-password" replace />
  }

  return <Outlet />
}

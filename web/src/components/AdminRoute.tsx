import { Navigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'

export function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuth()
  if (user?.global_role !== 'admin') {
    return <Navigate to="/projects" replace />
  }
  return <>{children}</>
}

import { Navigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespacePath } from '@/hooks/useNamespacePath'

export function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuth()
  const { p } = useNamespacePath()
  if (user?.global_role !== 'admin') {
    return <Navigate to={p('/projects')} replace />
  }
  return <>{children}</>
}

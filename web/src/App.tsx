import { Routes, Route, Navigate, Outlet, useParams } from 'react-router-dom'
import { ProtectedRoute } from '@/components/ProtectedRoute'
import { AdminRoute } from '@/components/AdminRoute'
import { NamespaceGuard } from '@/components/NamespaceGuard'
import { AppShell } from '@/components/AppShell'
import { LoginPage } from '@/pages/LoginPage'
import { OAuthCallbackPage } from '@/pages/OAuthCallbackPage'
import { ChangePasswordPage } from '@/pages/ChangePasswordPage'
import { ProjectListPage } from '@/pages/ProjectListPage'
import { ProjectDetailPage } from '@/pages/ProjectDetailPage'
import { PreferencesPage } from '@/pages/PreferencesPage'
import { SystemSettingsPage } from '@/pages/SystemSettingsPage'
import { InviteAcceptPage } from '@/pages/InviteAcceptPage'
import { CliAuthorizePage } from '@/pages/CliAuthorizePage'
import { RegisterPage } from '@/pages/RegisterPage'
import { VerifyEmailPage } from '@/pages/VerifyEmailPage'
import UserPage from '@/pages/InboxPage'
import { NamespaceSettingsPage } from '@/pages/NamespaceSettingsPage'
import { PortalShell } from '@/components/PortalShell'
import { PortalTicketListPage } from '@/pages/PortalTicketListPage'
import { PortalTicketDetailPage } from '@/pages/PortalTicketDetailPage'
import { PortalPreferencesPage } from '@/pages/PortalPreferencesPage'
import { PortalAppearancePage } from '@/pages/PortalAppearancePage'
import { ProfilePage } from '@/pages/ProfilePage'
import { useAuth } from '@/contexts/AuthContext'

/** True when the user is a portal-only customer (single customer project, no other memberships). */
function isPortalOnly(user: { portal_projects?: { project_key: string; namespace?: string }[]; total_project_count?: number; namespace_member_count?: number } | null): boolean {
  const pp = user?.portal_projects ?? []
  const total = user?.total_project_count ?? 0
  const nsMembers = user?.namespace_member_count ?? 0
  return pp.length === 1 && total === 1 && nsMembers === 0
}

/** Redirect to stored namespace or default — portal-only users go to portal */
function DefaultRedirect() {
  const { user } = useAuth()

  if (isPortalOnly(user)) {
    const first = user!.portal_projects![0]
    const segment = first.namespace || 'd'
    return <Navigate to={`/portal/${segment}/projects/${first.project_key}/tickets`} replace />
  }

  const stored = localStorage.getItem('taskwondo_namespace') || 'default'
  const segment = stored === 'default' ? 'd' : stored
  return <Navigate to={`/${segment}/projects`} state={{ autoRedirect: true }} replace />
}

/** Redirects non-portal-only users away from /portal routes to the AppShell equivalent */
function PortalOnlyGuard() {
  const { user } = useAuth()
  const { namespace, projectKey } = useParams<{ namespace: string; projectKey: string }>()

  if (!isPortalOnly(user)) {
    return <Navigate to={`/${namespace}/projects/${projectKey}/support`} replace />
  }
  return <Outlet />
}

/** Blocks portal-only users from regular routes — redirects them to portal */
function CustomerGuard() {
  const { user } = useAuth()

  if (isPortalOnly(user)) {
    const first = user!.portal_projects![0]
    const segment = first.namespace || 'd'
    return <Navigate to={`/portal/${segment}/projects/${first.project_key}/tickets`} replace />
  }
  return <Outlet />
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/auth/:provider/callback" element={<OAuthCallbackPage />} />
      <Route path="/change-password" element={<ChangePasswordPage />} />
      <Route path="/invite/:code" element={<InviteAcceptPage />} />
      <Route path="/auth/cli/authorize" element={<CliAuthorizePage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/verify-email" element={<VerifyEmailPage />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<CustomerGuard />}>
          <Route element={<AppShell />}>
            <Route path="/:namespace" element={<NamespaceGuard />}>
              <Route path="projects" element={<ProjectListPage />} />
              <Route path="projects/:projectKey/*" element={<ProjectDetailPage />} />
              <Route path="settings" element={<NamespaceSettingsPage />} />
            </Route>
            <Route path="/user/*" element={<UserPage />} />
            <Route path="/preferences/*" element={<PreferencesPage />} />
            <Route path="/admin/*" element={<AdminRoute><SystemSettingsPage /></AdminRoute>} />
          </Route>
        </Route>
      </Route>
      <Route element={<ProtectedRoute />}>
        <Route path="/portal/:namespace/projects/:projectKey" element={<PortalOnlyGuard />}>
          <Route element={<PortalShell />}>
            <Route path="tickets" element={<PortalTicketListPage />} />
            <Route path="tickets/:itemNumber" element={<PortalTicketDetailPage />} />
            <Route path="preferences" element={<PortalPreferencesPage />}>
              <Route index element={<Navigate to="profile" replace />} />
              <Route path="profile" element={<ProfilePage />} />
              <Route path="appearance" element={<PortalAppearancePage />} />
            </Route>
          </Route>
        </Route>
      </Route>
      <Route path="*" element={<DefaultRedirect />} />
    </Routes>
  )
}

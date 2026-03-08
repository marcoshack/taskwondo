import { Routes, Route, Navigate } from 'react-router-dom'
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

/** Redirect to stored namespace or default */
function DefaultRedirect() {
  const stored = localStorage.getItem('taskwondo_namespace') || 'default'
  const segment = stored === 'default' ? 'd' : stored
  return <Navigate to={`/${segment}/projects`} replace />
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
      <Route path="*" element={<DefaultRedirect />} />
    </Routes>
  )
}

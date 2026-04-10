import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation, Trans } from 'react-i18next'
import { Link2, Unlink, Check } from 'lucide-react'
import { Tabs } from '@/components/ui/Tabs'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { useAuth } from '@/contexts/AuthContext'
import { useConnectedAccounts, useUnlinkConnectedAccount } from '@/hooks/useConnectedAccounts'
import { setToken } from '@/api/client'
import * as authApi from '@/api/auth'
import type { ConnectedAccount } from '@/api/auth'
import { getLocalizedError } from '@/utils/apiError'
import { APIKeysPage } from './APIKeysPage'

const PROVIDER_ICONS: Record<string, React.ReactNode> = {
  google: (
    <svg className="w-5 h-5" viewBox="0 0 24 24">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
    </svg>
  ),
  discord: (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="#5865F2">
      <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286z" />
    </svg>
  ),
  github: (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
    </svg>
  ),
  microsoft: (
    <svg className="w-5 h-5" viewBox="0 0 23 23">
      <rect x="1" y="1" width="10" height="10" fill="#F25022"/>
      <rect x="12" y="1" width="10" height="10" fill="#7FBA00"/>
      <rect x="1" y="12" width="10" height="10" fill="#00A4EF"/>
      <rect x="12" y="12" width="10" height="10" fill="#FFB900"/>
    </svg>
  ),
}

function PasswordTab() {
  const { t } = useTranslation()
  const { user, updateUser } = useAuth()
  const hasPassword = user?.has_password ?? true
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [saved, setSaved] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setSaved(false)

    if (newPassword.length < 8) {
      setError(t('changePassword.tooShort'))
      return
    }

    if (newPassword !== confirmPassword) {
      setError(t('changePassword.mismatch'))
      return
    }

    setLoading(true)
    try {
      const { token } = await authApi.changePassword(oldPassword, newPassword)
      setToken(token)
      if (user) {
        updateUser({ ...user, has_password: true })
      }
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      setError(getLocalizedError(err, t, 'changePassword.error'))
    } finally {
      setLoading(false)
    }
  }

  const descriptionKey = hasPassword
    ? 'preferences.authentication.password.description'
    : 'preferences.authentication.password.setDescription'
  const submitKey = hasPassword
    ? 'changePassword.submit'
    : 'preferences.authentication.password.setSubmit'
  const passwordsMatch = newPassword.length > 0 && newPassword === confirmPassword

  return (
    <div className="mt-6">
      <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
        {t(descriptionKey)}
      </p>
      <form onSubmit={handleSubmit} className="space-y-4 max-w-md">
        {hasPassword && (
          <div>
            <div className="flex items-center justify-between mb-1">
              <label htmlFor="old-password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {t('changePassword.oldPassword')}
              </label>
              <Link
                to="/forgot-password"
                className="text-xs text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
              >
                {t('preferences.authentication.password.forgotLink')}
              </Link>
            </div>
            <Input
              id="old-password"
              type="password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              required
              autoComplete="current-password"
            />
          </div>
        )}
        <Input
          label={t('changePassword.newPassword')}
          type="password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          required
          autoComplete="new-password"
        />
        <Input
          label={t('changePassword.confirmPassword')}
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          required
          autoComplete="new-password"
          error={confirmPassword.length > 0 && !passwordsMatch ? t('changePassword.mismatch') : undefined}
        />
        {error && (
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        )}
        <div className="flex items-center gap-2">
          <Button type="submit" disabled={loading || !passwordsMatch}>
            {loading ? <Spinner className="h-4 w-4" /> : t(submitKey)}
          </Button>
          {saved && <Check className="h-4 w-4 text-green-500" />}
        </div>
      </form>
    </div>
  )
}

function ConnectedAccountsTab() {
  const { t } = useTranslation()
  const { data: accounts, isLoading } = useConnectedAccounts()
  const unlinkMutation = useUnlinkConnectedAccount()
  const [unlinkTarget, setUnlinkTarget] = useState<ConnectedAccount | null>(null)

  async function handleUnlink() {
    if (!unlinkTarget) return
    try {
      await unlinkMutation.mutateAsync(unlinkTarget.id)
      setUnlinkTarget(null)
    } catch {
      // error handled by mutation
    }
  }

  function providerLabel(provider: string): string {
    const labels: Record<string, string> = {
      google: 'Google',
      discord: 'Discord',
      github: 'GitHub',
      microsoft: 'Microsoft',
    }
    return labels[provider] ?? provider
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-8"><Spinner /></div>
    )
  }

  return (
    <div className="mt-6">
      <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
        {t('preferences.authentication.connectedAccounts.description')}
      </p>

      {!accounts || accounts.length === 0 ? (
        <div className="text-center py-8 text-gray-500 dark:text-gray-400">
          <Link2 className="h-8 w-8 mx-auto mb-2 opacity-40" />
          <p>{t('preferences.authentication.connectedAccounts.empty')}</p>
        </div>
      ) : (
        <div className="rounded-lg border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
          {accounts.map((account) => (
            <div key={account.id} className="p-4 bg-white dark:bg-gray-800 first:rounded-t-lg last:rounded-b-lg">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="shrink-0">
                    {PROVIDER_ICONS[account.provider] ?? <Link2 className="h-5 w-5 text-gray-400" />}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900 dark:text-gray-100">
                        {providerLabel(account.provider)}
                      </span>
                      <Badge color="gray">{t('preferences.authentication.connectedAccounts.linked')}</Badge>
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                      {account.provider_email && <span>{account.provider_email}</span>}
                      {account.provider_email && account.provider_username && <span> &middot; </span>}
                      {account.provider_username && <span>{account.provider_username}</span>}
                      {(account.provider_email || account.provider_username) && <span> &middot; </span>}
                      <span>{t('preferences.authentication.connectedAccounts.linkedOn')}: {new Date(account.created_at).toLocaleDateString()}</span>
                    </div>
                  </div>
                </div>
                <button
                  type="button"
                  className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                  onClick={() => setUnlinkTarget(account)}
                  title={t('preferences.authentication.connectedAccounts.unlink')}
                >
                  <Unlink className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Modal
        open={!!unlinkTarget}
        onClose={() => setUnlinkTarget(null)}
        title={t('preferences.authentication.connectedAccounts.unlinkConfirmTitle')}
      >
        <div className="p-4">
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            <Trans
              i18nKey="preferences.authentication.connectedAccounts.unlinkConfirmBody"
              values={{ provider: unlinkTarget ? providerLabel(unlinkTarget.provider) : '' }}
              components={{ bold: <strong className="font-semibold text-gray-900 dark:text-gray-100" /> }}
            />
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setUnlinkTarget(null)}>{t('common.cancel')}</Button>
            <Button variant="danger" onClick={handleUnlink} disabled={unlinkMutation.isPending}>
              {unlinkMutation.isPending ? <Spinner className="h-4 w-4" /> : t('preferences.authentication.connectedAccounts.unlink')}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

export function AuthenticationPage() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('password')

  const tabs = [
    { key: 'password', label: t('preferences.authentication.tabs.password') },
    { key: 'connected-accounts', label: t('preferences.authentication.tabs.connectedAccounts') },
    { key: 'api-keys', label: t('preferences.authentication.tabs.apiKeys') },
  ]

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          {t('preferences.authentication.title')}
        </h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {t('preferences.authentication.description')}
        </p>
      </div>

      <Tabs tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} />

      {activeTab === 'password' && <PasswordTab />}
      {activeTab === 'connected-accounts' && <ConnectedAccountsTab />}
      {activeTab === 'api-keys' && (
        <div className="mt-6">
          <APIKeysPage />
        </div>
      )}
    </div>
  )
}

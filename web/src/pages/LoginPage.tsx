import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import { Navigate, useSearchParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { isAxiosError } from 'axios'
import * as authApi from '@/api/auth'
import { useBrand } from '@/contexts/BrandContext'
import { usePublicSettings } from '@/hooks/useSystemSettings'
import { PoweredByFooter } from '@/components/PoweredByFooter'
import type { AuthProviders } from '@/api/auth'

const OAUTH_PROVIDERS: Record<string, { icon: React.ReactNode }> = {
  google: {
    icon: (
      <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24">
        <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
        <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
        <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
        <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
      </svg>
    ),
  },
  discord: {
    icon: (
      <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
        <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286z" />
      </svg>
    ),
  },
  github: {
    icon: (
      <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
      </svg>
    ),
  },
  microsoft: {
    icon: (
      <svg className="w-5 h-5 mr-2" viewBox="0 0 23 23">
        <rect x="1" y="1" width="10" height="10" fill="#F25022"/>
        <rect x="12" y="1" width="10" height="10" fill="#7FBA00"/>
        <rect x="1" y="12" width="10" height="10" fill="#00A4EF"/>
        <rect x="12" y="12" width="10" height="10" fill="#FFB900"/>
      </svg>
    ),
  },
}

const PENDING_INVITE_KEY = 'taskwondo_pending_invite'

export function LoginPage() {
  const { t } = useTranslation()
  const { brandName } = useBrand()
  const { user, login } = useAuth()
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [providers, setProviders] = useState<AuthProviders | null>(null)
  const { data: publicSettings } = usePublicSettings()

  useEffect(() => {
    authApi.getAuthProviders().then(setProviders).catch(() => {})
  }, [])

  if (user) {
    const next = searchParams.get('next')
    if (next && next.startsWith('/') && !next.startsWith('//')) {
      return <Navigate to={next} replace />
    }
    const pendingInvite = localStorage.getItem(PENDING_INVITE_KEY)
    if (pendingInvite) {
      // Don't remove here — InviteAcceptPage will clean up after accepting.
      // Removing here causes issues with React StrictMode double-rendering.
      return <Navigate to={`/invite/${pendingInvite}`} replace />
    }
    return <Navigate to="/projects" replace />
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(email, password)
      // Navigation is handled by the declarative if (user) block above,
      // which checks for pending invites and redirects accordingly.
    } catch (err) {
      if (isAxiosError(err) && err.response?.data?.error?.message) {
        setError(err.response.data.error.message)
      } else {
        setError(t('login.error'))
      }
    } finally {
      setLoading(false)
    }
  }

  const handleOAuthLogin = async (provider: string) => {
    try {
      const { url } = await authApi.getOAuthURL(provider)
      window.location.href = url
    } catch {
      setError(t(`login.${provider}.error`, t('login.oauth.error')))
    }
  }

  const providerOrder = Array.isArray(publicSettings?.oauth_provider_order)
    ? publicSettings.oauth_provider_order as string[]
    : ['discord', 'google', 'github', 'microsoft']

  const enabledProviders = providers
    ? Object.keys(OAUTH_PROVIDERS)
        .filter((p) => providers[p])
        .sort((a, b) => {
          const ai = providerOrder.indexOf(a)
          const bi = providerOrder.indexOf(b)
          return (ai === -1 ? Infinity : ai) - (bi === -1 ? Infinity : bi)
        })
    : []

  const emailLoginEnabled = providers ? providers.email_login !== false : true
  const emailRegistrationEnabled = providers?.email_registration === true

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
      <div className="flex-1 flex items-center justify-center">
      <div className="max-w-sm w-full">
        <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-gray-100 mb-8">
          {t('login.title', { brandName })}
        </h1>

        {emailLoginEnabled && (
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              label={t('login.email')}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
            />
            <Input
              label={t('login.password')}
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="current-password"
            />
            {error && (
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            )}
            <Button type="submit" disabled={loading} className="w-full">
              {loading ? t('login.submitting') : t('login.submit')}
            </Button>

            {emailRegistrationEnabled && (
              <p className="text-center text-sm text-gray-600 dark:text-gray-400">
                {t('login.noAccount')}{' '}
                <Link
                  to="/register"
                  className="text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
                >
                  {t('login.createAccount')}
                </Link>
              </p>
            )}
          </form>
        )}

        {!emailLoginEnabled && error && (
          <p className="text-sm text-red-600 dark:text-red-400 mb-4">{error}</p>
        )}

        {enabledProviders.length > 0 && (
          <>
            {emailLoginEnabled && (
              <div className="relative my-6">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-gray-300 dark:border-gray-600" />
                </div>
                <div className="relative flex justify-center text-sm">
                  <span className="px-2 bg-gray-50 dark:bg-gray-900 text-gray-500 dark:text-gray-400">
                    {t('login.or')}
                  </span>
                </div>
              </div>
            )}

            <div className="space-y-3">
              {enabledProviders.map((provider) => (
                <Button
                  key={provider}
                  type="button"
                  variant="secondary"
                  className="w-full"
                  onClick={() => handleOAuthLogin(provider)}
                >
                  {OAUTH_PROVIDERS[provider]?.icon}
                  {t(`login.${provider}.button`)}
                </Button>
              ))}
            </div>
          </>
        )}
      </div>
      </div>
      <PoweredByFooter />
    </div>
  )
}

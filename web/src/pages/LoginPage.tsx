import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import { Navigate, useSearchParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import * as authApi from '@/api/auth'
import { getLocalizedError } from '@/utils/apiError'
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
        <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6348.8731-1.3003 1.2157-1.9905a.077.077 0 00-.037-.0967 14.6185 14.6185 0 01-2.0922-.9973.0743.0743 0 00-.0785-.0031c-4.2526 1.9424-7.4501 7.2832-7.4501 12.3896 0 .0647.0031.1294.0062.1939a.0822.0822 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6348.8731-1.3003 1.2157-1.9905a.077.077 0 00-.037-.0967 14.6185 14.6185 0 01-2.0922-.9973.0743.0743 0 00-.0785-.0031c-4.2526 1.9424-7.4501 7.2832-7.4501 12.3896 0 .0647.0031.1294.0062.1939a.0822.0822 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6348.8731-1.3003 1.2157-1.9905a.077.077 0 00-.037-.0967 14.6185 14.6185 0 01-2.0922-.9973.0743.0743 0 00-.0785-.0031c-4.2526 1.9424-7.4501 7.2832-7.4501 12.3896 0 .0647.0031.1294.0062.1939a.0822.0822 0 00.0312.0561"/>
      </svg>
    ),
  },
  github: {
    icon: (
      <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
      </svg>
    ),
  },
  microsoft: {
    icon: (
      <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24">
        <path d="M11.4 24H0V12.6h11.4V24z" fill="#F25022"/>
        <path d="M24 24H12.6V12.6H24V24z" fill="#7FBA00"/>
        <path d="M11.4 11.4H0V0h11.4v11.4z" fill="#00A4EF"/>
        <path d="M24 11.4H12.6V0H24v11.4z" fill="#FFB900"/>
      </svg>
    ),
  },
  sso: {
    icon: (
      <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/>
      </svg>
    ),
  },
}

const PENDING_INVITE_KEY = 'taskwondo_pending_invite'
const OAUTH_NEXT_KEY = 'taskwondo_oauth_next'

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
      return <Navigate to={`/invite/${pendingInvite}`} replace />
    }
    return <Navigate to="/d/projects" replace />
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(email, password)
    } catch (err) {
      setError(getLocalizedError(err, t, 'login.error'))
    } finally {
      setLoading(false)
    }
  }

  const handleOAuthLogin = async (provider: string) => {
    try {
      const next = searchParams.get('next')
      if (next && next.startsWith('/') && !next.startsWith('//')) {
        sessionStorage.setItem(OAUTH_NEXT_KEY, next)
      } else {
        sessionStorage.removeItem(OAUTH_NEXT_KEY)
      }
      const { url } = await authApi.getOAuthURL(provider)
      window.location.href = url
    } catch {
      setError(t(`login.${provider}.error`, t('login.oauth.error')))
    }
  }

  const providerOrder = Array.isArray(publicSettings?.oauth_provider_order)
    ? publicSettings.oauth_provider_order as string[]
    : ['discord', 'google', 'github', 'microsoft', 'sso']

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
    <div className="min-h-screen flex bg-[var(--background)]">
      <div className="hidden lg:flex lg:w-1/2 bg-[var(--surface-secondary)] flex-col justify-between p-12">
        <div>
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-[var(--radius-md)] bg-[var(--primary)] flex items-center justify-center">
              <span className="text-[var(--primary-foreground)] font-bold text-sm">{brandName.charAt(0).toUpperCase()}</span>
            </div>
            <span className="text-lg font-semibold text-[var(--foreground)]">{brandName}</span>
          </div>
        </div>

        <div className="max-w-md">
          <h2 className="text-2xl font-semibold text-[var(--foreground)] tracking-tight">
            {t('login.heroTitle', 'Manage projects with clarity')}
          </h2>
          <p className="mt-3 text-[var(--foreground-secondary)] leading-relaxed">
            {t('login.heroDesc', 'Track work items, collaborate with your team, and deliver projects on time — all in one place.')}
          </p>
        </div>

        <PoweredByFooter />
      </div>

      <div className="flex-1 flex flex-col">
        <div className="flex-1 flex items-center justify-center px-6 py-12">
          <div className="w-full max-w-[360px]">
            <div className="lg:hidden flex items-center gap-2.5 mb-8">
              <div className="w-8 h-8 rounded-[var(--radius)] bg-[var(--primary)] flex items-center justify-center">
                <span className="text-[var(--primary-foreground)] font-bold text-xs">{brandName.charAt(0).toUpperCase()}</span>
              </div>
              <span className="text-base font-semibold text-[var(--foreground)]">{brandName}</span>
            </div>

            <h1 className="text-xl font-semibold text-[var(--foreground)] tracking-tight">
              {t('login.title', { brandName })}
            </h1>
            <p className="mt-1.5 text-sm text-[var(--foreground-secondary)]">
              {t('login.subtitle', 'Sign in to continue')}
            </p>

            {enabledProviders.length > 0 && (
              <div className="mt-6 space-y-2">
                {enabledProviders.map((p) => (
                  <button
                    key={p}
                    type="button"
                    onClick={() => handleOAuthLogin(p)}
                    className="w-full flex items-center justify-center gap-2 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--surface)] px-3 py-2 text-sm font-medium text-[var(--foreground)] hover:bg-[var(--surface-hover)] transition-colors"
                  >
                    {OAUTH_PROVIDERS[p]?.icon}
                    {p === 'sso' ? t('login.sso.button', 'Continue with SSO') : t(`login.${p}.button`, `Continue with ${p.charAt(0).toUpperCase() + p.slice(1)}`)}
                  </button>
                ))}
              </div>
            )}

            {enabledProviders.length > 0 && emailLoginEnabled && (
              <div className="relative my-6">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-[var(--border)]" />
                </div>
                <div className="relative flex justify-center">
                  <span className="bg-[var(--background)] px-3 text-xs text-[var(--foreground-muted)]">{t('login.orEmail', 'or')}</span>
                </div>
              </div>
            )}

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
                  <p className="text-sm text-[var(--danger)]">{error}</p>
                )}
                <Button type="submit" disabled={loading} className="w-full">
                  {loading ? t('login.submitting') : t('login.submit')}
                </Button>

                <div className="flex items-center justify-between pt-1">
                  <Link
                    to="/forgot-password"
                    className="text-xs text-[var(--foreground-secondary)] hover:text-[var(--primary)] transition-colors"
                  >
                    {t('login.forgotPassword')}
                  </Link>
                  {emailRegistrationEnabled && (
                    <Link
                      to="/register"
                      className="text-xs text-[var(--foreground-secondary)] hover:text-[var(--primary)] transition-colors"
                    >
                      {t('login.createAccount')}
                    </Link>
                  )}
                </div>
              </form>
            )}
          </div>
        </div>

        <div className="lg:hidden px-6 pb-6">
          <PoweredByFooter />
        </div>
      </div>
    </div>
  )
}

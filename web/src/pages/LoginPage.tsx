import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import { Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { isAxiosError } from 'axios'
import * as authApi from '@/api/auth'
import { useBrand } from '@/contexts/BrandContext'
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
}

export function LoginPage() {
  const { t } = useTranslation()
  const { brandName } = useBrand()
  const { user, login } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [providers, setProviders] = useState<AuthProviders | null>(null)

  useEffect(() => {
    authApi.getAuthProviders().then(setProviders).catch(() => {})
  }, [])

  if (user) {
    return <Navigate to="/projects" replace />
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(email, password)
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

  const enabledProviders = providers
    ? Object.keys(OAUTH_PROVIDERS).filter((p) => providers[p])
    : []

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
      <div className="flex-1 flex items-center justify-center">
      <div className="max-w-sm w-full">
        <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-gray-100 mb-8">
          {t('login.title', { brandName })}
        </h1>
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
        </form>

        {enabledProviders.length > 0 && (
          <>
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

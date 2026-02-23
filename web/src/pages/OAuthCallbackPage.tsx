import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { oauthCallback } from '@/api/auth'
import { Spinner } from '@/components/ui/Spinner'

export function OAuthCallbackPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { provider } = useParams<{ provider: string }>()
  const { loginWithToken } = useAuth()
  const [searchParams] = useSearchParams()
  const [error, setError] = useState('')
  const processed = useRef(false)

  useEffect(() => {
    if (processed.current) return
    processed.current = true

    const code = searchParams.get('code')
    const state = searchParams.get('state')

    if (!code || !state || !provider) {
      setError(t(`login.${provider ?? 'oauth'}.callbackError`, t('login.oauth.callbackError')))
      return
    }

    oauthCallback(provider, code, state)
      .then(({ token, user }) => {
        loginWithToken(token, user)
        navigate('/projects', { replace: true })
      })
      .catch(() => {
        setError(t(`login.${provider}.callbackError`, t('login.oauth.callbackError')))
      })
  }, [searchParams, loginWithToken, navigate, t, provider])

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="max-w-sm w-full text-center">
          <p className="text-sm text-red-600 dark:text-red-400 mb-4">{error}</p>
          <a href="/login" className="text-indigo-600 dark:text-indigo-400 hover:underline text-sm">
            {t(`login.${provider ?? 'oauth'}.backToLogin`, t('login.oauth.backToLogin'))}
          </a>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="text-center">
        <Spinner size="lg" />
        <p className="mt-4 text-sm text-gray-500 dark:text-gray-400">
          {t(`login.${provider ?? 'oauth'}.authenticating`, t('login.oauth.authenticating'))}
        </p>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { useTheme } from '@/contexts/ThemeContext'
import { useBrand } from '@/contexts/BrandContext'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'
import { createAPIKey } from '@/api/auth'
import { PoweredByFooter } from '@/components/PoweredByFooter'

function resolvedTheme(theme: 'light' | 'dark' | 'system'): 'light' | 'dark' {
  if (theme === 'system') return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  return theme
}

export function CliAuthorizePage() {
  const { t } = useTranslation()
  const { user, isLoading } = useAuth()
  const { theme } = useTheme()
  const { brandName } = useBrand()
  const [searchParams] = useSearchParams()
  const [status, setStatus] = useState<'idle' | 'authorizing' | 'success' | 'denied' | 'error'>('idle')

  const callbackPortRaw = searchParams.get('callback_port')
  const callbackPort = callbackPortRaw && /^\d+$/.test(callbackPortRaw) ? parseInt(callbackPortRaw, 10) : null
  const clientName = searchParams.get('client_name') || 'CLI'

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <Spinner />
      </div>
    )
  }

  if (!user) {
    const next = `/auth/cli/authorize?${searchParams.toString()}`
    return <Navigate to={`/login?next=${encodeURIComponent(next)}`} replace />
  }

  if (!callbackPort || callbackPort < 1024 || callbackPort > 65535) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="max-w-md w-full text-center">
          <p className="text-gray-600 dark:text-gray-400">{t('cliAuth.missingParams')}</p>
        </div>
      </div>
    )
  }

  const callbackParams = `&theme=${resolvedTheme(theme)}&brand=${encodeURIComponent(brandName)}`

  const handleAllow = async () => {
    setStatus('authorizing')
    try {
      const apiKey = await createAPIKey(`CLI: ${clientName}`)
      window.location.href = `http://localhost:${callbackPort}/callback?key=${encodeURIComponent(apiKey.key)}${callbackParams}`
      setStatus('success')
    } catch {
      setStatus('error')
    }
  }

  const handleDeny = () => {
    setStatus('denied')
    window.location.href = `http://localhost:${callbackPort}/callback?error=denied${callbackParams}`
  }

  if (status === 'success') {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="max-w-md w-full text-center">
          <p className="text-lg text-green-600 dark:text-green-400">{t('cliAuth.success')}</p>
        </div>
      </div>
    )
  }

  if (status === 'denied') {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="max-w-md w-full text-center">
          <p className="text-lg text-gray-600 dark:text-gray-400">{t('cliAuth.denied')}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
      <div className="flex-1 flex items-center justify-center">
        <div className="max-w-md w-full">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-8">
            <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100 mb-6 text-center">
              {t('cliAuth.title')}
            </h1>

            <p className="text-gray-700 dark:text-gray-300 mb-4 text-center">
              {t('cliAuth.description', { clientName })}
            </p>

            <p className="text-sm text-gray-500 dark:text-gray-400 mb-2 text-center">
              {t('cliAuth.loggedInAs', { displayName: user.display_name, email: user.email })}
            </p>

            <p className="text-sm text-gray-500 dark:text-gray-400 mb-8 text-center">
              {t('cliAuth.permissions')}
            </p>

            {status === 'error' && (
              <p className="text-sm text-red-600 dark:text-red-400 mb-4 text-center">{t('cliAuth.error')}</p>
            )}

            <div className="flex gap-3">
              <Button
                variant="secondary"
                className="flex-1"
                onClick={handleDeny}
                disabled={status === 'authorizing'}
              >
                {t('cliAuth.deny')}
              </Button>
              <Button
                className="flex-1"
                onClick={handleAllow}
                disabled={status === 'authorizing'}
              >
                {status === 'authorizing' ? t('cliAuth.authorizing') : t('cliAuth.allow')}
              </Button>
            </div>
          </div>
        </div>
      </div>
      <PoweredByFooter />
    </div>
  )
}

import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { useBrand } from '@/contexts/BrandContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { PoweredByFooter } from '@/components/PoweredByFooter'
import * as authApi from '@/api/auth'
import { getLocalizedError } from '@/utils/apiError'
import { usePublicSettings } from '@/hooks/useSystemSettings'

export function RegisterPage() {
  const { t } = useTranslation()
  const { brandName } = useBrand()
  const { user } = useAuth()
  const { data: publicSettings } = usePublicSettings()
  const [displayName, setDisplayName] = useState('')
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)

  // Check if registration is enabled
  const registrationEnabled = publicSettings?.auth_email_registration_enabled === true

  if (user) {
    return <Navigate to="/d/projects" replace />
  }

  if (publicSettings && !registrationEnabled) {
    return <Navigate to="/login" replace />
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const pendingInvite = localStorage.getItem('taskwondo_pending_invite') || undefined
      await authApi.register(email, displayName, pendingInvite)
      setSuccess(true)
    } catch (err) {
      setError(getLocalizedError(err, t, 'register.error'))
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
        <div className="flex-1 flex items-center justify-center">
          <div className="max-w-sm w-full text-center">
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
              {t('register.checkEmail')}
            </h1>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
              {t('register.checkEmailDescription', { email })}
            </p>
            <Link
              to="/login"
              className="text-sm text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              {t('register.backToLogin')}
            </Link>
          </div>
        </div>
        <PoweredByFooter />
      </div>
    )
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
      <div className="flex-1 flex items-center justify-center">
        <div className="max-w-sm w-full">
          <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-gray-100 mb-8">
            {t('register.title', { brandName })}
          </h1>
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              label={t('register.displayName')}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              required
              autoComplete="name"
            />
            <Input
              label={t('register.email')}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
            />
            {error && (
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            )}
            <Button type="submit" disabled={loading} className="w-full">
              {loading ? t('register.submitting') : t('register.submit')}
            </Button>
          </form>
          <p className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
            {t('register.haveAccount')}{' '}
            <Link
              to="/login"
              className="text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              {t('register.signIn')}
            </Link>
          </p>
        </div>
      </div>
      <PoweredByFooter />
    </div>
  )
}

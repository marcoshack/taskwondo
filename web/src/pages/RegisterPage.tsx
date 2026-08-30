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
              {t('register.heroTitle', 'Join the team')}
            </h2>
            <p className="mt-3 text-[var(--foreground-secondary)] leading-relaxed">
              {t('register.heroDesc', 'Start managing your projects more effectively.')}
            </p>
          </div>

          <PoweredByFooter />
        </div>

        <div className="flex-1 flex flex-col">
          <div className="flex-1 flex items-center justify-center px-6 py-12">
            <div className="w-full max-w-[360px] text-center">
              <div className="lg:hidden flex items-center gap-2.5 mb-8 justify-center">
                <div className="w-8 h-8 rounded-[var(--radius)] bg-[var(--primary)] flex items-center justify-center">
                  <span className="text-[var(--primary-foreground)] font-bold text-xs">{brandName.charAt(0).toUpperCase()}</span>
                </div>
                <span className="text-base font-semibold text-[var(--foreground)]">{brandName}</span>
              </div>

              <div className="w-12 h-12 rounded-full bg-[var(--success-bg)] flex items-center justify-center mx-auto mb-4">
                <svg className="w-6 h-6 text-[var(--success)]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h1 className="text-xl font-semibold text-[var(--foreground)] tracking-tight">
                {t('register.checkEmail')}
              </h1>
              <p className="mt-2 text-sm text-[var(--foreground-secondary)]">
                {t('register.checkEmailDescription', { email })}
              </p>
              <Link
                to="/login"
                className="inline-block mt-6 text-sm text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
              >
                {t('register.backToLogin')}
              </Link>
            </div>
          </div>

          <div className="lg:hidden px-6 pb-6">
            <PoweredByFooter />
          </div>
        </div>
      </div>
    )
  }

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
            {t('register.heroTitle', 'Join the team')}
          </h2>
          <p className="mt-3 text-[var(--foreground-secondary)] leading-relaxed">
            {t('register.heroDesc', 'Start managing your projects more effectively.')}
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
              {t('register.title', { brandName })}
            </h1>
            <p className="mt-1.5 text-sm text-[var(--foreground-secondary)]">
              {t('register.subtitle', 'Create your account')}
            </p>

            <form onSubmit={handleSubmit} className="mt-6 space-y-4">
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
                <p className="text-sm text-[var(--danger)]">{error}</p>
              )}
              <Button type="submit" disabled={loading} className="w-full">
                {loading ? t('register.submitting') : t('register.submit')}
              </Button>

              <p className="text-center text-sm text-[var(--foreground-secondary)]">
                {t('register.hasAccount')}{' '}
                <Link
                  to="/login"
                  className="text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
                >
                  {t('register.backToLogin')}
                </Link>
              </p>
            </form>
          </div>
        </div>

        <div className="lg:hidden px-6 pb-6">
          <PoweredByFooter />
        </div>
      </div>
    </div>
  )
}

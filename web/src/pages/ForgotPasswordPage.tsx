import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { PoweredByFooter } from '@/components/PoweredByFooter'
import * as authApi from '@/api/auth'
import { getLocalizedError } from '@/utils/apiError'

export function ForgotPasswordPage() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await authApi.forgotPassword(email)
      setSuccess(true)
    } catch (err) {
      setError(getLocalizedError(err, t, 'forgotPassword.error'))
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <div className="min-h-screen flex bg-[var(--background)]">
        <div className="flex-1 flex flex-col">
          <div className="flex-1 flex items-center justify-center px-6 py-12">
            <div className="w-full max-w-[360px] text-center">
              <div className="w-12 h-12 rounded-full bg-[var(--success-bg)] flex items-center justify-center mx-auto mb-4">
                <svg className="w-6 h-6 text-[var(--success)]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                </svg>
              </div>
              <h1 className="text-xl font-semibold text-[var(--foreground)] tracking-tight">
                {t('forgotPassword.checkEmail')}
              </h1>
              <p className="mt-2 text-sm text-[var(--foreground-secondary)]">
                {t('forgotPassword.checkEmailDescription')}
              </p>
              <Link
                to="/login"
                className="inline-block mt-6 text-sm text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
              >
                {t('forgotPassword.backToLogin')}
              </Link>
            </div>
          </div>

          <div className="px-6 pb-6">
            <PoweredByFooter />
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex bg-[var(--background)]">
      <div className="flex-1 flex flex-col">
        <div className="flex-1 flex items-center justify-center px-6 py-12">
          <div className="w-full max-w-[360px]">
            <div className="flex items-center gap-2.5 mb-8">
              <Link to="/login" className="text-[var(--foreground-muted)] hover:text-[var(--foreground)] transition-colors">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
                </svg>
              </Link>
            </div>

            <h1 className="text-xl font-semibold text-[var(--foreground)] tracking-tight">
              {t('forgotPassword.title')}
            </h1>
            <p className="mt-1.5 text-sm text-[var(--foreground-secondary)]">
              {t('forgotPassword.description')}
            </p>

            <form onSubmit={handleSubmit} className="mt-6 space-y-4">
              <Input
                label={t('forgotPassword.email')}
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
                {loading ? t('forgotPassword.submitting') : t('forgotPassword.submit')}
              </Button>

              <p className="text-center text-sm text-[var(--foreground-secondary)]">
                {t('forgotPassword.rememberPassword')}{' '}
                <Link
                  to="/login"
                  className="text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
                >
                  {t('forgotPassword.backToLogin')}
                </Link>
              </p>
            </form>
          </div>
        </div>

        <div className="px-6 pb-6">
          <PoweredByFooter />
        </div>
      </div>
    </div>
  )
}

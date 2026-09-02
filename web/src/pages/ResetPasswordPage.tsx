import { useState } from 'react'
import type { FormEvent } from 'react'
import { Navigate, useSearchParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { PoweredByFooter } from '@/components/PoweredByFooter'
import * as authApi from '@/api/auth'
import { getLocalizedError } from '@/utils/apiError'

export function ResetPasswordPage() {
  const { t } = useTranslation()
  const { user, loginWithToken } = useAuth()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  if (user) {
    return <Navigate to="/d/projects" replace />
  }

  if (!token) {
    return <Navigate to="/login" replace />
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')

    if (password !== confirmPassword) {
      setError(t('resetPassword.mismatch'))
      return
    }

    if (password.length < 8) {
      setError(t('resetPassword.tooShort'))
      return
    }

    setLoading(true)
    try {
      const result = await authApi.resetPassword(token, password)
      loginWithToken(result.token, result.user)
    } catch (err) {
      setError(getLocalizedError(err, t, 'resetPassword.error'))
    } finally {
      setLoading(false)
    }
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
              {t('resetPassword.title')}
            </h1>
            <p className="mt-1.5 text-sm text-[var(--foreground-secondary)]">
              {t('resetPassword.description')}
            </p>

            <form onSubmit={handleSubmit} className="mt-6 space-y-4">
              <Input
                label={t('resetPassword.password')}
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                autoComplete="new-password"
              />
              <Input
                label={t('resetPassword.confirmPassword')}
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                autoComplete="new-password"
              />
              {error && (
                <p className="text-sm text-[var(--danger)]">{error}</p>
              )}
              <Button type="submit" disabled={loading} className="w-full">
                {loading ? t('resetPassword.submitting') : t('resetPassword.submit')}
              </Button>
            </form>
            <p className="mt-6 text-center text-sm text-[var(--foreground-secondary)]">
              <Link
                to="/login"
                className="text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
              >
                {t('resetPassword.backToLogin')}
              </Link>
            </p>
          </div>
        </div>

        <div className="px-6 pb-6">
          <PoweredByFooter />
        </div>
      </div>
    </div>
  )
}

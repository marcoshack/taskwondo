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
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
      <div className="flex-1 flex items-center justify-center">
        <div className="max-w-sm w-full">
          <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-gray-100 mb-2">
            {t('resetPassword.title')}
          </h1>
          <p className="text-center text-sm text-gray-600 dark:text-gray-400 mb-8">
            {t('resetPassword.description')}
          </p>
          <form onSubmit={handleSubmit} className="space-y-4">
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
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            )}
            <Button type="submit" disabled={loading} className="w-full">
              {loading ? t('resetPassword.submitting') : t('resetPassword.submit')}
            </Button>
          </form>
          <p className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
            <Link
              to="/login"
              className="text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              {t('resetPassword.backToLogin')}
            </Link>
          </p>
        </div>
      </div>
      <PoweredByFooter />
    </div>
  )
}

import { useState } from 'react'
import type { FormEvent } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { useBrand } from '@/contexts/BrandContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { PoweredByFooter } from '@/components/PoweredByFooter'
import * as authApi from '@/api/auth'
import { getLocalizedError } from '@/utils/apiError'

export function VerifyEmailPage() {
  const { t } = useTranslation()
  const { brandName } = useBrand()
  const { user, loginWithToken } = useAuth()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [redirectTo, setRedirectTo] = useState<string | null>(null)

  if (user) {
    return <Navigate to={redirectTo || '/d/projects'} replace />
  }

  if (!token) {
    return <Navigate to="/login" replace />
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')

    if (password !== confirmPassword) {
      setError(t('verifyEmail.mismatch'))
      return
    }

    if (password.length < 8) {
      setError(t('verifyEmail.tooShort'))
      return
    }

    setLoading(true)
    try {
      const result = await authApi.verifyEmail(token, password)
      if (result.project_key) {
        setRedirectTo(`/d/projects/${result.project_key}`)
      }
      localStorage.removeItem('taskwondo_pending_invite')
      loginWithToken(result.token, result.user)
    } catch (err) {
      setError(getLocalizedError(err, t, 'verifyEmail.error'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
      <div className="flex-1 flex items-center justify-center">
        <div className="max-w-sm w-full">
          <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-gray-100 mb-2">
            {t('verifyEmail.title', { brandName })}
          </h1>
          <p className="text-center text-sm text-gray-600 dark:text-gray-400 mb-8">
            {t('verifyEmail.description')}
          </p>
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              label={t('verifyEmail.password')}
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="new-password"
            />
            <Input
              label={t('verifyEmail.confirmPassword')}
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
              {loading ? t('verifyEmail.submitting') : t('verifyEmail.submit')}
            </Button>
          </form>
        </div>
      </div>
      <PoweredByFooter />
    </div>
  )
}

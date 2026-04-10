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
      <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
        <div className="flex-1 flex items-center justify-center">
          <div className="max-w-sm w-full text-center">
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
              {t('forgotPassword.checkEmail')}
            </h1>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
              {t('forgotPassword.checkEmailDescription')}
            </p>
            <Link
              to="/login"
              className="text-sm text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              {t('forgotPassword.backToLogin')}
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
          <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-gray-100 mb-2">
            {t('forgotPassword.title')}
          </h1>
          <p className="text-center text-sm text-gray-600 dark:text-gray-400 mb-8">
            {t('forgotPassword.description')}
          </p>
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              label={t('forgotPassword.email')}
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
              {loading ? t('forgotPassword.submitting') : t('forgotPassword.submit')}
            </Button>
          </form>
          <p className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
            <Link
              to="/login"
              className="text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              {t('forgotPassword.backToLogin')}
            </Link>
          </p>
        </div>
      </div>
      <PoweredByFooter />
    </div>
  )
}

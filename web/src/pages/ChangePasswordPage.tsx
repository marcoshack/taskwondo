import { useState } from 'react'
import type { FormEvent } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { isAxiosError } from 'axios'
import * as authApi from '@/api/auth'
import { useBrand } from '@/contexts/BrandContext'
import { PoweredByFooter } from '@/components/PoweredByFooter'

export function ChangePasswordPage() {
  const { t } = useTranslation()
  const { brandName } = useBrand()
  const { user, forcePasswordChange, clearForcePasswordChange } = useAuth()
  const navigate = useNavigate()
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  if (!user) {
    return <Navigate to="/login" replace />
  }

  if (!forcePasswordChange) {
    return <Navigate to="/projects" replace />
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')

    if (newPassword.length < 8) {
      setError(t('changePassword.tooShort'))
      return
    }

    if (newPassword !== confirmPassword) {
      setError(t('changePassword.mismatch'))
      return
    }

    setLoading(true)
    try {
      const { token } = await authApi.changePassword(oldPassword, newPassword)
      clearForcePasswordChange(token)
      navigate('/projects', { replace: true })
    } catch (err) {
      if (isAxiosError(err) && err.response?.data?.error?.message) {
        setError(err.response.data.error.message)
      } else {
        setError(t('changePassword.error'))
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 px-4">
      <div className="flex-1 flex items-center justify-center">
        <div className="max-w-sm w-full">
          <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-gray-100 mb-2">
            {t('changePassword.title', { brandName })}
          </h1>
          <p className="text-sm text-center text-gray-600 dark:text-gray-400 mb-8">
            {t('changePassword.description')}
          </p>
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              label={t('changePassword.oldPassword')}
              type="password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              required
              autoComplete="current-password"
            />
            <Input
              label={t('changePassword.newPassword')}
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              autoComplete="new-password"
            />
            <Input
              label={t('changePassword.confirmPassword')}
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
              {loading ? t('changePassword.submitting') : t('changePassword.submit')}
            </Button>
          </form>
        </div>
      </div>
      <PoweredByFooter />
    </div>
  )
}

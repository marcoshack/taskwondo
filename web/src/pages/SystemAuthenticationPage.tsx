import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePublicSettings, useSetSystemSetting, useSMTPConfig } from '@/hooks/useSystemSettings'
import { getAuthProviders } from '@/api/auth'
import type { AuthProviders } from '@/api/auth'
import { Toggle } from '@/components/ui/Toggle'
import { Spinner } from '@/components/ui/Spinner'
import { TriangleAlert } from 'lucide-react'

export function SystemAuthenticationPage() {
  const { t } = useTranslation()
  const { data: publicSettings, isLoading: settingsLoading } = usePublicSettings()
  const { data: smtpConfig, isLoading: smtpLoading } = useSMTPConfig()
  const setSetting = useSetSystemSetting()
  const [providers, setProviders] = useState<AuthProviders | null>(null)

  useEffect(() => {
    getAuthProviders().then(setProviders).catch(() => {})
  }, [])

  if (settingsLoading || smtpLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  const settings = publicSettings ?? {}

  // Resolve current toggle values with backward-compatible defaults
  const emailLoginEnabled = settings.auth_email_login_enabled !== undefined
    ? settings.auth_email_login_enabled === true
    : true // default: enabled
  const emailRegistrationEnabled = settings.auth_email_registration_enabled !== undefined
    ? settings.auth_email_registration_enabled === true
    : false // default: disabled
  const discordEnabled = settings.auth_discord_enabled !== undefined
    ? settings.auth_discord_enabled === true
    : true // default: enabled if configured
  const googleEnabled = settings.auth_google_enabled !== undefined
    ? settings.auth_google_enabled === true
    : true // default: enabled if configured

  // OAuth providers are "configured" if they appear in the providers map
  // (backend only includes them when env vars are set)
  const discordConfigured = providers !== null && 'discord' in providers
  const googleConfigured = providers !== null && 'google' in providers

  // SMTP is configured if the config exists and is enabled
  const smtpConfigured = smtpConfig?.enabled === true

  const handleToggle = (key: string, value: boolean) => {
    setSetting.mutate({ key, value })
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          {t('admin.authentication.title')}
        </h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t('admin.authentication.description')}
        </p>
      </div>

      {/* Email/Password Login */}
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
              {t('admin.authentication.emailLogin.title')}
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {t('admin.authentication.emailLogin.description')}
            </p>
          </div>
          <Toggle
            enabled={emailLoginEnabled}
            onChange={(val) => handleToggle('auth_email_login_enabled', val)}
          />
        </div>
      </div>

      {/* Email Registration */}
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
              {t('admin.authentication.emailRegistration.title')}
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {t('admin.authentication.emailRegistration.description')}
            </p>
          </div>
          <Toggle
            enabled={emailRegistrationEnabled}
            onChange={(val) => handleToggle('auth_email_registration_enabled', val)}
            disabled={!smtpConfigured}
          />
        </div>
        {!smtpConfigured && (
          <div className="mt-3 flex items-center gap-2 text-sm text-amber-600 dark:text-amber-400">
            <TriangleAlert className="h-4 w-4 shrink-0" />
            <span>{t('admin.authentication.emailRegistration.smtpRequired')}</span>
          </div>
        )}
      </div>

      {/* Discord */}
      {discordConfigured && (
        <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
                {t('admin.authentication.discord.title')}
              </h3>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {t('admin.authentication.discord.description')}
              </p>
            </div>
            <Toggle
              enabled={discordEnabled}
              onChange={(val) => handleToggle('auth_discord_enabled', val)}
            />
          </div>
        </div>
      )}

      {/* Google */}
      {googleConfigured && (
        <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
                {t('admin.authentication.google.title')}
              </h3>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {t('admin.authentication.google.description')}
              </p>
            </div>
            <Toggle
              enabled={googleEnabled}
              onChange={(val) => handleToggle('auth_google_enabled', val)}
            />
          </div>
        </div>
      )}
    </div>
  )
}

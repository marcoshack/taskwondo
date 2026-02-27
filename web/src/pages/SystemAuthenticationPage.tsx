import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePublicSettings, useSetSystemSetting, useSMTPConfig, useOAuthConfig, useSetOAuthConfig } from '@/hooks/useSystemSettings'
import type { OAuthProviderConfig } from '@/api/systemSettings'
import { Toggle } from '@/components/ui/Toggle'
import { Input } from '@/components/ui/Input'
import { Spinner } from '@/components/ui/Spinner'
import { ExpandableConfigCard } from '@/components/ui/ExpandableConfigCard'
import { TriangleAlert } from 'lucide-react'

const PASSWORD_MASK = '••••••••'

const emptyConfig: OAuthProviderConfig = {
  client_id: '',
  client_secret: '',
  redirect_uri: '',
}

function OAuthProviderCard({
  provider,
  titleKey,
  descriptionKey,
  enabledSettingKey,
  enabled,
  onToggleEnabled,
}: {
  provider: string
  titleKey: string
  descriptionKey: string
  enabledSettingKey: string
  enabled: boolean
  onToggleEnabled: (key: string, value: boolean) => void
}) {
  const { t } = useTranslation()
  const { data: savedConfig, isLoading } = useOAuthConfig(provider)
  const setConfig = useSetOAuthConfig(provider)
  const [localConfig, setLocalConfig] = useState<OAuthProviderConfig | null>(null)
  const [expanded, setExpanded] = useState(false)
  const [saved, setSaved] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [secretTouched, setSecretTouched] = useState(false)

  const cfg = localConfig ?? savedConfig ?? emptyConfig
  const hasExistingConfig = !!(savedConfig && savedConfig.client_id)

  const updateField = <K extends keyof OAuthProviderConfig>(field: K, value: OAuthProviderConfig[K]) => {
    setLocalConfig((prev) => ({ ...(prev ?? savedConfig ?? emptyConfig), [field]: value }))
    setSaved(false)
    setSaveError('')
  }

  const isDirty = localConfig !== null || secretTouched

  const isFormComplete = () => {
    return (
      cfg.client_id.trim() !== '' &&
      cfg.redirect_uri.trim() !== '' &&
      (cfg.client_secret !== '' || (hasExistingConfig && savedConfig?.client_secret === PASSWORD_MASK && !secretTouched))
    )
  }

  const canSave = isDirty && isFormComplete()

  const handleSave = () => {
    setSaved(false)
    setSaveError('')

    const toSave = { ...cfg }
    if (!secretTouched && hasExistingConfig && savedConfig?.client_secret === PASSWORD_MASK) {
      toSave.client_secret = PASSWORD_MASK
    }

    setConfig.mutate(toSave, {
      onSuccess: () => {
        setSaved(true)
        setLocalConfig(null)
        setSecretTouched(false)
      },
      onError: () => setSaveError(t('admin.authentication.oauth.saveError')),
    })
  }

  const handleCancel = () => {
    setLocalConfig(null)
    setSecretTouched(false)
    setSaved(false)
    setSaveError('')
  }

  if (isLoading) return null

  return (
    <ExpandableConfigCard
      title={t(titleKey)}
      description={t(descriptionKey)}
      enabled={enabled}
      onToggle={(val) => onToggleEnabled(enabledSettingKey, val)}
      expanded={expanded}
      onToggleExpand={() => setExpanded((prev) => !prev)}
      onSave={handleSave}
      onCancel={isDirty ? handleCancel : undefined}
      canSave={canSave}
      saving={setConfig.isPending}
      saved={saved}
      error={saveError}
    >
      {!hasExistingConfig && !localConfig && (
        <p className="text-sm text-amber-600 dark:text-amber-400">
          {t('admin.authentication.oauth.notConfigured')}
        </p>
      )}
      <Input
        label={t('admin.authentication.oauth.clientId')}
        value={cfg.client_id}
        onChange={(e) => updateField('client_id', e.target.value)}
      />
      <Input
        label={t('admin.authentication.oauth.clientSecret')}
        type="password"
        value={secretTouched ? cfg.client_secret : (hasExistingConfig && savedConfig?.client_secret === PASSWORD_MASK ? PASSWORD_MASK : cfg.client_secret)}
        onChange={(e) => {
          setSecretTouched(true)
          updateField('client_secret', e.target.value)
        }}
        onFocus={() => {
          if (!secretTouched && hasExistingConfig && savedConfig?.client_secret === PASSWORD_MASK) {
            setSecretTouched(true)
            updateField('client_secret', '')
          }
        }}
      />
      <Input
        label={t('admin.authentication.oauth.redirectUri')}
        value={cfg.redirect_uri}
        onChange={(e) => updateField('redirect_uri', e.target.value)}
      />
    </ExpandableConfigCard>
  )
}

export function SystemAuthenticationPage() {
  const { t } = useTranslation()
  const { data: publicSettings, isLoading: settingsLoading } = usePublicSettings()
  const { data: smtpConfig, isLoading: smtpLoading } = useSMTPConfig()
  const setSetting = useSetSystemSetting()

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
      <OAuthProviderCard
        provider="discord"
        titleKey="admin.authentication.discord.title"
        descriptionKey="admin.authentication.discord.description"
        enabledSettingKey="auth_discord_enabled"
        enabled={discordEnabled}
        onToggleEnabled={handleToggle}
      />

      {/* Google */}
      <OAuthProviderCard
        provider="google"
        titleKey="admin.authentication.google.title"
        descriptionKey="admin.authentication.google.description"
        enabledSettingKey="auth_google_enabled"
        enabled={googleEnabled}
        onToggleEnabled={handleToggle}
      />
    </div>
  )
}

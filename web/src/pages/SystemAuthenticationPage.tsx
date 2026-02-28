import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePublicSettings, useSetSystemSetting, useSMTPConfig, useOAuthConfig, useSetOAuthConfig } from '@/hooks/useSystemSettings'
import type { OAuthProviderConfig } from '@/api/systemSettings'
import { Toggle } from '@/components/ui/Toggle'
import { Input } from '@/components/ui/Input'
import { Spinner } from '@/components/ui/Spinner'
import { ExpandableConfigCard } from '@/components/ui/ExpandableConfigCard'
import { Copy, Check, TriangleAlert, ArrowUp, ArrowDown } from 'lucide-react'

const PASSWORD_MASK = '••••••••'

const emptyConfig: OAuthProviderConfig = {
  client_id: '',
  client_secret: '',
}

function RedirectUriField({ provider }: { provider: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const redirectUri = `${window.location.origin}/auth/${provider}/callback`

  const handleCopy = () => {
    navigator.clipboard.writeText(redirectUri).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div>
      <label className="block text-sm mb-1">
        <span className="font-medium text-gray-700 dark:text-gray-300">
          {t('admin.authentication.oauth.redirectUri')}
        </span>
        <span className="ml-1.5 font-normal text-xs text-gray-400 dark:text-gray-500">
          ({t('admin.authentication.oauth.redirectUriHint')})
        </span>
      </label>
      <div className="relative">
        <input
          value={redirectUri}
          readOnly
          className="block w-full min-w-0 rounded-md border px-3 py-2 pr-10 text-sm shadow-sm border-gray-300 text-gray-500 bg-gray-50 dark:border-gray-600 dark:bg-gray-800/50 dark:text-gray-400 cursor-default"
        />
        <button
          type="button"
          onClick={handleCopy}
          className="group absolute inset-y-0 right-0 flex items-center px-2.5"
        >
          {copied ? (
            <Check className="h-4 w-4 text-green-500" />
          ) : (
            <Copy className="h-4 w-4 text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-300" />
          )}
          <span className="absolute bottom-full right-0 mb-1.5 hidden group-hover:block whitespace-nowrap rounded bg-gray-900 dark:bg-gray-700 px-2 py-1 text-xs text-white shadow-lg">
            {copied ? t('common.copied') : t('common.copy')}
          </span>
        </button>
      </div>
    </div>
  )
}

interface OAuthProviderDef {
  provider: string
  titleKey: string
  descriptionKey: string
  enabledSettingKey: string
}

const OAUTH_PROVIDERS: OAuthProviderDef[] = [
  {
    provider: 'discord',
    titleKey: 'admin.authentication.discord.title',
    descriptionKey: 'admin.authentication.discord.description',
    enabledSettingKey: 'auth_discord_enabled',
  },
  {
    provider: 'google',
    titleKey: 'admin.authentication.google.title',
    descriptionKey: 'admin.authentication.google.description',
    enabledSettingKey: 'auth_google_enabled',
  },
  {
    provider: 'github',
    titleKey: 'admin.authentication.github.title',
    descriptionKey: 'admin.authentication.github.description',
    enabledSettingKey: 'auth_github_enabled',
  },
]

const DEFAULT_PROVIDER_ORDER = ['discord', 'google', 'github']

function sortProviders(providers: OAuthProviderDef[], order: string[]): OAuthProviderDef[] {
  const orderMap = new Map(order.map((p, i) => [p, i]))
  return [...providers].sort((a, b) => {
    const ai = orderMap.get(a.provider) ?? Infinity
    const bi = orderMap.get(b.provider) ?? Infinity
    return ai - bi
  })
}

function OAuthProviderCard({
  provider,
  titleKey,
  descriptionKey,
  enabledSettingKey,
  enabled,
  onToggleEnabled,
  isFirst,
  isLast,
  onMoveUp,
  onMoveDown,
}: {
  provider: string
  titleKey: string
  descriptionKey: string
  enabledSettingKey: string
  enabled: boolean
  onToggleEnabled: (key: string, value: boolean) => void
  isFirst: boolean
  isLast: boolean
  onMoveUp: () => void
  onMoveDown: () => void
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
      <RedirectUriField provider={provider} />
      <div className="flex items-center gap-2 pt-1">
        <button
          type="button"
          disabled={isFirst}
          onClick={onMoveUp}
          className="group relative rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <ArrowUp className="h-4 w-4" />
          <span className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 hidden group-hover:block whitespace-nowrap rounded bg-gray-900 dark:bg-gray-700 px-2 py-1 text-xs text-white shadow-lg">
            {t('admin.authentication.oauth.changeOrder')}
          </span>
        </button>
        <button
          type="button"
          disabled={isLast}
          onClick={onMoveDown}
          className="group relative rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <ArrowDown className="h-4 w-4" />
          <span className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 hidden group-hover:block whitespace-nowrap rounded bg-gray-900 dark:bg-gray-700 px-2 py-1 text-xs text-white shadow-lg">
            {t('admin.authentication.oauth.changeOrder')}
          </span>
        </button>
      </div>
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

  // SMTP is configured if the config exists and is enabled
  const smtpConfigured = smtpConfig?.enabled === true

  const handleToggle = (key: string, value: boolean) => {
    setSetting.mutate({ key, value })
  }

  // Provider ordering
  const providerOrder = Array.isArray(settings.oauth_provider_order)
    ? settings.oauth_provider_order as string[]
    : DEFAULT_PROVIDER_ORDER
  const sortedProviders = sortProviders(OAUTH_PROVIDERS, providerOrder)

  const enabledMap: Record<string, boolean> = {
    auth_discord_enabled: settings.auth_discord_enabled !== undefined
      ? settings.auth_discord_enabled === true : true,
    auth_google_enabled: settings.auth_google_enabled !== undefined
      ? settings.auth_google_enabled === true : true,
    auth_github_enabled: settings.auth_github_enabled !== undefined
      ? settings.auth_github_enabled === true : true,
  }

  const handleReorder = (index: number, direction: 'up' | 'down') => {
    const currentOrder = sortedProviders.map((p) => p.provider)
    const swapIdx = direction === 'up' ? index - 1 : index + 1
    ;[currentOrder[index], currentOrder[swapIdx]] = [currentOrder[swapIdx], currentOrder[index]]
    setSetting.mutate({ key: 'oauth_provider_order', value: currentOrder })
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

      {/* Email & Password section */}
      <h3 className="text-base font-medium text-gray-700 dark:text-gray-300 pt-2">
        {t('admin.authentication.section.emailPassword')}
      </h3>

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

      {/* OAuth Providers section */}
      <h3 className="text-base font-medium text-gray-700 dark:text-gray-300 pt-2">
        {t('admin.authentication.section.oauth')}
      </h3>
      <p className="text-sm text-gray-500 dark:text-gray-400 -mt-4">
        {t('admin.authentication.section.oauthDescription')}
      </p>

      {sortedProviders.map((def, idx) => (
        <OAuthProviderCard
          key={def.provider}
          provider={def.provider}
          titleKey={def.titleKey}
          descriptionKey={def.descriptionKey}
          enabledSettingKey={def.enabledSettingKey}
          enabled={enabledMap[def.enabledSettingKey]}
          onToggleEnabled={handleToggle}
          isFirst={idx === 0}
          isLast={idx === sortedProviders.length - 1}
          onMoveUp={() => handleReorder(idx, 'up')}
          onMoveDown={() => handleReorder(idx, 'down')}
        />
      ))}
    </div>
  )
}

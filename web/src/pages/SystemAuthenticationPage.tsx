import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePublicSettings, useSetSystemSetting, useSMTPConfig, useOAuthConfig, useSetOAuthConfig } from '@/hooks/useSystemSettings'
import type { OAuthProviderConfig } from '@/api/systemSettings'
import { getOAuthConfigError } from '@/utils/oauthConfigError'
import { Toggle } from '@/components/ui/Toggle'
import { Input } from '@/components/ui/Input'
import { LoadingState } from '@/components/ui/LoadingState'
import { ExpandableConfigCard } from '@/components/ui/ExpandableConfigCard'
import { Copy, Check, TriangleAlert, ArrowUp, ArrowDown } from 'lucide-react'

const PASSWORD_MASK = '••••••••'

const emptyConfig: OAuthProviderConfig = {
  client_id: '',
  client_secret: '',
}

const emptySSOConfig: OAuthProviderConfig = {
  client_id: '',
  client_secret: '',
  issuer: '',
  scopes: [],
  button_label: '',
  disable_pkce: false,
  require_verified_email: true,
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
        <span className="font-medium text-[var(--foreground)]">
          {t('admin.authentication.oauth.redirectUri')}
        </span>
        <span className="ml-1.5 font-normal text-xs text-[var(--foreground-muted)]">
          ({t('admin.authentication.oauth.redirectUriHint')})
        </span>
      </label>
      <div className="relative">
        <input
          value={redirectUri}
          readOnly
          className="block w-full min-w-0 rounded-md border px-3 py-2 pr-10 text-sm border-[var(--border)] text-[var(--foreground-secondary)] bg-[var(--surface-secondary)] dark:border-[var(--border)] bg-[var(--surface)]/50 text-[var(--foreground-muted)] cursor-default"
        />
        <button
          type="button"
          onClick={handleCopy}
          className="group absolute inset-y-0 right-0 flex items-center px-2.5"
        >
          {copied ? (
            <Check className="h-4 w-4 text-green-500" />
          ) : (
            <Copy className="h-4 w-4 text-[var(--foreground-muted)] group-hover:text-[var(--foreground-secondary)] dark:group-hover:text-[var(--foreground-muted)]" />
          )}
          <span className="absolute bottom-full right-0 mb-1.5 hidden group-hover:block whitespace-nowrap rounded bg-[var(--background)] bg-[var(--surface-secondary)] px-2 py-1 text-xs text-white shadow-lg">
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
  /** Extra fields rendered for this provider beyond client id/secret. */
  sso?: boolean
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
  {
    provider: 'sso',
    titleKey: 'admin.authentication.sso.title',
    descriptionKey: 'admin.authentication.sso.description',
    enabledSettingKey: 'auth_sso_enabled',
    sso: true,
  },
]

const DEFAULT_PROVIDER_ORDER = ['discord', 'google', 'github', 'microsoft', 'sso']

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
  sso,
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
  sso: boolean
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

  const cfg = localConfig ?? savedConfig ?? (sso ? emptySSOConfig : emptyConfig)
  // A custom SSO provider is unusable until its issuer can be discovered, so
  // an issuer-less config counts as unconfigured even when credentials exist.
  const hasExistingConfig = !!(savedConfig && savedConfig.client_id && (!sso || savedConfig.issuer))

  const updateField = <K extends keyof OAuthProviderConfig>(field: K, value: OAuthProviderConfig[K]) => {
    setLocalConfig((prev) => ({ ...(prev ?? savedConfig ?? (sso ? emptySSOConfig : emptyConfig)), [field]: value }))
    setSaved(false)
    setSaveError('')
  }

  const isDirty = localConfig !== null || secretTouched

  const hasCredentials =
    cfg.client_id.trim() !== '' &&
    (cfg.client_secret !== '' ||
      (hasExistingConfig && savedConfig?.client_secret === PASSWORD_MASK && !secretTouched))

  const canSave =
    isDirty &&
    hasCredentials &&
    (!sso || (cfg.issuer ?? '').trim() !== '')

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
      onError: (err) => setSaveError(getOAuthConfigError(err, t)),
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
      enabled={hasExistingConfig && enabled}
      onToggle={(val) => onToggleEnabled(enabledSettingKey, val)}
      toggleDisabled={!hasExistingConfig}
      expanded={expanded}
      onToggleExpand={() => setExpanded((prev) => !prev)}
      onSave={handleSave}
      onCancel={isDirty ? handleCancel : undefined}
      canSave={canSave}
      saving={setConfig.isPending}
      saved={saved}
      error={saveError}
      headerExtra={
        <div className="flex items-center gap-1">
          <button
            type="button"
            disabled={isFirst}
            onClick={onMoveUp}
            className="group relative rounded p-0.5 text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)] disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ArrowUp className="h-4 w-4" />
            <span className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 hidden group-hover:block whitespace-nowrap rounded bg-[var(--background)] bg-[var(--surface-secondary)] px-2 py-1 text-xs text-white shadow-lg">
              {t('admin.authentication.oauth.changeOrder')}
            </span>
          </button>
          <button
            type="button"
            disabled={isLast}
            onClick={onMoveDown}
            className="group relative rounded p-0.5 text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)] disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ArrowDown className="h-4 w-4" />
            <span className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 hidden group-hover:block whitespace-nowrap rounded bg-[var(--background)] bg-[var(--surface-secondary)] px-2 py-1 text-xs text-white shadow-lg">
              {t('admin.authentication.oauth.changeOrder')}
            </span>
          </button>
        </div>
      }
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

      {sso && (
        <>
          <div className="border-t border-[var(--border)] my-4" />
          <div>
            <Input
              label={t('admin.authentication.sso.issuer')}
              placeholder={t('admin.authentication.sso.issuerPlaceholder')}
              value={cfg.issuer ?? ''}
              onChange={(e) => updateField('issuer', e.target.value)}
            />
            <p className="text-xs text-[var(--foreground-secondary)] mt-1">
              {t('admin.authentication.sso.issuerHelp')}
            </p>
          </div>
          <div>
            <Input
              label={t('admin.authentication.sso.scopes')}
              placeholder={t('admin.authentication.sso.scopesPlaceholder')}
              value={(cfg.scopes ?? []).join(', ')}
              onChange={(e) => {
                const scopes = e.target.value
                  .split(/[\s,]+/)
                  .map((s) => s.trim())
                  .filter((s) => s.length > 0)
                updateField('scopes', scopes)
              }}
            />
            <p className="text-xs text-[var(--foreground-secondary)] mt-1">
              {t('admin.authentication.sso.scopesHelp')}
            </p>
          </div>
          <div>
            <Input
              label={t('admin.authentication.sso.buttonLabel')}
              placeholder={t('admin.authentication.sso.buttonLabelPlaceholder')}
              value={cfg.button_label ?? ''}
              onChange={(e) => updateField('button_label', e.target.value)}
              maxLength={40}
            />
            <p className="text-xs text-[var(--foreground-secondary)] mt-1">
              {t('admin.authentication.sso.buttonLabelHelp')}
            </p>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <label className="block text-sm font-medium text-[var(--foreground)]">
                {t('admin.authentication.sso.disablePKCE')}
              </label>
              <p className="text-xs text-[var(--foreground-secondary)] mt-1">
                {t('admin.authentication.sso.disablePKCEHelp')}
              </p>
            </div>
            <Toggle
              enabled={cfg.disable_pkce ?? false}
              onChange={(val) => updateField('disable_pkce', val)}
            />
          </div>
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <label className="block text-sm font-medium text-[var(--foreground)]">
                {t('admin.authentication.sso.requireVerifiedEmail')}
              </label>
              <p className="text-xs text-[var(--foreground-secondary)] mt-1">
                {t('admin.authentication.sso.requireVerifiedEmailHelp')}
              </p>
            </div>
            <Toggle
              enabled={cfg.require_verified_email ?? true}
              onChange={(val) => updateField('require_verified_email', val)}
            />
          </div>
        </>
      )}
    </ExpandableConfigCard>
  )
}

export function SystemAuthenticationPage() {
  const { t } = useTranslation()
  const { data: publicSettings, isLoading: settingsLoading } = usePublicSettings()
  const { data: smtpConfig, isLoading: smtpLoading } = useSMTPConfig()
  const setSetting = useSetSystemSetting()

  if (settingsLoading || smtpLoading) {
    return <LoadingState />
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
    auth_discord_enabled: settings.auth_discord_enabled === true,
    auth_google_enabled: settings.auth_google_enabled === true,
    auth_github_enabled: settings.auth_github_enabled === true,
    auth_microsoft_enabled: settings.auth_microsoft_enabled === true,
    auth_sso_enabled: settings.auth_sso_enabled === true,
  }

  const ssoAutoProvisionEnabled = settings.sso_auto_provision_enabled === true

  const handleReorder = (index: number, direction: 'up' | 'down') => {
    const currentOrder = sortedProviders.map((p) => p.provider)
    const swapIdx = direction === 'up' ? index - 1 : index + 1
    ;[currentOrder[index], currentOrder[swapIdx]] = [currentOrder[swapIdx], currentOrder[index]]
    setSetting.mutate({ key: 'oauth_provider_order', value: currentOrder })
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--foreground)]">
          {t('admin.authentication.title')}
        </h2>
        <p className="mt-1 text-sm text-[var(--foreground-secondary)]">
          {t('admin.authentication.description')}
        </p>
      </div>

      {/* Email & Password section */}
      <h3 className="text-base font-medium text-[var(--foreground)] pt-2">
        {t('admin.authentication.section.emailPassword')}
      </h3>

      {/* Email/Password Login */}
      <div className="rounded-lg border border-[var(--border)] p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-[var(--foreground)]">
              {t('admin.authentication.emailLogin.title')}
            </h3>
            <p className="mt-1 text-sm text-[var(--foreground-secondary)]">
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
      <div className="rounded-lg border border-[var(--border)] p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-[var(--foreground)]">
              {t('admin.authentication.emailRegistration.title')}
            </h3>
            <p className="mt-1 text-sm text-[var(--foreground-secondary)]">
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
      <h3 className="text-base font-medium text-[var(--foreground)] pt-2">
        {t('admin.authentication.section.oauth')}
      </h3>
      <p className="text-sm text-[var(--foreground-secondary)] -mt-4">
        {t('admin.authentication.section.oauthDescription')}
      </p>

      {sortedProviders.map((def, idx) => (
        <OAuthProviderCard
          key={def.provider}
          provider={def.provider}
          titleKey={def.titleKey}
          descriptionKey={def.descriptionKey}
          enabledSettingKey={def.enabledSettingKey}
          sso={def.sso ?? false}
          enabled={enabledMap[def.enabledSettingKey]}
          onToggleEnabled={handleToggle}
          isFirst={idx === 0}
          isLast={idx === sortedProviders.length - 1}
          onMoveUp={() => handleReorder(idx, 'up')}
          onMoveDown={() => handleReorder(idx, 'down')}
        />
      ))}

      {/* SSO Auto-provision setting */}
      <div className="rounded-lg border border-[var(--border)] p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-[var(--foreground)]">
              {t('admin.authentication.sso.autoProvision')}
            </h3>
            <p className="mt-1 text-sm text-[var(--foreground-secondary)]">
              {t('admin.authentication.sso.autoProvisionHelp')}
            </p>
          </div>
          <Toggle
            enabled={ssoAutoProvisionEnabled}
            onChange={(val) => handleToggle('sso_auto_provision_enabled', val)}
          />
        </div>
      </div>
    </div>
  )
}

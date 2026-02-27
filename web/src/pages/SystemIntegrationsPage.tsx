import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSMTPConfig, useSetSMTPConfig, useTestSMTP } from '@/hooks/useSystemSettings'
import type { SMTPConfig } from '@/api/systemSettings'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'
import { ExpandableConfigCard } from '@/components/ui/ExpandableConfigCard'

const PASSWORD_MASK = '••••••••'

const defaultConfig: SMTPConfig = {
  enabled: false,
  smtp_host: '',
  smtp_port: 587,
  imap_host: '',
  imap_port: 993,
  username: '',
  password: '',
  encryption: 'starttls',
  from_address: '',
  from_name: '',
}

export function SystemIntegrationsPage() {
  const { t } = useTranslation()
  const { data: savedConfig, isLoading } = useSMTPConfig()
  const setConfigMutation = useSetSMTPConfig()
  const testMutation = useTestSMTP()

  // Local overrides — null means "use server data"
  const [localConfig, setLocalConfig] = useState<SMTPConfig | null>(null)
  const [expanded, setExpanded] = useState(false)
  const [saved, setSaved] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [testSuccess, setTestSuccess] = useState(false)
  const [testError, setTestError] = useState('')
  const [passwordTouched, setPasswordTouched] = useState(false)

  const cfg = localConfig ?? savedConfig ?? defaultConfig

  const updateField = <K extends keyof SMTPConfig>(field: K, value: SMTPConfig[K]) => {
    setLocalConfig((prev) => ({ ...(prev ?? savedConfig ?? defaultConfig), [field]: value }))
    setSaved(false)
    setSaveError('')
  }

  const isDirty = localConfig !== null || passwordTouched

  const isFormComplete = () => {
    if (!cfg.enabled) return true
    return (
      cfg.smtp_host.trim() !== '' &&
      cfg.smtp_port > 0 &&
      cfg.username.trim() !== '' &&
      cfg.from_address.trim() !== '' &&
      !!cfg.encryption &&
      (cfg.password !== '' || (savedConfig?.password === PASSWORD_MASK && !passwordTouched))
    )
  }

  const canSave = isDirty && isFormComplete()

  const handleSave = () => {
    setSaved(false)
    setSaveError('')

    const toSave = { ...cfg }
    if (!passwordTouched && savedConfig?.password === PASSWORD_MASK) {
      toSave.password = PASSWORD_MASK
    }

    setConfigMutation.mutate(toSave, {
      onSuccess: () => {
        setSaved(true)
        setLocalConfig(null)
        setPasswordTouched(false)
      },
      onError: () => setSaveError(t('admin.integrations.smtp.saveError')),
    })
  }

  const handleCancel = () => {
    setLocalConfig(null)
    setPasswordTouched(false)
    setSaved(false)
    setSaveError('')
  }

  const handleTest = () => {
    setTestSuccess(false)
    setTestError('')
    testMutation.mutate(undefined, {
      onSuccess: () => setTestSuccess(true),
      onError: (err) => {
        const msg = err instanceof Error ? err.message : String(err)
        setTestError(t('admin.integrations.smtp.testEmailError', { error: msg }))
      },
    })
  }

  const canTest = cfg.enabled && isFormComplete() && !isDirty

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  const encryptionOptions: { value: SMTPConfig['encryption']; label: string }[] = [
    { value: 'starttls', label: t('admin.integrations.smtp.encryptionStarttls') },
    { value: 'tls', label: t('admin.integrations.smtp.encryptionTls') },
    { value: 'none', label: t('admin.integrations.smtp.encryptionNone') },
  ]

  return (
    <div className="max-w-3xl space-y-6">
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          {t('admin.integrations.title')}
        </h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t('admin.integrations.description')}
        </p>
      </div>

      <ExpandableConfigCard
        title={t('admin.integrations.smtp.title')}
        description={t('admin.integrations.smtp.description')}
        enabled={cfg.enabled}
        onToggle={(val) => updateField('enabled', val)}
        expanded={expanded}
        onToggleExpand={() => setExpanded((prev) => !prev)}
        onSave={handleSave}
        onCancel={isDirty ? handleCancel : undefined}
        canSave={canSave}
        saving={setConfigMutation.isPending}
        saved={saved}
        savedMessage={t('admin.integrations.smtp.saved')}
        error={saveError}
        extraActions={
          <>
            <Button
              variant="secondary"
              onClick={handleTest}
              disabled={!canTest || testMutation.isPending}
            >
              {testMutation.isPending
                ? t('admin.integrations.smtp.testEmailSending')
                : t('admin.integrations.smtp.testEmail')}
            </Button>
            {testSuccess && (
              <span className="text-sm text-green-600 dark:text-green-400">
                {t('admin.integrations.smtp.testEmailSuccess')}
              </span>
            )}
            {testError && (
              <span className="text-sm text-red-600 dark:text-red-400">{testError}</span>
            )}
          </>
        }
      >
        <div className={`space-y-4 ${!cfg.enabled ? 'opacity-50 pointer-events-none' : ''}`}>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Input
              label={t('admin.integrations.smtp.smtpHost')}
              value={cfg.smtp_host}
              onChange={(e) => updateField('smtp_host', e.target.value)}
              placeholder={t('admin.integrations.smtp.smtpHostPlaceholder')}
            />
            <Input
              label={t('admin.integrations.smtp.smtpPort')}
              type="number"
              value={String(cfg.smtp_port)}
              onChange={(e) => updateField('smtp_port', parseInt(e.target.value) || 0)}
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Input
              label={t('admin.integrations.smtp.username')}
              value={cfg.username}
              onChange={(e) => updateField('username', e.target.value)}
              placeholder={t('admin.integrations.smtp.usernamePlaceholder')}
            />
            <Input
              label={t('admin.integrations.smtp.password')}
              type="password"
              value={passwordTouched ? cfg.password : (savedConfig?.password === PASSWORD_MASK ? PASSWORD_MASK : cfg.password)}
              onChange={(e) => {
                setPasswordTouched(true)
                updateField('password', e.target.value)
              }}
              onFocus={() => {
                if (!passwordTouched && savedConfig?.password === PASSWORD_MASK) {
                  setPasswordTouched(true)
                  updateField('password', '')
                }
              }}
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {t('admin.integrations.smtp.encryption')}
            </label>
            <select
              value={cfg.encryption}
              onChange={(e) => updateField('encryption', e.target.value as SMTPConfig['encryption'])}
              className="block w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
            >
              {encryptionOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Input
              label={t('admin.integrations.smtp.fromAddress')}
              type="email"
              value={cfg.from_address}
              onChange={(e) => updateField('from_address', e.target.value)}
              placeholder={t('admin.integrations.smtp.fromAddressPlaceholder')}
            />
            <Input
              label={t('admin.integrations.smtp.fromName')}
              value={cfg.from_name}
              onChange={(e) => updateField('from_name', e.target.value)}
              placeholder={t('admin.integrations.smtp.fromNamePlaceholder')}
            />
          </div>

          {/* IMAP section */}
          <div className="border-t border-gray-200 dark:border-gray-700 pt-4 mt-4">
            <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {t('admin.integrations.smtp.imapSection')}
            </h4>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
              {t('admin.integrations.smtp.imapHelp')}
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Input
                label={t('admin.integrations.smtp.imapHost')}
                value={cfg.imap_host}
                onChange={(e) => updateField('imap_host', e.target.value)}
                placeholder={t('admin.integrations.smtp.imapHostPlaceholder')}
              />
              <Input
                label={t('admin.integrations.smtp.imapPort')}
                type="number"
                value={String(cfg.imap_port)}
                onChange={(e) => updateField('imap_port', parseInt(e.target.value) || 0)}
              />
            </div>
          </div>
        </div>
      </ExpandableConfigCard>
    </div>
  )
}

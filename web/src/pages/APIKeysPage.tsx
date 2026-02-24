import { useState } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import { Trash2, Key, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { CopyButton } from '@/components/ui/CopyButton'
import { useAPIKeys, useCreateAPIKey, useDeleteAPIKey } from '@/hooks/useAPIKeys'
import type { APIKey, CreatedAPIKey } from '@/api/auth'

const EXPIRATION_OPTIONS = [
  { value: '7', days: 7, label: 'preferences.apiKeys.expires7d' },
  { value: '30', days: 30, label: 'preferences.apiKeys.expires30d' },
  { value: '60', days: 60, label: 'preferences.apiKeys.expires60d' },
  { value: '90', days: 90, label: 'preferences.apiKeys.expires90d' },
  { value: '365', days: 365, label: 'preferences.apiKeys.expires1y' },
  { value: 'never', days: 0, label: 'preferences.apiKeys.expiresNever' },
] as const

const PERMISSION_OPTIONS = [
  { value: '', label: 'preferences.apiKeys.permFullAccess' },
  { value: 'read', label: 'preferences.apiKeys.permReadOnly' },
  { value: 'read,write', label: 'preferences.apiKeys.permReadWrite' },
] as const

export function APIKeysPage() {
  const { t } = useTranslation()
  const { data: keys, isLoading } = useAPIKeys()
  const createMutation = useCreateAPIKey()
  const deleteMutation = useDeleteAPIKey()

  const [name, setName] = useState('')
  const [permission, setPermission] = useState('')
  const [expiration, setExpiration] = useState('30')
  const [error, setError] = useState('')
  const [createdKey, setCreatedKey] = useState<CreatedAPIKey | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<APIKey | null>(null)

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!name.trim()) {
      setError(t('preferences.apiKeys.nameRequired'))
      return
    }

    const permissions = permission ? permission.split(',') : []
    const expiresAt = expiration !== 'never'
      ? new Date(Date.now() + parseInt(expiration) * 86400000).toISOString()
      : undefined

    try {
      const result = await createMutation.mutateAsync({ name: name.trim(), permissions, expiresAt })
      setCreatedKey(result)
      setName('')
      setPermission('')
      setExpiration('30')
    } catch {
      setError(t('common.error'))
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      setDeleteTarget(null)
    } catch {
      // error handled by mutation
    }
  }

  function formatExpiration(key: APIKey): { text: string; expired: boolean } {
    if (!key.expires_at) return { text: t('preferences.apiKeys.noExpiration'), expired: false }
    const date = new Date(key.expires_at)
    if (date < new Date()) return { text: t('preferences.apiKeys.expired'), expired: true }
    return { text: date.toLocaleDateString(), expired: false }
  }

  function permissionLabel(perms: string[]): string {
    if (!perms || perms.length === 0) return t('preferences.apiKeys.permFullAccess')
    if (perms.includes('write')) return t('preferences.apiKeys.permReadWrite')
    if (perms.includes('read')) return t('preferences.apiKeys.permReadOnly')
    return perms.join(', ')
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{t('preferences.apiKeys.title')}</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('preferences.apiKeys.description')}</p>
      </div>

      {/* Key Reveal Card */}
      {createdKey && (
        <div className="mb-6 rounded-lg border-2 border-amber-400 bg-amber-50 dark:bg-amber-900/20 dark:border-amber-600 p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="font-medium text-amber-800 dark:text-amber-200 mb-1">{t('preferences.apiKeys.keyCreated')}</p>
              <p className="text-sm text-amber-700 dark:text-amber-300 mb-3">{t('preferences.apiKeys.keyWarning')}</p>
              <div className="flex items-center gap-2 bg-white dark:bg-gray-800 rounded-md border border-amber-300 dark:border-amber-700 px-3 py-2">
                <code className="text-sm font-mono text-gray-900 dark:text-gray-100 break-all flex-1">{createdKey.key}</code>
                <CopyButton text={createdKey.key} />
              </div>
            </div>
          </div>
          <div className="flex justify-end mt-3">
            <Button size="sm" onClick={() => setCreatedKey(null)}>{t('preferences.apiKeys.dismiss')}</Button>
          </div>
        </div>
      )}

      {/* Create Form */}
      <div className="mb-8 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4">
        <h2 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">{t('preferences.apiKeys.createNew')}</h2>
        <form onSubmit={handleCreate} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">{t('preferences.apiKeys.name')}</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('preferences.apiKeys.namePlaceholder')}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">{t('preferences.apiKeys.permissions')}</label>
              <select
                value={permission}
                onChange={(e) => setPermission(e.target.value)}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              >
                {PERMISSION_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{t(opt.label)}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">{t('preferences.apiKeys.expiration')}</label>
              <select
                value={expiration}
                onChange={(e) => setExpiration(e.target.value)}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              >
                {EXPIRATION_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{t(opt.label)}</option>
                ))}
              </select>
            </div>
          </div>
          {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? <Spinner className="h-4 w-4" /> : t('preferences.apiKeys.create')}
          </Button>
        </form>
      </div>

      {/* Key List */}
      {isLoading ? (
        <div className="flex justify-center py-8"><Spinner /></div>
      ) : !keys || keys.length === 0 ? (
        <div className="text-center py-8 text-gray-500 dark:text-gray-400">
          <Key className="h-8 w-8 mx-auto mb-2 opacity-40" />
          <p>{t('preferences.apiKeys.empty')}</p>
        </div>
      ) : (
        <div className="rounded-lg border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
          {keys.map((key) => {
            const exp = formatExpiration(key)
            return (
              <div key={key.id} className="p-4 bg-white dark:bg-gray-800 first:rounded-t-lg last:rounded-b-lg">
                <div className="flex items-center justify-between">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-medium text-gray-900 dark:text-gray-100">{key.name}</span>
                      <Badge color="gray">{permissionLabel(key.permissions)}</Badge>
                      {exp.expired && <Badge color="red">{exp.text}</Badge>}
                    </div>
                    <div className="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
                      <span><code>{key.key_prefix}...</code></span>
                      <span>{t('preferences.apiKeys.createdAt')}: {new Date(key.created_at).toLocaleDateString()}</span>
                      <span>{t('preferences.apiKeys.lastUsed')}: {key.last_used_at ? new Date(key.last_used_at).toLocaleDateString() : t('preferences.apiKeys.lastUsedNever')}</span>
                      {!exp.expired && key.expires_at && <span>{t('preferences.apiKeys.expiresAt')}: {exp.text}</span>}
                      {!key.expires_at && <span>{t('preferences.apiKeys.expiresAt')}: {t('preferences.apiKeys.noExpiration')}</span>}
                    </div>
                  </div>
                  <button
                    type="button"
                    className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                    onClick={() => setDeleteTarget(key)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Delete Confirmation Modal */}
      <Modal
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title={t('preferences.apiKeys.deleteConfirmTitle')}
      >
        <div className="p-4">
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            <Trans
              i18nKey="preferences.apiKeys.deleteConfirmBody"
              values={{ name: deleteTarget?.name }}
              components={{ bold: <strong className="font-semibold text-gray-900 dark:text-gray-100" /> }}
            />
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setDeleteTarget(null)}>{t('common.cancel')}</Button>
            <Button variant="danger" onClick={handleDelete} disabled={deleteMutation.isPending}>
              {deleteMutation.isPending ? <Spinner className="h-4 w-4" /> : t('common.delete')}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

import { useState } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import { Trash2, Key, AlertTriangle, Pencil, Check, X } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { CopyButton } from '@/components/ui/CopyButton'
import {
  useSystemAPIKeys,
  useCreateSystemAPIKey,
  useRenameSystemAPIKey,
  useDeleteSystemAPIKey,
} from '@/hooks/useSystemAPIKeys'
import type { SystemAPIKey, CreatedSystemAPIKey } from '@/api/auth'

const EXPIRATION_OPTIONS = [
  { value: '7', days: 7, label: 'admin.apiKeys.expires7d' },
  { value: '30', days: 30, label: 'admin.apiKeys.expires30d' },
  { value: '60', days: 60, label: 'admin.apiKeys.expires60d' },
  { value: '90', days: 90, label: 'admin.apiKeys.expires90d' },
  { value: '365', days: 365, label: 'admin.apiKeys.expires1y' },
  { value: 'never', days: 0, label: 'admin.apiKeys.expiresNever' },
] as const

interface ResourcePermission {
  resource: string
  labelKey: string
  accessOptions: { value: string; labelKey: string }[]
}

const RESOURCE_PERMISSIONS: ResourcePermission[] = [
  {
    resource: 'metrics',
    labelKey: 'admin.apiKeys.resourceMetrics',
    accessOptions: [
      { value: 'r', labelKey: 'admin.apiKeys.accessRead' },
    ],
  },
  {
    resource: 'items',
    labelKey: 'admin.apiKeys.resourceItems',
    accessOptions: [
      { value: 'r', labelKey: 'admin.apiKeys.accessRead' },
      { value: 'w', labelKey: 'admin.apiKeys.accessWrite' },
      { value: 'rw', labelKey: 'admin.apiKeys.accessReadWrite' },
    ],
  },
]

export function SystemAPIKeysPage() {
  const { t } = useTranslation()
  const { data: keys, isLoading } = useSystemAPIKeys()
  const createMutation = useCreateSystemAPIKey()
  const renameMutation = useRenameSystemAPIKey()
  const deleteMutation = useDeleteSystemAPIKey()

  const [name, setName] = useState('')
  const [expiration, setExpiration] = useState('30')
  const [error, setError] = useState('')
  const [createdKey, setCreatedKey] = useState<CreatedSystemAPIKey | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<SystemAPIKey | null>(null)
  const [editingKeyId, setEditingKeyId] = useState<string | null>(null)
  const [editingName, setEditingName] = useState('')
  const [savedId, setSavedId] = useState<string | null>(null)

  // Resource permission state: { "metrics": "r", "items": "rw" }
  const [resourcePerms, setResourcePerms] = useState<Record<string, string>>({})

  function toggleResourcePerm(resource: string, access: string) {
    setResourcePerms((prev) => {
      const current = prev[resource]
      if (current === access) {
        // Deselect
        const next = { ...prev }
        delete next[resource]
        return next
      }
      return { ...prev, [resource]: access }
    })
  }

  function buildPermissions(): string[] {
    return Object.entries(resourcePerms).map(([resource, access]) => `${resource}:${access}`)
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!name.trim()) {
      setError(t('admin.apiKeys.nameRequired'))
      return
    }

    const permissions = buildPermissions()
    if (permissions.length === 0) {
      setError(t('admin.apiKeys.permissions') + ' required')
      return
    }

    const expiresAt =
      expiration !== 'never'
        ? new Date(Date.now() + parseInt(expiration) * 86400000).toISOString()
        : undefined

    try {
      const result = await createMutation.mutateAsync({
        name: name.trim(),
        permissions,
        expiresAt,
      })
      setCreatedKey(result)
      setName('')
      setExpiration('30')
      setResourcePerms({})
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

  async function handleRename(keyId: string) {
    const trimmed = editingName.trim()
    if (!trimmed) return
    try {
      await renameMutation.mutateAsync({ id: keyId, name: trimmed })
      setEditingKeyId(null)
      setEditingName('')
      setSavedId(keyId)
      setTimeout(() => setSavedId(null), 2000)
    } catch {
      // error handled by mutation
    }
  }

  function formatExpiration(key: SystemAPIKey): { text: string; expired: boolean } {
    if (!key.expires_at) return { text: t('admin.apiKeys.noExpiration'), expired: false }
    const date = new Date(key.expires_at)
    if (date < new Date()) return { text: t('admin.apiKeys.expired'), expired: true }
    return { text: date.toLocaleDateString(), expired: false }
  }

  function permissionLabels(perms: string[]): string {
    return perms
      .map((p) => {
        const [resource, access] = p.split(':')
        const res = RESOURCE_PERMISSIONS.find((r) => r.resource === resource)
        const acc = res?.accessOptions.find((a) => a.value === access)
        const resLabel = res ? t(res.labelKey) : resource
        const accLabel = acc ? t(acc.labelKey) : access
        return `${resLabel}: ${accLabel}`
      })
      .join(', ')
  }

  return (
    <div className="max-w-3xl">
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          {t('admin.apiKeys.title')}
        </h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {t('admin.apiKeys.description')}
        </p>
      </div>

      {/* Key Reveal Card */}
      {createdKey && (
        <div className="mb-6 rounded-lg border-2 border-amber-400 bg-amber-50 dark:bg-amber-900/20 dark:border-amber-600 p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="font-medium text-amber-800 dark:text-amber-200 mb-1">
                {t('admin.apiKeys.keyCreated')}
              </p>
              <p className="text-sm text-amber-700 dark:text-amber-300 mb-3">
                {t('admin.apiKeys.keyWarning')}
              </p>
              <div className="flex items-center gap-2 bg-white dark:bg-gray-800 rounded-md border border-amber-300 dark:border-amber-700 px-3 py-2">
                <code className="text-sm font-mono text-gray-900 dark:text-gray-100 break-all flex-1">
                  {createdKey.key}
                </code>
                <CopyButton text={createdKey.key} />
              </div>
            </div>
          </div>
          <div className="flex justify-end mt-3">
            <Button size="sm" onClick={() => setCreatedKey(null)}>
              {t('admin.apiKeys.dismiss')}
            </Button>
          </div>
        </div>
      )}

      {/* Create Form */}
      <div className="mb-8 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4">
        <h2 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
          {t('admin.apiKeys.createNew')}
        </h2>
        <form onSubmit={handleCreate} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                {t('admin.apiKeys.name')}
              </label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t('admin.apiKeys.namePlaceholder')}
              />
            </div>
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                {t('admin.apiKeys.expiration')}
              </label>
              <select
                value={expiration}
                onChange={(e) => setExpiration(e.target.value)}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              >
                {EXPIRATION_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {t(opt.label)}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Resource Permission Checkboxes */}
          <div>
            <label className="block text-sm text-gray-600 dark:text-gray-400 mb-2">
              {t('admin.apiKeys.permissions')}
            </label>
            <div className="space-y-3">
              {RESOURCE_PERMISSIONS.map((rp) => (
                <div
                  key={rp.resource}
                  data-testid={`resource-${rp.resource}`}
                  className="flex items-center gap-4 rounded-md border border-gray-200 dark:border-gray-700 px-3 py-2"
                >
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300 w-28">
                    {t(rp.labelKey)}
                  </span>
                  <div className="flex gap-2">
                    {rp.accessOptions.map((opt) => {
                      const isSelected = resourcePerms[rp.resource] === opt.value
                      return (
                        <button
                          key={opt.value}
                          type="button"
                          onClick={() => toggleResourcePerm(rp.resource, opt.value)}
                          className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${
                            isSelected
                              ? 'bg-indigo-100 text-indigo-700 ring-1 ring-indigo-500 dark:bg-indigo-900/40 dark:text-indigo-300 dark:ring-indigo-400'
                              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-400 dark:hover:bg-gray-600'
                          }`}
                        >
                          {t(opt.labelKey)}
                        </button>
                      )
                    })}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? (
              <Spinner className="h-4 w-4" />
            ) : (
              t('admin.apiKeys.create')
            )}
          </Button>
        </form>
      </div>

      {/* Key List */}
      {isLoading ? (
        <div className="flex justify-center py-8">
          <Spinner />
        </div>
      ) : !keys || keys.length === 0 ? (
        <div className="text-center py-8 text-gray-500 dark:text-gray-400">
          <Key className="h-8 w-8 mx-auto mb-2 opacity-40" />
          <p>{t('admin.apiKeys.empty')}</p>
        </div>
      ) : (
        <div className="rounded-lg border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
          {keys.map((key) => {
            const exp = formatExpiration(key)
            const isEditing = editingKeyId === key.id
            return (
              <div
                key={key.id}
                className="p-4 bg-white dark:bg-gray-800 first:rounded-t-lg last:rounded-b-lg"
              >
                <div className="flex items-center justify-between">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      {isEditing ? (
                        <form
                          className="flex items-center gap-1"
                          onSubmit={(e) => {
                            e.preventDefault()
                            handleRename(key.id)
                          }}
                        >
                          <Input
                            value={editingName}
                            onChange={(e) => setEditingName(e.target.value)}
                            className="h-7 text-sm w-48"
                            autoFocus
                            onKeyDown={(e) => {
                              if (e.key === 'Escape') {
                                setEditingKeyId(null)
                                setEditingName('')
                              }
                            }}
                          />
                          <button
                            type="submit"
                            className="p-1 text-gray-400 hover:text-green-600 dark:hover:text-green-400 transition-colors"
                            disabled={renameMutation.isPending}
                          >
                            <Check className="h-4 w-4" />
                          </button>
                          <button
                            type="button"
                            className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                            onClick={() => {
                              setEditingKeyId(null)
                              setEditingName('')
                            }}
                          >
                            <X className="h-4 w-4" />
                          </button>
                        </form>
                      ) : (
                        <>
                          <span className="font-medium text-gray-900 dark:text-gray-100">
                            {key.name}
                          </span>
                          {savedId === key.id && <Check className="h-4 w-4 text-green-500" />}
                        </>
                      )}
                      <Badge color="blue">{permissionLabels(key.permissions)}</Badge>
                      {exp.expired && <Badge color="red">{exp.text}</Badge>}
                    </div>
                    <div className="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
                      <span>
                        <code>{key.key_prefix}...</code>
                      </span>
                      <span>
                        {t('admin.apiKeys.createdAt')}:{' '}
                        {new Date(key.created_at).toLocaleDateString()}
                      </span>
                      <span>
                        {t('admin.apiKeys.lastUsed')}:{' '}
                        {key.last_used_at
                          ? new Date(key.last_used_at).toLocaleDateString()
                          : t('admin.apiKeys.lastUsedNever')}
                      </span>
                      {!exp.expired && key.expires_at && (
                        <span>
                          {t('admin.apiKeys.expiresAt')}: {exp.text}
                        </span>
                      )}
                      {!key.expires_at && (
                        <span>
                          {t('admin.apiKeys.expiresAt')}: {t('admin.apiKeys.noExpiration')}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    {!isEditing && (
                      <button
                        type="button"
                        className="p-1.5 text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors"
                        onClick={() => {
                          setEditingKeyId(key.id)
                          setEditingName(key.name)
                        }}
                        title={t('admin.apiKeys.rename')}
                      >
                        <Pencil className="h-4 w-4" />
                      </button>
                    )}
                    <button
                      type="button"
                      className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                      onClick={() => setDeleteTarget(key)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
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
        title={t('admin.apiKeys.deleteConfirmTitle')}
      >
        <div className="p-4">
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            <Trans
              i18nKey="admin.apiKeys.deleteConfirmBody"
              values={{ name: deleteTarget?.name }}
              components={{
                bold: (
                  <strong className="font-semibold text-gray-900 dark:text-gray-100" />
                ),
              }}
            />
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setDeleteTarget(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="danger"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? (
                <Spinner className="h-4 w-4" />
              ) : (
                t('common.delete')
              )}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

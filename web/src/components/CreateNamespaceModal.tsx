import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { useCreateNamespace, useOwnedNamespaceCount, useMaxNamespaces } from '@/hooks/useNamespaces'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import type { Namespace } from '@/api/namespaces'

interface Props {
  open: boolean
  onClose: () => void
  onCreated: (ns: Namespace) => void
}

export function CreateNamespaceModal({ open, onClose, onCreated }: Props) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const createMutation = useCreateNamespace()
  const { data: ownedCount } = useOwnedNamespaceCount()
  const { data: maxNamespaces } = useMaxNamespaces()

  const isAdmin = user?.global_role === 'admin'
  const limit = maxNamespaces ?? 1
  const count = ownedCount ?? 0
  const atLimit = !isAdmin && limit > 0 && count >= limit

  const [displayName, setDisplayName] = useState('')
  const [slug, setSlug] = useState('')
  const [formError, setFormError] = useState('')

  function handleClose() {
    setDisplayName('')
    setSlug('')
    setFormError('')
    onClose()
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!displayName.trim() || !slug.trim()) return

    createMutation.mutate(
      { display_name: displayName.trim(), slug: slug.trim() },
      {
        onSuccess: (created) => {
          setDisplayName('')
          setSlug('')
          setFormError('')
          onCreated(created)
        },
        onError: (err) => {
          if (err && typeof err === 'object' && 'response' in err) {
            const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
            setFormError(axiosErr.response?.data?.error?.message ?? t('namespaces.createError'))
          } else {
            setFormError(t('namespaces.createError'))
          }
        },
      },
    )
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={
        <span className="flex items-center gap-3">
          {t('namespaces.createTitle')}
          {!isAdmin && limit > 0 && (
            <span className={`text-sm font-normal ${atLimit ? 'text-amber-500 dark:text-amber-400' : 'text-gray-500 dark:text-gray-400'}`}>
              {t('namespaces.limitCounter', { count, limit })}
            </span>
          )}
        </span>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {atLimit && (
          <p className="text-sm text-amber-600 dark:text-amber-400">{t('namespaces.limitReached')}</p>
        )}
        <p className="text-sm text-gray-500 dark:text-gray-400">{t('namespaces.createDescription')}</p>
        <Input
          label={t('namespaces.displayName')}
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          placeholder={t('namespaces.displayNamePlaceholder')}
          required
          disabled={atLimit}
        />
        <Input
          label={t('namespaces.slug')}
          value={slug}
          onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
          placeholder={t('namespaces.slugPlaceholder')}
          maxLength={30}
          required
          disabled={atLimit}
        />
        <p className="text-xs text-gray-400 dark:text-gray-500 -mt-3">{t('namespaces.slugHint')}</p>
        {formError && <p className="text-sm text-red-600 dark:text-red-400">{formError}</p>}
        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" disabled={atLimit || createMutation.isPending || !displayName.trim() || !slug.trim()}>
            {createMutation.isPending ? t('common.creating') : t('common.create')}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

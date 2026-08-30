import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { Tooltip } from '@/components/ui/Tooltip'
import type { SavedSearch } from '@/api/savedSearches'

interface SaveSearchModalProps {
  open: boolean
  onClose: () => void
  onSaveNew: (name: string, shared: boolean) => void
  onUpdateExisting: () => void
  activeSearch: SavedSearch | null
  hasUnsavedChanges: boolean
  canManageShared: boolean
}

export function SaveSearchModal({
  open,
  onClose,
  onSaveNew,
  onUpdateExisting,
  activeSearch,
  hasUnsavedChanges,
  canManageShared,
}: SaveSearchModalProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [shared, setShared] = useState(false)

  function handleClose() {
    setName('')
    setShared(false)
    onClose()
  }

  function handleSaveNew() {
    if (!name.trim()) return
    onSaveNew(name.trim(), shared)
    setName('')
    setShared(false)
  }

  function handleUpdate() {
    onUpdateExisting()
  }

  const showUpdateOption = activeSearch && hasUnsavedChanges
  const canUpdateActive = activeSearch?.scope === 'shared' ? canManageShared : true

  return (
    <Modal open={open} onClose={handleClose} title={t('savedSearches.save')}>
      <div className="space-y-4">
        {showUpdateOption && (
          <>
            <Tooltip content={!canUpdateActive ? t('savedSearches.updateSharedAdminOnly') : undefined}>
              <Button onClick={handleUpdate} className="w-full" disabled={!canUpdateActive}>
                {t('savedSearches.updateExisting', { name: activeSearch.name })}
              </Button>
            </Tooltip>
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-[var(--border)]" />
              </div>
              <div className="relative flex justify-center text-xs">
                <span className="bg-[var(--surface)] px-2 text-[var(--foreground-secondary)] uppercase">
                  {t('common.or')}
                </span>
              </div>
            </div>
          </>
        )}

        <div className="space-y-3">
          {showUpdateOption && (
            <p className="text-sm font-medium text-[var(--foreground)]">
              {t('savedSearches.saveAsNew')}
            </p>
          )}
          <div>
            <label className="block text-sm font-medium text-[var(--foreground)] mb-1">
              {t('savedSearches.nameLabel')}
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('savedSearches.namePlaceholder')}
              onKeyDown={(e) => { if (e.key === 'Enter') handleSaveNew() }}
              autoFocus={!showUpdateOption}
            />
          </div>

          {canManageShared && (
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={shared}
                onChange={(e) => setShared(e.target.checked)}
                className="rounded border-[var(--border)] text-[var(--primary)] focus:ring-[var(--focus-ring)] dark:border-[var(--border)] bg-[var(--surface-secondary)]"
              />
              <span className="text-sm text-[var(--foreground)]">
                {t('savedSearches.sharedToggle')}
              </span>
            </label>
          )}

          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={handleClose}>{t('common.cancel')}</Button>
            <Button onClick={handleSaveNew} disabled={!name.trim()}>
              {showUpdateOption ? t('savedSearches.saveNew') : t('savedSearches.save')}
            </Button>
          </div>
        </div>
      </div>
    </Modal>
  )
}

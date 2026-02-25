import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
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

  return (
    <Modal open={open} onClose={handleClose} title={t('savedSearches.save')}>
      <div className="space-y-4">
        {showUpdateOption && (
          <>
            <Button onClick={handleUpdate} className="w-full">
              {t('savedSearches.updateExisting', { name: activeSearch.name })}
            </Button>
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-gray-200 dark:border-gray-700" />
              </div>
              <div className="relative flex justify-center text-xs">
                <span className="bg-white dark:bg-gray-800 px-2 text-gray-500 dark:text-gray-400 uppercase">
                  {t('common.or')}
                </span>
              </div>
            </div>
          </>
        )}

        <div className="space-y-3">
          {showUpdateOption && (
            <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('savedSearches.saveAsNew')}
            </p>
          )}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
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
                className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">
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

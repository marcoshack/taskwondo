import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Trans } from 'react-i18next'
import { ChevronDown, Pencil, Trash2, Search, FolderSearch } from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import type { SavedSearch } from '@/api/savedSearches'

interface SavedSearchSelectorProps {
  searches: SavedSearch[]
  activeSearchId: string | null
  hasUnsavedChanges: boolean
  onSelect: (search: SavedSearch) => void
  onRename: (search: SavedSearch, newName: string) => void
  onDelete: (search: SavedSearch) => void
  canManageShared: boolean
  variant?: 'desktop' | 'mobile'
}

export function SavedSearchSelector({
  searches,
  activeSearchId,
  hasUnsavedChanges,
  onSelect,
  onRename,
  onDelete,
  canManageShared,
  variant = 'desktop',
}: SavedSearchSelectorProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [filterText, setFilterText] = useState('')
  const [renaming, setRenaming] = useState<SavedSearch | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [deleting, setDeleting] = useState<SavedSearch | null>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const [dropdownTop, setDropdownTop] = useState(0)

  const activeSearch = searches.find((s) => s.id === activeSearchId) ?? null

  // Close on outside click
  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  const userSearches = searches.filter((s) => s.scope === 'user')
  const sharedSearches = searches.filter((s) => s.scope === 'shared')

  const filtered = (list: SavedSearch[]) =>
    filterText ? list.filter((s) => s.name.toLowerCase().includes(filterText.toLowerCase())) : list

  const filteredUser = filtered(userSearches)
  const filteredShared = filtered(sharedSearches)

  function canModify(search: SavedSearch) {
    return search.scope === 'user' || canManageShared
  }

  function handleRenameSubmit() {
    if (!renaming || !renameValue.trim()) return
    onRename(renaming, renameValue.trim())
    setRenaming(null)
  }

  function handleDeleteConfirm() {
    if (!deleting) return
    onDelete(deleting)
    setDeleting(null)
  }

  const buttonLabel = activeSearch ? activeSearch.name : t('savedSearches.placeholder')

  function handleSelect(s: SavedSearch) {
    onSelect(s)
    setOpen(false)
    setFilterText('')
  }

  const searchListContent = (
    <>
      <div className={variant === 'mobile' ? 'px-0 pb-2' : 'p-2 border-b border-gray-200 dark:border-gray-700'}>
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-gray-400" />
          <input
            type="text"
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            placeholder={t('savedSearches.searchPlaceholder')}
            className="w-full pl-8 pr-3 py-1.5 text-sm rounded border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            autoFocus
          />
        </div>
      </div>

      <div className="max-h-64 overflow-y-auto py-1">
        {filteredUser.length === 0 && filteredShared.length === 0 && (
          <p className="px-3 py-4 text-sm text-gray-500 dark:text-gray-400 text-center">
            {t('savedSearches.empty')}
          </p>
        )}

        {filteredUser.length > 0 && (
          <>
            <p className="px-3 py-1 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('savedSearches.mySearches')}
            </p>
            {filteredUser.map((s) => (
              <SearchEntry
                key={s.id}
                search={s}
                isActive={s.id === activeSearchId}
                canModify={canModify(s)}
                onSelect={() => handleSelect(s)}
                onRename={() => { setRenaming(s); setRenameValue(s.name) }}
                onDelete={() => setDeleting(s)}
              />
            ))}
          </>
        )}

        {filteredShared.length > 0 && (
          <>
            {filteredUser.length > 0 && <div className="my-1 border-t border-gray-200 dark:border-gray-700" />}
            <p className="px-3 py-1 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('savedSearches.shared')}
            </p>
            {filteredShared.map((s) => (
              <SearchEntry
                key={s.id}
                search={s}
                isActive={s.id === activeSearchId}
                canModify={canModify(s)}
                onSelect={() => handleSelect(s)}
                onRename={() => { setRenaming(s); setRenameValue(s.name) }}
                onDelete={() => setDeleting(s)}
              />
            ))}
          </>
        )}
      </div>
    </>
  )

  return (
    <>
      <div className="relative" ref={dropdownRef}>
        {variant === 'mobile' ? (
          <button
            ref={buttonRef}
            onClick={() => {
              if (!open && buttonRef.current) {
                const rect = buttonRef.current.getBoundingClientRect()
                setDropdownTop(rect.bottom + 4)
              }
              setOpen(!open)
            }}
            className="relative shrink-0 p-2.5 rounded-md border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
            aria-label={t('savedSearches.placeholder')}
          >
            <FolderSearch className="h-5 w-5" />
            {activeSearch && (
              <span className="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-indigo-600 text-[10px] font-bold text-white">
                1
              </span>
            )}
          </button>
        ) : (
          <button
            onClick={() => setOpen(!open)}
            className={`flex items-center gap-1.5 px-3 h-[39px] text-sm font-medium rounded-md border transition-colors ${
              activeSearch
                ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700'
                : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-600 dark:hover:bg-gray-700'
            }`}
          >
            <span className="truncate max-w-[160px]">{buttonLabel}</span>
            {hasUnsavedChanges && activeSearch && (
              <span className="flex h-2 w-2 rounded-full bg-amber-500 shrink-0" />
            )}
            <ChevronDown className="h-3.5 w-3.5 shrink-0" />
          </button>
        )}

        {open && (
          <div
            className={`z-50 rounded-md border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800 ${variant === 'mobile' ? 'fixed left-4 right-4' : 'absolute left-0 top-full mt-1 w-72'}`}
            style={variant === 'mobile' ? { top: dropdownTop } : undefined}
          >
            {searchListContent}
          </div>
        )}
      </div>

      {/* Rename modal */}
      <Modal open={!!renaming} onClose={() => setRenaming(null)} title={t('savedSearches.renameTitle')}>
        <div className="space-y-4">
          <Input
            value={renameValue}
            onChange={(e) => setRenameValue(e.target.value)}
            placeholder={t('savedSearches.namePlaceholder')}
            onKeyDown={(e) => { if (e.key === 'Enter') handleRenameSubmit() }}
            autoFocus
          />
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setRenaming(null)}>{t('common.cancel')}</Button>
            <Button onClick={handleRenameSubmit} disabled={!renameValue.trim()}>{t('common.save')}</Button>
          </div>
        </div>
      </Modal>

      {/* Delete confirmation modal */}
      <Modal open={!!deleting} onClose={() => setDeleting(null)} title={t('savedSearches.deleteConfirmTitle')}>
        <div className="space-y-4">
          <p className="text-sm text-gray-700 dark:text-gray-300">
            <Trans i18nKey="savedSearches.deleteConfirmBody" values={{ name: deleting?.name }} components={{ strong: <strong /> }} />
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setDeleting(null)}>{t('common.cancel')}</Button>
            <Button variant="danger" onClick={handleDeleteConfirm}>{t('common.delete')}</Button>
          </div>
        </div>
      </Modal>
    </>
  )
}

function SearchEntry({
  search,
  isActive,
  canModify,
  onSelect,
  onRename,
  onDelete,
}: {
  search: SavedSearch
  isActive: boolean
  canModify: boolean
  onSelect: () => void
  onRename: () => void
  onDelete: () => void
}) {
  return (
    <div
      className={`group flex items-center gap-1 px-3 py-1.5 cursor-pointer ${
        isActive
          ? 'bg-indigo-50 dark:bg-indigo-900/20'
          : 'hover:bg-gray-100 dark:hover:bg-gray-700'
      }`}
    >
      <button
        onClick={onSelect}
        className={`flex-1 text-left text-sm truncate ${
          isActive ? 'text-indigo-700 dark:text-indigo-300 font-medium' : 'text-gray-700 dark:text-gray-300'
        }`}
      >
        {search.name}
      </button>
      {canModify && (
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
          <button
            onClick={(e) => { e.stopPropagation(); onRename() }}
            className="p-1 rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            <Pencil className="h-3 w-3" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); onDelete() }}
            className="p-1 rounded text-gray-400 hover:text-red-500 dark:hover:text-red-400"
          >
            <Trash2 className="h-3 w-3" />
          </button>
        </div>
      )}
    </div>
  )
}

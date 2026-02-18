import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useDebounce } from '@/hooks/useDebounce'
import { useSearchUsers } from '@/hooks/useUsers'
import type { UserSearchResult } from '@/api/auth'

interface UserSearchInputProps {
  excludeUserIds: string[]
  onSelect: (user: UserSearchResult) => void
}

export function UserSearchInput({ excludeUserIds, onSelect }: UserSearchInputProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const debouncedSearch = useDebounce(search, 300)
  const { data: results, isLoading } = useSearchUsers(debouncedSearch)
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const filtered = (results ?? []).filter((u) => !excludeUserIds.includes(u.id))

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  // Open dropdown when there are results
  useEffect(() => {
    if (debouncedSearch.length >= 2) {
      setOpen(true)
    }
  }, [debouncedSearch])

  function handleSelect(user: UserSearchResult) {
    onSelect(user)
    setSearch('')
    setOpen(false)
  }

  return (
    <div ref={ref} className="relative flex-1">
      <input
        ref={inputRef}
        className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        placeholder={t('projects.settings.addMemberPlaceholder')}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        onFocus={() => { if (debouncedSearch.length >= 2) setOpen(true) }}
      />

      {open && search.length >= 2 && (
        <div className="absolute z-20 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg">
          <ul className="max-h-48 overflow-auto">
            {isLoading && (
              <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">...</li>
            )}
            {!isLoading && filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">
                {t('projects.settings.noUsersFound')}
              </li>
            )}
            {!isLoading && filtered.map((user) => (
              <li key={user.id}>
                <button
                  type="button"
                  className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100"
                  onClick={() => handleSelect(user)}
                >
                  <div className="font-medium">{user.display_name}</div>
                  <div className="text-xs text-gray-400">{user.email}</div>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}

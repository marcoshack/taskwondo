import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { useDebounce } from '@/hooks/useDebounce'
import { useSearchUsers } from '@/hooks/useUsers'
import type { UserSearchResult } from '@/api/users'
import { Mail } from 'lucide-react'
import { Avatar } from '@/components/ui/Avatar'

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

interface UserSearchInputProps {
  excludeUserIds: string[]
  currentNamespaceSlug?: string
  onSelectMember: (user: UserSearchResult) => void
  onInviteEmail: (email: string) => void
  placeholder?: string
}

export function UserSearchInput({
  excludeUserIds,
  currentNamespaceSlug,
  onSelectMember,
  onInviteEmail,
  placeholder,
}: UserSearchInputProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({})
  const debouncedSearch = useDebounce(search, 300)
  const { data: results, isLoading } = useSearchUsers(debouncedSearch)
  const ref = useRef<HTMLDivElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const filtered = useMemo(
    () => (results ?? []).filter((u) => !excludeUserIds.includes(u.id)),
    [results, excludeUserIds],
  )

  const trimmedSearch = search.trim()
  const isEmail = EMAIL_REGEX.test(trimmedSearch)
  const exactEmailMatch = filtered.find(
    (u) => u.email.toLowerCase() === trimmedSearch.toLowerCase(),
  )
  const showInviteRow = isEmail && !exactEmailMatch

  function isMemberOfNamespace(u: UserSearchResult): boolean {
    if (!currentNamespaceSlug) return true
    return (u.namespace_slugs ?? []).includes(currentNamespaceSlug)
  }

  const updateDropdownPosition = useCallback(() => {
    if (!inputRef.current) return
    const rect = inputRef.current.getBoundingClientRect()
    setDropdownStyle({
      position: 'fixed',
      top: rect.bottom + 4,
      left: rect.left,
      width: rect.width,
      zIndex: 60,
    })
  }, [])

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      const target = e.target as Node
      if (ref.current?.contains(target) || dropdownRef.current?.contains(target)) return
      setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  useEffect(() => {
    if (debouncedSearch.length >= 2) {
      updateDropdownPosition()
      setOpen(true)
    }
  }, [debouncedSearch, updateDropdownPosition])

  function handleSelectMember(user: UserSearchResult) {
    onSelectMember(user)
    setSearch('')
    setOpen(false)
  }

  function handleInviteUser(user: UserSearchResult) {
    onInviteEmail(user.email)
    setSearch('')
    setOpen(false)
  }

  function handleInviteTyped() {
    if (!isEmail) return
    onInviteEmail(trimmedSearch)
    setSearch('')
    setOpen(false)
  }

  return (
    <div ref={ref} className="relative flex-1">
      <input
        ref={inputRef}
        className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        placeholder={placeholder ?? t('projects.settings.addMemberPlaceholder')}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        onFocus={() => { if (debouncedSearch.length >= 2) { updateDropdownPosition(); setOpen(true) } }}
      />

      {open && search.length >= 2 && createPortal(
        <div
          ref={dropdownRef}
          style={dropdownStyle}
          className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg"
        >
          <ul className="max-h-64 overflow-auto">
            {isLoading && (
              <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">...</li>
            )}
            {!isLoading && filtered.length === 0 && !showInviteRow && (
              <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">
                {t('projects.settings.noUsersFound')}
              </li>
            )}
            {!isLoading && filtered.map((user) => {
              const isMember = isMemberOfNamespace(user)
              return (
                <li key={user.id}>
                  <button
                    type="button"
                    className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100 flex items-center gap-2"
                    onClick={() => (isMember ? handleSelectMember(user) : handleInviteUser(user))}
                  >
                    <Avatar name={user.display_name} avatarUrl={user.avatar_url} size="xs" />
                    <div className="min-w-0 flex-1">
                      <div className="font-medium truncate">{user.display_name}</div>
                      <div className="text-xs text-gray-400 truncate">{user.email}</div>
                    </div>
                    {!isMember && (
                      <span className="text-xs text-gray-500 dark:text-gray-400 shrink-0">
                        {t('projects.settings.notInNamespaceInvite')}
                      </span>
                    )}
                  </button>
                </li>
              )
            })}
            {!isLoading && showInviteRow && (
              <li className={filtered.length > 0 ? 'border-t border-gray-200 dark:border-gray-600' : ''}>
                <button
                  type="button"
                  className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100 flex items-center gap-2"
                  onClick={handleInviteTyped}
                >
                  <Mail className="h-4 w-4 text-gray-400 shrink-0" />
                  <span className="truncate">
                    {t('projects.settings.inviteByEmailRow', { email: trimmedSearch })}
                  </span>
                </button>
              </li>
            )}
          </ul>
        </div>,
        document.body
      )}
    </div>
  )
}

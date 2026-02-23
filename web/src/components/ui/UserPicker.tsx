import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import type { ProjectMember } from '@/api/projects'

interface UserPickerProps {
  members: ProjectMember[]
  value: string | null
  onChange: (userId: string | null) => void
  placeholder?: string
  disabled?: boolean
}

export function UserPicker({ members, value, onChange, placeholder, disabled }: UserPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const selected = members.find((m) => m.user_id === value)

  const filtered = members.filter((m) => {
    if (!search) return true
    const q = search.toLowerCase()
    return m.display_name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q)
  })

  // Close on click outside
  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <div ref={ref} className="relative">
      {/* Display / trigger */}
      <button
        type="button"
        className={`block w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm text-left shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-800 ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
        onClick={() => { if (disabled) return; setOpen(!open); setTimeout(() => inputRef.current?.focus(), 0) }}
        disabled={disabled}
      >
        {selected ? (
          <span className="text-gray-900 dark:text-gray-100">{selected.display_name}</span>
        ) : (
          <span className="text-gray-400 dark:text-gray-500">{value ? t('userPicker.unknownUser') : t('userPicker.unassigned')}</span>
        )}
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-20 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg">
          <div className="p-2">
            <input
              ref={inputRef}
              className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-500"
              placeholder={placeholder ?? t('userPicker.searchMembers')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <ul className="max-h-48 overflow-auto">
            {/* Unassign option */}
            <li>
              <button
                type="button"
                className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 ${
                  !value ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300' : 'text-gray-500 dark:text-gray-400 italic'
                }`}
                onClick={() => { onChange(null); setOpen(false); setSearch('') }}
              >
                {t('userPicker.unassigned')}
              </button>
            </li>
            {filtered.map((m) => (
              <li key={m.user_id}>
                <button
                  type="button"
                  className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 ${
                    m.user_id === value ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300' : 'text-gray-900 dark:text-gray-100'
                  }`}
                  onClick={() => { onChange(m.user_id); setOpen(false); setSearch('') }}
                >
                  <div className="font-medium">{m.display_name}</div>
                  <div className="text-xs text-gray-400">{m.email}</div>
                </button>
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">{t('userPicker.noMembersFound')}</li>
            )}
          </ul>
        </div>
      )}
    </div>
  )
}

import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import type { ProjectMember } from '@/api/projects'
import { Avatar } from '@/components/ui/Avatar'

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

  // Exclude viewers — they cannot be assigned to work items
  const assignableMembers = members.filter((m) => m.role !== 'viewer')
  // Use full members list for display in case current assignee was demoted to viewer
  const selected = members.find((m) => m.user_id === value)

  const filtered = assignableMembers.filter((m) => {
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
        className={`block w-full rounded-md border border-[var(--border)] px-3 py-2 text-sm text-left shadow-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--focus-ring)] bg-[var(--surface)] ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
        onClick={() => { if (disabled) return; setOpen(!open); setTimeout(() => inputRef.current?.focus(), 0) }}
        disabled={disabled}
      >
        {selected ? (
          <span className="flex items-center gap-2 text-[var(--foreground)]">
            <Avatar name={selected.display_name} avatarUrl={selected.avatar_url} size="xs" />
            {selected.display_name}
          </span>
        ) : (
          <span className="text-[var(--foreground-muted)]">{value ? t('userPicker.unknownUser') : t('userPicker.unassigned')}</span>
        )}
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-20 mt-1 w-full bg-[var(--surface)] border border-[var(--border)] rounded-md shadow-[var(--shadow-md)]">
          <div className="p-2">
            <input
              ref={inputRef}
              className="block w-full rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-[var(--focus-ring)]"
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
                className={`w-full text-left px-3 py-2 text-sm hover:bg-[var(--surface-hover)] ${
                  !value ? 'bg-[var(--primary-muted)] text-[var(--primary)]' : 'text-[var(--foreground-secondary)] italic'
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
                  className={`w-full text-left px-3 py-2 text-sm hover:bg-[var(--surface-hover)] ${
                    m.user_id === value ? 'bg-[var(--primary-muted)] text-[var(--primary)]' : 'text-[var(--foreground)]'
                  }`}
                  onClick={() => { onChange(m.user_id); setOpen(false); setSearch('') }}
                >
                  <div className="flex items-center gap-2">
                    <Avatar name={m.display_name} avatarUrl={m.avatar_url} size="xs" />
                    <div>
                      <div className="font-medium">{m.display_name}</div>
                      <div className="text-xs text-[var(--foreground-muted)]">{m.email}</div>
                    </div>
                  </div>
                </button>
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-[var(--foreground-muted)]">{t('userPicker.noMembersFound')}</li>
            )}
          </ul>
        </div>
      )}
    </div>
  )
}

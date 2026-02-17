import { useState, useRef, useEffect } from 'react'
import type { ProjectMember } from '@/api/projects'

interface UserPickerProps {
  members: ProjectMember[]
  value: string | null
  onChange: (userId: string | null) => void
  placeholder?: string
}

export function UserPicker({ members, value, onChange, placeholder = 'Search members...' }: UserPickerProps) {
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
        className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm text-left shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white"
        onClick={() => { setOpen(!open); setTimeout(() => inputRef.current?.focus(), 0) }}
      >
        {selected ? (
          <span className="text-gray-900">{selected.display_name}</span>
        ) : (
          <span className="text-gray-400">{value ? 'Unknown user' : 'Unassigned'}</span>
        )}
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-20 mt-1 w-full bg-white border border-gray-200 rounded-md shadow-lg">
          <div className="p-2">
            <input
              ref={inputRef}
              className="block w-full rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-500"
              placeholder={placeholder}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <ul className="max-h-48 overflow-auto">
            {/* Unassign option */}
            <li>
              <button
                type="button"
                className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 ${
                  !value ? 'bg-indigo-50 text-indigo-700' : 'text-gray-500 italic'
                }`}
                onClick={() => { onChange(null); setOpen(false); setSearch('') }}
              >
                Unassigned
              </button>
            </li>
            {filtered.map((m) => (
              <li key={m.user_id}>
                <button
                  type="button"
                  className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 ${
                    m.user_id === value ? 'bg-indigo-50 text-indigo-700' : 'text-gray-900'
                  }`}
                  onClick={() => { onChange(m.user_id); setOpen(false); setSearch('') }}
                >
                  <div className="font-medium">{m.display_name}</div>
                  <div className="text-xs text-gray-400">{m.email}</div>
                </button>
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-gray-400">No members found</li>
            )}
          </ul>
        </div>
      )}
    </div>
  )
}

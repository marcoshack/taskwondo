import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import type { Project } from '@/api/projects'

interface ProjectPickerProps {
  projects: Project[]
  value: string
  onChange: (projectKey: string) => void
  disabled?: boolean
}

export function ProjectPicker({ projects, value, onChange, disabled }: ProjectPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const selected = projects.find((p) => p.key === value)

  const filtered = projects.filter((p) => {
    if (!search) return true
    const q = search.toLowerCase()
    return p.name.toLowerCase().includes(q) || p.key.toLowerCase().includes(q)
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
      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
        {t('workitems.form.project')} <span className="text-red-500">*</span>
      </label>
      <button
        type="button"
        className={`block w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm text-left shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-800 ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
        onClick={() => { if (disabled) return; setOpen(!open); setTimeout(() => inputRef.current?.focus(), 0) }}
        disabled={disabled}
      >
        {selected ? (
          <span className="text-gray-900 dark:text-gray-100">{selected.name} ({selected.key})</span>
        ) : (
          <span className="text-gray-400 dark:text-gray-500">{t('workitems.form.projectPlaceholder')}</span>
        )}
      </button>

      {open && (
        <div className="absolute z-20 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg">
          <div className="p-2">
            <input
              ref={inputRef}
              className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-500"
              placeholder={t('workitems.form.projectSearchPlaceholder')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <ul className="max-h-48 overflow-auto">
            {filtered.map((p) => (
              <li key={p.key}>
                <button
                  type="button"
                  className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 ${
                    p.key === value ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300' : 'text-gray-900 dark:text-gray-100'
                  }`}
                  onClick={() => { onChange(p.key); setOpen(false); setSearch('') }}
                >
                  <div className="font-medium">{p.name}</div>
                  <div className="text-xs text-gray-400">{p.key}</div>
                </button>
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">{t('workitems.form.noProjectsFound')}</li>
            )}
          </ul>
        </div>
      )}
    </div>
  )
}

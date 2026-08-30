import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import type { Project } from '@/api/projects'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { useNamespaceContext } from '@/contexts/NamespaceContext'

interface ProjectPickerProps {
  projects: Project[]
  value: string
  onChange: (projectKey: string) => void
  disabled?: boolean
  /** Validation message; also turns the control's border red. */
  error?: string
}

export function ProjectPicker({ projects, value, onChange, disabled, error }: ProjectPickerProps) {
  const { t } = useTranslation()
  const { showSwitcher: showNamespaces } = useNamespaceContext()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const selected = projects.find((p) => p.key === value)

  const filtered = projects.filter((p) => {
    if (!search) return true
    const q = search.toLowerCase()
    return p.name.toLowerCase().includes(q) || p.key.toLowerCase().includes(q) || (p.namespace_slug ?? '').toLowerCase().includes(q)
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
      <label className="block text-sm font-medium text-[var(--foreground)] mb-1">
        {t('workitems.form.project')} <span className="text-[var(--danger)]">*</span>
      </label>
      <button
        type="button"
        className={`block w-full rounded-md border px-3 py-2 text-sm text-left shadow-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--focus-ring)] bg-[var(--surface)] ${
          error ? 'border-[var(--danger)]' : 'border-[var(--border)]'
        } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
        onClick={() => { if (disabled) return; setOpen(!open); setTimeout(() => inputRef.current?.focus(), 0) }}
        disabled={disabled}
      >
        {selected ? (
          <span className="flex items-center gap-2 text-[var(--foreground)]">
            <ProjectKeyBadge size="icon">{selected.key}</ProjectKeyBadge>
            <span className="truncate">{selected.name}</span>
            {showNamespaces && selected.namespace_slug && (
              <span className="ml-auto flex items-center gap-1 text-[0.7rem] text-[var(--foreground-muted)] shrink-0">
                <span>{selected.namespace_slug}</span>
                <NamespaceIcon icon={selected.namespace_icon ?? 'building2'} color={selected.namespace_color ?? 'slate'} className="h-3 w-3" />
              </span>
            )}
          </span>
        ) : (
          <span className="text-[var(--foreground-muted)]">{t('workitems.form.projectPlaceholder')}</span>
        )}
      </button>

      {open && (
        <div className="absolute z-20 mt-1 w-full bg-[var(--surface)] border border-[var(--border)] rounded-md shadow-[var(--shadow-md)]">
          <div className="p-2">
            <input
              ref={inputRef}
              className="block w-full rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-[var(--focus-ring)]"
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
                  className={`w-full text-left flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--surface-hover)] ${
                    p.key === value ? 'bg-[var(--primary-muted)]' : ''
                  }`}
                  onClick={() => { onChange(p.key); setOpen(false); setSearch('') }}
                >
                  <ProjectKeyBadge size="icon">{p.key}</ProjectKeyBadge>
                  <span className="text-[var(--foreground)] font-medium truncate">{p.name}</span>
                  {showNamespaces && p.namespace_slug && (
                    <span className="ml-auto flex items-center gap-1 text-[0.7rem] text-[var(--foreground-muted)] shrink-0">
                      <span>{p.namespace_slug}</span>
                      <NamespaceIcon icon={p.namespace_icon ?? 'building2'} color={p.namespace_color ?? 'slate'} className="h-3 w-3" />
                    </span>
                  )}
                </button>
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="px-3 py-2 text-sm text-[var(--foreground-muted)]">{t('workitems.form.noProjectsFound')}</li>
            )}
          </ul>
        </div>
      )}

      {error && <p className="mt-1 text-sm text-[var(--danger)]">{error}</p>}
    </div>
  )
}

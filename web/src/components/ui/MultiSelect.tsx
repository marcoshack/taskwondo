import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'

export interface MultiSelectOption {
  value: string
  label: string
  group?: string
}

interface MultiSelectProps {
  options: MultiSelectOption[]
  selected: string[]
  onChange: (selected: string[]) => void
  placeholder?: string
  className?: string
}

export function MultiSelect({ options, selected, onChange, placeholder = 'All', className = '' }: MultiSelectProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  function toggle(value: string) {
    if (selected.includes(value)) {
      onChange(selected.filter((v) => v !== value))
    } else {
      onChange([...selected, value])
    }
  }

  function selectAll() {
    onChange(options.map((o) => o.value))
  }

  function clearAll() {
    onChange([])
  }

  const label = selected.length === 0
    ? placeholder
    : selected.length === options.length
      ? placeholder
      : selected.length === 1
        ? options.find((o) => o.value === selected[0])?.label ?? selected[0]
        : t('multiSelect.selected', { count: selected.length })

  // Group options if any have a group
  const hasGroups = options.some((o) => o.group)
  const groups: { name: string | null; items: MultiSelectOption[] }[] = []
  if (hasGroups) {
    const groupMap = new Map<string | null, MultiSelectOption[]>()
    for (const opt of options) {
      const g = opt.group ?? null
      if (!groupMap.has(g)) groupMap.set(g, [])
      groupMap.get(g)!.push(opt)
    }
    for (const [name, items] of groupMap) {
      groups.push({ name, items })
    }
  }

  return (
    <div ref={ref} className={`relative ${className}`}>
      <button
        type="button"
        className={`flex items-center justify-between w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm shadow-sm hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
          selected.length > 0 && selected.length < options.length ? 'text-gray-900 dark:text-gray-100' : 'text-gray-500 dark:text-gray-400'
        }`}
        onClick={() => setOpen(!open)}
      >
        <span className="truncate">{label}</span>
        <svg className="ml-1 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {open && (
        <div className="absolute z-20 mt-1 w-full min-w-[180px] rounded-md border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 shadow-lg">
          <div className="flex items-center justify-between px-3 py-1.5 border-b border-gray-100 dark:border-gray-700">
            <button type="button" className="text-xs text-indigo-600 hover:text-indigo-800" onClick={selectAll}>
              {t('common.all')}
            </button>
            <button type="button" className="text-xs text-gray-400 hover:text-gray-600" onClick={clearAll}>
              {t('common.none')}
            </button>
          </div>
          <div className="max-h-60 overflow-y-auto py-1">
            {hasGroups ? (
              groups.map((group) => (
                <div key={group.name ?? '__default'}>
                  {group.name && (
                    <div className="px-3 pt-2 pb-1 text-[0.65rem] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                      {group.name}
                    </div>
                  )}
                  {group.items.map((opt) => (
                    <OptionRow key={opt.value} option={opt} checked={selected.includes(opt.value)} onToggle={toggle} />
                  ))}
                </div>
              ))
            ) : (
              options.map((opt) => (
                <OptionRow key={opt.value} option={opt} checked={selected.includes(opt.value)} onToggle={toggle} />
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function OptionRow({ option, checked, onToggle }: { option: MultiSelectOption; checked: boolean; onToggle: (value: string) => void }) {
  return (
    <label className="flex items-center gap-2 px-3 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer text-sm text-gray-700 dark:text-gray-300">
      <input
        type="checkbox"
        checked={checked}
        onChange={() => onToggle(option.value)}
        className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
      />
      {option.label}
    </label>
  )
}

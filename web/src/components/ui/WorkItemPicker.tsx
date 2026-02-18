import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useWorkItems } from '@/hooks/useWorkItems'
import { useDebounce } from '@/hooks/useDebounce'
import { Spinner } from '@/components/ui/Spinner'

interface WorkItemPickerProps {
  projectKey: string
  excludeItemNumber?: number
  value: string
  onChange: (value: string) => void
  onSelect: (displayId: string) => void
  placeholder?: string
}

export function WorkItemPicker({
  projectKey,
  excludeItemNumber,
  value,
  onChange,
  onSelect,
  placeholder,
}: WorkItemPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const debouncedSearch = useDebounce(value, 300)
  const shouldSearch = open && debouncedSearch.length >= 2

  const { data, isFetching } = useWorkItems(
    projectKey,
    shouldSearch ? { q: debouncedSearch, limit: 10 } : { limit: 0 },
  )

  const items = shouldSearch
    ? (data?.data ?? []).filter((item) => item.item_number !== excludeItemNumber)
    : []

  // Close on click outside
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

  return (
    <div ref={ref} className="relative">
      <input
        ref={inputRef}
        className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        placeholder={placeholder ?? t('workItemPicker.searchPlaceholder')}
        value={value}
        onChange={(e) => {
          onChange(e.target.value)
          if (!open) setOpen(true)
        }}
        onFocus={() => setOpen(true)}
      />

      {open && value.length >= 2 && (
        <div className="absolute z-20 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg">
          {isFetching ? (
            <div className="flex items-center justify-center py-3">
              <Spinner size="sm" />
            </div>
          ) : items.length > 0 ? (
            <ul className="max-h-48 overflow-auto py-1">
              {items.map((item) => (
                <li key={item.id}>
                  <button
                    type="button"
                    className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700"
                    onClick={() => {
                      onSelect(item.display_id)
                      setOpen(false)
                    }}
                  >
                    <span className="font-mono font-medium text-indigo-600 dark:text-indigo-400">
                      {item.display_id}
                    </span>
                    <span className="ml-2 text-gray-700 dark:text-gray-300 truncate">
                      {item.title}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <div className="px-3 py-3 text-sm text-gray-400 dark:text-gray-500">
              {t('workItemPicker.noItems')}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

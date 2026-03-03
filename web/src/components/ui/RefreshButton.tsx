import { useState, useRef, useEffect, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { RefreshCw, ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export type RefreshInterval = 0 | 5000 | 10000 | 30000 | 60000 | 300000

interface RefreshButtonProps {
  interval: RefreshInterval
  onIntervalChange: (interval: RefreshInterval) => void
  onRefresh: () => void
  isRefreshing?: boolean
}

const INTERVAL_OPTIONS: { value: RefreshInterval; labelKey: string }[] = [
  { value: 0, labelKey: 'inbox.autoRefreshOff' },
  { value: 5000, labelKey: 'inbox.autoRefresh5s' },
  { value: 10000, labelKey: 'inbox.autoRefresh10s' },
  { value: 30000, labelKey: 'inbox.autoRefresh30s' },
  { value: 60000, labelKey: 'inbox.autoRefresh1m' },
  { value: 300000, labelKey: 'inbox.autoRefresh5m' },
]

function getIntervalLabel(t: (key: string) => string, interval: RefreshInterval): string {
  const opt = INTERVAL_OPTIONS.find((o) => o.value === interval)
  return opt ? t(opt.labelKey) : t('inbox.autoRefreshOff')
}

export function RefreshButton({ interval, onIntervalChange, onRefresh, isRefreshing }: RefreshButtonProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const buttonRef = useRef<HTMLDivElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (
      buttonRef.current && !buttonRef.current.contains(e.target as Node) &&
      dropdownRef.current && !dropdownRef.current.contains(e.target as Node)
    ) {
      setOpen(false)
    }
  }, [])

  useEffect(() => {
    if (open) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [open, handleClickOutside])

  // Close on Escape
  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open])

  const isActive = interval > 0
  const label = isActive ? getIntervalLabel(t, interval) : t('inbox.refresh')

  return (
    <div ref={buttonRef} className="relative inline-flex">
      <div className="inline-flex items-stretch rounded-lg border border-gray-300 dark:border-gray-600 overflow-hidden">
        {/* Refresh button */}
        <button
          onClick={onRefresh}
          className="flex items-center gap-1.5 px-2.5 py-2 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          aria-label={t('inbox.refresh')}
        >
          <RefreshCw className={`h-5 w-5 ${isRefreshing ? 'animate-spin' : ''}`} />
          <span className="hidden sm:inline">{label}</span>
        </button>
        {/* Dropdown toggle */}
        <button
          onClick={() => setOpen((v) => !v)}
          className="flex items-center px-1.5 border-l border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          aria-label={t('inbox.autoRefresh')}
          aria-expanded={open}
        >
          <ChevronDown className={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-180' : ''}`} />
        </button>
      </div>

      {/* Dropdown portal */}
      {open && <DropdownMenu
        triggerRef={buttonRef}
        dropdownRef={dropdownRef}
        interval={interval}
        onSelect={(val) => {
          onIntervalChange(val)
          setOpen(false)
        }}
      />}
    </div>
  )
}

function DropdownMenu({
  triggerRef,
  dropdownRef,
  interval,
  onSelect,
}: {
  triggerRef: React.RefObject<HTMLDivElement | null>
  dropdownRef: React.RefObject<HTMLDivElement | null>
  interval: RefreshInterval
  onSelect: (val: RefreshInterval) => void
}) {
  const { t } = useTranslation()
  const [style, setStyle] = useState<React.CSSProperties>({ position: 'fixed', opacity: 0 })

  useEffect(() => {
    const trigger = triggerRef.current
    if (!trigger) return
    const rect = trigger.getBoundingClientRect()
    setStyle({
      position: 'fixed',
      top: rect.bottom + 4,
      right: window.innerWidth - rect.right,
      opacity: 1,
    })
  }, [triggerRef])

  return createPortal(
    <div
      ref={dropdownRef}
      className="z-50 min-w-[120px] rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-lg py-1 animate-in fade-in duration-100"
      style={style}
    >
      {INTERVAL_OPTIONS.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onSelect(opt.value)}
          className={`w-full text-left px-3 py-1.5 text-sm transition-colors ${
            interval === opt.value
              ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400 font-medium'
              : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
          }`}
        >
          {t(opt.labelKey)}
        </button>
      ))}
    </div>,
    document.body,
  )
}

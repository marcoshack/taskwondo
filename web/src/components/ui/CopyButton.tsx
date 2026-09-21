import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'

interface CopyButtonProps {
  text: string
  className?: string
  tooltip?: string
}

export function CopyButton({ text, className = '', tooltip }: CopyButtonProps) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // fallback
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }, [text])

  return (
    <button
      type="button"
      className={`group/copy relative inline-flex items-center justify-center w-7 h-7 rounded-md text-[var(--text-tertiary)] hover:text-[var(--text-secondary)] hover:bg-[var(--hover)] transition-colors ${className}`}
      onClick={handleCopy}
      aria-label={tooltip ?? t('common.copyToClipboard')}
    >
      {copied ? (
        <svg className="w-4 h-4 text-[var(--success)]" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
          <path strokeLinecap="round" strokeLinejoin="round" d="M3.5 8.5l3 3 6-7" />
        </svg>
      ) : (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
          <rect x="5.5" y="5.5" width="5" height="7" rx="0.5" />
          <path strokeLinecap="round" strokeLinejoin="round" d="M3.5 10.5V4.5a1 1 0 011-1h5" />
        </svg>
      )}
      <span className="pointer-events-none absolute -top-8 left-1/2 -translate-x-1/2 whitespace-nowrap rounded bg-[var(--tooltip-bg)] px-2 py-1 text-xs text-[var(--tooltip-text)] opacity-0 group-hover/copy:opacity-100 transition-opacity">
        {copied ? t('common.copied') : (tooltip ?? t('common.copy'))}
      </span>
    </button>
  )
}

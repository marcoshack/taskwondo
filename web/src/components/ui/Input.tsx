import { forwardRef } from 'react'
import type { InputHTMLAttributes } from 'react'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  /**
   * Shows a red asterisk after the label. Opt-in rather than driven by
   * `required`, so existing forms keep their exact label text.
   */
  requiredMarker?: boolean
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, requiredMarker, id, className = '', ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-')
    return (
      <div className="overflow-hidden">
        {label && (
          <label htmlFor={inputId} className="block text-sm font-medium text-[var(--foreground)] mb-1">
            {label}{requiredMarker && <span className="text-[var(--danger)]" aria-hidden="true"> *</span>}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          className={`block w-full min-w-0 rounded-[var(--radius)] border px-3 py-2 text-sm bg-[var(--surface)] focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)] placeholder:text-[var(--foreground-muted)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors ${
            error ? 'border-[var(--danger)] text-[var(--danger)]' : 'border-[var(--border)] text-[var(--foreground)]'
          } ${className}`}
          {...props}
        />
        {error && <p className="mt-1 text-sm text-[var(--danger)]">{error}</p>}
      </div>
    )
  },
)

Input.displayName = 'Input'

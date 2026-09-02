import { forwardRef } from 'react'
import type { SelectHTMLAttributes } from 'react'
import { ChevronDown } from 'lucide-react'

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
  error?: string
  /**
   * Shows a red asterisk after the label. Opt-in rather than driven by
   * `required`, so existing forms keep their exact label text.
   */
  requiredMarker?: boolean
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, error, requiredMarker, id, className = '', children, ...props }, ref) => {
    const selectId = id ?? label?.toLowerCase().replace(/\s+/g, '-')
    return (
      <div>
        {label && (
          <label htmlFor={selectId} className="block text-sm font-medium text-[var(--foreground)] mb-1">
            {label}{requiredMarker && <span className="text-[var(--danger)]" aria-hidden="true"> *</span>}
          </label>
        )}
        <div className="relative">
          <select
            ref={ref}
            id={selectId}
            // appearance-none keeps the box model consistent across browsers — Safari's
            // native select control ignores vertical padding and renders shorter than
            // text inputs. We render our own chevron below in its place.
            className={`block w-full appearance-none rounded-[var(--radius)] border px-3 py-2 pr-10 text-sm bg-[var(--surface)] focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors ${
              error ? 'border-[var(--danger)]' : 'border-[var(--border)]'
            } ${className}`}
            {...props}
          >
            {children}
          </select>
          <ChevronDown
            className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--foreground-muted)]"
            aria-hidden="true"
          />
        </div>
        {error && <p className="mt-1 text-sm text-[var(--danger)]">{error}</p>}
      </div>
    )
  },
)

Select.displayName = 'Select'

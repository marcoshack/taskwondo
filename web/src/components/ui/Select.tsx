import { forwardRef } from 'react'
import type { SelectHTMLAttributes } from 'react'
import { ChevronDown } from 'lucide-react'

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
  error?: string
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, error, id, className = '', children, ...props }, ref) => {
    const selectId = id ?? label?.toLowerCase().replace(/\s+/g, '-')
    return (
      <div>
        {label && (
          <label htmlFor={selectId} className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {label}
          </label>
        )}
        <div className="relative">
          <select
            ref={ref}
            id={selectId}
            // appearance-none keeps the box model consistent across browsers — Safari's
            // native select control ignores vertical padding and renders shorter than
            // text inputs. We render our own chevron below in its place.
            className={`block w-full appearance-none rounded-md border px-3 py-2 pr-10 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-100 disabled:opacity-50 disabled:cursor-not-allowed ${
              error ? 'border-red-300' : 'border-gray-300 dark:border-gray-600'
            } ${className}`}
            {...props}
          >
            {children}
          </select>
          <ChevronDown
            className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            aria-hidden="true"
          />
        </div>
        {error && <p className="mt-1 text-sm text-red-600">{error}</p>}
      </div>
    )
  },
)

Select.displayName = 'Select'

import type { ButtonHTMLAttributes } from 'react'

const variants = {
  primary: 'bg-[var(--primary)] text-[var(--primary-foreground)] hover:bg-[var(--primary-hover)] focus:ring-[var(--focus-ring)]',
  secondary: 'bg-[var(--surface)] text-[var(--foreground)] border border-[var(--border)] hover:bg-[var(--hover)] focus:ring-[var(--focus-ring)]',
  danger: 'bg-[var(--danger)] text-white hover:bg-red-700 focus:ring-[var(--danger)]',
  ghost: 'text-[var(--foreground)] hover:bg-[var(--hover)] focus:ring-[var(--focus-ring)]',
} as const

const sizes = {
  sm: 'px-2.5 py-1.5 text-xs',
  md: 'px-4 py-2 text-sm',
  lg: 'px-6 py-3 text-base',
} as const

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: keyof typeof variants
  size?: keyof typeof sizes
}

export function Button({
  variant = 'primary',
  size = 'md',
  className = '',
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={`inline-flex items-center justify-center font-medium rounded-[var(--radius)] focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors ${variants[variant]} ${sizes[size]} ${className}`}
      disabled={disabled}
      {...props}
    />
  )
}

const colors = {
  gray: 'bg-[var(--surface-tertiary)] text-[var(--foreground-secondary)]',
  blue: 'bg-[var(--info-bg)] text-[var(--info)]',
  green: 'bg-[var(--success-bg)] text-[var(--success)]',
  yellow: 'bg-[var(--warning-bg)] text-[var(--warning)]',
  red: 'bg-[var(--danger-bg)] text-[var(--danger)]',
  indigo: 'bg-[var(--primary-muted)] text-[var(--primary)]',
} as const

interface BadgeProps {
  color?: keyof typeof colors
  children: React.ReactNode
}

export function Badge({ color = 'gray', children }: BadgeProps) {
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${colors[color]}`}>
      {children}
    </span>
  )
}

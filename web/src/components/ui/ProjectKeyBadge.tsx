const sizeClasses = {
  default: 'text-sm px-2.5 py-0.5 min-w-[4.5rem]',
  nav: 'text-sm px-2.5 py-1',
  'nav-mobile': 'text-base px-2.5 py-1',
  icon: 'text-xs leading-none px-1.5 py-0.5',
} as const

interface ProjectKeyBadgeProps {
  children: React.ReactNode
  size?: keyof typeof sizeClasses
}

export function ProjectKeyBadge({ children, size = 'default' }: ProjectKeyBadgeProps) {
  return (
    <span className={`inline-flex items-center justify-center rounded-md bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 font-bold shrink-0 ${sizeClasses[size]}`}>
      {children}
    </span>
  )
}

const sizeClasses = {
  sm: 'h-6 w-6 text-xs',
  md: 'h-8 w-8 text-sm',
  lg: 'h-10 w-10 text-base',
} as const

interface AvatarProps {
  name: string
  size?: keyof typeof sizeClasses
}

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((p) => p[0])
    .filter(Boolean)
    .slice(0, 2)
    .join('')
    .toUpperCase()
}

export function Avatar({ name, size = 'md' }: AvatarProps) {
  return (
    <span
      className={`inline-flex items-center justify-center rounded-full bg-indigo-600 text-white font-medium ${sizeClasses[size]}`}
      title={name}
    >
      {getInitials(name)}
    </span>
  )
}

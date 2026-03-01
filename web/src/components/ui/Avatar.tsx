import { useState } from 'react'
import { Tooltip } from './Tooltip'

const sizeClasses = {
  xs: 'h-4.5 w-4.5 text-[0.6rem]',
  sm: 'h-6 w-6 text-xs',
  md: 'h-8 w-8 text-sm',
  lg: 'h-10 w-10 text-base',
  xl: 'h-28 w-28 text-3xl',
} as const

interface AvatarProps {
  name: string
  avatarUrl?: string
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

export function Avatar({ name, avatarUrl, size = 'md' }: AvatarProps) {
  const [imgError, setImgError] = useState(false)

  if (avatarUrl && !imgError) {
    return (
      <Tooltip content={name}>
        <img
          src={avatarUrl}
          alt={name}
          onError={() => setImgError(true)}
          className={`inline-flex rounded-full object-cover ${sizeClasses[size]}`}
        />
      </Tooltip>
    )
  }

  return (
    <Tooltip content={name}>
      <span
        className={`inline-flex items-center justify-center rounded-full bg-indigo-600 text-white font-medium ${sizeClasses[size]}`}
      >
        {getInitials(name)}
      </span>
    </Tooltip>
  )
}

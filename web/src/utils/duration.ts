const UNITS: Record<string, number> = {
  w: 7 * 24 * 3600,
  d: 24 * 3600,
  h: 3600,
  m: 60,
}

/**
 * Parse a human-readable duration string into seconds.
 * Supports: "15m", "2h", "1d", "1w", "1d 2h", "2h 30m", etc.
 * Returns null if the input is invalid.
 */
export function parseDuration(input: string): number | null {
  const trimmed = input.trim()
  if (!trimmed) return null

  // Try pure number (treat as seconds)
  const asNumber = Number(trimmed)
  if (!isNaN(asNumber) && asNumber > 0) return Math.floor(asNumber)

  const pattern = /(\d+)\s*(w|d|h|m)/gi
  let match: RegExpExecArray | null
  let total = 0
  let matched = false

  while ((match = pattern.exec(trimmed)) !== null) {
    const value = parseInt(match[1], 10)
    const unit = match[2].toLowerCase()
    if (UNITS[unit] && value > 0) {
      total += value * UNITS[unit]
      matched = true
    }
  }

  return matched && total > 0 ? total : null
}

/**
 * Format seconds into a human-readable duration string.
 * Examples: "1d 2h", "45m", "2w 3d", "1h 30m"
 */
export function formatDuration(seconds: number): string {
  if (seconds <= 0) return '0m'

  const parts: string[] = []
  let remaining = seconds

  const weeks = Math.floor(remaining / UNITS.w)
  if (weeks > 0) {
    parts.push(`${weeks}w`)
    remaining -= weeks * UNITS.w
  }

  const days = Math.floor(remaining / UNITS.d)
  if (days > 0) {
    parts.push(`${days}d`)
    remaining -= days * UNITS.d
  }

  const hours = Math.floor(remaining / UNITS.h)
  if (hours > 0) {
    parts.push(`${hours}h`)
    remaining -= hours * UNITS.h
  }

  const minutes = Math.floor(remaining / UNITS.m)
  if (minutes > 0) {
    parts.push(`${minutes}m`)
  }

  return parts.length > 0 ? parts.join(' ') : '< 1m'
}

/**
 * Format remaining seconds as a countdown string.
 * Positive: "2h 15m left"
 * Negative: "1h 30m overdue"
 */
export function formatRemaining(seconds: number): string {
  if (seconds >= 0) {
    return `${formatDuration(seconds)} left`
  }
  return `${formatDuration(Math.abs(seconds))} overdue`
}

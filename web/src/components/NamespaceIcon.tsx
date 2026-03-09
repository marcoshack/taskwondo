import {
  Building2, Users, Briefcase, Code, Rocket, Shield, Heart, Zap,
  BookOpen, Star, Layers, Compass, Target, Lightbulb, Globe,
  Palette, Cpu, Leaf, Music, Anchor,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

const ICON_MAP: Record<string, LucideIcon> = {
  'building2': Building2,
  'users': Users,
  'briefcase': Briefcase,
  'code': Code,
  'rocket': Rocket,
  'shield': Shield,
  'heart': Heart,
  'zap': Zap,
  'book-open': BookOpen,
  'star': Star,
  'layers': Layers,
  'compass': Compass,
  'target': Target,
  'lightbulb': Lightbulb,
  'globe': Globe,
  'palette': Palette,
  'cpu': Cpu,
  'leaf': Leaf,
  'music': Music,
  'anchor': Anchor,
}

export const NAMESPACE_ICONS = Object.keys(ICON_MAP)

export const NAMESPACE_COLORS = [
  'slate', 'red', 'orange', 'amber', 'green',
  'teal', 'blue', 'indigo', 'purple', 'pink',
] as const

export type NamespaceColor = typeof NAMESPACE_COLORS[number]

/** Tailwind color classes for each namespace color. */
const COLOR_CLASSES: Record<string, { text: string; bg: string; ring: string }> = {
  slate:  { text: 'text-slate-500',  bg: 'bg-slate-500',  ring: 'ring-slate-500' },
  red:    { text: 'text-red-500',    bg: 'bg-red-500',    ring: 'ring-red-500' },
  orange: { text: 'text-orange-500', bg: 'bg-orange-500', ring: 'ring-orange-500' },
  amber:  { text: 'text-amber-500',  bg: 'bg-amber-500',  ring: 'ring-amber-500' },
  green:  { text: 'text-green-500',  bg: 'bg-green-500',  ring: 'ring-green-500' },
  teal:   { text: 'text-teal-500',   bg: 'bg-teal-500',   ring: 'ring-teal-500' },
  blue:   { text: 'text-blue-500',   bg: 'bg-blue-500',   ring: 'ring-blue-500' },
  indigo: { text: 'text-indigo-500', bg: 'bg-indigo-500', ring: 'ring-indigo-500' },
  purple: { text: 'text-purple-500', bg: 'bg-purple-500', ring: 'ring-purple-500' },
  pink:   { text: 'text-pink-500',   bg: 'bg-pink-500',   ring: 'ring-pink-500' },
}

export function getColorClasses(color: string) {
  return COLOR_CLASSES[color] ?? COLOR_CLASSES.blue
}

interface NamespaceIconProps {
  icon: string
  color?: string
  className?: string
}

export function NamespaceIcon({ icon, color = 'slate', className = 'h-4 w-4' }: NamespaceIconProps) {
  const IconComponent = ICON_MAP[icon] ?? Building2
  const colorClass = getColorClasses(color).text
  return <IconComponent className={`${className} ${colorClass}`} />
}

export function getIconComponent(icon: string): LucideIcon {
  return ICON_MAP[icon] ?? Building2
}

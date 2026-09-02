import type { ComponentType } from 'react'

interface Tab {
  key: string
  label: string
  icon?: ComponentType<{ className?: string }>
}

interface TabsProps {
  tabs: Tab[]
  activeTab: string
  onTabChange: (key: string) => void
}

export function Tabs({ tabs, activeTab, onTabChange }: TabsProps) {
  return (
    <div className="flex overflow-x-auto overflow-y-hidden border-b border-[var(--border)]">
      {tabs.map((tab) => {
        const isActive = tab.key === activeTab
        const Icon = tab.icon
        return (
          <button
            key={tab.key}
            onClick={() => onTabChange(tab.key)}
            className={`px-4 py-2 text-sm font-medium transition-colors -mb-px inline-flex items-center gap-1.5 shrink-0 whitespace-nowrap ${
              isActive
                ? 'text-[var(--primary)] border-b-2 border-[var(--primary)] font-semibold'
                : 'text-[var(--foreground-secondary)] hover:text-[var(--foreground)] border-b-2 border-transparent'
            }`}
          >
            {Icon && <Icon className="h-3.5 w-3.5" />}
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}

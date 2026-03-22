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
    <div className="flex border-b border-gray-200 dark:border-gray-700">
      {tabs.map((tab) => {
        const isActive = tab.key === activeTab
        const Icon = tab.icon
        return (
          <button
            key={tab.key}
            onClick={() => onTabChange(tab.key)}
            className={`px-4 py-2 text-sm font-medium transition-colors -mb-px inline-flex items-center gap-1.5 ${
              isActive
                ? 'text-indigo-600 dark:text-indigo-400 border-b-2 border-indigo-600 dark:border-indigo-400 font-semibold'
                : 'text-gray-500 dark:text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 border-b-2 border-transparent'
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

import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { Toggle } from '@/components/ui/Toggle'
import { Button } from '@/components/ui/Button'

interface ExpandableConfigCardProps {
  title: string
  description: string
  enabled: boolean
  onToggle: (enabled: boolean) => void
  toggleDisabled?: boolean
  expanded: boolean
  onToggleExpand: () => void
  children: ReactNode
  onSave?: () => void
  onCancel?: () => void
  canSave?: boolean
  saving?: boolean
  saved?: boolean
  savedMessage?: string
  error?: string
  extraActions?: ReactNode
}

export function ExpandableConfigCard({
  title,
  description,
  enabled,
  onToggle,
  toggleDisabled,
  expanded,
  onToggleExpand,
  children,
  onSave,
  onCancel,
  canSave = false,
  saving = false,
  saved = false,
  savedMessage,
  error,
  extraActions,
}: ExpandableConfigCardProps) {
  const { t } = useTranslation()

  return (
    <div className="rounded-lg border border-gray-200 dark:border-gray-700">
      {/* Header — always visible */}
      <div className="flex items-center justify-between p-6">
        <button
          type="button"
          className="flex flex-1 items-center gap-2 text-left"
          onClick={onToggleExpand}
        >
          {expanded ? (
            <ChevronUp className="h-4 w-4 shrink-0 text-gray-400" />
          ) : (
            <ChevronDown className="h-4 w-4 shrink-0 text-gray-400" />
          )}
          <div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
              {title}
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {description}
            </p>
          </div>
        </button>
        <div className="ml-4 shrink-0" onClick={(e) => e.stopPropagation()}>
          <Toggle enabled={enabled} onChange={onToggle} disabled={toggleDisabled} />
        </div>
      </div>

      {/* Expandable body */}
      {expanded && (
        <div className="border-t border-gray-200 dark:border-gray-700 px-6 pb-6 pt-4">
          <div className="space-y-4">{children}</div>

          {/* Actions */}
          {(onSave || extraActions) && (
            <div className="mt-6 flex flex-wrap items-center gap-3">
              {onSave && (
                <Button onClick={onSave} disabled={!canSave || saving}>
                  {saving ? t('common.saving') : t('common.save')}
                </Button>
              )}
              {onCancel && (
                <Button variant="secondary" onClick={onCancel}>
                  {t('common.cancel')}
                </Button>
              )}
              {extraActions}
              {saved && (
                <span className="text-sm text-green-600 dark:text-green-400">
                  {savedMessage ?? t('common.saved')}
                </span>
              )}
              {error && (
                <span className="text-sm text-red-600 dark:text-red-400">
                  {error}
                </span>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

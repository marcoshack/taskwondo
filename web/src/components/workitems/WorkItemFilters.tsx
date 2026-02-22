import { useRef, useState } from 'react'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useTranslation } from 'react-i18next'
import { SlidersHorizontal, ArrowUpDown, Settings, X } from 'lucide-react'
import { Input } from '@/components/ui/Input'
import { Tooltip } from '@/components/ui/Tooltip'
import { Modal } from '@/components/ui/Modal'
import { MultiSelect, type MultiSelectOption } from '@/components/ui/MultiSelect'
import { Select } from '@/components/ui/Select'
import type { WorkItemFilter } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'
import type { Milestone } from '@/api/milestones'

interface WorkItemFiltersProps {
  filter: WorkItemFilter
  onFilterChange: (filter: WorkItemFilter) => void
  statuses: WorkflowStatus[]
  milestones?: Milestone[]
  search: string
  onSearchChange: (value: string) => void
  sort?: string
  order?: 'asc' | 'desc'
  onSort?: (sortKey: string) => void
  onOrderChange?: (order: 'asc' | 'desc') => void
  showDates?: boolean
  onShowDatesChange?: (value: boolean) => void
}

const closedCategories = new Set(['done', 'cancelled'])

export function WorkItemFilters({ filter, onFilterChange, statuses, milestones = [], search, onSearchChange, sort, order, onSort, onOrderChange, showDates, onShowDatesChange }: WorkItemFiltersProps) {
  const { t } = useTranslation()
  const searchRef = useRef<HTMLInputElement>(null)
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [sortOpen, setSortOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)

  const typeOptions: MultiSelectOption[] = [
    { value: 'task', label: t('workitems.types.task') },
    { value: 'ticket', label: t('workitems.types.ticket') },
    { value: 'bug', label: t('workitems.types.bug') },
    { value: 'feedback', label: t('workitems.types.feedback') },
    { value: 'epic', label: t('workitems.types.epic') },
  ]

  const priorityOptions: MultiSelectOption[] = [
    { value: 'critical', label: t('workitems.priorities.critical') },
    { value: 'high', label: t('workitems.priorities.high') },
    { value: 'medium', label: t('workitems.priorities.medium') },
    { value: 'low', label: t('workitems.priorities.low') },
  ]

  const assigneeOptions: MultiSelectOption[] = [
    { value: 'me', label: t('workitems.filters.assignedToMe') },
    { value: 'unassigned', label: t('workitems.filters.unassigned') },
  ]

  function buildStatusOptions(ss: WorkflowStatus[]): MultiSelectOption[] {
    const open = ss.filter((s) => !closedCategories.has(s.category))
    const closed = ss.filter((s) => closedCategories.has(s.category))
    return [...open, ...closed].map((s) => ({
      value: s.name,
      label: t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name }),
      group: closedCategories.has(s.category) ? t('workitems.filters.statusGroupClosed') : t('workitems.filters.statusGroupOpen'),
    }))
  }

  useKeyboardShortcut({ key: '/' }, () => searchRef.current?.focus())

  function setArray(key: 'type' | 'status' | 'priority' | 'assignee', values: string[]) {
    onFilterChange({ ...filter, [key]: values.length > 0 ? values : undefined, cursor: undefined })
  }

  const statusOptions = buildStatusOptions(statuses)

  const activeFilterCount =
    (filter.type?.length ? 1 : 0) +
    (filter.priority?.length ? 1 : 0) +
    (filter.status?.length ? 1 : 0) +
    (filter.assignee?.length ? 1 : 0) +
    (filter.milestone ? 1 : 0)

  const sortOptions: { key: string; label: string }[] = [
    { key: 'created_at', label: t('workitems.sort.created') },
    { key: 'updated_at', label: t('workitems.sort.updated') },
    { key: 'priority', label: t('workitems.sort.priority') },
    { key: 'title', label: t('workitems.sort.name') },
    { key: 'type', label: t('workitems.sort.type') },
    { key: 'status', label: t('workitems.sort.status') },
    { key: 'item_number', label: t('workitems.sort.number') },
    { key: 'sla_target_at', label: t('workitems.sort.sla') },
  ]

  const isDefaultSort = !sort || (sort === 'created_at' && order === 'desc')

  return (
    <>
      {/* Desktop: inline layout */}
      <div className="hidden sm:flex flex-wrap items-end gap-3">
        <div className="flex-1 min-w-[200px] relative">
          <Input
            ref={searchRef}
            placeholder={t('workitems.filters.search')}
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Escape') searchRef.current?.blur() }}
            className="pr-8"
          />
          {search && (
            <button
              onClick={() => { onSearchChange(''); searchRef.current?.focus() }}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              aria-label={t('common.clear')}
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
        <div className="w-36">
          <MultiSelect options={typeOptions} selected={filter.type ?? []} onChange={(v) => setArray('type', v)} placeholder={t('workitems.filters.allTypes')} />
        </div>
        <div className="w-36">
          <MultiSelect options={priorityOptions} selected={filter.priority ?? []} onChange={(v) => setArray('priority', v)} placeholder={t('workitems.filters.allPriorities')} />
        </div>
        <div className="w-40">
          <MultiSelect options={statusOptions} selected={filter.status ?? []} onChange={(v) => setArray('status', v)} placeholder={t('workitems.filters.allStatuses')} />
        </div>
        <div className="w-40">
          <MultiSelect options={assigneeOptions} selected={filter.assignee ?? []} onChange={(v) => setArray('assignee', v)} placeholder={t('workitems.filters.allAssignees')} />
        </div>
        {milestones.length > 0 && (
          <div className="w-40">
            <Select
              value={filter.milestone ?? ''}
              onChange={(e) => onFilterChange({ ...filter, milestone: e.target.value || undefined, cursor: undefined })}
              className={filter.milestone ? '' : 'text-gray-500 dark:text-gray-400'}
            >
              <option value="">{t('workitems.filters.allMilestones')}</option>
              <option value="none" className="text-gray-900 dark:text-gray-100">{t('workitems.filters.noMilestone')}</option>
              {milestones.filter((m) => m.status === 'open').map((m) => (
                <option key={m.id} value={m.id} className="text-gray-900 dark:text-gray-100">{m.name}</option>
              ))}
              {milestones.some((m) => m.status === 'closed') && (
                <optgroup label={t('milestones.statusClosed')}>
                  {milestones.filter((m) => m.status === 'closed').map((m) => (
                    <option key={m.id} value={m.id} className="text-gray-900 dark:text-gray-100">{m.name}</option>
                  ))}
                </optgroup>
              )}
            </Select>
          </div>
        )}
      </div>

      {/* Mobile: search + sort icon + filter icon */}
      <div className="flex sm:hidden items-center gap-2">
        <div className="flex-1 min-w-0 relative">
          <Input
            placeholder={t('workitems.filters.search')}
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            className="pr-8"
          />
          {search && (
            <button
              onClick={() => onSearchChange('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              aria-label={t('common.clear')}
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
        {onSort && (
          <button
            onClick={() => setSortOpen(true)}
            className="relative shrink-0 p-2.5 rounded-md border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
            aria-label={t('workitems.sort.title')}
          >
            <ArrowUpDown className="h-5 w-5" />
            {!isDefaultSort && (
              <span className="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-indigo-600 text-[10px] font-bold text-white">
                !
              </span>
            )}
          </button>
        )}
        <button
          onClick={() => setFiltersOpen(true)}
          className="relative shrink-0 p-2.5 rounded-md border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
          aria-label={t('workitems.filters.title')}
        >
          <SlidersHorizontal className="h-5 w-5" />
          {activeFilterCount > 0 && (
            <span className="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-indigo-600 text-[10px] font-bold text-white">
              {activeFilterCount}
            </span>
          )}
        </button>
        {onShowDatesChange && (
          <button
            onClick={() => setSettingsOpen(true)}
            className="relative shrink-0 p-2.5 rounded-md border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
            aria-label={t('workitems.settings.title')}
          >
            <Settings className="h-5 w-5" />
          </button>
        )}
      </div>

      {/* Mobile sort modal */}
      {onSort && onOrderChange && (
        <Modal open={sortOpen} onClose={() => setSortOpen(false)} title={t('workitems.sort.title')}>
          <div className="space-y-1">
            {sortOptions.map((opt) => {
              const isActive = sort === opt.key
              return (
                <div key={opt.key} className="flex items-center gap-1">
                  <button
                    onClick={() => onSort(opt.key)}
                    className={`flex-1 text-left px-3 py-2.5 rounded-md text-sm ${
                      isActive
                        ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300 font-medium'
                        : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
                    }`}
                  >
                    {opt.label}
                  </button>
                  {isActive && (
                    <div className="flex shrink-0">
                      <Tooltip content={t('workitems.sort.asc')}>
                        <button
                          onClick={() => onOrderChange('asc')}
                          className={`px-3 py-2.5 rounded-l-md text-base font-medium border ${
                            order === 'asc'
                              ? 'bg-indigo-100 text-indigo-700 border-indigo-300 dark:bg-indigo-900/40 dark:text-indigo-300 dark:border-indigo-700'
                              : 'bg-white text-gray-500 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-600 dark:hover:bg-gray-700'
                          }`}
                        >
                          {'\u2191'}
                        </button>
                      </Tooltip>
                      <Tooltip content={t('workitems.sort.desc')}>
                        <button
                          onClick={() => onOrderChange('desc')}
                          className={`px-3 py-2.5 rounded-r-md text-base font-medium border-t border-r border-b ${
                            order === 'desc'
                              ? 'bg-indigo-100 text-indigo-700 border-indigo-300 dark:bg-indigo-900/40 dark:text-indigo-300 dark:border-indigo-700'
                              : 'bg-white text-gray-500 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-600 dark:hover:bg-gray-700'
                          }`}
                        >
                          {'\u2193'}
                        </button>
                      </Tooltip>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </Modal>
      )}

      {/* Mobile settings modal */}
      {onShowDatesChange && (
        <Modal open={settingsOpen} onClose={() => setSettingsOpen(false)} title={t('workitems.settings.title')}>
          <label className="flex items-center justify-between cursor-pointer">
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('workitems.settings.showDates')}</span>
            <button
              type="button"
              role="switch"
              aria-checked={showDates}
              onClick={() => onShowDatesChange(!showDates)}
              className={`relative inline-flex h-6 w-11 shrink-0 rounded-full border-2 border-transparent transition-colors ${
                showDates ? 'bg-indigo-600' : 'bg-gray-200 dark:bg-gray-600'
              }`}
            >
              <span className={`pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform ${
                showDates ? 'translate-x-5' : 'translate-x-0'
              }`} />
            </button>
          </label>
        </Modal>
      )}

      {/* Mobile filter modal */}
      <Modal open={filtersOpen} onClose={() => setFiltersOpen(false)} title={t('workitems.filters.title')}>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.filters.allTypes')}</label>
            <MultiSelect options={typeOptions} selected={filter.type ?? []} onChange={(v) => setArray('type', v)} placeholder={t('workitems.filters.allTypes')} />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.filters.allPriorities')}</label>
            <MultiSelect options={priorityOptions} selected={filter.priority ?? []} onChange={(v) => setArray('priority', v)} placeholder={t('workitems.filters.allPriorities')} />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.filters.allStatuses')}</label>
            <MultiSelect options={statusOptions} selected={filter.status ?? []} onChange={(v) => setArray('status', v)} placeholder={t('workitems.filters.allStatuses')} />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.filters.allAssignees')}</label>
            <MultiSelect options={assigneeOptions} selected={filter.assignee ?? []} onChange={(v) => setArray('assignee', v)} placeholder={t('workitems.filters.allAssignees')} />
          </div>
          {milestones.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.form.milestone')}</label>
              <Select
                value={filter.milestone ?? ''}
                onChange={(e) => onFilterChange({ ...filter, milestone: e.target.value || undefined, cursor: undefined })}
                className={filter.milestone ? '' : 'text-gray-500 dark:text-gray-400'}
              >
                <option value="">{t('workitems.filters.allMilestones')}</option>
                <option value="none" className="text-gray-900 dark:text-gray-100">{t('workitems.filters.noMilestone')}</option>
                {milestones.filter((m) => m.status === 'open').map((m) => (
                  <option key={m.id} value={m.id} className="text-gray-900 dark:text-gray-100">{m.name}</option>
                ))}
                {milestones.some((m) => m.status === 'closed') && (
                  <optgroup label={t('milestones.statusClosed')}>
                    {milestones.filter((m) => m.status === 'closed').map((m) => (
                      <option key={m.id} value={m.id} className="text-gray-900 dark:text-gray-100">{m.name}</option>
                    ))}
                  </optgroup>
                )}
              </Select>
            </div>
          )}
        </div>
      </Modal>
    </>
  )
}

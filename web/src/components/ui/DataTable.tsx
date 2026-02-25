import { useEffect, useRef, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

export interface Column<T> {
  key: string
  header: string
  render: (row: T) => ReactNode
  className?: string
  width?: string
  sortKey?: string
}

interface DataTableProps<T> {
  columns: Column<T>[]
  data: T[]
  onRowClick?: (row: T) => void
  emptyMessage?: string
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
  onSort?: (sortKey: string) => void
  activeRowIndex?: number
}

function SortIndicator({ active, direction }: { active: boolean; direction?: 'asc' | 'desc' }) {
  if (!active) {
    return (
      <svg className="w-3 h-3 text-gray-300 dark:text-gray-600" viewBox="0 0 10 14" fill="currentColor">
        <path d="M5 0L9 5H1L5 0Z" />
        <path d="M5 14L1 9H9L5 14Z" />
      </svg>
    )
  }
  if (direction === 'asc') {
    return (
      <svg className="w-3 h-3 text-indigo-600 dark:text-indigo-400" viewBox="0 0 10 7" fill="currentColor">
        <path d="M5 0L10 7H0L5 0Z" />
      </svg>
    )
  }
  return (
    <svg className="w-3 h-3 text-indigo-600 dark:text-indigo-400" viewBox="0 0 10 7" fill="currentColor">
      <path d="M5 7L0 0H10L5 7Z" />
    </svg>
  )
}

export function DataTable<T>({
  columns, data, onRowClick, emptyMessage,
  sortBy, sortOrder, onSort, activeRowIndex,
}: DataTableProps<T>) {
  const { t } = useTranslation()
  const resolvedEmptyMessage = emptyMessage ?? t('common.noData')
  const activeRowRef = useRef<HTMLTableRowElement>(null)

  useEffect(() => {
    if (activeRowIndex != null && activeRowIndex >= 0) {
      activeRowRef.current?.scrollIntoView({ block: 'nearest' })
    }
  }, [activeRowIndex])

  return (
    <div className="overflow-hidden">
      <table className="w-full table-fixed sm:divide-y sm:divide-gray-200 sm:dark:divide-gray-700">
        <colgroup>
          {columns.map((col) => (
            <col key={col.key} className={col.className ?? ''} style={col.width ? { width: col.width } : undefined} />
          ))}
        </colgroup>
        <thead className="hidden sm:table-header-group bg-gray-50 dark:bg-gray-800">
          <tr>
            {columns.map((col) => {
              const isSortable = !!col.sortKey && !!onSort
              const isActive = col.sortKey === sortBy
              return (
                <th
                  key={col.key}
                  style={col.width ? { width: col.width } : undefined}
                  className={`px-3 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider whitespace-nowrap ${col.className ?? ''} ${isSortable ? 'cursor-pointer select-none hover:text-gray-700 dark:hover:text-gray-200' : ''}`}
                  onClick={isSortable ? () => onSort!(col.sortKey!) : undefined}
                >
                  <span className="inline-flex items-center gap-1">
                    {col.header}
                    {isSortable && <SortIndicator active={isActive} direction={isActive ? sortOrder : undefined} />}
                  </span>
                </th>
              )
            })}
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-900 sm:divide-y sm:divide-gray-200 sm:dark:divide-gray-700">
          {data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-6 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
                {resolvedEmptyMessage}
              </td>
            </tr>
          ) : (
            data.map((row, i) => {
              const isActive = activeRowIndex === i
              return (
              <tr
                key={i}
                ref={isActive ? activeRowRef : undefined}
                onClick={() => onRowClick?.(row)}
                className={`group ${onRowClick ? 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800' : ''} ${isActive ? 'ring-2 ring-inset ring-indigo-500 bg-indigo-50/50 dark:bg-indigo-900/20' : ''}`}
              >
                {columns.map((col) => (
                  <td key={col.key} className={`px-3 sm:px-6 py-4 whitespace-nowrap text-sm overflow-hidden ${col.className ?? ''}`}>
                    {col.render(row)}
                  </td>
                ))}
              </tr>
              )
            })
          )}
        </tbody>
      </table>
    </div>
  )
}

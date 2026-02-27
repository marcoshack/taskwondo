import { useEffect, useRef, useCallback, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

export interface Column<T> {
  key: string
  header: string
  render: (row: T) => ReactNode
  className?: string
  width?: string
  sortKey?: string
  resizable?: boolean
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
  resizable?: boolean
  columnWidths?: Record<string, number>
  onColumnResize?: (key: string, width: number) => void
  onColumnResetWidth?: (key: string) => void
}

const MIN_COL_WIDTH = 40

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
  resizable, columnWidths, onColumnResize, onColumnResetWidth,
}: DataTableProps<T>) {
  const { t } = useTranslation()
  const resolvedEmptyMessage = emptyMessage ?? t('common.noData')
  const activeRowRef = useRef<HTMLTableRowElement>(null)
  const resizingRef = useRef<{ key: string; startX: number; startWidth: number } | null>(null)
  const onResizeRef = useRef(onColumnResize)
  onResizeRef.current = onColumnResize

  useEffect(() => {
    if (activeRowIndex != null && activeRowIndex >= 0) {
      activeRowRef.current?.scrollIntoView({ block: 'nearest' })
    }
  }, [activeRowIndex])

  const handleResizeMove = useRef((e: MouseEvent) => {
    if (!resizingRef.current) return
    const { key, startX, startWidth } = resizingRef.current
    const diff = e.clientX - startX
    const newWidth = Math.max(MIN_COL_WIDTH, startWidth + diff)
    onResizeRef.current?.(key, newWidth)
  }).current

  const handleResizeEnd = useRef(() => {
    resizingRef.current = null
    document.removeEventListener('mousemove', handleResizeMove)
    document.removeEventListener('mouseup', handleResizeEnd)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    // Swallow the click event that follows mouseup to prevent sort from firing
    document.addEventListener('click', suppressClick, true)
  }).current

  const suppressClick = useRef((e: MouseEvent) => {
    e.stopPropagation()
    e.preventDefault()
    document.removeEventListener('click', suppressClick, true)
  }).current

  useEffect(() => {
    return () => {
      document.removeEventListener('mousemove', handleResizeMove)
      document.removeEventListener('mouseup', handleResizeEnd)
      document.removeEventListener('click', suppressClick, true)
    }
  }, [handleResizeMove, handleResizeEnd, suppressClick])

  const handleResizeStart = useCallback((e: React.MouseEvent, colKey: string) => {
    e.preventDefault()
    e.stopPropagation()
    const th = (e.target as HTMLElement).closest('th')
    if (!th) return
    const startWidth = th.getBoundingClientRect().width
    resizingRef.current = { key: colKey, startX: e.clientX, startWidth }
    document.addEventListener('mousemove', handleResizeMove)
    document.addEventListener('mouseup', handleResizeEnd)
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }, [handleResizeMove, handleResizeEnd])

  function getColStyle(col: Column<T>): React.CSSProperties | undefined {
    if (columnWidths?.[col.key]) return { width: columnWidths[col.key] }
    if (col.width) return { width: col.width }
    return undefined
  }

  function isColResizable(col: Column<T>): boolean {
    return !!resizable && col.resizable !== false
  }

  return (
    <div className="overflow-hidden">
      <table className="w-full table-fixed sm:divide-y sm:divide-gray-200 sm:dark:divide-gray-700">
        <colgroup>
          {columns.map((col) => (
            <col
              key={col.key}
              className={columnWidths?.[col.key] ? '' : (col.className ?? '')}
              style={getColStyle(col)}
            />
          ))}
        </colgroup>
        <thead className="hidden sm:table-header-group bg-gray-50 dark:bg-gray-800 group/thead">
          <tr>
            {columns.map((col) => {
              const isSortable = !!col.sortKey && !!onSort
              const isActive = col.sortKey === sortBy
              const colResizable = isColResizable(col)
              return (
                <th
                  key={col.key}
                  style={getColStyle(col)}
                  className={`px-3 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider whitespace-nowrap ${columnWidths?.[col.key] ? '' : (col.className ?? '')} ${isSortable ? 'cursor-pointer select-none hover:text-gray-700 dark:hover:text-gray-200' : ''} ${colResizable ? 'relative' : ''}`}
                  onClick={isSortable ? () => onSort!(col.sortKey!) : undefined}
                >
                  <span className="inline-flex items-center gap-1">
                    {col.header}
                    {isSortable && <SortIndicator active={isActive} direction={isActive ? sortOrder : undefined} />}
                  </span>
                  {colResizable && (
                    <div
                      className="absolute right-0.5 top-0 bottom-0 w-1.5 cursor-col-resize opacity-0 group-hover/thead:opacity-100 bg-indigo-300/40 hover:!bg-indigo-400/60 active:!bg-indigo-500/60 dark:bg-indigo-500/30 dark:hover:!bg-indigo-400/50 dark:active:!bg-indigo-500/50 transition-opacity z-10"
                      onMouseDown={(e) => handleResizeStart(e, col.key)}
                      onDoubleClick={(e) => {
                        e.stopPropagation()
                        onColumnResetWidth?.(col.key)
                      }}
                    />
                  )}
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
                  <td key={col.key} className={`px-3 sm:px-6 py-4 whitespace-nowrap text-sm overflow-hidden ${columnWidths?.[col.key] ? '' : (col.className ?? '')}`}>
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

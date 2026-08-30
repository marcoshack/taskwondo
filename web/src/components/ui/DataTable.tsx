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
  /** Hide this column on mobile (below sm breakpoint) */
  hiddenOnMobile?: boolean
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
  /** Show column headers on all screen sizes (default: hidden on mobile) */
  alwaysShowHeader?: boolean
}

const MIN_COL_WIDTH = 40

function SortIndicator({ active, direction }: { active: boolean; direction?: 'asc' | 'desc' }) {
  if (!active) {
    return (
      <svg className="w-3 h-3 text-[var(--foreground-muted)]" viewBox="0 0 10 14" fill="currentColor">
        <path d="M5 0L9 5H1L5 0Z" />
        <path d="M5 14L1 9H9L5 14Z" />
      </svg>
    )
  }
  if (direction === 'asc') {
    return (
      <svg className="w-3 h-3 text-[var(--primary)]" viewBox="0 0 10 7" fill="currentColor">
        <path d="M5 0L10 7H0L5 0Z" />
      </svg>
    )
  }
  return (
    <svg className="w-3 h-3 text-[var(--primary)]" viewBox="0 0 10 7" fill="currentColor">
      <path d="M5 7L0 0H10L5 7Z" />
    </svg>
  )
}

export function DataTable<T>({
  columns, data, onRowClick, emptyMessage,
  sortBy, sortOrder, onSort, activeRowIndex,
  resizable, columnWidths, onColumnResize, onColumnResetWidth,
  alwaysShowHeader,
}: DataTableProps<T>) {
  const { t } = useTranslation()
  const resolvedEmptyMessage = emptyMessage ?? t('common.noData')
  const activeRowRef = useRef<HTMLTableRowElement>(null)
  const resizingRef = useRef<{ key: string; startX: number; startWidth: number } | null>(null)
  const onResizeRef = useRef(onColumnResize)
  useEffect(() => { onResizeRef.current = onColumnResize }, [onColumnResize])

  useEffect(() => {
    if (activeRowIndex != null && activeRowIndex >= 0) {
      activeRowRef.current?.scrollIntoView({ block: 'nearest' })
    }
  }, [activeRowIndex])

  const handleResizeStart = useCallback((e: React.MouseEvent, colKey: string) => {
    e.preventDefault()
    e.stopPropagation()
    const th = (e.target as HTMLElement).closest('th')
    if (!th) return
    const startWidth = th.getBoundingClientRect().width
    resizingRef.current = { key: colKey, startX: e.clientX, startWidth }
    const handleMove = (e: MouseEvent) => {
      if (!resizingRef.current) return
      const { key, startX, startWidth } = resizingRef.current
      const diff = e.clientX - startX
      const newWidth = Math.max(MIN_COL_WIDTH, startWidth + diff)
      onResizeRef.current?.(key, newWidth)
    }
    const handleEnd = () => {
      resizingRef.current = null
      document.removeEventListener('mousemove', handleMove)
      document.removeEventListener('mouseup', handleEnd)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
      const suppress = (e: MouseEvent) => {
        e.stopPropagation()
        e.preventDefault()
        document.removeEventListener('click', suppress, true)
      }
      document.addEventListener('click', suppress, true)
    }
    document.addEventListener('mousemove', handleMove)
    document.addEventListener('mouseup', handleEnd)
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    return () => {
      document.removeEventListener('mousemove', handleMove)
      document.removeEventListener('mouseup', handleEnd)
    }
  }, [onColumnResize])

  function mobileHideClass(col: Column<T>): string {
    return col.hiddenOnMobile ? 'hidden sm:table-cell' : ''
  }

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
      <table className="w-full table-fixed sm:divide-y sm:divide-[var(--border)] sm:border-b sm:border-[var(--border)]">
        <colgroup>
          {columns.map((col) => (
            <col
              key={col.key}
              className={`${columnWidths?.[col.key] ? '' : (col.className ?? '')} ${mobileHideClass(col)}`}
              style={getColStyle(col)}
            />
          ))}
        </colgroup>
        <thead className={`${alwaysShowHeader ? '' : 'hidden sm:table-header-group'} bg-[var(--background-secondary)] group/thead`}>
          <tr>
            {columns.map((col) => {
              const isSortable = !!col.sortKey && !!onSort
              const isActive = col.sortKey === sortBy
              const colResizable = isColResizable(col)
              return (
                <th
                  key={col.key}
                  style={getColStyle(col)}
                  className={`px-3 sm:px-6 py-2.5 text-left text-xs font-medium text-[var(--foreground-secondary)] uppercase tracking-wider whitespace-nowrap ${columnWidths?.[col.key] ? '' : (col.className ?? '')} ${isSortable ? 'cursor-pointer select-none hover:text-[var(--foreground)]' : ''} ${colResizable ? 'relative' : ''} ${mobileHideClass(col)}`}
                  onClick={isSortable ? () => onSort!(col.sortKey!) : undefined}
                >
                  <span className="inline-flex items-center gap-1">
                    {col.header}
                    {isSortable && <SortIndicator active={isActive} direction={isActive ? sortOrder : undefined} />}
                  </span>
                  {colResizable && (
                    <div
                      className="absolute right-0.5 top-0 bottom-0 w-1.5 cursor-col-resize opacity-0 group-hover/thead:opacity-100 bg-[var(--primary)]/40 hover:!bg-[var(--primary)]/60 active:!bg-[var(--primary)]/80 transition-opacity z-10"
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
        <tbody className="bg-[var(--surface)] sm:divide-y sm:divide-[var(--border)]">
          {data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-6 py-10 text-center text-sm text-[var(--foreground-secondary)]">
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
                className={`group ${onRowClick ? 'cursor-pointer hover:bg-[var(--hover)]' : ''} ${isActive ? 'ring-2 ring-inset ring-[var(--primary)] bg-[var(--primary-muted)]' : ''}`}
              >
                {columns.map((col) => (
                  <td key={col.key} className={`px-3 sm:px-6 py-3 whitespace-nowrap text-sm overflow-hidden ${columnWidths?.[col.key] ? '' : (col.className ?? '')} ${mobileHideClass(col)}`}>
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

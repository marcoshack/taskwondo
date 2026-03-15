import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useRelations, useCreateRelation, useDeleteRelation } from '@/hooks/useWorkItems'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { Button } from '@/components/ui/Button'
import { WorkItemPicker } from '@/components/ui/WorkItemPicker'
import { Select } from '@/components/ui/Select'
import { Spinner } from '@/components/ui/Spinner'
import type { Relation } from '@/api/workitems'

const RELATION_TYPES = ['blocks', 'blocked_by', 'relates_to', 'duplicates', 'caused_by', 'parent_of', 'child_of']

/** Maps a relation type to its inverse key for translation lookup */
const INVERSE_TYPE_KEY: Record<string, string> = {
  blocks: 'blocked_by',
  blocked_by: 'blocks',
  parent_of: 'child_of',
  child_of: 'parent_of',
  relates_to: 'relates_to',
  duplicates: 'duplicated_by',
  caused_by: 'causes',
}

function isCompleted(category: string) {
  return category === 'done' || category === 'cancelled'
}

interface RelationListProps {
  projectKey: string
  itemNumber: number
  readOnly?: boolean
}

export function RelationList({ projectKey, itemNumber, readOnly = false }: RelationListProps) {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const { data: relations, isLoading } = useRelations(projectKey, itemNumber)

  function displayIdToPath(displayId: string) {
    const idx = displayId.lastIndexOf('-')
    if (idx < 0) return '#'
    return p(`/projects/${displayId.slice(0, idx)}/items/${displayId.slice(idx + 1)}`)
  }
  const createMutation = useCreateRelation(projectKey, itemNumber)
  const deleteMutation = useDeleteRelation(projectKey, itemNumber)

  const [targetId, setTargetId] = useState('')
  const [relationType, setRelationType] = useState('relates_to')

  const currentDisplayId = `${projectKey}-${itemNumber}`

  function getLinkedInfo(r: Relation) {
    const isSource = r.source_display_id === currentDisplayId
    return {
      displayId: isSource ? r.target_display_id : r.source_display_id,
      title: isSource ? r.target_title : r.source_title,
      statusCategory: isSource ? r.target_status_category : r.source_status_category,
      label: isSource
        ? t(`relations.types.${r.relation_type}`)
        : t(`relations.types.${INVERSE_TYPE_KEY[r.relation_type] ?? r.relation_type}`),
    }
  }

  /** Check if this relation represents a child (current item is parent) */
  function isChildRelation(r: Relation) {
    const isSource = r.source_display_id === currentDisplayId
    return (isSource && r.relation_type === 'parent_of') || (!isSource && r.relation_type === 'child_of')
  }

  const { children, others } = useMemo(() => {
    const childList: Relation[] = []
    const otherList: Relation[] = []
    for (const r of relations ?? []) {
      if (isChildRelation(r)) {
        childList.push(r)
      } else {
        otherList.push(r)
      }
    }
    return { children: childList, others: otherList }
  }, [relations, currentDisplayId])

  const childrenCompleted = children.filter((r) => {
    const linked = getLinkedInfo(r)
    return isCompleted(linked.statusCategory)
  }).length

  if (isLoading) return <Spinner size="sm" />

  function renderRelationRow(r: Relation) {
    const linked = getLinkedInfo(r)
    const done = isCompleted(linked.statusCategory)
    return (
      <div key={r.id} className="group/relation flex items-center justify-between text-sm py-1.5">
        <div className="flex items-center gap-2 min-w-0">
          <span className={`shrink-0 ${done ? 'text-gray-300 dark:text-gray-600' : 'text-gray-500 dark:text-gray-400'}`}>{linked.label}</span>
          <Link
            to={displayIdToPath(linked.displayId)}
            className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-bold transition-colors shrink-0 ${
              done
                ? 'bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-500'
                : 'bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 hover:bg-indigo-200 dark:hover:bg-indigo-900/60'
            }`}
          >
            {linked.displayId}
          </Link>
          <Link
            to={displayIdToPath(linked.displayId)}
            className={`truncate transition-colors ${
              done
                ? 'line-through text-gray-400 dark:text-gray-500'
                : 'text-gray-700 dark:text-gray-300 hover:text-indigo-600 dark:hover:text-indigo-400'
            }`}
          >
            {linked.title}
          </Link>
        </div>
        {!readOnly && (
          <button
            className="group/del relative inline-flex items-center justify-center w-7 h-7 rounded-md text-red-400 hover:text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/30 transition-colors sm:opacity-0 sm:group-hover/relation:opacity-100 shrink-0 ml-2"
            onClick={() => deleteMutation.mutate(r.id)}
            aria-label={t('common.remove')}
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
              <path strokeLinecap="round" strokeLinejoin="round" d="M3 4.5h10M6.5 4.5V3a1 1 0 011-1h1a1 1 0 011 1v1.5M5 4.5v8a1 1 0 001 1h4a1 1 0 001-1v-8" />
            </svg>
            <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 px-2 py-1 text-xs text-white bg-gray-900 dark:bg-gray-700 rounded whitespace-nowrap opacity-0 group-hover/del:opacity-100 transition-opacity">
              {t('common.remove')}
            </span>
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {!readOnly && (
        <div className="flex flex-col sm:flex-row sm:items-center gap-2 pb-3 border-b border-gray-100 dark:border-gray-700">
          <div className="sm:flex-1">
            <WorkItemPicker
              projectKey={projectKey}
              excludeItemNumber={itemNumber}
              value={targetId}
              onChange={setTargetId}
              onSelect={setTargetId}
            />
          </div>
          <div className="flex items-center gap-2">
            <div className="flex-1 sm:w-40 sm:flex-none">
              <Select value={relationType} onChange={(e) => setRelationType(e.target.value)}>
                {RELATION_TYPES.map((tp) => (
                  <option key={tp} value={tp}>{t(`relations.types.${tp}`)}</option>
                ))}
              </Select>
            </div>
            <Button
              className="py-2 text-sm shrink-0"
              onClick={() => {
                createMutation.mutate({ targetDisplayId: targetId, relationType }, {
                  onSuccess: () => setTargetId(''),
                })
              }}
              disabled={!targetId.trim() || createMutation.isPending}
            >
              {t('common.add')}
            </Button>
          </div>
        </div>
      )}

      {/* Children section */}
      {children.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <h4 className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
              {t('relations.children')}
            </h4>
            <span className="text-xs text-gray-400 dark:text-gray-500">
              {t('relations.childrenProgress', { completed: childrenCompleted, total: children.length })}
            </span>
          </div>
          <div className="mb-2">
            <div className="h-1.5 w-full rounded-full bg-gray-200 dark:bg-gray-700 overflow-hidden">
              <div
                className="h-full rounded-full bg-green-500 dark:bg-green-400 transition-all duration-300"
                style={{ width: `${children.length > 0 ? (childrenCompleted / children.length) * 100 : 0}%` }}
              />
            </div>
          </div>
          <div className="space-y-1">
            {children.map(renderRelationRow)}
          </div>
        </div>
      )}

      {/* Other relations section */}
      {others.length > 0 && (
        <div>
          {children.length > 0 && (
            <h4 className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400 mb-2">
              {t('relations.otherRelations')}
            </h4>
          )}
          <div className="space-y-1">
            {others.map(renderRelationRow)}
          </div>
        </div>
      )}
    </div>
  )
}

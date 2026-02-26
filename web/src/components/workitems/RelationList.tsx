import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useRelations, useCreateRelation, useDeleteRelation } from '@/hooks/useWorkItems'
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

function displayIdToPath(displayId: string) {
  const idx = displayId.lastIndexOf('-')
  if (idx < 0) return '#'
  return `/projects/${displayId.slice(0, idx)}/items/${displayId.slice(idx + 1)}`
}

interface RelationListProps {
  projectKey: string
  itemNumber: number
  readOnly?: boolean
}

export function RelationList({ projectKey, itemNumber, readOnly = false }: RelationListProps) {
  const { t } = useTranslation()
  const { data: relations, isLoading } = useRelations(projectKey, itemNumber)
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
      label: isSource
        ? t(`relations.types.${r.relation_type}`)
        : t(`relations.types.${INVERSE_TYPE_KEY[r.relation_type] ?? r.relation_type}`),
    }
  }

  if (isLoading) return <Spinner size="sm" />

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

      <div className="space-y-1">
        {(relations ?? []).map((r) => {
          const linked = getLinkedInfo(r)
          return (
            <div key={r.id} className="flex items-center justify-between text-sm py-1.5">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-gray-500 dark:text-gray-400 shrink-0">{linked.label}</span>
                <Link
                  to={displayIdToPath(linked.displayId)}
                  className="inline-flex items-center rounded-md bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 px-2 py-0.5 text-xs font-bold hover:bg-indigo-200 dark:hover:bg-indigo-900/60 transition-colors shrink-0"
                >
                  {linked.displayId}
                </Link>
                <Link
                  to={displayIdToPath(linked.displayId)}
                  className="text-gray-700 dark:text-gray-300 truncate hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors"
                >
                  {linked.title}
                </Link>
              </div>
              {!readOnly && (
                <button
                  className="text-xs text-red-400 hover:text-red-600 dark:hover:text-red-300 shrink-0 ml-2"
                  onClick={() => deleteMutation.mutate(r.id)}
                >
                  {t('common.remove')}
                </button>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

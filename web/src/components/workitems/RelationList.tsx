import { useState } from 'react'
import { useRelations, useCreateRelation, useDeleteRelation } from '@/hooks/useWorkItems'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Spinner } from '@/components/ui/Spinner'

const RELATION_TYPES = ['blocks', 'blocked_by', 'relates_to', 'duplicates', 'caused_by', 'parent_of', 'child_of']

interface RelationListProps {
  projectKey: string
  itemNumber: number
}

export function RelationList({ projectKey, itemNumber }: RelationListProps) {
  const { data: relations, isLoading } = useRelations(projectKey, itemNumber)
  const createMutation = useCreateRelation(projectKey, itemNumber)
  const deleteMutation = useDeleteRelation(projectKey, itemNumber)

  const [targetId, setTargetId] = useState('')
  const [relationType, setRelationType] = useState('relates_to')

  if (isLoading) return <Spinner size="sm" />

  return (
    <div className="space-y-3">
      {(relations ?? []).map((r) => (
        <div key={r.id} className="flex items-center justify-between text-sm py-1">
          <div>
            <span className="text-gray-500 dark:text-gray-400">{r.relation_type.replace(/_/g, ' ')}</span>
            {' '}
            <span className="font-mono font-medium text-indigo-600 dark:text-indigo-400">{r.target_display_id}</span>
          </div>
          <button
            className="text-xs text-red-400 hover:text-red-600 dark:hover:text-red-300"
            onClick={() => deleteMutation.mutate(r.id)}
          >
            Remove
          </button>
        </div>
      ))}

      <div className="flex items-end gap-2 pt-2 border-t border-gray-100 dark:border-gray-700">
        <div className="flex-1">
          <Input
            placeholder="TARGET-123"
            value={targetId}
            onChange={(e) => setTargetId(e.target.value)}
          />
        </div>
        <div className="w-40">
          <Select value={relationType} onChange={(e) => setRelationType(e.target.value)}>
            {RELATION_TYPES.map((t) => (
              <option key={t} value={t}>{t.replace(/_/g, ' ')}</option>
            ))}
          </Select>
        </div>
        <Button
          size="sm"
          onClick={() => {
            createMutation.mutate({ targetDisplayId: targetId, relationType }, {
              onSuccess: () => setTargetId(''),
            })
          }}
          disabled={!targetId.trim() || createMutation.isPending}
        >
          Add
        </Button>
      </div>
    </div>
  )
}

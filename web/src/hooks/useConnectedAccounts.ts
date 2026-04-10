import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listConnectedAccounts, unlinkConnectedAccount } from '@/api/auth'

export function useConnectedAccounts() {
  return useQuery({
    queryKey: ['connectedAccounts'],
    queryFn: listConnectedAccounts,
  })
}

export function useUnlinkConnectedAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => unlinkConnectedAccount(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['connectedAccounts'] }),
  })
}

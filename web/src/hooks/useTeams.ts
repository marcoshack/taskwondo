import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listTeams,
  getTeam,
  createTeam,
  updateTeam,
  deleteTeam,
  listTeamMembers,
  addTeamMember,
  removeTeamMember,
  type CreateTeamInput,
  type UpdateTeamInput,
} from '@/api/teams'

export function useTeams(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'teams'],
    queryFn: () => listTeams(projectKey),
    enabled: !!projectKey,
  })
}

export function useTeam(projectKey: string, teamId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'teams', teamId],
    queryFn: () => getTeam(projectKey, teamId),
    enabled: !!projectKey && !!teamId,
  })
}

export function useTeamMembers(projectKey: string, teamId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'teams', teamId, 'members'],
    queryFn: () => listTeamMembers(projectKey, teamId),
    enabled: !!projectKey && !!teamId,
  })
}

export function useCreateTeam(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateTeamInput) => createTeam(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams'] })
    },
  })
}

export function useUpdateTeam(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ teamId, input }: { teamId: string; input: UpdateTeamInput }) =>
      updateTeam(projectKey, teamId, input),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', vars.teamId] })
    },
  })
}

export function useDeleteTeam(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (teamId: string) => deleteTeam(projectKey, teamId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams'] })
    },
  })
}

export function useAddTeamMember(projectKey: string, teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => addTeamMember(projectKey, teamId, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', teamId, 'members'] })
    },
  })
}

export function useRemoveTeamMember(projectKey: string, teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => removeTeamMember(projectKey, teamId, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', teamId, 'members'] })
    },
  })
}

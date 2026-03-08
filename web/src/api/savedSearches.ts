import { api, nsPrefix } from './client'

// --- Types ---

export interface SavedSearchFilters {
  q?: string
  type?: string[]
  status?: string[]
  priority?: string[]
  assignee?: string[]
  milestone?: string[]
}

export interface SavedSearch {
  id: string
  project_id: string
  user_id: string | null
  scope: 'user' | 'shared'
  name: string
  filters: SavedSearchFilters
  view_mode: 'list' | 'board'
  position: number
  created_at: string
  updated_at: string
}

export interface CreateSavedSearchInput {
  name: string
  filters: SavedSearchFilters
  view_mode: string
  shared: boolean
}

export interface UpdateSavedSearchInput {
  name?: string
  filters?: SavedSearchFilters
  view_mode?: string
  position?: number
}

// --- API Functions ---

export async function listSavedSearches(projectKey: string): Promise<SavedSearch[]> {
  const res = await api.get<{ data: SavedSearch[] }>(`${nsPrefix()}/projects/${projectKey}/saved-searches`)
  return res.data.data
}

export async function createSavedSearch(projectKey: string, input: CreateSavedSearchInput): Promise<SavedSearch> {
  const res = await api.post<{ data: SavedSearch }>(`${nsPrefix()}/projects/${projectKey}/saved-searches`, input)
  return res.data.data
}

export async function updateSavedSearch(projectKey: string, searchId: string, input: UpdateSavedSearchInput): Promise<SavedSearch> {
  const res = await api.patch<{ data: SavedSearch }>(`${nsPrefix()}/projects/${projectKey}/saved-searches/${searchId}`, input)
  return res.data.data
}

export async function deleteSavedSearch(projectKey: string, searchId: string): Promise<void> {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/saved-searches/${searchId}`)
}

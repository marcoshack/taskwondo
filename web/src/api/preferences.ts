import { api } from './client'

export interface UserPreference {
  key: string
  value: unknown
}

interface DataResponse<T> {
  data: T
}

export async function getPreferences(): Promise<UserPreference[]> {
  const res = await api.get<DataResponse<UserPreference[]>>('/user/preferences')
  return res.data.data ?? []
}

export async function getPreference(key: string): Promise<UserPreference> {
  const res = await api.get<DataResponse<UserPreference>>(`/user/preferences/${key}`)
  return res.data.data
}

export async function setPreference(key: string, value: unknown): Promise<UserPreference> {
  const res = await api.put<DataResponse<UserPreference>>(`/user/preferences/${key}`, { value })
  return res.data.data
}

export async function deletePreference(key: string): Promise<void> {
  await api.delete(`/user/preferences/${key}`)
}

import { api, nsPrefix } from './client'

export interface UserSetting {
  key: string
  value: unknown
}

interface DataResponse<T> {
  data: T
}

export async function getUserSettings(projectKey: string): Promise<UserSetting[]> {
  const res = await api.get<DataResponse<UserSetting[]>>(`${nsPrefix()}/projects/${projectKey}/user-settings`)
  return res.data.data ?? []
}

export async function getUserSetting(projectKey: string, key: string): Promise<UserSetting> {
  const res = await api.get<DataResponse<UserSetting>>(`${nsPrefix()}/projects/${projectKey}/user-settings/${key}`)
  return res.data.data
}

export async function setUserSetting(projectKey: string, key: string, value: unknown): Promise<UserSetting> {
  const res = await api.put<DataResponse<UserSetting>>(`${nsPrefix()}/projects/${projectKey}/user-settings/${key}`, { value })
  return res.data.data
}

export async function deleteUserSetting(projectKey: string, key: string): Promise<void> {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/user-settings/${key}`)
}

// Global preferences (not scoped to a project)

export async function getGlobalPreference(key: string): Promise<UserSetting> {
  const res = await api.get<DataResponse<UserSetting>>(`/user/preferences/${key}`)
  return res.data.data
}

export async function setGlobalPreference(key: string, value: unknown): Promise<UserSetting> {
  const res = await api.put<DataResponse<UserSetting>>(`/user/preferences/${key}`, { value })
  return res.data.data
}

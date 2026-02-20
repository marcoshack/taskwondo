import { api } from './client'
import axios from 'axios'

export interface SystemSetting {
  key: string
  value: unknown
}

interface DataResponse<T> {
  data: T
}

export async function getSystemSettings(): Promise<SystemSetting[]> {
  const res = await api.get<DataResponse<SystemSetting[]>>('/admin/settings')
  return res.data.data ?? []
}

export async function getSystemSetting(key: string): Promise<SystemSetting> {
  const res = await api.get<DataResponse<SystemSetting>>(`/admin/settings/${key}`)
  return res.data.data
}

export async function setSystemSetting(key: string, value: unknown): Promise<SystemSetting> {
  const res = await api.put<DataResponse<SystemSetting>>(`/admin/settings/${key}`, { value })
  return res.data.data
}

export async function deleteSystemSetting(key: string): Promise<void> {
  await api.delete(`/admin/settings/${key}`)
}

export async function getPublicSettings(): Promise<Record<string, unknown>> {
  const res = await axios.get<DataResponse<Record<string, unknown>>>('/api/v1/settings/public')
  return res.data.data ?? {}
}

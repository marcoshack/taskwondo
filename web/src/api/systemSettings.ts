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

// SMTP Configuration

export interface SMTPConfig {
  enabled: boolean
  smtp_host: string
  smtp_port: number
  imap_host: string
  imap_port: number
  username: string
  password: string
  encryption: 'starttls' | 'tls' | 'none'
  from_address: string
  from_name: string
}

export async function getSMTPConfig(): Promise<SMTPConfig> {
  const res = await api.get<DataResponse<SMTPConfig>>('/admin/settings/smtp_config')
  return res.data.data
}

export async function setSMTPConfig(config: SMTPConfig): Promise<SMTPConfig> {
  const res = await api.put<DataResponse<SMTPConfig>>('/admin/settings/smtp_config', config)
  return res.data.data
}

export async function testSMTPConfig(): Promise<{ message: string }> {
  const res = await api.post<DataResponse<{ message: string }>>('/admin/settings/smtp_config/test')
  return res.data.data
}

// OAuth Provider Configuration

export interface OAuthProviderConfig {
  client_id: string
  client_secret: string
}

export async function getOAuthConfig(provider: string): Promise<OAuthProviderConfig> {
  const res = await api.get<DataResponse<OAuthProviderConfig>>(`/admin/settings/oauth_config/${provider}`)
  return res.data.data
}

export async function setOAuthConfig(provider: string, config: OAuthProviderConfig): Promise<OAuthProviderConfig> {
  const res = await api.put<DataResponse<OAuthProviderConfig>>(`/admin/settings/oauth_config/${provider}`, config)
  return res.data.data
}

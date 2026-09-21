import type { TFunction } from 'i18next'
import { isAxiosError } from 'axios'

interface OAuthErrorBody {
  error?: {
    code?: string
    error_key?: string
    message?: string
  }
}

/**
 * Localise a failed `PUT /admin/settings/oauth_config/{provider}`.
 *
 * The API rejects invalid configs as
 * `{ code: 'VALIDATION_ERROR', message: 'validation error: issuer must use https' }`
 * — the sentinel prefix followed by the field-level detail. The detail is
 * English server text, so we translate the wrapper and append it only when no
 * dedicated `errors.<error_key>` entry exists, otherwise the raw message would
 * leak untranslated server internals into the admin UI.
 */
export function getOAuthConfigError(err: unknown, t: TFunction): string {
  const fallback = t('admin.authentication.oauth.saveError')
  if (!isAxiosError<OAuthErrorBody>(err)) return fallback

  const body = err.response?.data?.error
  if (!body) return fallback

  if (body.error_key) {
    const keyed = t(`errors.${body.error_key}`)
    if (keyed !== `errors.${body.error_key}`) return keyed
  }

  const message = body.message?.trim()
  if (!message) return fallback

  if (body.code === 'VALIDATION_ERROR') {
    const detail = message.replace(/^validation error:\s*/, '')
    if (detail) return t('admin.authentication.oauth.errorValidation', { detail })
  }
  return fallback
}

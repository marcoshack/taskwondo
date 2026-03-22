import { isAxiosError } from 'axios'
import type { TFunction } from 'i18next'

interface APIErrorBody {
  error?: {
    code?: string
    error_key?: string
    params?: Record<string, string>
    message?: string
  }
}

/**
 * Extract a localized error message from an API error response.
 *
 * Resolution order:
 * 1. If the response contains `error_key`, look up `errors.<error_key>` in i18n
 *    with interpolation params. If the key exists, return the translated string.
 * 2. Fall back to the raw `error.message` from the API response.
 * 3. Fall back to the provided `fallbackKey` translation.
 */
export function getLocalizedError(err: unknown, t: TFunction, fallbackKey: string): string {
  if (!isAxiosError<APIErrorBody>(err) || !err.response?.data?.error) {
    return t(fallbackKey)
  }

  const { error_key, params, message } = err.response.data.error

  if (error_key) {
    const i18nKey = `errors.${error_key}`
    const translated = t(i18nKey, params ?? {})
    // i18next returns the key itself if no translation exists
    if (translated !== i18nKey) {
      return translated
    }
  }

  return message || t(fallbackKey)
}

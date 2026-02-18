import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'
import { LANGUAGE_KEY } from '@/i18n'

export type Language = 'en' | 'pt' | 'es'

interface LanguageContextValue {
  language: Language
  setLanguage: (lang: Language) => void
  availableLanguages: { value: Language; label: string; nativeLabel: string }[]
}

const SUPPORTED_LANGUAGES: LanguageContextValue['availableLanguages'] = [
  { value: 'en', label: 'English', nativeLabel: 'English' },
  { value: 'pt', label: 'Portuguese', nativeLabel: 'Português' },
  { value: 'es', label: 'Spanish', nativeLabel: 'Español' },
]

const LanguageContext = createContext<LanguageContextValue | null>(null)

function isValidLanguage(v: unknown): v is Language {
  return SUPPORTED_LANGUAGES.some((l) => l.value === v)
}

function getStoredLanguage(): Language {
  const stored = localStorage.getItem(LANGUAGE_KEY)
  if (isValidLanguage(stored)) return stored
  return 'en'
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const { i18n } = useTranslation()
  const [language, setLanguageState] = useState<Language>(getStoredLanguage)

  const { data: apiLanguage } = usePreference<string>(user ? 'language' : '')
  const setPreferenceMutation = useSetPreference()

  useEffect(() => {
    if (isValidLanguage(apiLanguage)) {
      setLanguageState(apiLanguage)
      localStorage.setItem(LANGUAGE_KEY, apiLanguage)
      i18n.changeLanguage(apiLanguage)
    }
  }, [apiLanguage, i18n])

  useEffect(() => {
    if (i18n.language !== language) {
      i18n.changeLanguage(language)
    }
  }, [language, i18n])

  const setLanguage = useCallback(
    (newLang: Language) => {
      setLanguageState(newLang)
      localStorage.setItem(LANGUAGE_KEY, newLang)
      i18n.changeLanguage(newLang)
      if (user) {
        setPreferenceMutation.mutate({ key: 'language', value: newLang })
      }
    },
    [user, setPreferenceMutation, i18n],
  )

  return (
    <LanguageContext.Provider value={{ language, setLanguage, availableLanguages: SUPPORTED_LANGUAGES }}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useLanguage() {
  const ctx = useContext(LanguageContext)
  if (!ctx) throw new Error('useLanguage must be used within LanguageProvider')
  return ctx
}

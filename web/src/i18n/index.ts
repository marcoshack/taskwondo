import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import en from './en.json'
import pt from './pt.json'
import es from './es.json'
import fr from './fr.json'
import de from './de.json'
import ja from './ja.json'
import ko from './ko.json'
import zh from './zh.json'
import ar from './ar.json'

export const LANGUAGE_KEY = 'trackforge_language'

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      pt: { translation: pt },
      es: { translation: es },
      fr: { translation: fr },
      de: { translation: de },
      ja: { translation: ja },
      ko: { translation: ko },
      zh: { translation: zh },
      ar: { translation: ar },
    },
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: LANGUAGE_KEY,
      caches: ['localStorage'],
    },
  })

export default i18n

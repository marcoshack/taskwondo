import { describe, it, expect } from 'vitest'
import en from './en.json'
import ar from './ar.json'
import de from './de.json'
import es from './es.json'
import fr from './fr.json'
import ja from './ja.json'
import ko from './ko.json'
import pt from './pt.json'
import zh from './zh.json'

const translations: Record<string, Record<string, string>> = {
  ar, de, es, fr, ja, ko, pt, zh,
}

const enKeys = Object.keys(en).sort()

describe('i18n completeness', () => {
  for (const [lang, data] of Object.entries(translations)) {
    describe(lang, () => {
      it('has no missing keys', () => {
        const langKeys = new Set(Object.keys(data))
        const missing = enKeys.filter((k) => !langKeys.has(k))
        expect(missing, `Missing keys in ${lang}`).toEqual([])
      })

      it('has no extra keys', () => {
        const enKeySet = new Set(enKeys)
        const extra = Object.keys(data).filter((k) => !enKeySet.has(k))
        expect(extra, `Extra keys in ${lang}`).toEqual([])
      })

      it('preserves interpolation placeholders', () => {
        const placeholderRe = /\{\{(\w+)\}\}/g
        const mismatched: string[] = []

        for (const key of enKeys) {
          if (!(key in data)) continue
          const enPlaceholders = [...(en as Record<string, string>)[key].matchAll(placeholderRe)]
            .map((m) => m[1])
            .sort()
          const langPlaceholders = [...data[key].matchAll(placeholderRe)]
            .map((m) => m[1])
            .sort()

          if (JSON.stringify(enPlaceholders) !== JSON.stringify(langPlaceholders)) {
            mismatched.push(
              `${key}: expected {{${enPlaceholders.join(', ')}}} but got {{${langPlaceholders.join(', ')}}}`
            )
          }
        }

        expect(mismatched, `Placeholder mismatches in ${lang}`).toEqual([])
      })

      it('preserves HTML-like tags for Trans component', () => {
        const tagRe = /<(\w+)>/g
        const mismatched: string[] = []

        for (const key of enKeys) {
          if (!(key in data)) continue
          const enTags = [...(en as Record<string, string>)[key].matchAll(tagRe)]
            .map((m) => m[1])
            .sort()
          const langTags = [...data[key].matchAll(tagRe)]
            .map((m) => m[1])
            .sort()

          if (JSON.stringify(enTags) !== JSON.stringify(langTags)) {
            mismatched.push(
              `${key}: expected <${enTags.join(', ')}> but got <${langTags.join(', ')}>`
            )
          }
        }

        expect(mismatched, `Tag mismatches in ${lang}`).toEqual([])
      })

      it('has no empty values', () => {
        const empty = Object.entries(data)
          .filter(([, v]) => v.trim() === '')
          .map(([k]) => k)
        expect(empty, `Empty values in ${lang}`).toEqual([])
      })
    })
  }
})

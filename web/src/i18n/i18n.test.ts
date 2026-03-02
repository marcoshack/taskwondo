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

// Keys where the translated value is legitimately the same as English.
// Includes brand names, technical terms, cognates, and format-only strings.
const SAME_VALUE_ALLOWED = new Set([
  // Brand / proper nouns
  'admin.authentication.discord.title',
  'admin.authentication.google.title',
  'admin.authentication.github.title',
  'admin.authentication.microsoft.title',
  'brand.name',

  // Placeholder examples (emails, domains, format strings)
  'admin.integrations.smtp.fromAddressPlaceholder',
  'admin.integrations.smtp.fromNamePlaceholder',
  'admin.integrations.smtp.imapHostPlaceholder',
  'admin.integrations.smtp.smtpHostPlaceholder',
  'admin.integrations.smtp.usernamePlaceholder',
  'admin.integrations.smtp.encryptionStarttls',
  'sla.durationPlaceholder',
  'timeTracking.durationPlaceholder',
  'projects.create.keyPlaceholder',

  // Format-only strings (only interpolation variables, no translatable words)
  'inbox.autoRefresh5s',
  'inbox.autoRefresh10s',
  'inbox.autoRefresh30s',
  'inbox.autoRefresh1m',
  'inbox.autoRefresh5m',
  'milestones.timeEstimated',
  'milestones.timeSpent',
  'milestones.timeProgress',
  'projects.limitCounter',
  'projects.overview.range_24h',
  'projects.overview.range_3d',
  'projects.overview.range_7d',
  'sla.mode24x7',

  // Technical terms / loanwords commonly kept in target languages
  'activity.fields.description',
  'activity.fields.labels',
  'activity.fields.status',
  'activity.fields.type',
  'admin.general.title',
  'admin.sidebar.general',
  'admin.sidebar.workflows',
  'admin.titleShort',
  'admin.users.name',
  'admin.users.status',
  'admin.workflows.title',
  'common.description',
  'common.system',
  'milestone.dashboard.totalItems',
  'milestones.name',
  'preferences.apiKeys.expiration',
  'preferences.apiKeys.permissions',
  'preferences.fontSizes.normal',
  'preferences.notifications.title',
  'preferences.sidebar.notifications',
  'preferences.themes.system',
  'projects.create.description',
  'projects.create.name',
  'projects.overview.bugs',
  'projects.overview.tickets',
  'projects.overview.total',
  'projects.settings.general',
  'projects.settings.workflows',
  'projects.table.name',
  'projects.table.total',
  'shortcuts.actions',
  'shortcuts.general',
  'shortcuts.navigation',
  'sidebar.menu',
  'sidebar.workflows',
  'sla.columnHeader',
  'sla.status',
  'tabs.relations',
  'timeTracking.date',
  'timeTracking.minutes',
  'user.sidebar.feed',
  'welcome.feed.title',
  'welcome.workflows.title',
  'workflows.title',
  'workflows.transitions',
  'workitems.detail.description',
  'workitems.filters.allStatuses',
  'workitems.filters.allTypes',
  'workitems.form.description',
  'workitems.form.labels',
  'workitems.form.status',
  'workitems.form.type',
  'workitems.sort.status',
  'workitems.sort.type',
  'workitems.statuses.backlog',
  'workitems.table.id',
  'workitems.table.status',
  'workitems.table.type',
  'workitems.types.bug',
  'workitems.types.epic',
  'workitems.types.feedback',
  'workitems.types.ticket',
  'workitems.view.board',
  'workitems.visibilities.portal',
  'workitems.visibilities.public',
])

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

      it('has no untranslated values (same as English)', () => {
        const untranslated = Object.entries(data)
          .filter(([k, v]) => (en as Record<string, string>)[k] === v && !SAME_VALUE_ALLOWED.has(k))
          .map(([k]) => k)
        expect(untranslated, `Untranslated keys in ${lang}`).toEqual([])
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

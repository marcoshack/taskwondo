import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'

interface ShortcutEntry {
  keys: string[]
  label: string
}

interface ShortcutCategory {
  title: string
  shortcuts: ShortcutEntry[]
}

function Kbd({ children }: { children: string }) {
  return (
    <kbd className="inline-flex items-center justify-center min-w-[1.5rem] px-1.5 py-0.5 text-xs font-mono font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-500 rounded shadow-sm">
      {children}
    </kbd>
  )
}

function ShortcutRow({ keys, label, thenLabel }: ShortcutEntry & { thenLabel: string }) {
  return (
    <div className="flex items-center justify-between py-1.5">
      <span className="text-sm text-gray-600 dark:text-gray-300">{label}</span>
      <span className="flex items-center gap-1 shrink-0 ml-4">
        {keys.map((k, i) => (
          <span key={i} className="flex items-center gap-1">
            {i > 0 && <span className="text-xs text-gray-400">{thenLabel}</span>}
            <Kbd>{k}</Kbd>
          </span>
        ))}
      </span>
    </div>
  )
}

export function KeyboardShortcutsModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation()

  const categories: ShortcutCategory[] = [
    {
      title: t('shortcuts.navigation'),
      shortcuts: [
        { keys: ['g', 'p'], label: t('shortcuts.navigation.switchProject') },
        { keys: ['g', 'o'], label: t('shortcuts.navigation.goToItems') },
        { keys: ['g', 'i'], label: t('shortcuts.navigation.goToInbox') },
        { keys: ['Ctrl', ','], label: t('shortcuts.navigation.preferences') },
        { keys: ['['], label: t('shortcuts.navigation.toggleSidebar') },
      ],
    },
    {
      title: t('shortcuts.lists'),
      shortcuts: [
        { keys: ['j / \u2193'], label: t('shortcuts.lists.moveDown') },
        { keys: ['k / \u2191'], label: t('shortcuts.lists.moveUp') },
        { keys: ['o / \u21B5'], label: t('shortcuts.lists.open') },
        { keys: ['x'], label: t('shortcuts.lists.select') },
        { keys: ['Esc'], label: t('shortcuts.lists.deselect') },
      ],
    },
    {
      title: t('shortcuts.actions'),
      shortcuts: [
        { keys: ['c'], label: t('shortcuts.actions.createItem') },
        { keys: ['n'], label: t('shortcuts.actions.createProject') },
        { keys: ['#'], label: t('shortcuts.actions.delete') },
        { keys: ['/'], label: t('shortcuts.actions.search') },
      ],
    },
    {
      title: t('shortcuts.inbox'),
      shortcuts: [
        { keys: ['j / \u2193'], label: t('shortcuts.inbox.moveDown') },
        { keys: ['k / \u2191'], label: t('shortcuts.inbox.moveUp') },
        { keys: ['o / \u21B5'], label: t('shortcuts.inbox.open') },
        { keys: ['x'], label: t('shortcuts.inbox.select') },
        { keys: ['Shift', '#'], label: t('shortcuts.inbox.remove') },
      ],
    },
    {
      title: t('shortcuts.editing'),
      shortcuts: [
        { keys: ['\u21B5'], label: t('shortcuts.editing.saveField') },
        { keys: ['Ctrl', '\u21B5'], label: t('shortcuts.editing.saveMultiline') },
        { keys: ['Esc'], label: t('shortcuts.editing.cancel') },
      ],
    },
    {
      title: t('shortcuts.general'),
      shortcuts: [
        { keys: ['?'], label: t('shortcuts.general.showShortcuts') },
        { keys: ['Esc'], label: t('shortcuts.general.closeModal') },
      ],
    },
  ]

  return (
    <Modal open={open} onClose={onClose} title={t('shortcuts.title')} className="!max-w-[57.6rem]">
      <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-8 gap-y-4">
        {categories.map((cat) => (
          <div key={cat.title}>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500 mb-1">
              {cat.title}
            </h3>
            <div className="divide-y divide-gray-100 dark:divide-gray-700">
              {cat.shortcuts.map((s, i) => (
                <ShortcutRow key={i} keys={s.keys} label={s.label} thenLabel={t('shortcuts.then')} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </Modal>
  )
}

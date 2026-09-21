import { Trans } from 'react-i18next'

export function PoweredByFooter() {
  return (
    <footer className="py-3 text-center">
      <p className="text-xs text-[var(--foreground-muted)]">
        <Trans i18nKey="footer.poweredBy" components={{ brand: <a href="https://github.com/marcoshack/taskwondo" target="_blank" rel="noopener noreferrer" className="font-medium hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)] underline-offset-2 hover:underline" /> }} />
      </p>
    </footer>
  )
}

import { Trans } from 'react-i18next'

export function PoweredByFooter() {
  return (
    <footer className="py-3 text-center">
      <p className="text-xs text-gray-400 dark:text-gray-500">
        <Trans i18nKey="footer.poweredBy" components={{ brand: <a href="https://github.com/marcoshack/taskwondo" target="_blank" rel="noopener noreferrer" className="font-medium hover:text-gray-600 dark:hover:text-gray-400 underline-offset-2 hover:underline" /> }} />
      </p>
    </footer>
  )
}

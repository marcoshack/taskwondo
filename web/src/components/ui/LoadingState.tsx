import { Spinner } from './Spinner'

interface LoadingStateProps {
  message?: string
}

export function LoadingState({ message }: LoadingStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16">
      <Spinner size="lg" />
      {message && (
        <p className="mt-4 text-sm text-gray-500 dark:text-gray-400">
          {message}
        </p>
      )}
    </div>
  )
}

interface ConfirmCheckProps {
  visible: boolean
}

export function ConfirmCheck({ visible }: ConfirmCheckProps) {
  if (!visible) return null
  return (
    <span className="inline-flex items-center text-green-500 animate-confirm">
      <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="2">
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.5 8.5l3 3 6-7" />
      </svg>
    </span>
  )
}

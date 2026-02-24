import { useEffect, useRef, useState } from 'react'
import mermaid from 'mermaid'

function initMermaid(dark: boolean) {
  mermaid.initialize({
    startOnLoad: false,
    theme: dark ? 'dark' : 'default',
    securityLevel: 'strict',
  })
}

let idCounter = 0

interface MermaidProps {
  chart: string
}

export function Mermaid({ chart }: MermaidProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [svg, setSvg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const dark = document.documentElement.classList.contains('dark')
    initMermaid(dark)

    const id = `mermaid-${idCounter++}`
    mermaid
      .render(id, chart)
      .then(({ svg }) => {
        if (!cancelled) {
          setSvg(svg)
          setError(null)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setSvg(null)
          setError(chart)
        }
      })

    return () => {
      cancelled = true
    }
  }, [chart])

  if (error) {
    return (
      <pre className="bg-gray-100 dark:bg-gray-800 p-3 rounded text-sm overflow-x-auto">
        <code>{error}</code>
      </pre>
    )
  }

  if (!svg) return null

  return (
    <div
      ref={containerRef}
      className="my-4 flex justify-center"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}

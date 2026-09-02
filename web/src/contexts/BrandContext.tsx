/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useRef } from 'react'
import type { ReactNode } from 'react'
import { usePublicSettings } from '@/hooks/useSystemSettings'

const DEFAULT_BRAND_NAME = 'Taskwondo'

interface BrandContextValue {
  brandName: string
}

const BrandContext = createContext<BrandContextValue>({ brandName: DEFAULT_BRAND_NAME })

export function BrandProvider({ children }: { children: ReactNode }) {
  const { data: publicSettings } = usePublicSettings()

  const brandName =
    typeof publicSettings?.brand_name === 'string' && publicSettings.brand_name
      ? publicSettings.brand_name
      : DEFAULT_BRAND_NAME

  // Keep the browser tab title in sync with the configured brand, without
  // clobbering page-specific titles (e.g. the work item detail page).
  const lastBaseTitle = useRef(DEFAULT_BRAND_NAME)
  useEffect(() => {
    if (document.title === lastBaseTitle.current) {
      document.title = brandName
    }
    lastBaseTitle.current = brandName
  }, [brandName])

  return (
    <BrandContext.Provider value={{ brandName }}>
      {children}
    </BrandContext.Provider>
  )
}

export function useBrand() {
  return useContext(BrandContext)
}

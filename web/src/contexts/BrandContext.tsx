import { createContext, useContext } from 'react'
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

  return (
    <BrandContext.Provider value={{ brandName }}>
      {children}
    </BrandContext.Provider>
  )
}

export function useBrand() {
  return useContext(BrandContext)
}

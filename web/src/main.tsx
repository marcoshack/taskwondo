import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '@/contexts/AuthContext'
import { ThemeProvider } from '@/contexts/ThemeContext'
import { LanguageProvider } from '@/contexts/LanguageContext'
import { LayoutProvider } from '@/contexts/LayoutContext'
import { SidebarProvider } from '@/contexts/SidebarContext'
import { KeyboardShortcutProvider } from '@/contexts/KeyboardShortcutContext'
import { NavigationGuardProvider } from '@/contexts/NavigationGuardContext'
import { NotificationProvider } from '@/contexts/NotificationContext'
import { BrandProvider } from '@/contexts/BrandContext'
import '@/i18n'
import App from '@/App'
import '@/index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrandProvider>
        <BrowserRouter>
          <AuthProvider>
            <ThemeProvider>
              <LanguageProvider>
                <LayoutProvider>
                <KeyboardShortcutProvider>
                  <NavigationGuardProvider>
                    <NotificationProvider>
                      <SidebarProvider>
                        <App />
                      </SidebarProvider>
                    </NotificationProvider>
                  </NavigationGuardProvider>
                </KeyboardShortcutProvider>
                </LayoutProvider>
              </LanguageProvider>
            </ThemeProvider>
          </AuthProvider>
        </BrowserRouter>
      </BrandProvider>
    </QueryClientProvider>
  </StrictMode>,
)

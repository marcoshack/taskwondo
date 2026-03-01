import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import type { User } from '@/api/auth'
import * as authApi from '@/api/auth'
import { setToken, clearToken, getToken } from '@/api/client'

interface AuthContextValue {
  user: User | null
  isLoading: boolean
  forcePasswordChange: boolean
  login: (email: string, password: string) => Promise<void>
  loginWithToken: (token: string, user: User) => void
  clearForcePasswordChange: (newToken: string) => void
  updateUser: (user: User) => void
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [forcePasswordChange, setForcePasswordChange] = useState(false)

  useEffect(() => {
    const token = getToken()
    if (!token) {
      setIsLoading(false)
      return
    }
    authApi
      .getMe()
      .then((u) => setUser(u))
      .catch(() => clearToken())
      .finally(() => setIsLoading(false))
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const result = await authApi.login(email, password)
    setToken(result.token)
    setUser(result.user)
    if (result.force_password_change) {
      setForcePasswordChange(true)
    }
  }, [])

  const loginWithToken = useCallback((token: string, user: User) => {
    setToken(token)
    setUser(user)
  }, [])

  const clearForcePasswordChange = useCallback((newToken: string) => {
    setToken(newToken)
    setForcePasswordChange(false)
  }, [])

  const updateUser = useCallback((updatedUser: User) => {
    setUser(updatedUser)
  }, [])

  const logout = useCallback(() => {
    clearToken()
    setUser(null)
    setForcePasswordChange(false)
  }, [])

  return (
    <AuthContext.Provider value={{ user, isLoading, forcePasswordChange, login, loginWithToken, clearForcePasswordChange, updateUser, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

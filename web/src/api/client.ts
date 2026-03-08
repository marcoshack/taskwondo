import axios from 'axios'

const TOKEN_KEY = 'taskwondo_token'

// Active namespace slug (defaults to 'default')
// Initialize from localStorage so the first request after refresh uses the correct namespace
const NAMESPACE_KEY = 'taskwondo_namespace'
const storedNs = localStorage.getItem(NAMESPACE_KEY)
let activeNamespaceSlug: string | null = storedNs && storedNs !== 'default' ? storedNs : null

export function setNamespaceSlug(slug: string | null) {
  activeNamespaceSlug = slug
}

export function getNamespaceSlug(): string | null {
  return activeNamespaceSlug
}

/** Returns the namespace path prefix for project-scoped API routes. */
export function nsPrefix(): string {
  return `/${activeNamespaceSlug || 'default'}`
}

export const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      const url = error.config?.url || ''
      const isAuthEndpoint = url.startsWith('/auth/login') || url.startsWith('/auth/discord')
      if (!isAuthEndpoint) {
        localStorage.removeItem(TOKEN_KEY)
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  },
)

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

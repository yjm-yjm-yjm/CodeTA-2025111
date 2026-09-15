import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api } from '../api/client'
import type { Admin } from '../api/types'

type AuthState = {
  admin: Admin | null
  loading: boolean
  mockLogin: () => Promise<void>
  logout: () => void
  acceptToken: (token: string) => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [admin, setAdmin] = useState<Admin | null>(null)
  const [loading, setLoading] = useState(true)

  const loadMe = useCallback(async () => {
    const t = localStorage.getItem('admin_token')
    if (!t) {
      setAdmin(null)
      return
    }
    const me = await api.me()
    setAdmin(me)
  }, [])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const qToken = params.get('token')
    const boot = async () => {
      try {
        if (qToken) {
          localStorage.setItem('admin_token', qToken)
          params.delete('token')
          const next = `${window.location.pathname}${params.toString() ? `?${params}` : ''}`
          window.history.replaceState({}, '', next)
        }
        await loadMe()
      } catch {
        localStorage.removeItem('admin_token')
        setAdmin(null)
      } finally {
        setLoading(false)
      }
    }
    void boot()
  }, [loadMe])

  const mockLogin = useCallback(async () => {
    const res = await api.mockLogin()
    localStorage.setItem('admin_token', res.token)
    setAdmin(res.admin)
    await loadMe().catch(() => setAdmin(res.admin))
  }, [loadMe])

  const acceptToken = useCallback(async (token: string) => {
    localStorage.setItem('admin_token', token)
    await loadMe()
  }, [loadMe])

  const logout = useCallback(() => {
    localStorage.removeItem('admin_token')
    setAdmin(null)
  }, [])

  const value = useMemo(
    () => ({ admin, loading, mockLogin, logout, acceptToken }),
    [admin, loading, mockLogin, logout, acceptToken],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

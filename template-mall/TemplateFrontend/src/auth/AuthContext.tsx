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
import type { User } from '../api/types'

type AuthState = {
  user: User | null
  loading: boolean
  login: (phone: string, password: string) => Promise<void>
  register: (phone: string, password: string, nickname: string) => Promise<void>
  logout: () => void
  refreshMe: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const refreshMe = useCallback(async () => {
    const t = localStorage.getItem('c_token')
    if (!t) {
      setUser(null)
      return
    }
    const me = await api.me()
    setUser(me)
  }, [])

  useEffect(() => {
    refreshMe()
      .catch(() => {
        localStorage.removeItem('c_token')
        setUser(null)
      })
      .finally(() => setLoading(false))
  }, [refreshMe])

  const login = useCallback(async (phone: string, password: string) => {
    const res = await api.login(phone, password)
    localStorage.setItem('c_token', res.token)
    setUser({ ...res.user })
    await refreshMe().catch(() => setUser(res.user))
  }, [refreshMe])

  const register = useCallback(async (phone: string, password: string, nickname: string) => {
    const res = await api.register(phone, password, nickname)
    localStorage.setItem('c_token', res.token)
    setUser({ ...res.user })
    await refreshMe().catch(() => setUser(res.user))
  }, [refreshMe])

  const logout = useCallback(() => {
    localStorage.removeItem('c_token')
    setUser(null)
  }, [])

  const value = useMemo(
    () => ({ user, loading, login, register, logout, refreshMe }),
    [user, loading, login, register, logout, refreshMe],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

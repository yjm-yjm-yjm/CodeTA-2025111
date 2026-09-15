import { useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { AuthBackdrop } from '../components/AuthBackdrop'

export function LoginPage() {
  const { admin, loading, logout, mockLogin } = useAuth()
  const navigate = useNavigate()
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)

  if (!loading && admin) return <Navigate to="/" replace />

  function startWpsLogin() {
    logout()
    navigate('/auth/wps')
  }

  async function startMockLogin() {
    setError('')
    setPending(true)
    try {
      await mockLogin()
      navigate('/', { replace: true })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Mock 登录失败')
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthBackdrop>
      <div className="auth-card">
        <h1>管理员登录</h1>
        <button type="button" className="btn primary" onClick={startWpsLogin}>
          WPS 统一登录
        </button>
        <button type="button" className="btn ghost" disabled={pending} onClick={() => void startMockLogin()}>
          {pending ? '登录中…' : 'Mock 登录（本地）'}
        </button>
        {error && <p className="error">{error}</p>}
      </div>
    </AuthBackdrop>
  )
}

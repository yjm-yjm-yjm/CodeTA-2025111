import { useState, type FormEvent } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { AuthBackdrop } from '../components/AuthBackdrop'

const PHONE_RE = /^1[3-9]\d{9}$/

export function LoginPage() {
  const { user, loading, login, register } = useAuth()
  const navigate = useNavigate()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)

  if (!loading && user) return <Navigate to="/" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    const p = phone.trim()
    if (!PHONE_RE.test(p)) {
      setError('请输入正确的 11 位手机号')
      return
    }
    if (password.length < 6 || password.length > 64) {
      setError('密码长度需为 6–64 位')
      return
    }
    setPending(true)
    try {
      if (mode === 'login') await login(p, password)
      else await register(p, password, nickname.trim() || p)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败')
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthBackdrop>
      <form className="auth-card" onSubmit={onSubmit}>
        <h1>{mode === 'login' ? '用户登录' : '注册账号'}</h1>
        <label>
          手机号
          <input
            type="tel"
            inputMode="numeric"
            pattern="1[3-9][0-9]{9}"
            maxLength={11}
            placeholder="11 位手机号"
            value={phone}
            onChange={(e) => setPhone(e.target.value.replace(/\D/g, '').slice(0, 11))}
            required
            autoComplete="tel"
          />
        </label>
        {mode === 'register' && (
          <label>
            昵称
            <input value={nickname} onChange={(e) => setNickname(e.target.value)} autoComplete="nickname" />
          </label>
        )}
        <label>
          密码
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={6}
            maxLength={64}
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
          />
        </label>
        {error && <p className="error">{error}</p>}
        <button className="btn primary" type="submit" disabled={pending}>
          {pending ? '提交中…' : mode === 'login' ? '登录' : '注册并登录'}
        </button>
        <button
          type="button"
          className="btn link"
          onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
        >
          {mode === 'login' ? '没有账号？去注册' : '已有账号？去登录'}
        </button>
      </form>
    </AuthBackdrop>
  )
}

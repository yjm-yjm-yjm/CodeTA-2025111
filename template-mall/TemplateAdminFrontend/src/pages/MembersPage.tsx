import { useState, type FormEvent } from 'react'
import { Navigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../auth/AuthContext'

export function MembersPage() {
  const { admin, loading } = useAuth()
  const [userId, setUserId] = useState('')
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)

  if (!loading && !admin) return <Navigate to="/login" replace />

  async function setMember(isMember: boolean) {
    setError('')
    setMsg('')
    if (!userId.trim()) {
      setError('请填写用户 user_id')
      return
    }
    setPending(true)
    try {
      const u = await api.setMembership(userId.trim(), isMember)
      setMsg(
        u.is_member
          ? `操作成功！用户${u.user_id} 当前已是会员状态！`
          : `操作成功！用户${u.user_id} 当前已不是会员状态！`,
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : '操作失败')
    } finally {
      setPending(false)
    }
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
  }

  return (
    <section>
      <div className="page-head">
        <h1>会员管理</h1>
      </div>
      <form className="form-card" onSubmit={onSubmit}>
        <label>
          用户 ID
          <input
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            placeholder="来自 C 端注册后的 user_id"
          />
        </label>
        {error && <p className="error">{error}</p>}
        {msg && <p className="ok">{msg}</p>}
        <div className="actions">
          <button
            type="button"
            className="btn primary"
            disabled={pending}
            onClick={() => void setMember(true)}
          >
            设为会员
          </button>
          <button
            type="button"
            className="btn danger"
            disabled={pending}
            onClick={() => void setMember(false)}
          >
            取消会员
          </button>
        </div>
      </form>
    </section>
  )
}

import { useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { api, formatFen } from '../api/client'
import type { MembershipPlan } from '../api/types'
import { useAuth } from '../auth/AuthContext'

export function MembershipPage() {
  const { user, loading, refreshMe } = useAuth()
  const [plans, setPlans] = useState<MembershipPlan[]>([])
  const [error, setError] = useState('')
  const [msg, setMsg] = useState('')
  const [busy, setBusy] = useState('')

  useEffect(() => {
    if (!user) return
    void refreshMe().catch(() => {})
    api
      .listMembershipPlans()
      .then((res) => setPlans(res.items || []))
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [user, refreshMe])

  if (!loading && !user) return <Navigate to="/login" replace />

  async function onSubscribe(plan: MembershipPlan) {
    setError('')
    setMsg('')
    setBusy(plan.plan)
    try {
      const res = await api.subscribeMembership(plan.plan)
      if (res.payment?.pay_url) {
        setMsg(`正在打开支付页，完成支付后即可成为会员（${plan.name}）`)
        window.open(res.payment.pay_url, '_blank', 'noopener,noreferrer')
      } else {
        setMsg('已创建订阅订单，请稍后在「我的订单」中完成支付')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : '订阅失败')
    } finally {
      setBusy('')
    }
  }

  return (
    <section>
      <div className="page-head">
        <h1>会员订阅</h1>
      </div>

      {user?.is_member ? (
        <div className="notice-bar notice-ok" role="status">
          你当前已是会员，可直接下载商城内付费模板。
        </div>
      ) : (
        <div className="notice-bar" role="note">
          <p>开通会员后，付费模板可直接下载，无需逐个购买。</p>
          <p>当前为非会员。选择套餐完成支付后自动开通。</p>
        </div>
      )}

      {error && <p className="error">{error}</p>}
      {msg && <p className="ok">{msg}</p>}

      <div className="plan-grid">
        {plans.map((p) => (
          <article key={p.plan} className={`plan-card plan-${p.plan}`}>
            <h2>{p.name}</h2>
            <p className="plan-price">{formatFen(p.price_fen)}</p>
            <p className="muted">{p.desc}</p>
            <button
              type="button"
              className="btn primary"
              disabled={busy === p.plan || Boolean(user?.is_member)}
              onClick={() => void onSubscribe(p)}
            >
              {user?.is_member ? '已是会员' : busy === p.plan ? '处理中…' : '立即订阅'}
            </button>
          </article>
        ))}
      </div>
    </section>
  )
}

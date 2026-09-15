import { useCallback, useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { api, formatFen, shortEnum } from '../api/client'
import type { Order } from '../api/types'
import { useAuth } from '../auth/AuthContext'

function orderKindLabel(o: Order): string {
  const tid = o.template_id || ''
  if (tid.startsWith('membership_')) return '会员订阅'
  if (shortEnum(o.order_type) === 'member') return '会员下载'
  if (o.price_fen_snapshot <= 0 || shortEnum(o.order_type) === 'free') return '免费'
  return '付费'
}

function orderStatusLabel(status: string): string {
  const s = shortEnum(status)
  switch (s) {
    case 'ready':
    case 'download':
    case 'download_ready':
      return '已完成'
    case 'unpaid':
      return '待支付'
    case 'cancelled':
      return '已取消'
    default:
      if (status.includes('READY') || status.includes('ready')) return '已完成'
      return status
  }
}

export function OrdersPage() {
  const { admin, loading } = useAuth()
  const [items, setItems] = useState<Order[]>([])
  const [error, setError] = useState('')

  const reload = useCallback(() => {
    return api
      .listOrders()
      .then((res) => setItems(res.items || []))
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [])

  useEffect(() => {
    if (!admin) return
    void reload()
  }, [admin, reload])

  if (!loading && !admin) return <Navigate to="/login" replace />

  return (
    <section>
      <div className="page-head">
        <h1>订单列表</h1>
      </div>
      {error && <p className="error">{error}</p>}
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>订单号</th>
              <th>用户</th>
              <th>用户名</th>
              <th>模版名</th>
              <th>订阅状态</th>
              <th>类型</th>
              <th>状态</th>
              <th>金额</th>
            </tr>
          </thead>
          <tbody>
            {items.map((o) => (
              <tr key={o.order_id}>
                <td className="mono">{o.order_id}</td>
                <td className="mono">{o.user_id}</td>
                <td>{o.user_name || '-'}</td>
                <td>{o.template_name || '-'}</td>
                <td>{o.is_member ? '会员' : '非会员'}</td>
                <td>{orderKindLabel(o)}</td>
                <td>{orderStatusLabel(o.status)}</td>
                <td>{formatFen(o.price_fen_snapshot)}</td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td colSpan={8} className="muted">
                  暂无订单
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  )
}

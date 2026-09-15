import { useCallback, useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { api, formatFen, shortEnum } from '../api/client'
import type { Order } from '../api/types'
import { useAuth } from '../auth/AuthContext'

function orderKindLabel(o: Order): string {
  if ((o.template_id || '').startsWith('membership_')) return '会员订阅'
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
  const { user, loading } = useAuth()
  const [items, setItems] = useState<Order[]>([])
  const [error, setError] = useState('')
  const [busyId, setBusyId] = useState('')

  const reload = useCallback(() => {
    return api
      .listOrders()
      .then((res) => setItems(res.items || []))
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [])

  useEffect(() => {
    if (!user) return
    void reload()
  }, [user, reload])

  if (!loading && !user) return <Navigate to="/login" replace />

  async function onCancel(orderId: string) {
    setError('')
    setBusyId(orderId)
    try {
      await api.cancelOrder(orderId)
      await reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : '取消失败')
    } finally {
      setBusyId('')
    }
  }

  function canCancel(o: Order) {
    return shortEnum(o.order_type) === 'retail' && shortEnum(o.status) === 'unpaid'
  }

  return (
    <section>
      <div className="page-head">
        <h1>我的订单</h1>
        <p className="muted">可取消自己的未支付零售订单。</p>
      </div>
      {error && <p className="error">{error}</p>}
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>订单号</th>
              <th>模板</th>
              <th>类型</th>
              <th>状态</th>
              <th>金额</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {items.map((o) => (
              <tr key={o.order_id}>
                <td className="mono">{o.order_id}</td>
                <td>{o.template_name || o.template_id}</td>
                <td>{orderKindLabel(o)}</td>
                <td>{orderStatusLabel(o.status)}</td>
                <td>{formatFen(o.price_fen_snapshot)}</td>
                <td className="actions">
                  {canCancel(o) && (
                    <>
                      {o.pay_url && (
                        <a className="btn ghost" href={o.pay_url} target="_blank" rel="noreferrer">
                          去支付
                        </a>
                      )}
                      <button
                        type="button"
                        className="btn danger"
                        disabled={busyId === o.order_id}
                        onClick={() => void onCancel(o.order_id)}
                      >
                        {busyId === o.order_id ? '取消中…' : '取消'}
                      </button>
                    </>
                  )}
                </td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td colSpan={6} className="muted">
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

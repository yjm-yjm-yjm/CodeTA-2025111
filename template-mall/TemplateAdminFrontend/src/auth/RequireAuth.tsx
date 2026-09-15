import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from './AuthContext'

/** 未登录禁止进入管理功能页，避免「没走 OAuth 却像进了系统」。 */
export function RequireAuth() {
  const { admin, loading } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div className="auth-page">
        <p className="muted">加载登录状态…</p>
      </div>
    )
  }
  if (!admin) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  return <Outlet />
}

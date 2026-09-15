import { Link, NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

export function Layout() {
  const { admin, logout } = useAuth()
  return (
    <div className="app-shell">
      <header className="topbar">
        <Link to="/" className="brand">
          模板商城 · 管理端
        </Link>
        <nav className="nav">
          <NavLink to="/" end>
            模板
          </NavLink>
          <NavLink to="/upload">上传</NavLink>
          <NavLink to="/orders">订单</NavLink>
          <NavLink to="/members">会员</NavLink>
        </nav>
        <div className="userbox">
          {admin && <span className="muted">{admin.nickname}（管理员）</span>}
          <button type="button" className="btn ghost" onClick={logout}>
            退出
          </button>
        </div>
      </header>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}

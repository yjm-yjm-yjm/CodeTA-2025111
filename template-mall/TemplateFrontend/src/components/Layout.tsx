import { Link, NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

export function Layout() {
  const { user, logout } = useAuth()

  return (
    <div className="app-shell">
      <header className="topbar">
        <Link to="/" className="brand">
          模板商城
        </Link>
        <nav className="nav">
          <NavLink to="/" end>
            模板
          </NavLink>
          <NavLink to="/membership">会员订阅</NavLink>
          <NavLink to="/orders">我的订单</NavLink>
        </nav>
        <div className="userbox">
          {user ? (
            <>
              <span className="muted">
                {user.nickname}
                {user.is_member ? ' · 会员' : ''}
              </span>
              <button type="button" className="btn ghost" onClick={logout}>
                退出
              </button>
            </>
          ) : (
            <NavLink to="/login">登录</NavLink>
          )}
        </div>
      </header>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}

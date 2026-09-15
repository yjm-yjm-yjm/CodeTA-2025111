import type { ReactNode } from 'react'

function FloatIcon({
  className,
  children,
}: {
  className: string
  children: ReactNode
}) {
  return (
    <span className={`auth-float-icon ${className}`} aria-hidden="true">
      {children}
    </span>
  )
}

/** 登录页：上方品牌大字，下方登录框；背景为彩色渐变 + 浮动小图标 */
export function AuthBackdrop({ children }: { children: ReactNode }) {
  return (
    <div className="auth-page auth-page--admin">
      <div className="auth-backdrop" aria-hidden="true">
        <div className="auth-wash auth-wash-a" />
        <div className="auth-wash auth-wash-b" />
        <div className="auth-wash auth-wash-c" />
        <div className="auth-dots" />
        <div className="auth-ribbon auth-ribbon-a" />
        <div className="auth-ribbon auth-ribbon-b" />
        <div className="auth-float-layer">
          <FloatIcon className="fi-1">
            <svg viewBox="0 0 24 24" fill="none">
              <path
                d="M7 3.5h7.2L19 8.3V20a1.5 1.5 0 0 1-1.5 1.5h-10A1.5 1.5 0 0 1 6 20V5A1.5 1.5 0 0 1 7.5 3.5H7Z"
                stroke="currentColor"
                strokeWidth="1.6"
              />
              <path d="M14 3.5V8h5" stroke="currentColor" strokeWidth="1.6" />
              <path d="M9 12h6M9 15.5h6" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
            </svg>
          </FloatIcon>
          <FloatIcon className="fi-2">
            <svg viewBox="0 0 24 24" fill="none">
              <rect x="3.5" y="5" width="17" height="13" rx="2" stroke="currentColor" strokeWidth="1.6" />
              <path d="M10 9.5v5l4.5-2.5L10 9.5Z" fill="currentColor" />
            </svg>
          </FloatIcon>
          <FloatIcon className="fi-3">
            <svg viewBox="0 0 24 24" fill="none">
              <rect x="4" y="4" width="16" height="16" rx="2" stroke="currentColor" strokeWidth="1.6" />
              <path d="M4 10h16M4 15h16M10 4v16" stroke="currentColor" strokeWidth="1.6" />
            </svg>
          </FloatIcon>
          <FloatIcon className="fi-4">
            <svg viewBox="0 0 24 24" fill="none">
              <path
                d="M12 3.8 14.2 9l5.5.5-4.2 3.7 1.3 5.3L12 15.8 7.2 18.5l1.3-5.3L4.3 9.5 9.8 9 12 3.8Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinejoin="round"
              />
            </svg>
          </FloatIcon>
          <FloatIcon className="fi-5">
            <svg viewBox="0 0 24 24" fill="none">
              <path
                d="M12 4v10m0 0 3.5-3.5M12 14l-3.5-3.5"
                stroke="currentColor"
                strokeWidth="1.7"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
              <path d="M5 18.5h14" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
            </svg>
          </FloatIcon>
          <FloatIcon className="fi-6">
            <svg viewBox="0 0 24 24" fill="none">
              <path
                d="M7 17.5V8.2A2.2 2.2 0 0 1 9.2 6h8.3v9.8A1.7 1.7 0 0 1 15.8 17.5H7Z"
                stroke="currentColor"
                strokeWidth="1.6"
              />
              <path d="M7 17.5A1.7 1.7 0 0 1 5.3 15.8V7" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
            </svg>
          </FloatIcon>
        </div>
      </div>
      <div className="auth-stage">
        <header className="auth-brand-block">
          <p className="auth-brand">
            <span style={{ animationDelay: '0s' }}>模</span>
            <span style={{ animationDelay: '0.14s' }}>版</span>
            <span style={{ animationDelay: '0.28s' }}>商</span>
            <span style={{ animationDelay: '0.42s' }}>城</span>
          </p>
          <p className="auth-brand-sub">Admin · Template Mall</p>
        </header>
        <div className="auth-foreground">{children}</div>
      </div>
    </div>
  )
}

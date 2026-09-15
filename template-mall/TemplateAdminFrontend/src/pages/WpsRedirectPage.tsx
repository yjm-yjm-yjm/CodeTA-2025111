import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { AuthBackdrop } from '../components/AuthBackdrop'

/**
 * WPS OAuth 跳转中间页：先展示提示，再离开本站前往 openapi.wps.cn。
 */
export function WpsRedirectPage() {
  const [error, setError] = useState('')
  const [authorizeUrl, setAuthorizeUrl] = useState('')
  const [status, setStatus] = useState('正在准备 WPS 统一认证…')

  useEffect(() => {
    let cancelled = false
    let timer: number | undefined

    const run = async () => {
      // 清掉旧会话，避免带着本地 token「看起来已登录」
      localStorage.removeItem('admin_token')
      try {
        setStatus('正在向服务端申请 WPS 授权地址…')
        const { authorize_url: url } = await api.wpsAuthorize()
        if (cancelled) return
        if (!url || !/^https:\/\/openapi\.wps\.cn\//i.test(url)) {
          throw new Error('授权地址无效（须为 https://openapi.wps.cn/...）')
        }
        setAuthorizeUrl(url)
        setStatus('即将跳转到 WPS 授权页（openapi.wps.cn）…')
        timer = window.setTimeout(() => {
          window.location.assign(url)
        }, 1200)
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : '获取 WPS 授权地址失败')
          setStatus('跳转失败')
        }
      }
    }
    void run()
    return () => {
      cancelled = true
      if (timer !== undefined) window.clearTimeout(timer)
    }
  }, [])

  return (
    <AuthBackdrop>
      <div className="auth-card">
        <h1>跳转 WPS 统一认证</h1>
        <p className="muted">{status}</p>
        <div className="oauth-redirect-box" aria-live="polite">
          <p>
            下一跳目标域名：<strong>openapi.wps.cn</strong>
          </p>
          <p className="muted small">
            若浏览器已登录 WPS，授权页可能一闪而过；未登录时应出现 WPS 账号登录界面。
          </p>
        </div>
        {error && <p className="error">{error}</p>}
        {authorizeUrl && !error && (
          <a className="btn primary" href={authorizeUrl}>
            若未自动跳转，点击前往 WPS
          </a>
        )}
        {error && (
          <Link className="btn ghost" to="/login">
            返回登录
          </Link>
        )}
      </div>
    </AuthBackdrop>
  )
}

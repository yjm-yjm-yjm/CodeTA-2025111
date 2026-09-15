import type { Admin, Order, PageInfo, Template, UploadCredential } from './types'

// 空字符串表示走当前页同源（Vite 代理 /api → AdminWebServer），便于 WPS redirect_uri 用 localhost:3001
const API_BASE =
  import.meta.env.VITE_API_BASE !== undefined
    ? String(import.meta.env.VITE_API_BASE)
    : ''

function token(): string | null {
  return localStorage.getItem('admin_token')
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type') && init.body && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }
  const t = token()
  if (t) headers.set('Authorization', `Bearer ${t}`)

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers, redirect: 'manual' })
  // 避免 Vite/代理把 OAuth 302 自动跟飞；授权地址必须走 JSON
  if (res.type === 'opaqueredirect' || (res.status >= 300 && res.status < 400)) {
    throw new Error('收到意外重定向，请使用 /api/auth/login?format=json')
  }
  const text = await res.text()
  let json: { data?: T; error?: string; mode?: string; message?: string } = {}
  if (text) {
    try {
      json = JSON.parse(text)
    } catch {
      throw new Error(text || res.statusText)
    }
  }
  if (!res.ok) throw new Error(json.error || res.statusText || 'request failed')
  return (json.data ?? json) as T
}

export const api = {
  base: API_BASE,
  mockLogin(adminId = 'mock-admin', nickname = 'MockAdmin') {
    return request<{ token: string; admin: Admin }>('/api/auth/mock-login', {
      method: 'POST',
      body: JSON.stringify({ admin_id: adminId, nickname }),
    })
  },
  /** 获取 WPS 授权 URL（由 /auth/wps 中间页再跳转） */
  async wpsAuthorize() {
    const res = await fetch(`${API_BASE}/api/auth/login?format=json`, {
      headers: { Accept: 'application/json' },
      redirect: 'manual',
      cache: 'no-store',
    })
    if (res.type === 'opaqueredirect' || (res.status >= 300 && res.status < 400)) {
      throw new Error('登录接口返回了重定向而非 authorize_url，请检查服务端 AUTH_MODE=wps')
    }
    const text = await res.text()
    let json: { data?: { mode: string; authorize_url: string }; error?: string } = {}
    try {
      json = JSON.parse(text)
    } catch {
      throw new Error(text || '无法解析授权响应')
    }
    if (!res.ok) throw new Error(json.error || res.statusText || 'request failed')
    const data = json.data
    if (!data?.authorize_url) throw new Error('响应缺少 authorize_url')
    return data
  },
  me() {
    return request<Admin>('/api/me')
  },
  uploadCredential(filename: string, contentType: string) {
    return request<UploadCredential>('/api/upload/credential', {
      method: 'POST',
      body: JSON.stringify({ filename, content_type: contentType }),
    })
  },
  async putFile(uploadUrl: string, file: File, contentType: string) {
    // 签名直传：upload_url 为 OSS 预签名地址时直连 OSS；mock 模式则为同源 /api/upload/mock
    let url = uploadUrl
    try {
      if (url.startsWith('http://') || url.startsWith('https://')) {
        const u = new URL(url)
        // mock 凭证指向本机 AdminWebServer，走 Vite 代理
        if (u.pathname.startsWith('/api/upload/mock')) {
          url = `${u.pathname}${u.search}`
        }
      }
    } catch {
      /* keep original */
    }
    const ct = contentType || file.type || 'application/octet-stream'
    let res: Response
    try {
      res = await fetch(url, {
        method: 'PUT',
        headers: { 'Content-Type': ct },
        body: file,
      })
    } catch {
      throw new Error(
        '上传失败：无法直传对象存储（请确认 OSS 桶已配置 CORS，允许本机管理端来源发起 PUT）',
      )
    }
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || `upload failed: ${res.status}`)
    }
  },
  confirmAndCreate(body: {
    object_key: string
    cover_object_key?: string
    original_filename: string
    file_type: string
    file_size: number
    name: string
    price_fen: number
    price_type: string
    publish: boolean
  }) {
    return request<Template>('/api/upload/confirm', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  listTemplates(page = 1, pageSize = 50) {
    return request<{ items: Template[]; page: PageInfo }>(
      `/api/templates?page=${page}&page_size=${pageSize}`,
    )
  },
  updateTemplate(id: string, body: { name?: string; price_fen?: number; price_type?: string }) {
    return request<Template>(`/api/templates/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  publish(id: string) {
    return request<Template>(`/api/templates/${encodeURIComponent(id)}/publish`, { method: 'POST' })
  },
  unpublish(id: string) {
    return request<Template>(`/api/templates/${encodeURIComponent(id)}/unpublish`, { method: 'POST' })
  },
  listOrders(page = 1, pageSize = 50) {
    return request<{ items: Order[]; page: PageInfo }>(
      `/api/orders?page=${page}&page_size=${pageSize}`,
    )
  },
  setMembership(userId: string, isMember: boolean) {
    return request<{ user_id: string; nickname: string; is_member: boolean }>(
      `/api/users/${encodeURIComponent(userId)}/membership`,
      { method: 'POST', body: JSON.stringify({ is_member: isMember }) },
    )
  },
}

export function formatFen(fen: number) {
  return `¥${(fen / 100).toFixed(2)}`
}

export function shortEnum(s: string) {
  const parts = s.split('_')
  return parts[parts.length - 1]?.toLowerCase() || s
}

export function fileExt(name: string) {
  const i = name.lastIndexOf('.')
  return i >= 0 ? name.slice(i + 1).toLowerCase() : ''
}

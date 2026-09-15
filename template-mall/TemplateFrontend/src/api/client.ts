import type {
  DownloadResult,
  MembershipPlan,
  Order,
  PageInfo,
  SubscribeResult,
  Template,
  User,
} from './types'

const API_BASE = (import.meta.env.VITE_API_BASE as string) || 'http://127.0.0.1:8080'

function token(): string | null {
  return localStorage.getItem('c_token')
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type') && init.body) {
    headers.set('Content-Type', 'application/json')
  }
  const t = token()
  if (t) headers.set('Authorization', `Bearer ${t}`)

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers })
  const text = await res.text()
  let json: { data?: T; error?: string } = {}
  if (text) {
    try {
      json = JSON.parse(text)
    } catch {
      throw new Error(text || res.statusText)
    }
  }
  if (!res.ok) {
    throw new Error(json.error || res.statusText || 'request failed')
  }
  return json.data as T
}

export const api = {
  register(phone: string, password: string, nickname: string) {
    return request<{ token: string; user: User }>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ phone, password, nickname }),
    })
  },
  login(phone: string, password: string) {
    return request<{ token: string; user: User }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ phone, password }),
    })
  },
  me() {
    return request<User>('/api/me')
  },
  listTemplates(page = 1, pageSize = 50, fileType = '') {
    const q = new URLSearchParams({
      page: String(page),
      page_size: String(pageSize),
    })
    if (fileType) q.set('file_type', fileType)
    return request<{ items: Template[]; page: PageInfo }>(`/api/templates?${q}`)
  },
  download(templateId: string) {
    return request<DownloadResult>(`/api/templates/${encodeURIComponent(templateId)}/download`, {
      method: 'POST',
    })
  },
  listOrders(page = 1, pageSize = 20) {
    return request<{ items: Order[]; page: PageInfo }>(
      `/api/orders?page=${page}&page_size=${pageSize}`,
    )
  },
  cancelOrder(orderId: string) {
    return request<Order>(`/api/orders/${encodeURIComponent(orderId)}/cancel`, {
      method: 'POST',
    })
  },
  listMembershipPlans() {
    return request<{ items: MembershipPlan[] }>('/api/membership/plans')
  },
  subscribeMembership(plan: string) {
    return request<SubscribeResult>('/api/membership/subscribe', {
      method: 'POST',
      body: JSON.stringify({ plan }),
    })
  },
}

export function formatFen(fen: number): string {
  return `¥${(fen / 100).toFixed(2)}`
}

export function shortEnum(s: string): string {
  const parts = s.split('_')
  return parts[parts.length - 1]?.toLowerCase() || s
}

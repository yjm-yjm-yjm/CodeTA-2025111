export type User = {
  user_id: string
  phone: string
  nickname: string
  is_member?: boolean
}

export type Template = {
  template_id: string
  name: string
  file_type: string
  price_fen: number
  price_type: string
  status: string
  cover_url?: string
}

export type Order = {
  order_id: string
  user_id: string
  template_id: string
  template_name?: string
  order_type: string
  status: string
  price_fen_snapshot: number
  pay_order_id?: string
  pay_url?: string
  created_at_unix?: number
}

export type PageInfo = {
  page: number
  page_size: number
  total: number
}

export type MembershipPlan = {
  plan: 'month' | 'quarter' | 'year' | string
  name: string
  price_fen: number
  desc: string
}

export type SubscribeResult = {
  order: Order
  payment: {
    pay_order_id: string
    order_id: string
    amount_fen: number
    pay_url: string
    status: string
  }
}

export type DownloadGranted = {
  result: 'granted'
  order_id: string
  download_url: string
  expire_at_unix: number
  order: Order
}

export type DownloadPaymentRequired = {
  result: 'payment_required'
  order: Order
  payment: {
    pay_order_id: string
    order_id: string
    amount_fen: number
    pay_url: string
    status: string
  }
}

export type DownloadResult = DownloadGranted | DownloadPaymentRequired

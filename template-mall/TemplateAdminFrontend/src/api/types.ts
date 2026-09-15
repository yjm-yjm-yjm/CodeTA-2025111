export type Admin = {
  admin_id: string
  nickname: string
  role?: string
}

export type Template = {
  template_id: string
  name: string
  file_type: string
  price_fen: number
  price_type: string
  status: string
  cover_url?: string
  cover_object_key?: string
  storage_provider?: string
  bucket?: string
  object_key?: string
  original_filename?: string
  file_size?: number
}

export type Order = {
  order_id: string
  user_id: string
  user_name?: string
  template_id: string
  template_name?: string
  is_member?: boolean
  order_type: string
  status: string
  price_fen_snapshot: number
  pay_order_id?: string
  pay_url?: string
}

export type UploadCredential = {
  provider: string
  bucket: string
  object_key: string
  upload_url: string
  method: string
  expire_at_unix: number
  content_type: string
}

export type PageInfo = {
  page: number
  page_size: number
  total: number
}

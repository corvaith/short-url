export interface User {
  id: string
  email: string
  created_at: string
}

export type UrlStatus = 'active' | 'expired' | 'disabled'

export interface UrlItem {
  id: string
  code: string
  short_url: string
  target_url: string
  clicks: number
  status: UrlStatus
  is_active: boolean
  expires_at: string | null
  last_accessed_at: string | null
  created_at: string
}

export interface UrlListResponse {
  items: UrlItem[]
  next_cursor: string | null
}

export interface CreateUrlPayload {
  target_url: string
  alias?: string
  expires_at?: string
}

export interface StatsPoint {
  date: string
  clicks: number
}

export interface UrlStats {
  id: string
  days: number
  range_clicks: number
  total_clicks: number
  series: StatsPoint[]
}

export interface Overview {
  total_urls: number
  total_clicks: number
  active_urls: number
}

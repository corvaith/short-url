import { api } from './api'
import type { CreateUrlPayload, Overview, UrlItem, UrlListResponse, UrlStats } from '../types'

export const urlsApi = {
  create(payload: CreateUrlPayload): Promise<UrlItem> {
    return api<UrlItem>('/urls', { method: 'POST', body: payload })
  },
  list(params: { limit?: number; cursor?: string } = {}): Promise<UrlListResponse> {
    return api<UrlListResponse>('/urls', {
      params: { limit: params.limit, cursor: params.cursor },
    })
  },
  get(id: string): Promise<UrlItem> {
    return api<UrlItem>(`/urls/${encodeURIComponent(id)}`)
  },
  setActive(id: string, isActive: boolean): Promise<UrlItem> {
    return api<UrlItem>(`/urls/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: { is_active: isActive },
    })
  },
  remove(id: string): Promise<void> {
    return api<void>(`/urls/${encodeURIComponent(id)}`, { method: 'DELETE' })
  },
  stats(id: string, days: 7 | 30): Promise<UrlStats> {
    return api<UrlStats>(`/urls/${encodeURIComponent(id)}/stats`, { params: { days } })
  },
  overview(): Promise<Overview> {
    return api<Overview>('/stats/overview')
  },
}

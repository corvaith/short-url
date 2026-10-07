import { api } from './api'
import type { User } from '../types'

export const authApi = {
  me(): Promise<{ user: User }> {
    return api<{ user: User }>('/auth/me')
  },
  login(email: string, password: string): Promise<{ user: User }> {
    return api<{ user: User }>('/auth/login', { method: 'POST', body: { email, password } })
  },
  register(email: string, password: string): Promise<{ user: User }> {
    return api<{ user: User }>('/auth/register', { method: 'POST', body: { email, password } })
  },
  logout(): Promise<void> {
    return api<void>('/auth/logout', { method: 'POST' })
  },
  changePassword(currentPassword: string, newPassword: string): Promise<void> {
    return api<void>('/auth/password', {
      method: 'PUT',
      body: { current_password: currentPassword, new_password: newPassword },
    })
  },
}

import type { LoginResponse } from '../types/auth'
import { apiRequest } from './client'

export function login(username: string, password: string) {
  return apiRequest<LoginResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}


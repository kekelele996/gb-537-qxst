import type { ApiEnvelope } from '../types/common'

const ACCESS_TOKEN_KEY = 'certrollover-access-token'

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly requestId = '',
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export function setAccessToken(token: string | null) {
  if (token) localStorage.setItem(ACCESS_TOKEN_KEY, token)
  else localStorage.removeItem(ACCESS_TOKEN_KEY)
}

export function getAccessToken() {
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

export async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  const token = getAccessToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')

  let response: Response
  try {
    response = await fetch(`/api/v1${path}`, { ...init, headers })
  } catch {
    throw new ApiError(0, 'NETWORK_UNAVAILABLE', '无法连接服务，请检查网络或服务状态。')
  }

  const envelope = (await response.json().catch(() => ({
    code: 'INVALID_RESPONSE',
    message: '服务返回了无法解析的响应。',
    request_id: '',
  }))) as Partial<ApiEnvelope<T>>

  if (!response.ok) {
    if (response.status === 401) window.dispatchEvent(new Event('certrollover:unauthorized'))
    throw new ApiError(response.status, envelope.code ?? 'REQUEST_FAILED', envelope.message ?? '请求失败。', envelope.request_id)
  }
  return envelope.data as T
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '发生未知错误。'
}


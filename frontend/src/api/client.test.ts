import { ApiError, apiRequest, setAccessToken } from './client'

describe('apiRequest', () => {
  it('unwraps data and sends the stored bearer token', async () => {
    setAccessToken('signed-token')
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ code: 'OK', message: 'success', data: { id: 7 }, request_id: 'req-1' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    await expect(apiRequest<{ id: number }>('/trust-anchors/7')).resolves.toEqual({ id: 7 })
    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers)
    expect(headers.get('Authorization')).toBe('Bearer signed-token')
  })

  it('surfaces the safe API error and emits unauthorized', async () => {
    const unauthorized = vi.fn()
    window.addEventListener('certrollover:unauthorized', unauthorized)
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ code: 'UNAUTHORIZED', message: 'access token is invalid or expired', request_id: 'req-401' }), { status: 401, headers: { 'Content-Type': 'application/json' } }))

    const request = apiRequest('/trust-anchors')
    await expect(request).rejects.toMatchObject({ status: 401, code: 'UNAUTHORIZED', requestId: 'req-401' })
    expect(unauthorized).toHaveBeenCalledOnce()
    window.removeEventListener('certrollover:unauthorized', unauthorized)
  })

  it('does not expose fetch internals in network errors', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('socket details'))
    await expect(apiRequest('/trust-anchors')).rejects.toMatchObject({ code: 'NETWORK_UNAVAILABLE', message: '无法连接服务，请检查网络或服务状态。' })
  })
})

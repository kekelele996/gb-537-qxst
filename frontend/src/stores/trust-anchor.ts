import { create } from 'zustand'
import { trustAnchorApi } from '../api/trust-anchor'
import { errorMessage } from '../api/client'
import type { TrustAnchor } from '../types/trust-anchor'
import type { LoadState } from '../types/common'

interface TrustAnchorState {
  items: TrustAnchor[]
  total: number
  status: LoadState
  error: string
  fetchAnchors: (query?: string) => Promise<void>
  lifecycle: (id: number, action: 'revoke' | 'archive' | 'restore') => Promise<void>
}

export const useTrustAnchorStore = create<TrustAnchorState>((set, get) => ({
  items: [], total: 0, status: 'idle', error: '',
  fetchAnchors: async (query = '') => {
    set({ status: 'loading', error: '' })
    try {
      const result = await trustAnchorApi.list(query)
      set({ items: result.items, total: result.total, status: 'ready' })
    } catch (error) { set({ status: 'error', error: errorMessage(error) }) }
  },
  lifecycle: async (id, action) => {
    const updated = await trustAnchorApi.lifecycle(id, action)
    set({ items: get().items.map((item) => item.id === id ? updated : item) })
  },
}))


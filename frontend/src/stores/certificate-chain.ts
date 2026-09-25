import { create } from 'zustand'
import { certificateChainApi } from '../api/certificate-chain'
import { errorMessage } from '../api/client'
import type { CertificateChain } from '../types/certificate-chain'
import type { LoadState } from '../types/common'

interface CertificateChainState {
  items: CertificateChain[]
  total: number
  status: LoadState
  error: string
  fetchChains: (query?: string) => Promise<void>
  transition: (id: number, toState: 'validated' | 'deprecated' | 'revoked') => Promise<void>
}

export const useCertificateChainStore = create<CertificateChainState>((set, get) => ({
  items: [], total: 0, status: 'idle', error: '',
  fetchChains: async (query = '') => {
    set({ status: 'loading', error: '' })
    try {
      const result = await certificateChainApi.list(query)
      set({ items: result.items, total: result.total, status: 'ready' })
    } catch (error) { set({ status: 'error', error: errorMessage(error) }) }
  },
  transition: async (id, toState) => {
    const updated = await certificateChainApi.transition(id, toState)
    set({ items: get().items.map((item) => item.id === id ? updated : item) })
  },
}))


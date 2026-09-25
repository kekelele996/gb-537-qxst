import { create } from 'zustand'
import { auditApi } from '../api/audit'
import { errorMessage } from '../api/client'
import type { AuditLog } from '../types/audit'
import type { LoadState } from '../types/common'

interface AuditState {
  items: AuditLog[]
  total: number
  status: LoadState
  error: string
  fetchAudit: (query?: string) => Promise<void>
}

export const useAuditStore = create<AuditState>((set) => ({
  items: [], total: 0, status: 'idle', error: '',
  fetchAudit: async (query = '') => {
    set({ status: 'loading', error: '' })
    try {
      const result = await auditApi.list(query)
      set({ items: result.items, total: result.total, status: 'ready' })
    } catch (error) { set({ status: 'error', error: errorMessage(error) }) }
  },
}))


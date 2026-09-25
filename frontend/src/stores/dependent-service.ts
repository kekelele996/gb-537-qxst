import { create } from 'zustand'
import { dependentServiceApi } from '../api/dependent-service'
import { errorMessage } from '../api/client'
import type { LoadState } from '../types/common'
import type { DependentService, DependentServiceInput } from '../types/dependent-service'

interface DependentServiceState {
  items: DependentService[]
  total: number
  status: LoadState
  error: string
  fetchServices: (query?: string) => Promise<void>
  createService: (input: DependentServiceInput & { service_code: string }) => Promise<DependentService>
  updateService: (id: number, input: DependentServiceInput) => Promise<DependentService>
  deactivate: (id: number) => Promise<void>
}

export const useDependentServiceStore = create<DependentServiceState>((set, get) => ({
  items: [], total: 0, status: 'idle', error: '',
  fetchServices: async (query = '') => {
    set({ status: 'loading', error: '' })
    try {
      const result = await dependentServiceApi.list(query)
      set({ items: result.items, total: result.total, status: 'ready' })
    } catch (error) { set({ status: 'error', error: errorMessage(error) }) }
  },
  createService: async (input) => {
    const created = await dependentServiceApi.create(input)
    set({ items: [...get().items, created], total: get().total + 1 })
    return created
  },
  updateService: async (id, input) => {
    const updated = await dependentServiceApi.update(id, input)
    set({ items: get().items.map((item) => item.id === id ? updated : item) })
    return updated
  },
  deactivate: async (id) => {
    const updated = await dependentServiceApi.deactivate(id)
    set({ items: get().items.map((item) => item.id === id ? updated : item) })
  },
}))


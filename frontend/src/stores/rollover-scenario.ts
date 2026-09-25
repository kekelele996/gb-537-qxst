import { create } from 'zustand'
import { rolloverScenarioApi } from '../api/rollover-scenario'
import { errorMessage } from '../api/client'
import type { LoadState } from '../types/common'
import type { CreateRolloverScenarioInput, RolloverScenario } from '../types/rollover-scenario'
import type { ScenarioState } from '../types/enums/scenario-state'

interface RolloverScenarioState {
  items: RolloverScenario[]
  total: number
  status: LoadState
  error: string
  active: RolloverScenario | null
  fetchScenarios: (query?: string) => Promise<void>
  createScenario: (input: CreateRolloverScenarioInput) => Promise<RolloverScenario>
  simulate: (id: number, key: string) => Promise<RolloverScenario>
  transition: (id: number, state: ScenarioState, comment?: string) => Promise<RolloverScenario>
  replay: (id: number) => Promise<RolloverScenario>
  select: (scenario: RolloverScenario | null) => void
}

export const useRolloverScenarioStore = create<RolloverScenarioState>((set, get) => {
  const merge = (updated: RolloverScenario) => set({
    active: updated,
    items: get().items.map((item) => item.id === updated.id ? updated : item),
  })
  return {
    items: [], total: 0, status: 'idle', error: '', active: null,
    fetchScenarios: async (query = '') => {
      set({ status: 'loading', error: '' })
      try {
        const result = await rolloverScenarioApi.list(query)
        set({ items: result.items, total: result.total, status: 'ready' })
      } catch (error) { set({ status: 'error', error: errorMessage(error) }) }
    },
    createScenario: async (input) => {
      const created = await rolloverScenarioApi.create(input)
      set({ items: [created, ...get().items], total: get().total + 1, active: created })
      return created
    },
    simulate: async (id, key) => { const updated = await rolloverScenarioApi.simulate(id, key); merge(updated); return updated },
    transition: async (id, state, comment) => { const updated = await rolloverScenarioApi.transition(id, state, comment); merge(updated); return updated },
    replay: async (id) => { const updated = await rolloverScenarioApi.replay(id); merge(updated); return updated },
    select: (scenario) => set({ active: scenario }),
  }
})


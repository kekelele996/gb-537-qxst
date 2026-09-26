import { useRolloverScenarioStore } from './rollover-scenario'
import type { ScenarioComparison } from '../types/rollover-scenario'

const comparison: ScenarioComparison = {
  first_id: 1,
  first_name: 'plan-a',
  second_id: 2,
  second_name: 'plan-b',
  algorithm_version: 'trust-path-window-v1.0.0',
  inventory_hash: 'abc123',
  first_critical_affected: 2,
  second_critical_affected: 1,
  new_impacts: [{ id: 0, code: '', state: '', service_id: 9, service_code: 'billing', criticality: 'critical', at: '2026-09-08T12:00:00Z', reason: 'trust set does not include new anchor' }],
  recovered_impacts: [],
  new_broken_paths: [],
  resolved_broken_paths: [{ at: '2026-09-08T12:00:00Z', service_codes: ['audit'], reason: 'chain expired' }],
}

function respond(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

describe('rollover scenario compare store', () => {
  beforeEach(() => {
    useRolloverScenarioStore.setState({ items: [], total: 0, status: 'idle', error: '', active: null, comparison: null, comparisonError: '' })
  })

  it('stores the comparison delta on success', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(respond(200, { code: 'OK', message: 'success', data: comparison, request_id: 'req-compare' }))

    const result = await useRolloverScenarioStore.getState().compareScenarios(1, 2)
    expect(result?.second_critical_affected).toBe(1)
    expect(useRolloverScenarioStore.getState().comparison?.new_impacts[0]?.service_code).toBe('billing')
    expect(useRolloverScenarioStore.getState().comparisonError).toBe('')
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/rollover-scenarios/1/compare/2')
  })

  it('surfaces the refusal reason when inputs cannot be combined', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(respond(409, { code: 'COMPARISON_INPUT_MISMATCH', message: 'scenarios froze different trust inventories', request_id: 'req-mismatch' }))

    const result = await useRolloverScenarioStore.getState().compareScenarios(1, 2)
    expect(result).toBeNull()
    expect(useRolloverScenarioStore.getState().comparison).toBeNull()
    expect(useRolloverScenarioStore.getState().comparisonError).toBe('scenarios froze different trust inventories')
  })

  it('clears the comparison when another scenario is selected', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(respond(200, { code: 'OK', message: 'success', data: comparison, request_id: 'req-compare' }))
    await useRolloverScenarioStore.getState().compareScenarios(1, 2)
    expect(useRolloverScenarioStore.getState().comparison).not.toBeNull()

    useRolloverScenarioStore.getState().select(null)
    expect(useRolloverScenarioStore.getState().comparison).toBeNull()
    expect(useRolloverScenarioStore.getState().comparisonError).toBe('')
  })
})

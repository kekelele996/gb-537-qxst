import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { vi } from 'vitest'
import type { RolloverScenario, ScenarioComparison } from '../../types/rollover-scenario'
import { ScenarioComparisonPanel } from './ScenarioComparisonPanel'

vi.mock('../../api/rollover-scenario', () => ({
  rolloverScenarioApi: {
    compare: vi.fn(),
  },
}))

import { rolloverScenarioApi } from '../../api/rollover-scenario'

function makeScenario(overrides: Partial<RolloverScenario>): RolloverScenario {
  return {
    id: 1,
    name: '基准方案',
    old_anchor_id: 1,
    new_anchor_id: 2,
    overlap_start: '2026-10-10T00:00:00Z',
    overlap_end: '2026-10-26T00:00:00Z',
    candidate_chain_ids: [1, 2],
    algorithm_version: 'trust-path-window-v1.0.0',
    input_hash: 'full-hash-a',
    frozen_input_hash: 'frozen-topology',
    simulation_time: '2026-10-18T00:00:00Z',
    affected_services_json: [],
    broken_paths_json: [],
    path_evidence_json: [],
    scenario_state: 'simulated',
    explanation: '',
    created_by: 1,
    created_by_name: 'operator',
    verified_by_name: '',
    replay_verified: false,
    duration_ms: 1,
    rollback_record: '',
    created_at: '2026-09-01T00:00:00Z',
    updated_at: '2026-09-01T00:00:00Z',
    ...overrides,
  } as RolloverScenario
}

const comparison: ScenarioComparison = {
  first: {
    scenario_id: 1, name: '基准方案', scenario_state: 'simulated',
    overlap_start: '2026-10-10T00:00:00Z', overlap_end: '2026-10-26T00:00:00Z',
    affected_service_count: 2, broken_path_count: 2, distinct_affected_count: 2,
    critical_affected_count: 1, criticality_breakdown: { critical: 1, high: 1, medium: 0, low: 0 },
  },
  second: {
    scenario_id: 2, name: '加宽交叠方案', scenario_state: 'simulated',
    overlap_start: '2026-10-03T00:00:00Z', overlap_end: '2026-11-09T00:00:00Z',
    affected_service_count: 1, broken_path_count: 1, distinct_affected_count: 1,
    critical_affected_count: 0, criticality_breakdown: { critical: 0, high: 1, medium: 0, low: 0 },
  },
  new_impacts: [],
  resolved_impacts: [
    { service_id: 201, service_code: 'PAYMENTS-API', criticality: 'critical', at: '2026-11-09T00:01:00Z', reason: 'anchor unavailable' },
  ],
  eliminated_broken_paths: [
    { at: '2026-11-09T00:01:00Z', service_codes: ['PAYMENTS-API'], reason: 'anchor unavailable' },
  ],
  introduced_broken_paths: [],
}

function openSelect(label: string) {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: label }))
}

describe('ScenarioComparisonPanel', () => {
  it('blocks incompatible scenarios before calling the API', () => {
    const baseline = makeScenario({})
    const mismatched = makeScenario({ id: 3, name: '旧算法方案', algorithm_version: 'trust-path-window-v0.9.0' })
    render(<ScenarioComparisonPanel baseline={baseline} scenarios={[baseline, mismatched]} />)
    openSelect('选择另一条场景')
    fireEvent.click(screen.getByRole('option', { name: /旧算法方案/ }))
    expect(screen.getByText(/算法版本不一致/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '生成差异评审' })).toBeDisabled()
  })

  it('renders new and resolved impacts with critical counts for both sides', async () => {
    const baseline = makeScenario({})
    const widened = makeScenario({ id: 2, name: '加宽交叠方案' })
    vi.mocked(rolloverScenarioApi.compare).mockResolvedValue(comparison)
    render(<ScenarioComparisonPanel baseline={baseline} scenarios={[baseline, widened]} />)
    openSelect('选择另一条场景')
    fireEvent.click(screen.getByRole('option', { name: /加宽交叠方案/ }))
    fireEvent.click(screen.getByRole('button', { name: '生成差异评审' }))
    await waitFor(() => expect(rolloverScenarioApi.compare).toHaveBeenCalledWith(1, 2))
    expect(screen.getAllByText('PAYMENTS-API').length).toBeGreaterThan(1)
    expect(screen.getAllByText('0').length).toBeGreaterThan(0)
    expect(screen.getAllByText('1').length).toBeGreaterThan(0)
    expect(screen.getByText('恢复项（故障消失）')).toBeInTheDocument()
    expect(screen.getByText('消除的断裂路径')).toBeInTheDocument()
  })

  it('shows the server rejection reason instead of spliced results', async () => {
    const baseline = makeScenario({})
    const drifted = makeScenario({ id: 4, name: '拓扑已变方案' })
    vi.mocked(rolloverScenarioApi.compare).mockRejectedValue(new Error('refusing to splice results built on different frozen trust inputs'))
    render(<ScenarioComparisonPanel baseline={baseline} scenarios={[baseline, drifted]} />)
    openSelect('选择另一条场景')
    fireEvent.click(screen.getByRole('option', { name: /拓扑已变方案/ }))
    fireEvent.click(screen.getByRole('button', { name: '生成差异评审' }))
    expect(await screen.findByText(/接口拒绝拼接/)).toBeInTheDocument()
    expect(screen.getByText(/different frozen trust inputs/)).toBeInTheDocument()
  })
})

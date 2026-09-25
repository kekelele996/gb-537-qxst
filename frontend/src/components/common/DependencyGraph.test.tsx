import { render, screen } from '@testing-library/react'
import { DependencyGraph } from './DependencyGraph'
import type { DependentService } from '../../types/dependent-service'

const base: Omit<DependentService, 'id' | 'service_code' | 'name' | 'dependency_edges_json'> = {
  owner_team: 'PKI',
  environment: 'production',
  chain_id: 1,
  client_trust_refs_json: [1],
  protocol: 'mtls',
  criticality: 'critical',
  service_state: 'active',
  created_at: '2030-01-01T00:00:00Z',
  updated_at: '2030-01-01T00:00:00Z',
}
const services: DependentService[] = [
  { ...base, id: 1, service_code: 'EDGE-AUTH', name: 'Edge Authentication', dependency_edges_json: [] },
  { ...base, id: 2, service_code: 'PAYMENTS-API', name: 'Payments', dependency_edges_json: [1] },
]

describe('DependencyGraph', () => {
  it('renders real dependency edges and affected semantics', () => {
    const { container } = render(<DependencyGraph services={services} highlightedIds={[2]} />)
    expect(screen.getByText('PAYMENTS-API')).toBeInTheDocument()
    expect(screen.getAllByText('EDGE-AUTH')).toHaveLength(2)
    expect(container.querySelector('.graph-row.is-affected')).toHaveTextContent('PAYMENTS-API')
  })

  it('renders a useful empty state', () => {
    render(<DependencyGraph services={[]} />)
    expect(screen.getByText('暂无依赖路径')).toBeInTheDocument()
  })
})

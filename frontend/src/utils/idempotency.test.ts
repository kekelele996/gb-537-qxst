import { simulationKey } from './idempotency'

describe('simulationKey', () => {
  it('scopes every key to a scenario and produces unique attempts', () => {
    const first = simulationKey(42)
    const second = simulationKey(42)
    expect(first).toMatch(/^ui-scenario-42-/)
    expect(second).toMatch(/^ui-scenario-42-/)
    expect(first).not.toBe(second)
  })
})


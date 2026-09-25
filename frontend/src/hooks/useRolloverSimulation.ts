import { useState } from 'react'
import { errorMessage } from '../api/client'
import { useRolloverScenarioStore } from '../stores/rollover-scenario'
import { simulationKey } from '../utils/idempotency'

export function useRolloverSimulation() {
  const simulate = useRolloverScenarioStore((state) => state.simulate)
  const [runningId, setRunningId] = useState<number | null>(null)
  const [error, setError] = useState('')

  const run = async (scenarioId: number) => {
    setRunningId(scenarioId)
    setError('')
    try {
      return await simulate(scenarioId, simulationKey(scenarioId))
    } catch (cause) {
      setError(errorMessage(cause))
      throw cause
    } finally {
      setRunningId(null)
    }
  }
  return { run, runningId, error, clearError: () => setError('') }
}


export function simulationKey(scenarioId: number) {
  const random = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `ui-scenario-${scenarioId}-${random}`
}


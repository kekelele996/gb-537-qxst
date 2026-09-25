export const scenarioStates = ['draft', 'simulated', 'ready', 'executing', 'verified', 'rollback'] as const
export type ScenarioState = (typeof scenarioStates)[number]


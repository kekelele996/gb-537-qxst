import { Box } from '@mui/material'
import type { ScenarioState } from '../../types/enums/scenario-state'

const labels: Record<ScenarioState, string> = {
  draft: '草稿', simulated: '已推演', ready: '待执行', executing: '演练中', verified: '已复核', rollback: '已回滚',
}

export function ScenarioStateBadge({ state }: { state: ScenarioState }) {
  return <Box component="span" className={`state-badge scenario-${state}`}><span className="state-dot" />{labels[state]}</Box>
}


import { CompareArrowsRounded, ErrorOutlineRounded, FactCheckRounded, RouteRounded } from '@mui/icons-material'
import { Alert, Box, Button, CircularProgress, FormControl, InputLabel, MenuItem, Select, Typography } from '@mui/material'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { errorMessage } from '../../api/client'
import { rolloverScenarioApi } from '../../api/rollover-scenario'
import { ToneBadge } from './ToneBadge'
import type { ImpactChange, RolloverScenario, ScenarioComparison } from '../../types/rollover-scenario'
import { formatDateTime } from '../../utils/date'

interface Props {
  baseline: RolloverScenario
  scenarios: RolloverScenario[]
}

function groupByService(changes: ImpactChange[]) {
  const groups = new Map<string, { criticality: string; items: ImpactChange[] }>()
  for (const change of changes) {
    const key = change.service_code
    const group = groups.get(key) ?? { criticality: change.criticality, items: [] }
    group.criticality = change.criticality
    group.items.push(change)
    groups.set(key, group)
  }
  return [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0]))
}

function precheckReason(baseline: RolloverScenario, other: RolloverScenario): string {
  if (other.scenario_state === 'draft') {
    return '该场景尚未运行离线推演，没有可比对的冻结结果。'
  }
  if (other.algorithm_version !== baseline.algorithm_version) {
    return `算法版本不一致：${baseline.algorithm_version} 与 ${other.algorithm_version}，拒绝拼接结果。`
  }
  if (other.frozen_input_hash && baseline.frozen_input_hash && other.frozen_input_hash !== baseline.frozen_input_hash) {
    return '冻结输入不一致：两侧的信任锚、候选证书链或服务依赖图存在差异，只有调整交叠时间的方案才能比对。'
  }
  return ''
}

export function ScenarioComparisonPanel({ baseline, scenarios }: Props) {
  const [otherId, setOtherId] = useState(0)
  const [comparison, setComparison] = useState<ScenarioComparison | null>(null)
  const [loading, setLoading] = useState(false)
  const [failure, setFailure] = useState('')

  const candidates = useMemo(
    () => scenarios.filter((scenario) => scenario.id !== baseline.id).sort((a, b) => a.name.localeCompare(b.name)),
    [scenarios, baseline.id],
  )
  const selected = candidates.find((scenario) => scenario.id === otherId) ?? null
  const clientBlocker = selected ? precheckReason(baseline, selected) : ''

  useEffect(() => {
    setComparison(null)
    setFailure('')
  }, [baseline.id, otherId])

  const runCompare = async () => {
    if (!selected) return
    setLoading(true)
    setFailure('')
    setComparison(null)
    try {
      setComparison(await rolloverScenarioApi.compare(baseline.id, selected.id))
    } catch (cause) {
      setFailure(errorMessage(cause))
    } finally {
      setLoading(false)
    }
  }

  return <Box className="scenario-compare">
    <Box className="detail-section-head"><Typography variant="h3"><CompareArrowsRounded fontSize="small" /> 双方案轮换评审</Typography><span>仅允许比对同算法、同冻结输入、仅调整交叠时间的方案</span></Box>
    <Box className="compare-controls">
      <Box className="compare-side-label"><span>基准方案</span><strong>{baseline.name}</strong><small>{formatDateTime(baseline.overlap_start)} 至 {formatDateTime(baseline.overlap_end)}</small></Box>
      <FormControl size="small" className="compare-select">
        <InputLabel id="compare-scenario-label">选择另一条场景</InputLabel>
        <Select labelId="compare-scenario-label" label="选择另一条场景" value={otherId || ''} onChange={(event) => setOtherId(Number(event.target.value))}>
          {candidates.map((scenario) => <MenuItem key={scenario.id} value={scenario.id}>{scenario.name} · {formatDateTime(scenario.overlap_start)} 起</MenuItem>)}
        </Select>
      </FormControl>
      <Button variant="contained" startIcon={loading ? <CircularProgress size={14} color="inherit" /> : <CompareArrowsRounded />} disabled={!selected || loading || !!clientBlocker} onClick={runCompare}>{loading ? '比对中…' : '生成差异评审'}</Button>
    </Box>
    {!candidates.length && <Alert severity="info" icon={<ErrorOutlineRounded fontSize="inherit" />}>当前只有一条冻结场景，无法进行双方案评审，请先创建一个仅调整交叠时间的新方案。</Alert>}
    {clientBlocker && <Alert severity="warning" icon={<ErrorOutlineRounded fontSize="inherit" />}>{clientBlocker}</Alert>}
    {failure && <Alert severity="error" icon={<ErrorOutlineRounded fontSize="inherit" />}>接口拒绝拼接：{failure}</Alert>}
    {comparison && <ComparisonResult comparison={comparison} />}
  </Box>
}

function ComparisonResult({ comparison }: { comparison: ScenarioComparison }) {
  const { first, second } = comparison
  return <Box className="compare-result">
    <Box className="compare-sides">
      <Box className="compare-side">
        <Typography className="eyebrow">基准方案 #{first.scenario_id}</Typography>
        <strong>{first.name}</strong>
        <small>{formatDateTime(first.overlap_start)} 至 {formatDateTime(first.overlap_end)}</small>
        <Box className="compare-metric is-risk"><span>关键服务受损</span><strong>{first.critical_affected_count}</strong><em>{first.affected_service_count} 个服务时间点 · {first.broken_path_count} 条断裂路径</em></Box>
      </Box>
      <CompareArrowsRounded className="compare-arrow" />
      <Box className="compare-side">
        <Typography className="eyebrow">待评方案 #{second.scenario_id}</Typography>
        <strong>{second.name}</strong>
        <small>{formatDateTime(second.overlap_start)} 至 {formatDateTime(second.overlap_end)}</small>
        <Box className={`compare-metric ${second.critical_affected_count > first.critical_affected_count ? 'is-risk' : second.critical_affected_count < first.critical_affected_count ? 'is-pass' : ''}`}><span>关键服务受损</span><strong>{second.critical_affected_count}</strong><em>{second.affected_service_count} 个服务时间点 · {second.broken_path_count} 条断裂路径</em></Box>
      </Box>
    </Box>
    <Box className="compare-criticality-strip">
      <span><ToneBadge value="critical" /> 基准 {first.criticality_breakdown.critical} → 待评 {second.criticality_breakdown.critical}</span>
      <span><ToneBadge value="high" /> 基准 {first.criticality_breakdown.high} → 待评 {second.criticality_breakdown.high}</span>
      <span><ToneBadge value="medium" /> 基准 {first.criticality_breakdown.medium} → 待评 {second.criticality_breakdown.medium}</span>
      <span><ToneBadge value="low" /> 基准 {first.criticality_breakdown.low} → 待评 {second.criticality_breakdown.low}</span>
    </Box>
    <Box className="compare-grid">
      <ImpactGroup title="新增关键服务故障" tone="is-risk" icon={<ErrorOutlineRounded />} emptyText="未新增服务时间点故障。" groups={groupByService(comparison.new_impacts)} />
      <ImpactGroup title="恢复项（故障消失）" tone="is-pass" icon={<FactCheckRounded />} emptyText="没有恢复的服务时间点。" groups={groupByService(comparison.resolved_impacts)} />
    </Box>
    <Box className="compare-grid">
      <PathGroup title="消除的断裂路径" tone="is-pass" icon={<RouteRounded />} emptyText="没有断裂路径被消除。" paths={comparison.eliminated_broken_paths} />
      <PathGroup title="新增的断裂路径" tone="is-risk" icon={<RouteRounded />} emptyText="未引入新的断裂路径。" paths={comparison.introduced_broken_paths} />
    </Box>
  </Box>
}

function ImpactGroup({ title, tone, icon, emptyText, groups }: { title: string; tone: string; icon: ReactNode; emptyText: string; groups: [string, { criticality: string; items: ImpactChange[] }][] }) {
  return <section className={`compare-group ${tone}`}>
    <Box className="compare-group-head">{icon}<Typography variant="h4">{title}</Typography><span>{groups.reduce((sum, [, group]) => sum + group.items.length, 0)}</span></Box>
    {groups.length === 0 && <Box className="compare-empty">{emptyText}</Box>}
    <Box className="impact-groups">
      {groups.map(([code, group]) => <Box className="impact-service" key={code}>
        <Box className="impact-service-head"><strong>{code}</strong><ToneBadge value={group.criticality || 'low'} /></Box>
        {group.items.map((item, index) => <Box className="impact-row" key={`${item.at}-${index}`}><span>{formatDateTime(item.at)}</span><Typography>{item.reason}</Typography></Box>)}
      </Box>)}
    </Box>
  </section>
}

function PathGroup({ title, tone, icon, emptyText, paths }: { title: string; tone: string; icon: ReactNode; emptyText: string; paths: ScenarioComparison['eliminated_broken_paths'] }) {
  return <section className={`compare-group ${tone}`}>
    <Box className="compare-group-head">{icon}<Typography variant="h4">{title}</Typography><span>{paths.length}</span></Box>
    {paths.length === 0 && <Box className="compare-empty">{emptyText}</Box>}
    <Box className="broken-paths compare-paths">{paths.map((path, index) => <Box key={`${path.at}-${index}`}><span>{formatDateTime(path.at)}</span><strong>{path.service_codes.join(' → ')}</strong><Typography>{path.reason}</Typography></Box>)}</Box>
  </section>
}

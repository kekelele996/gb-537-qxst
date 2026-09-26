import { AddRounded, ArrowForwardRounded, AutorenewRounded, CompareArrowsRounded, FactCheckRounded, KeyboardArrowRightRounded, PlayArrowRounded, RefreshRounded, ReplayRounded, RouteRounded, ScienceRounded } from '@mui/icons-material'
import { Alert, Box, Button, Checkbox, FormControl, IconButton, InputLabel, ListItemText, MenuItem, Select, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography } from '@mui/material'
import { FormEvent, useEffect, useState } from 'react'
import { errorMessage } from '../api/client'
import { DependencyGraph } from '../components/common/DependencyGraph'
import { EmptyState, ErrorState, LoadingState } from '../components/common/DataState'
import { Fingerprint } from '../components/common/Fingerprint'
import { FormDrawer } from '../components/common/FormDrawer'
import { PageHeader } from '../components/common/PageHeader'
import { ScenarioStateBadge } from '../components/common/ScenarioStateBadge'
import { StatStrip } from '../components/common/StatStrip'
import { ValidationEvidenceDrawer } from '../components/common/ValidationEvidenceDrawer'
import { useAuth } from '../hooks/useAuth'
import { useRolloverSimulation } from '../hooks/useRolloverSimulation'
import { useCertificateChainStore } from '../stores/certificate-chain'
import { useDependentServiceStore } from '../stores/dependent-service'
import { useRolloverScenarioStore } from '../stores/rollover-scenario'
import { useTrustAnchorStore } from '../stores/trust-anchor'
import type { CreateRolloverScenarioInput, RolloverScenario } from '../types/rollover-scenario'
import type { ScenarioState } from '../types/enums/scenario-state'
import { formatDateTime, toLocalInput } from '../utils/date'

function defaultScenario(): CreateRolloverScenarioInput {
  const start = new Date(Date.now() + 7 * 86_400_000)
  const end = new Date(Date.now() + 21 * 86_400_000)
  const simulation = new Date(Date.now() + 14 * 86_400_000)
  return { name: '', old_anchor_id: 0, new_anchor_id: 0, overlap_start: toLocalInput(start), overlap_end: toLocalInput(end), candidate_chain_ids: [], simulation_time: toLocalInput(simulation) }
}

const transitionCopy: Partial<Record<ScenarioState, { to: ScenarioState; label: string }>> = {
  simulated: { to: 'ready', label: '标记待执行' },
  ready: { to: 'executing', label: '记录演练开始' },
  executing: { to: 'verified', label: '独立复核通过' },
}

export function RolloversPage() {
  const { items, total, status, error, active, comparison, comparisonError, fetchScenarios, createScenario, transition, replay, select, compareScenarios, clearComparison } = useRolloverScenarioStore()
  const { items: anchors, fetchAnchors } = useTrustAnchorStore()
  const { items: chains, fetchChains } = useCertificateChainStore()
  const { items: services, fetchServices } = useDependentServiceStore()
  const { can, user } = useAuth()
  const simulation = useRolloverSimulation()
  const [createOpen, setCreateOpen] = useState(false)
  const [evidenceOpen, setEvidenceOpen] = useState(false)
  const [form, setForm] = useState<CreateRolloverScenarioInput>(defaultScenario)
  const [feedback, setFeedback] = useState('')
  const [success, setSuccess] = useState('')
  const [busy, setBusy] = useState(false)
  const [compareTarget, setCompareTarget] = useState<number | ''>('')
  const [compareBusy, setCompareBusy] = useState(false)

  useEffect(() => { void fetchScenarios(); void fetchAnchors(); void fetchChains(); void fetchServices() }, [fetchAnchors, fetchChains, fetchScenarios, fetchServices])
  useEffect(() => { if (!active && items.length) select(items[0]) }, [active, items, select])
  useEffect(() => { setCompareTarget('') }, [active?.id])
  const affectedIds = active?.affected_services_json.map((item) => item.service_id ?? item.id).filter(Boolean) as number[] | undefined

  const openCreate = () => {
    const next = defaultScenario()
    next.old_anchor_id = anchors[0]?.id ?? 0
    next.new_anchor_id = anchors[1]?.id ?? 0
    next.candidate_chain_ids = chains.map((chain) => chain.id)
    setForm(next); setFeedback(''); setCreateOpen(true)
  }
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setFeedback(''); setSuccess('')
    try {
      const created = await createScenario({ ...form, overlap_start: new Date(form.overlap_start).toISOString(), overlap_end: new Date(form.overlap_end).toISOString(), simulation_time: new Date(form.simulation_time).toISOString() })
      setCreateOpen(false); setSuccess(`${created.name} 已冻结输入，可运行离线推演。`)
    } catch (cause) { setFeedback(errorMessage(cause)) } finally { setBusy(false) }
  }
  const runSimulation = async (scenario: RolloverScenario) => {
    setFeedback(''); setSuccess('')
    try { const updated = await simulation.run(scenario.id); setSuccess(`推演完成：记录 ${updated.path_evidence_json.length} 个时间点，发现 ${updated.broken_paths_json.length} 条断裂路径。`) }
    catch (cause) { setFeedback(errorMessage(cause)) }
  }
  const transitionActive = async (to: ScenarioState) => {
    if (!active) return; setBusy(true); setFeedback(''); setSuccess('')
    try { const updated = await transition(active.id, to); setSuccess(`场景状态已更新为 ${updated.scenario_state}。`) }
    catch (cause) { setFeedback(errorMessage(cause)) } finally { setBusy(false) }
  }
  const replayActive = async () => {
    if (!active) return; setBusy(true); setFeedback(''); setSuccess('')
    try { const updated = await replay(active.id); setSuccess(updated.replay_verified ? '重放结果与冻结历史证据一致。' : '重放结果不一致。') }
    catch (cause) { setFeedback(errorMessage(cause)) } finally { setBusy(false) }
  }
  const runCompare = async () => {
    if (!active || !compareTarget) return
    setCompareBusy(true)
    try { await compareScenarios(active.id, compareTarget) }
    finally { setCompareBusy(false) }
  }

  const comparableItems = items.filter((item) => item.id !== active?.id && item.scenario_state !== 'draft')

  const next = active ? transitionCopy[active.scenario_state] : undefined
  const canAdvance = next && ((next.to === 'verified' && can('scenario.verify')) || (next.to !== 'verified' && can('scenario.write')))
  const reviewerConflict = active?.scenario_state === 'executing' && active.created_by === user?.user_id

  return <Box className="page-shell rollover-page">
    <PageHeader eyebrow="ROLLOVER REHEARSAL / FROZEN SNAPSHOTS" title="轮换推演" summary="在旧根、新根和交叠窗口的关键时间点重放服务信任路径。executing 仅记录演练步骤，不执行生产变更。" actions={<><Tooltip title="刷新"><IconButton onClick={() => fetchScenarios()} aria-label="刷新轮换推演"><RefreshRounded /></IconButton></Tooltip>{can('scenario.write') && <Button variant="contained" startIcon={<AddRounded />} onClick={openCreate}>新建冻结场景</Button>}</>} />
    <StatStrip items={[{ label: '场景总数', value: total }, { label: '待推演', value: items.filter((item) => item.scenario_state === 'draft').length }, { label: '断裂路径', value: items.reduce((sum, item) => sum + item.broken_paths_json.length, 0), tone: 'is-danger' }, { label: '已独立复核', value: items.filter((item) => item.scenario_state === 'verified').length, tone: 'is-good' }]} />
    {feedback && <Alert severity="error" onClose={() => setFeedback('')}>{feedback}</Alert>}{success && <Alert severity="success" onClose={() => setSuccess('')}>{success}</Alert>}{reviewerConflict && <Alert severity="warning">当前账号是场景创建者，不能复核自己的推演。请由独立安全复核员完成验证。</Alert>}
    <Box className="scenario-layout">
      <section className="scenario-rail">
        <Box className="section-title compact"><Box><Typography className="eyebrow">SCENARIO REGISTER</Typography><Typography variant="h2">冻结场景</Typography></Box><span>{items.length}</span></Box>
        {status === 'loading' && <LoadingState />}{status === 'error' && <ErrorState message={error} retry={() => fetchScenarios()} />}{status === 'ready' && !items.length && <EmptyState title="还没有轮换场景" detail="冻结锚点、证书链和服务图后才能运行推演。" />}
        <Box className="scenario-list">{items.map((scenario) => <Button className={active?.id === scenario.id ? 'scenario-row selected' : 'scenario-row'} key={scenario.id} onClick={() => select(scenario)}><Box><Typography component="strong">{scenario.name}</Typography><Typography>{formatDateTime(scenario.simulation_time)}</Typography></Box><Box><ScenarioStateBadge state={scenario.scenario_state} /><KeyboardArrowRightRounded /></Box></Button>)}</Box>
      </section>
      <section className="scenario-detail">
        {active ? <>
          <Box className="scenario-title"><Box><Typography className="eyebrow">SCENARIO #{active.id} · {active.algorithm_version}</Typography><Typography variant="h2">{active.name}</Typography><Typography>由 {active.created_by_name} 创建 · {formatDateTime(active.created_at)}</Typography></Box><ScenarioStateBadge state={active.scenario_state} /></Box>
          <Box className="anchor-transition"><Box><span>旧信任锚</span><strong>{active.old_anchor?.anchor_code ?? `#${active.old_anchor_id}`}</strong><small>{active.old_anchor?.fingerprint_sha256 && <Fingerprint value={active.old_anchor.fingerprint_sha256} compact />}</small></Box><Box className="transition-axis"><ArrowForwardRounded /><span>{formatDateTime(active.overlap_start)}<br />至 {formatDateTime(active.overlap_end)}</span></Box><Box><span>新信任锚</span><strong>{active.new_anchor?.anchor_code ?? `#${active.new_anchor_id}`}</strong><small>{active.new_anchor?.fingerprint_sha256 && <Fingerprint value={active.new_anchor.fingerprint_sha256} compact />}</small></Box></Box>
          <Box className="scenario-evidence-grid"><Box><Typography className="eyebrow">SIMULATION TIME</Typography><strong>{formatDateTime(active.simulation_time)}</strong><span>耗时 {active.duration_ms} ms</span></Box><Box><Typography className="eyebrow">INPUT HASH</Typography><Fingerprint value={active.input_hash} compact /></Box><Box className={active.broken_paths_json.length ? 'is-risk' : 'is-pass'}><Typography className="eyebrow">BROKEN PATHS</Typography><strong>{active.broken_paths_json.length}</strong><span>{active.affected_services_json.length} 个受影响服务</span></Box></Box>
          <Box className="simulation-explanation"><ScienceRounded /><Typography>{active.explanation}</Typography></Box>
          <Box className="scenario-toolbar">
            {active.scenario_state === 'draft' && can('scenario.run') && <Button variant="contained" startIcon={<PlayArrowRounded />} disabled={simulation.runningId === active.id} onClick={() => runSimulation(active)}>{simulation.runningId === active.id ? '正在推演…' : '运行离线推演'}</Button>}
            {canAdvance && !reviewerConflict && <Button variant="contained" startIcon={next?.to === 'verified' ? <FactCheckRounded /> : <ArrowForwardRounded />} disabled={busy} onClick={() => next && transitionActive(next.to)}>{next?.label}</Button>}
            {active.scenario_state !== 'draft' && can('scenario.run') && <Button variant="outlined" startIcon={<ReplayRounded />} disabled={busy} onClick={replayActive}>重放一致性</Button>}
            {!!active.path_evidence_json.length && <Button variant="outlined" startIcon={<RouteRounded />} onClick={() => setEvidenceOpen(true)}>逐路径证据</Button>}
            {active.scenario_state === 'executing' && can('scenario.write') && <Button color="error" variant="text" startIcon={<AutorenewRounded />} onClick={() => transitionActive('rollback')}>记录回滚</Button>}
          </Box>
          <Box className="rollover-lower-grid"><section><Box className="detail-section-head"><Typography variant="h3">服务可达性</Typography><span>{affectedIds?.length ?? 0} 受影响</span></Box><DependencyGraph services={services} highlightedIds={affectedIds} /></section><section><Box className="detail-section-head"><Typography variant="h3">断裂路径</Typography><span>{active.broken_paths_json.length}</span></Box><Box className="broken-paths">{active.broken_paths_json.map((path, index) => <Box key={`${path.at}-${index}`}><span>{formatDateTime(path.at)}</span><strong>{path.service_codes.join(' → ')}</strong><Typography>{path.reason}</Typography></Box>)}{!active.broken_paths_json.length && <Box className="no-broken-paths"><FactCheckRounded /><span>当前证据未发现断裂路径</span></Box>}</Box></section></Box>
          <section className="compare-panel">
            <Box className="detail-section-head"><Typography variant="h3">方案对比</Typography><span>{comparison ? `#${comparison.first_id} vs #${comparison.second_id}` : '选择另一条已推演场景'}</span></Box>
            <Box className="compare-toolbar">
              <FormControl size="small" className="compare-select">
                <InputLabel>对比场景</InputLabel>
                <Select label="对比场景" value={compareTarget} onChange={(event) => setCompareTarget(Number(event.target.value))}>
                  {comparableItems.map((scenario) => <MenuItem key={scenario.id} value={scenario.id}>#{scenario.id} · {scenario.name}</MenuItem>)}
                </Select>
              </FormControl>
              <Button variant="outlined" startIcon={<CompareArrowsRounded />} disabled={!compareTarget || compareBusy} onClick={runCompare}>{compareBusy ? '正在对比…' : '对比方案'}</Button>
            </Box>
            {!comparableItems.length && <Alert severity="info">没有其他已完成推演的场景可对比。先冻结并运行另一条调整过交叠时间的方案。</Alert>}
            {comparisonError && <Alert severity="error" onClose={clearComparison}>{comparisonError}</Alert>}
            {comparison && <>
              <Box className="compare-grid">
                <Box><Typography className="eyebrow">{comparison.first_name} · 关键服务受损</Typography><strong>{comparison.first_critical_affected}</strong><span>场景 #{comparison.first_id}</span></Box>
                <Box className={comparison.second_critical_affected > comparison.first_critical_affected ? 'is-risk' : 'is-pass'}><Typography className="eyebrow">{comparison.second_name} · 关键服务受损</Typography><strong>{comparison.second_critical_affected}</strong><span>场景 #{comparison.second_id}</span></Box>
                <Box><Typography className="eyebrow">新增影响</Typography><strong>{comparison.new_impacts.length}</strong><span>第二方案新出现的故障</span></Box>
                <Box><Typography className="eyebrow">恢复项</Typography><strong>{comparison.recovered_impacts.length}</strong><span>第二方案消除的故障</span></Box>
              </Box>
              <Box className="compare-lists">
                <section>
                  <Box className="detail-section-head"><Typography variant="h3">新增影响</Typography><span>{comparison.new_impacts.length}</span></Box>
                  <Box className="impact-list is-new">{comparison.new_impacts.map((impact, index) => <Box key={`${impact.service_id}-${impact.at}-${index}`}><span>{formatDateTime(impact.at)} · {impact.criticality}</span><strong>{impact.service_code}</strong><Typography>{impact.reason}</Typography></Box>)}{!comparison.new_impacts.length && <Box className="impact-empty"><FactCheckRounded /><span>第二方案没有新增服务故障</span></Box>}</Box>
                  <Box className="detail-section-head"><Typography variant="h3">新增断裂路径</Typography><span>{comparison.new_broken_paths.length}</span></Box>
                  <Box className="broken-paths">{comparison.new_broken_paths.map((path, index) => <Box key={`${path.at}-${index}`}><span>{formatDateTime(path.at)}</span><strong>{path.service_codes.join(' → ')}</strong><Typography>{path.reason}</Typography></Box>)}{!comparison.new_broken_paths.length && <Box className="no-broken-paths"><FactCheckRounded /><span>没有新增断裂路径</span></Box>}</Box>
                </section>
                <section>
                  <Box className="detail-section-head"><Typography variant="h3">恢复项</Typography><span>{comparison.recovered_impacts.length}</span></Box>
                  <Box className="impact-list is-recovered">{comparison.recovered_impacts.map((impact, index) => <Box key={`${impact.service_id}-${impact.at}-${index}`}><span>{formatDateTime(impact.at)} · {impact.criticality}</span><strong>{impact.service_code}</strong><Typography>{impact.reason}</Typography></Box>)}{!comparison.recovered_impacts.length && <Box className="impact-empty"><AutorenewRounded /><span>第二方案没有恢复任何故障</span></Box>}</Box>
                  <Box className="detail-section-head"><Typography variant="h3">已消除断裂路径</Typography><span>{comparison.resolved_broken_paths.length}</span></Box>
                  <Box className="broken-paths is-resolved">{comparison.resolved_broken_paths.map((path, index) => <Box key={`${path.at}-${index}`}><span>{formatDateTime(path.at)}</span><strong>{path.service_codes.join(' → ')}</strong><Typography>{path.reason}</Typography></Box>)}{!comparison.resolved_broken_paths.length && <Box className="no-broken-paths"><FactCheckRounded /><span>没有断裂路径被消除</span></Box>}</Box>
                </section>
              </Box>
            </>}
          </section>
        </> : <Box className="detail-placeholder"><Typography>选择一个冻结场景查看推演证据。</Typography></Box>}
      </section>
    </Box>
    {active && <ValidationEvidenceDrawer open={evidenceOpen} onClose={() => setEvidenceOpen(false)} title={active.name} subtitle={`${active.algorithm_version} · ${active.input_hash.slice(0, 16)}`} pathEvidence={active.path_evidence_json} />}
    <FormDrawer open={createOpen} onClose={() => setCreateOpen(false)} eyebrow="FREEZE INPUT SNAPSHOT" title="新建轮换场景">
      <Alert severity="warning">场景创建后会冻结当前锚点、证书链和服务依赖图。推演只提供决策证据，不能替代生产变更评审与双人复核。</Alert>
      <Box component="form" className="drawer-form" onSubmit={submit}>{feedback && <Alert severity="error">{feedback}</Alert>}<TextField label="场景名称" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
        <Box className="form-grid"><FormControl required><InputLabel>旧信任锚</InputLabel><Select label="旧信任锚" value={form.old_anchor_id || ''} onChange={(event) => setForm({ ...form, old_anchor_id: Number(event.target.value) })}>{anchors.map((anchor) => <MenuItem key={anchor.id} value={anchor.id}>{anchor.anchor_code}</MenuItem>)}</Select></FormControl><FormControl required><InputLabel>新信任锚</InputLabel><Select label="新信任锚" value={form.new_anchor_id || ''} onChange={(event) => setForm({ ...form, new_anchor_id: Number(event.target.value) })}>{anchors.map((anchor) => <MenuItem key={anchor.id} value={anchor.id}>{anchor.anchor_code}</MenuItem>)}</Select></FormControl></Box>
        <FormControl required><InputLabel>候选证书链</InputLabel><Select multiple label="候选证书链" value={form.candidate_chain_ids} onChange={(event) => setForm({ ...form, candidate_chain_ids: event.target.value as number[] })} renderValue={(values) => values.map((id) => chains.find((chain) => chain.id === id)?.chain_code ?? id).join(', ')}>{chains.map((chain) => <MenuItem key={chain.id} value={chain.id}><Checkbox checked={form.candidate_chain_ids.includes(chain.id)} /><ListItemText primary={chain.chain_code} secondary={chain.leaf_subject} /></MenuItem>)}</Select></FormControl>
        <Box className="form-grid"><TextField label="交叠开始" type="datetime-local" value={form.overlap_start} onChange={(event) => setForm({ ...form, overlap_start: event.target.value })} slotProps={{ inputLabel: { shrink: true } }} required /><TextField label="交叠结束" type="datetime-local" value={form.overlap_end} onChange={(event) => setForm({ ...form, overlap_end: event.target.value })} slotProps={{ inputLabel: { shrink: true } }} required /></Box><TextField label="模拟时间" type="datetime-local" value={form.simulation_time} onChange={(event) => setForm({ ...form, simulation_time: event.target.value })} slotProps={{ inputLabel: { shrink: true } }} required />
        <Button type="submit" variant="contained" disabled={busy}>{busy ? '正在冻结输入…' : '冻结输入并创建草稿'}</Button>
      </Box>
    </FormDrawer>
  </Box>
}

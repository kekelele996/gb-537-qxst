import { AddRounded, EditRounded, HubRounded, KeyboardArrowRightRounded, PowerSettingsNewRounded, RefreshRounded, SearchRounded } from '@mui/icons-material'
import { Alert, Box, Button, Checkbox, FormControl, IconButton, InputAdornment, InputLabel, ListItemText, MenuItem, Select, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography } from '@mui/material'
import { FormEvent, useEffect, useMemo, useState } from 'react'
import { errorMessage } from '../api/client'
import { DependencyGraph } from '../components/common/DependencyGraph'
import { EmptyState, ErrorState, LoadingState } from '../components/common/DataState'
import { FormDrawer } from '../components/common/FormDrawer'
import { PageHeader } from '../components/common/PageHeader'
import { StatStrip } from '../components/common/StatStrip'
import { ToneBadge } from '../components/common/ToneBadge'
import { useAuth } from '../hooks/useAuth'
import { useCertificateChainStore } from '../stores/certificate-chain'
import { useDependentServiceStore } from '../stores/dependent-service'
import { useTrustAnchorStore } from '../stores/trust-anchor'
import type { Criticality, DependentService, DependentServiceInput, Environment, Protocol } from '../types/dependent-service'
import { formatDateTime } from '../utils/date'

interface ServiceForm extends DependentServiceInput { service_code: string }
const emptyForm: ServiceForm = { service_code: '', name: '', owner_team: '', environment: 'production', chain_id: 0, client_trust_refs_json: [], protocol: 'mtls', criticality: 'medium', dependency_edges_json: [] }

export function DependenciesPage() {
  const { items, total, status, error, fetchServices, createService, updateService, deactivate } = useDependentServiceStore()
  const { items: chains, fetchChains } = useCertificateChainStore()
  const { items: anchors, fetchAnchors } = useTrustAnchorStore()
  const { can, user } = useAuth()
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<DependentService | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<DependentService | null>(null)
  const [form, setForm] = useState<ServiceForm>(emptyForm)
  const [feedback, setFeedback] = useState('')
  const [success, setSuccess] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => { void fetchServices(); void fetchChains(); void fetchAnchors() }, [fetchAnchors, fetchChains, fetchServices])
  useEffect(() => { if (!selected && items.length) setSelected(items[0]) }, [items, selected])
  const visible = useMemo(() => items.filter((item) => `${item.service_code} ${item.name} ${item.owner_team}`.toLowerCase().includes(search.toLowerCase())), [items, search])
  const cycleHint = items.some((service) => service.dependency_edges_json.includes(service.id))

  const openCreate = () => {
    setEditing(null)
    setForm({ ...emptyForm, owner_team: user?.team ?? '', chain_id: chains[0]?.id ?? 0, client_trust_refs_json: anchors.map((anchor) => anchor.id) })
    setFeedback(''); setFormOpen(true)
  }
  const openEdit = (service: DependentService) => {
    setEditing(service)
    setForm({ service_code: service.service_code, name: service.name, owner_team: service.owner_team, environment: service.environment, chain_id: service.chain_id, client_trust_refs_json: service.client_trust_refs_json, protocol: service.protocol, criticality: service.criticality, dependency_edges_json: service.dependency_edges_json })
    setFeedback(''); setFormOpen(true)
  }
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setFeedback(''); setSuccess('')
    try {
      const saved = editing ? await updateService(editing.id, form) : await createService(form)
      setSelected(saved); setFormOpen(false); setSuccess(editing ? `${saved.service_code} 已更新，依赖图已刷新。` : `${saved.service_code} 已登记，依赖图已刷新。`)
    } catch (cause) { setFeedback(errorMessage(cause)) } finally { setBusy(false) }
  }
  const deactivateSelected = async () => {
    if (!selected) return; setBusy(true); setFeedback('')
    try { await deactivate(selected.id); await fetchServices(); setSelected({ ...selected, service_state: 'inactive' }); setSuccess(`${selected.service_code} 已停用。`) }
    catch (cause) { setFeedback(errorMessage(cause)) } finally { setBusy(false) }
  }

  return <Box className="page-shell">
    <PageHeader eyebrow="SERVICE REACHABILITY / DEPENDENCY GRAPH" title="服务依赖" summary="登记服务的证书链、客户端信任集合和下游依赖。写入前执行存在性、团队归属与循环依赖检查。" actions={<><Tooltip title="刷新"><IconButton onClick={() => fetchServices()} aria-label="刷新服务依赖"><RefreshRounded /></IconButton></Tooltip>{can('dependency.write') && <Button variant="contained" startIcon={<AddRounded />} onClick={openCreate}>登记服务</Button>}</>} />
    <StatStrip items={[{ label: '依赖服务', value: total }, { label: '生产环境', value: items.filter((item) => item.environment === 'production').length }, { label: '关键服务', value: items.filter((item) => item.criticality === 'critical').length, tone: 'is-warning' }, { label: '循环依赖', value: cycleHint ? '需处理' : '未发现', tone: cycleHint ? 'is-danger' : 'is-good' }]} />
    {feedback && <Alert severity="error" onClose={() => setFeedback('')}>{feedback}</Alert>}{success && <Alert severity="success" onClose={() => setSuccess('')}>{success}</Alert>}
    <section className="graph-band"><Box className="section-title"><Box><Typography className="eyebrow">CURRENT REACHABILITY</Typography><Typography variant="h2">当前服务路径</Typography></Box><HubRounded /></Box><DependencyGraph services={items} /></section>
    <Box className="split-workbench dependency-split">
      <section className="workbench-list">
        <Box className="list-toolbar"><TextField size="small" placeholder="搜索服务代码、名称或团队" value={search} onChange={(event) => setSearch(event.target.value)} slotProps={{ input: { startAdornment: <InputAdornment position="start"><SearchRounded fontSize="small" /></InputAdornment> } }} /><Typography>{visible.length} 条记录</Typography></Box>
        {status === 'loading' && <LoadingState />}{status === 'error' && <ErrorState message={error} retry={() => fetchServices()} />}{status === 'ready' && !visible.length && <EmptyState title="没有匹配的服务" detail="调整检索条件，或登记一个使用现有证书链的服务。" />}
        {!!visible.length && <TableContainer className="evidence-table"><Table size="small"><TableHead><TableRow><TableCell>服务</TableCell><TableCell>环境</TableCell><TableCell>重要度</TableCell><TableCell>依赖</TableCell><TableCell /></TableRow></TableHead><TableBody>{visible.map((service) => <TableRow key={service.id} hover selected={selected?.id === service.id} onClick={() => setSelected(service)}><TableCell><strong>{service.service_code}</strong><span>{service.name}</span></TableCell><TableCell><ToneBadge value={service.environment} /></TableCell><TableCell><ToneBadge value={service.criticality} /></TableCell><TableCell>{service.dependency_edges_json.length}</TableCell><TableCell><KeyboardArrowRightRounded fontSize="small" /></TableCell></TableRow>)}</TableBody></Table></TableContainer>}
      </section>
      <aside className="workbench-detail">
        {selected ? <><Box className="detail-heading"><Box><Typography className="eyebrow">SERVICE #{selected.id}</Typography><Typography variant="h2">{selected.service_code}</Typography></Box><ToneBadge value={selected.service_state} /></Box><Typography className="detail-lead">{selected.name}</Typography>
          <dl className="detail-grid"><div><dt>责任团队</dt><dd>{selected.owner_team}</dd></div><div><dt>环境 / 协议</dt><dd><ToneBadge value={selected.environment} /> {selected.protocol.toUpperCase()}</dd></div><div><dt>证书链</dt><dd>{selected.chain?.chain_code ?? `#${selected.chain_id}`}</dd></div><div><dt>重要度</dt><dd><ToneBadge value={selected.criticality} /></dd></div><div className="wide"><dt>客户端信任锚</dt><dd className="id-list">{selected.client_trust_refs_json.map((id) => <span key={id}>{anchors.find((anchor) => anchor.id === id)?.anchor_code ?? `#${id}`}</span>)}</dd></div><div className="wide"><dt>下游依赖</dt><dd className="id-list">{selected.dependency_edges_json.length ? selected.dependency_edges_json.map((id) => <span key={id}>{items.find((item) => item.id === id)?.service_code ?? `#${id}`}</span>) : '无'}</dd></div></dl>
          <Box className="detail-footer"><Typography>更新于 {formatDateTime(selected.updated_at)}</Typography>{can('dependency.write') && <Box className="button-pair"><Button variant="outlined" startIcon={<EditRounded />} onClick={() => openEdit(selected)}>编辑关系</Button>{selected.service_state === 'active' && <Button color="error" variant="text" startIcon={<PowerSettingsNewRounded />} disabled={busy} onClick={deactivateSelected}>停用</Button>}</Box>}</Box>
        </> : <Box className="detail-placeholder"><Typography>选择一个服务查看信任集合和依赖边。</Typography></Box>}
      </aside>
    </Box>
    <FormDrawer open={formOpen} onClose={() => setFormOpen(false)} eyebrow={editing ? 'DEPENDENCY CHANGE' : 'SERVICE REGISTRATION'} title={editing ? `编辑 ${editing.service_code}` : '登记依赖服务'}>
      <Alert severity="info">保存时会校验责任团队、证书链、信任锚和所有依赖边；形成环路的变更会被拒绝。</Alert>
      <Box component="form" className="drawer-form" onSubmit={submit}>{feedback && <Alert severity="error">{feedback}</Alert>}<TextField label="服务代码" value={form.service_code} disabled={!!editing} onChange={(event) => setForm({ ...form, service_code: event.target.value.toUpperCase() })} required /><TextField label="服务名称" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required /><TextField label="责任团队" value={form.owner_team} onChange={(event) => setForm({ ...form, owner_team: event.target.value })} required />
        <Box className="form-grid"><FormControl required><InputLabel>环境</InputLabel><Select label="环境" value={form.environment} onChange={(event) => setForm({ ...form, environment: event.target.value as Environment })}>{['production', 'staging', 'development', 'disaster_recovery'].map((value) => <MenuItem key={value} value={value}><ToneBadge value={value} /></MenuItem>)}</Select></FormControl><FormControl required><InputLabel>重要度</InputLabel><Select label="重要度" value={form.criticality} onChange={(event) => setForm({ ...form, criticality: event.target.value as Criticality })}>{['critical', 'high', 'medium', 'low'].map((value) => <MenuItem key={value} value={value}><ToneBadge value={value} /></MenuItem>)}</Select></FormControl></Box>
        <Box className="form-grid"><FormControl required><InputLabel>证书链</InputLabel><Select label="证书链" value={form.chain_id || ''} onChange={(event) => setForm({ ...form, chain_id: Number(event.target.value) })}>{chains.map((chain) => <MenuItem key={chain.id} value={chain.id}>{chain.chain_code}</MenuItem>)}</Select></FormControl><FormControl required><InputLabel>协议</InputLabel><Select label="协议" value={form.protocol} onChange={(event) => setForm({ ...form, protocol: event.target.value as Protocol })}>{['tls', 'mtls', 'ldaps', 'smtps', 'kafka_tls'].map((value) => <MenuItem key={value} value={value}>{value.toUpperCase()}</MenuItem>)}</Select></FormControl></Box>
        <FormControl required><InputLabel>客户端信任锚</InputLabel><Select multiple label="客户端信任锚" value={form.client_trust_refs_json} onChange={(event) => setForm({ ...form, client_trust_refs_json: event.target.value as number[] })} renderValue={(values) => values.map((id) => anchors.find((anchor) => anchor.id === id)?.anchor_code ?? id).join(', ')}>{anchors.map((anchor) => <MenuItem key={anchor.id} value={anchor.id}><Checkbox checked={form.client_trust_refs_json.includes(anchor.id)} /><ListItemText primary={anchor.anchor_code} secondary={anchor.subject_dn} /></MenuItem>)}</Select></FormControl>
        <FormControl><InputLabel>下游依赖</InputLabel><Select multiple label="下游依赖" value={form.dependency_edges_json} onChange={(event) => setForm({ ...form, dependency_edges_json: event.target.value as number[] })} renderValue={(values) => values.map((id) => items.find((item) => item.id === id)?.service_code ?? id).join(', ')}>{items.filter((item) => item.id !== editing?.id).map((service) => <MenuItem key={service.id} value={service.id}><Checkbox checked={form.dependency_edges_json.includes(service.id)} /><ListItemText primary={service.service_code} secondary={service.name} /></MenuItem>)}</Select></FormControl>
        <Button type="submit" variant="contained" disabled={busy}>{busy ? '正在校验…' : editing ? '保存依赖关系' : '登记服务'}</Button>
      </Box>
    </FormDrawer>
  </Box>
}

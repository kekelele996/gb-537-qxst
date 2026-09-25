import { FactCheckRounded, FilterAltRounded, KeyboardArrowRightRounded, RefreshRounded, SearchRounded, ShieldRounded } from '@mui/icons-material'
import { Alert, Box, Button, FormControl, IconButton, InputAdornment, InputLabel, MenuItem, Select, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography } from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import { EmptyState, ErrorState, LoadingState } from '../components/common/DataState'
import { FormDrawer } from '../components/common/FormDrawer'
import { PageHeader } from '../components/common/PageHeader'
import { StatStrip } from '../components/common/StatStrip'
import { ToneBadge } from '../components/common/ToneBadge'
import { ValidationEvidenceDrawer } from '../components/common/ValidationEvidenceDrawer'
import { useAuth } from '../hooks/useAuth'
import { useAuditStore } from '../stores/audit'
import type { AuditLog } from '../types/audit'
import type { TimepointEvidence } from '../types/rollover-scenario'
import { formatDateTime } from '../utils/date'

const entityLabels: Record<string, string> = { trust_anchor: '信任锚', certificate_chain: '证书链', dependent_service: '依赖服务', rollover_scenario: '轮换场景' }

function extractPathEvidence(entry: AuditLog | null): TimepointEvidence[] {
  if (!entry || entry.entity_type !== 'rollover_scenario') return []
  try {
    const snapshot = JSON.parse(entry.after_snapshot) as { path_evidence_json?: string | TimepointEvidence[] }
    if (Array.isArray(snapshot.path_evidence_json)) return snapshot.path_evidence_json
    return snapshot.path_evidence_json ? JSON.parse(snapshot.path_evidence_json) as TimepointEvidence[] : []
  } catch { return [] }
}

export function AuditPage() {
  const { items, total, status, error, fetchAudit } = useAuditStore()
  const { can } = useAuth()
  const [search, setSearch] = useState('')
  const [entity, setEntity] = useState('')
  const [selected, setSelected] = useState<AuditLog | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [evidenceOpen, setEvidenceOpen] = useState(false)

  useEffect(() => { if (can('audit.read')) void fetchAudit() }, [can, fetchAudit])
  const visible = useMemo(() => items.filter((entry) => {
    const matchesSearch = `${entry.request_id} ${entry.actor_name} ${entry.action} ${entry.entity_type}`.toLowerCase().includes(search.toLowerCase())
    return matchesSearch && (!entity || entry.entity_type === entity)
  }), [entity, items, search])
  const openDetail = (entry: AuditLog) => { setSelected(entry); setDetailOpen(true) }
  const evidence = extractPathEvidence(selected)

  if (!can('audit.read')) return <Box className="page-shell"><PageHeader eyebrow="RBAC / AUDIT" title="审计中心" summary="该页面只对管理员、安全复核员和审计员开放。" /><Alert severity="warning">当前职责无权读取不可变审计记录。</Alert></Box>

  return <Box className="page-shell">
    <PageHeader eyebrow="IMMUTABLE AUDIT / REQUEST TRACE" title="审计中心" summary="按 request ID、实体和操作者追溯所有写操作。快照在落库前脱敏，审计记录只追加、不修改。" actions={<Tooltip title="刷新"><IconButton onClick={() => fetchAudit()} aria-label="刷新审计记录"><RefreshRounded /></IconButton></Tooltip>} />
    <StatStrip items={[{ label: '审计记录', value: total }, { label: '轮换场景', value: items.filter((item) => item.entity_type === 'rollover_scenario').length }, { label: '推演 / 重放', value: items.filter((item) => item.action === 'simulate' || item.action === 'replay').length }, { label: '脱敏存储', value: '已启用', tone: 'is-good' }]} />
    <section className="audit-register">
      <Box className="audit-toolbar"><Box><TextField size="small" placeholder="搜索 request ID、操作者或动作" value={search} onChange={(event) => setSearch(event.target.value)} slotProps={{ input: { startAdornment: <InputAdornment position="start"><SearchRounded fontSize="small" /></InputAdornment> } }} /><FormControl size="small"><InputLabel>实体</InputLabel><Select label="实体" value={entity} onChange={(event) => setEntity(event.target.value)}><MenuItem value="">全部实体</MenuItem>{Object.entries(entityLabels).map(([value, label]) => <MenuItem key={value} value={value}>{label}</MenuItem>)}</Select></FormControl></Box><Box><FilterAltRounded fontSize="small" /><Typography>{visible.length} 条结果</Typography></Box></Box>
      {status === 'loading' && <LoadingState />}{status === 'error' && <ErrorState message={error} retry={() => fetchAudit()} />}{status === 'ready' && !visible.length && <EmptyState title="没有匹配的审计记录" detail="调整 request ID、操作者或实体筛选条件。" />}
      {!!visible.length && <TableContainer className="evidence-table audit-table"><Table size="small"><TableHead><TableRow><TableCell>时间 / Request ID</TableCell><TableCell>操作者</TableCell><TableCell>实体</TableCell><TableCell>动作</TableCell><TableCell>算法证据</TableCell><TableCell /></TableRow></TableHead><TableBody>{visible.map((entry) => <TableRow key={entry.id} hover onClick={() => openDetail(entry)}><TableCell><strong>{formatDateTime(entry.created_at)}</strong><span className="request-id">{entry.request_id}</span></TableCell><TableCell><strong>{entry.actor_name}</strong><span>{entry.actor_role}</span></TableCell><TableCell>{entityLabels[entry.entity_type] ?? entry.entity_type} #{entry.entity_id}</TableCell><TableCell><ToneBadge value={entry.action} /></TableCell><TableCell>{entry.algorithm_version ? <><strong>{entry.algorithm_version}</strong><span>{entry.duration_ms} ms</span></> : '—'}</TableCell><TableCell><KeyboardArrowRightRounded fontSize="small" /></TableCell></TableRow>)}</TableBody></Table></TableContainer>}
    </section>
    <FormDrawer open={detailOpen} onClose={() => setDetailOpen(false)} eyebrow="IMMUTABLE AUDIT ENTRY" title={selected ? `${entityLabels[selected.entity_type] ?? selected.entity_type} #${selected.entity_id}` : '审计记录'}>
      {selected && <Box className="audit-detail"><Box className="audit-verification"><ShieldRounded /><Box><strong>不可变追加记录</strong><span>记录 #{selected.id} · {formatDateTime(selected.created_at)}</span></Box></Box><dl className="detail-grid"><div className="wide"><dt>Request ID</dt><dd className="request-id">{selected.request_id}</dd></div><div><dt>操作者</dt><dd>{selected.actor_name}</dd></div><div><dt>角色</dt><dd>{selected.actor_role}</dd></div><div><dt>动作</dt><dd>{selected.action}</dd></div><div><dt>算法版本</dt><dd>{selected.algorithm_version || '—'}</dd></div><div className="wide"><dt>输入哈希</dt><dd className="request-id">{selected.input_hash || '—'}</dd></div><div className="wide"><dt>路径摘要</dt><dd>{selected.path_summary || '—'}</dd></div></dl><Box className="snapshot-block"><Typography variant="h3">变更前快照</Typography><pre>{JSON.stringify(JSON.parse(selected.before_snapshot || '{}'), null, 2)}</pre></Box><Box className="snapshot-block"><Typography variant="h3">变更后快照</Typography><pre>{JSON.stringify(JSON.parse(selected.after_snapshot || '{}'), null, 2)}</pre></Box>{selected.entity_type === 'rollover_scenario' && <Button variant="outlined" startIcon={<FactCheckRounded />} onClick={() => setEvidenceOpen(true)}>查看推演证据</Button>}</Box>}
    </FormDrawer>
    {selected && <ValidationEvidenceDrawer open={evidenceOpen} onClose={() => setEvidenceOpen(false)} title={`${entityLabels[selected.entity_type] ?? selected.entity_type} #${selected.entity_id}`} subtitle={`Request ${selected.request_id}`} pathEvidence={evidence} />}
  </Box>
}

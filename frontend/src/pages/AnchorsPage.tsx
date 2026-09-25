import { AddRounded, ArchiveOutlined, KeyboardArrowRightRounded, RefreshRounded, RestoreRounded, SearchRounded } from '@mui/icons-material'
import { Alert, Box, Button, IconButton, InputAdornment, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography } from '@mui/material'
import { FormEvent, useEffect, useMemo, useState } from 'react'
import { trustAnchorApi } from '../api/trust-anchor'
import { CertificateStateBadge } from '../components/common/CertificateStateBadge'
import { EmptyState, ErrorState, LoadingState } from '../components/common/DataState'
import { Fingerprint } from '../components/common/Fingerprint'
import { FormDrawer } from '../components/common/FormDrawer'
import { PageHeader } from '../components/common/PageHeader'
import { StatStrip } from '../components/common/StatStrip'
import { useAuth } from '../hooks/useAuth'
import { useCertificateChainStore } from '../stores/certificate-chain'
import { useTrustAnchorStore } from '../stores/trust-anchor'
import type { TrustAnchor } from '../types/trust-anchor'
import { formatDate, formatDateTime } from '../utils/date'
import { errorMessage } from '../api/client'

export function AnchorsPage() {
  const { items, total, status, error, fetchAnchors, lifecycle } = useTrustAnchorStore()
  const { items: chains, fetchChains } = useCertificateChainStore()
  const { can } = useAuth()
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<TrustAnchor | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [anchorCode, setAnchorCode] = useState('')
  const [pem, setPem] = useState('')
  const [mutationError, setMutationError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => { void fetchAnchors(); void fetchChains() }, [fetchAnchors, fetchChains])
  useEffect(() => { if (!selected && items.length) setSelected(items[0]) }, [items, selected])
  const visible = useMemo(() => items.filter((item) => `${item.anchor_code} ${item.subject_dn} ${item.fingerprint_sha256}`.toLowerCase().includes(search.toLowerCase())), [items, search])
  const relatedChains = chains.filter((chain) => chain.trust_anchor_id === selected?.id)

  const importAnchor = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setMutationError('')
    try { const created = await trustAnchorApi.import({ anchor_code: anchorCode, certificate_pem: pem }); await fetchAnchors(); setSelected(created); setDrawerOpen(false); setAnchorCode(''); setPem('') }
    catch (cause) { setMutationError(errorMessage(cause)) } finally { setBusy(false) }
  }
  const changeLifecycle = async (action: 'archive' | 'restore') => {
    if (!selected) return
    setBusy(true); setMutationError('')
    try { await lifecycle(selected.id, action); const refreshed = await trustAnchorApi.get(selected.id); setSelected(refreshed) }
    catch (cause) { setMutationError(errorMessage(cause)) } finally { setBusy(false) }
  }

  return <Box className="page-shell">
    <PageHeader eyebrow="TRUST INVENTORY / ROOTS" title="信任锚" summary="核对根证书身份、有效窗口与被引用范围。证书状态由服务端按日期和撤销记录计算。" actions={<><Tooltip title="刷新"><IconButton onClick={() => fetchAnchors()} aria-label="刷新信任锚"><RefreshRounded /></IconButton></Tooltip>{can('anchor.write') && <Button variant="contained" startIcon={<AddRounded />} onClick={() => setDrawerOpen(true)}>导入公钥证书</Button>}</>} />
    <StatStrip items={[{ label: '信任锚总数', value: total }, { label: '有效', value: items.filter((item) => item.certificate_state === 'valid').length, tone: 'is-good' }, { label: '需关注', value: items.filter((item) => item.certificate_state !== 'valid').length, tone: 'is-warning' }, { label: '链引用', value: chains.length }]} />
    {mutationError && <Alert severity="error" onClose={() => setMutationError('')}>{mutationError}</Alert>}
    <Box className="split-workbench">
      <section className="workbench-list">
        <Box className="list-toolbar"><TextField size="small" placeholder="搜索代码、Subject DN 或指纹" value={search} onChange={(event) => setSearch(event.target.value)} slotProps={{ input: { startAdornment: <InputAdornment position="start"><SearchRounded fontSize="small" /></InputAdornment> } }} /><Typography>{visible.length} 条记录</Typography></Box>
        {status === 'loading' && <LoadingState />}{status === 'error' && <ErrorState message={error} retry={() => fetchAnchors()} />}{status === 'ready' && !visible.length && <EmptyState title="没有匹配的信任锚" detail="调整检索条件，或导入一张仅含公钥材料的 CA 证书。" />}
        {!!visible.length && <TableContainer className="evidence-table"><Table size="small"><TableHead><TableRow><TableCell>锚点身份</TableCell><TableCell>状态</TableCell><TableCell>有效期至</TableCell><TableCell>链引用</TableCell><TableCell /></TableRow></TableHead><TableBody>{visible.map((anchor) => <TableRow key={anchor.id} hover selected={selected?.id === anchor.id} onClick={() => setSelected(anchor)}><TableCell><strong>{anchor.anchor_code}</strong><span>{anchor.subject_dn}</span></TableCell><TableCell><CertificateStateBadge state={anchor.certificate_state} /></TableCell><TableCell>{formatDate(anchor.not_after)}</TableCell><TableCell>{anchor.chain_count}</TableCell><TableCell><KeyboardArrowRightRounded fontSize="small" /></TableCell></TableRow>)}</TableBody></Table></TableContainer>}
      </section>
      <aside className="workbench-detail">
        {!selected ? <Box className="detail-placeholder"><Typography>选择一个信任锚查看证书身份与引用关系。</Typography></Box> : <>
          <Box className="detail-heading"><Box><Typography className="eyebrow">ANCHOR #{selected.id}</Typography><Typography variant="h2">{selected.anchor_code}</Typography></Box><CertificateStateBadge state={selected.certificate_state} /></Box>
          <dl className="detail-grid"><div><dt>Subject DN</dt><dd>{selected.subject_dn}</dd></div><div><dt>序列号</dt><dd>{selected.serial_number}</dd></div><div><dt>密钥算法</dt><dd>{selected.key_algorithm}</dd></div><div><dt>有效窗口</dt><dd>{formatDate(selected.not_before)} — {formatDate(selected.not_after)}</dd></div><div className="wide"><dt>SHA-256 指纹</dt><dd><Fingerprint value={selected.fingerprint_sha256} /></dd></div><div className="wide"><dt>材料边界</dt><dd>{selected.pem_redacted}</dd></div></dl>
          <Box className="detail-section-head"><Typography variant="h3">引用证书链</Typography><span>{relatedChains.length}</span></Box>
          <Box className="related-list">{relatedChains.map((chain) => <Box key={chain.id}><Box><strong>{chain.chain_code}</strong><span>{chain.leaf_subject}</span></Box><span>{chain.service_count} 个服务</span></Box>)}{!relatedChains.length && <Typography color="text.secondary">暂无证书链引用该锚点。</Typography>}</Box>
          <Box className="detail-footer"><Typography>更新于 {formatDateTime(selected.updated_at)}</Typography>{can('anchor.write') && <Button variant="outlined" disabled={busy} startIcon={selected.archived ? <RestoreRounded /> : <ArchiveOutlined />} onClick={() => changeLifecycle(selected.archived ? 'restore' : 'archive')}>{selected.archived ? '恢复锚点' : '归档锚点'}</Button>}</Box>
        </>}
      </aside>
    </Box>
    <FormDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} eyebrow="PUBLIC CERTIFICATE IMPORT" title="导入信任锚">
      <Alert severity="warning">只接受 CA 公钥证书。私钥、加密私钥和混合材料会被拒绝，PEM 全文不会进入日志。</Alert>
      <Box component="form" className="drawer-form" onSubmit={importAnchor}>{mutationError && <Alert severity="error">{mutationError}</Alert>}<TextField label="锚点代码" value={anchorCode} onChange={(event) => setAnchorCode(event.target.value)} required helperText="例如 ROOT-NEXT-2035" /><TextField label="公钥证书 PEM" value={pem} onChange={(event) => setPem(event.target.value)} required multiline minRows={12} placeholder="-----BEGIN CERTIFICATE-----" /><Button type="submit" variant="contained" disabled={busy}>{busy ? '正在校验…' : '离线校验并导入'}</Button></Box>
    </FormDrawer>
  </Box>
}

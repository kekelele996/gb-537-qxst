import { AddRounded, CompareArrowsRounded, KeyboardArrowRightRounded, RefreshRounded, SearchRounded, VerifiedRounded } from '@mui/icons-material'
import { Alert, Box, Button, FormControl, IconButton, InputAdornment, InputLabel, MenuItem, Select, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography } from '@mui/material'
import { FormEvent, useEffect, useMemo, useState } from 'react'
import { certificateChainApi } from '../api/certificate-chain'
import { errorMessage } from '../api/client'
import { CertificateStateBadge } from '../components/common/CertificateStateBadge'
import { EmptyState, ErrorState, LoadingState } from '../components/common/DataState'
import { Fingerprint } from '../components/common/Fingerprint'
import { FormDrawer } from '../components/common/FormDrawer'
import { PageHeader } from '../components/common/PageHeader'
import { StatStrip } from '../components/common/StatStrip'
import { ToneBadge } from '../components/common/ToneBadge'
import { ValidationEvidenceDrawer } from '../components/common/ValidationEvidenceDrawer'
import { useAuth } from '../hooks/useAuth'
import { useCertificateChainStore } from '../stores/certificate-chain'
import { useDependentServiceStore } from '../stores/dependent-service'
import { useTrustAnchorStore } from '../stores/trust-anchor'
import type { CertificateChain } from '../types/certificate-chain'
import { formatDate, formatDateTime } from '../utils/date'

export function ChainsPage() {
  const { items, total, status, error, fetchChains, transition } = useCertificateChainStore()
  const { items: anchors, fetchAnchors } = useTrustAnchorStore()
  const { items: services, fetchServices } = useDependentServiceStore()
  const { can } = useAuth()
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<CertificateChain | null>(null)
  const [evidenceOpen, setEvidenceOpen] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  const [chainCode, setChainCode] = useState('')
  const [anchorId, setAnchorId] = useState<number | ''>('')
  const [certificatePem, setCertificatePem] = useState('')
  const [feedback, setFeedback] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => { void fetchChains(); void fetchAnchors(); void fetchServices() }, [fetchAnchors, fetchChains, fetchServices])
  useEffect(() => { if (!selected && items.length) setSelected(items[0]) }, [items, selected])
  const visible = useMemo(() => items.filter((item) => `${item.chain_code} ${item.leaf_subject}`.toLowerCase().includes(search.toLowerCase())), [items, search])
  const relatedServices = services.filter((service) => service.chain_id === selected?.id)

  const importChain = async (event: FormEvent) => {
    event.preventDefault(); if (!anchorId) return; setBusy(true); setFeedback('')
    try { const created = await certificateChainApi.import({ chain_code: chainCode, trust_anchor_id: anchorId, certificates_pem: [certificatePem] }); await fetchChains(); setSelected(created); setImportOpen(false); setChainCode(''); setCertificatePem('') }
    catch (cause) { setFeedback(errorMessage(cause)) } finally { setBusy(false) }
  }
  const changeState = async (toState: 'deprecated' | 'revoked') => {
    if (!selected) return; setBusy(true); setFeedback('')
    try { await transition(selected.id, toState); const refreshed = await certificateChainApi.get(selected.id); setSelected(refreshed) }
    catch (cause) { setFeedback(errorMessage(cause)) } finally { setBusy(false) }
  }

  return <Box className="page-shell">
    <PageHeader eyebrow="CHAIN VALIDATION / PUBLIC MATERIAL" title="证书链" summary="检查叶证书到受信根的签名路径、有效窗口和依赖服务。所有验证使用 Go 标准库 crypto/x509。" actions={<><Tooltip title="刷新"><IconButton onClick={() => fetchChains()} aria-label="刷新证书链"><RefreshRounded /></IconButton></Tooltip>{can('chain.write') && <Button variant="contained" startIcon={<AddRounded />} onClick={() => setImportOpen(true)}>导入公钥链</Button>}</>} />
    <StatStrip items={[{ label: '证书链总数', value: total }, { label: '验证通过', value: items.filter((item) => item.validation_result.valid).length, tone: 'is-good' }, { label: '生产服务引用', value: services.filter((service) => service.environment === 'production').length }, { label: '需迁移链', value: items.filter((item) => item.chain_state === 'deprecated' || item.chain_state === 'revoked').length, tone: 'is-warning' }]} />
    {feedback && <Alert severity="error" onClose={() => setFeedback('')}>{feedback}</Alert>}
    <Box className="split-workbench">
      <section className="workbench-list">
        <Box className="list-toolbar"><TextField size="small" placeholder="搜索链代码或叶证书 Subject" value={search} onChange={(event) => setSearch(event.target.value)} slotProps={{ input: { startAdornment: <InputAdornment position="start"><SearchRounded fontSize="small" /></InputAdornment> } }} /><Typography>{visible.length} 条记录</Typography></Box>
        {status === 'loading' && <LoadingState />}{status === 'error' && <ErrorState message={error} retry={() => fetchChains()} />}{status === 'ready' && !visible.length && <EmptyState title="没有匹配的证书链" detail="调整检索条件，或导入一条仅含公钥材料的证书链。" />}
        {!!visible.length && <TableContainer className="evidence-table"><Table size="small"><TableHead><TableRow><TableCell>证书链</TableCell><TableCell>链状态</TableCell><TableCell>离线验证</TableCell><TableCell>服务</TableCell><TableCell /></TableRow></TableHead><TableBody>{visible.map((chain) => <TableRow key={chain.id} hover selected={selected?.id === chain.id} onClick={() => setSelected(chain)}><TableCell><strong>{chain.chain_code}</strong><span>{chain.leaf_subject}</span></TableCell><TableCell><ToneBadge value={chain.chain_state} /></TableCell><TableCell>{chain.validation_result.valid ? <span className="inline-verdict is-pass"><VerifiedRounded fontSize="small" /> 通过</span> : <span className="inline-verdict is-fail">失败</span>}</TableCell><TableCell>{chain.service_count}</TableCell><TableCell><KeyboardArrowRightRounded fontSize="small" /></TableCell></TableRow>)}</TableBody></Table></TableContainer>}
      </section>
      <aside className="workbench-detail">
        {selected ? <>
          <Box className="detail-heading"><Box><Typography className="eyebrow">CHAIN #{selected.id}</Typography><Typography variant="h2">{selected.chain_code}</Typography></Box><ToneBadge value={selected.chain_state} /></Box>
          <dl className="detail-grid"><div className="wide"><dt>叶证书 Subject</dt><dd>{selected.leaf_subject}</dd></div><div><dt>信任锚</dt><dd>{selected.trust_anchor?.anchor_code ?? `#${selected.trust_anchor_id}`}{selected.trust_anchor && <CertificateStateBadge state={selected.trust_anchor.certificate_state} />}</dd></div><div><dt>有效窗口</dt><dd>{formatDate(selected.valid_from)} — {formatDate(selected.valid_to)}</dd></div><div className="wide"><dt>链指纹</dt><dd><Fingerprint value={selected.chain_fingerprint} /></dd></div><div className="wide"><dt>来源校验和</dt><dd><Fingerprint value={selected.source_checksum} /></dd></div></dl>
          <Box className="evidence-callout"><Box><VerifiedRounded /><div><strong>{selected.validation_result.valid ? '签名路径验证通过' : '签名路径验证失败'}</strong><span>{selected.validation_result.message}</span></div></Box><Button size="small" startIcon={<CompareArrowsRounded />} onClick={() => setEvidenceOpen(true)}>查看逐证据</Button></Box>
          <Box className="detail-section-head"><Typography variant="h3">依赖服务</Typography><span>{relatedServices.length}</span></Box>
          <Box className="related-list">{relatedServices.map((service) => <Box key={service.id}><Box><strong>{service.service_code}</strong><span>{service.owner_team}</span></Box><ToneBadge value={service.criticality} /></Box>)}{!relatedServices.length && <Typography color="text.secondary">暂无服务引用该证书链。</Typography>}</Box>
          <Box className="detail-footer"><Typography>验证于 {formatDateTime(selected.validation_result.verified_at)}</Typography>{can('chain.write') && selected.chain_state !== 'revoked' && <Box className="button-pair">{selected.chain_state === 'validated' && <Button variant="outlined" disabled={busy} onClick={() => changeState('deprecated')}>标记废止</Button>}<Button color="error" variant="text" disabled={busy} onClick={() => changeState('revoked')}>撤销链记录</Button></Box>}</Box>
        </> : <Box className="detail-placeholder"><Typography>选择一条证书链查看验证证据。</Typography></Box>}
      </aside>
    </Box>
    {selected && <ValidationEvidenceDrawer open={evidenceOpen} onClose={() => setEvidenceOpen(false)} title={selected.chain_code} subtitle={selected.leaf_subject} chainEvidence={selected.validation_result} />}
    <FormDrawer open={importOpen} onClose={() => setImportOpen(false)} eyebrow="X.509 OFFLINE VALIDATION" title="导入证书链">
      <Alert severity="warning">仅粘贴公钥证书 PEM。系统会拒绝任何私钥材料，并在持久化前验证签名链与有效期。</Alert>
      <Box component="form" className="drawer-form" onSubmit={importChain}>{feedback && <Alert severity="error">{feedback}</Alert>}<TextField label="证书链代码" value={chainCode} onChange={(event) => setChainCode(event.target.value)} required /><FormControl required><InputLabel id="chain-anchor-label">信任锚</InputLabel><Select labelId="chain-anchor-label" value={anchorId} label="信任锚" onChange={(event) => setAnchorId(Number(event.target.value))}>{anchors.map((anchor) => <MenuItem key={anchor.id} value={anchor.id}>{anchor.anchor_code}</MenuItem>)}</Select></FormControl><TextField label="证书 PEM" value={certificatePem} onChange={(event) => setCertificatePem(event.target.value)} multiline minRows={12} required placeholder="-----BEGIN CERTIFICATE-----" /><Button type="submit" variant="contained" disabled={busy}>{busy ? '正在验证…' : '验证并导入证书链'}</Button></Box>
    </FormDrawer>
  </Box>
}

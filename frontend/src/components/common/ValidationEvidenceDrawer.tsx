import { CheckCircleRounded, CloseRounded, ErrorRounded, RouteRounded } from '@mui/icons-material'
import { Box, Drawer, IconButton, Typography } from '@mui/material'
import type { ChainValidationEvidence } from '../../types/certificate-chain'
import type { TimepointEvidence } from '../../types/rollover-scenario'
import { formatDateTime } from '../../utils/date'

interface Props {
  open: boolean
  onClose: () => void
  title: string
  subtitle?: string
  chainEvidence?: ChainValidationEvidence
  pathEvidence?: TimepointEvidence[]
}

export function ValidationEvidenceDrawer({ open, onClose, title, subtitle, chainEvidence, pathEvidence = [] }: Props) {
  return (
    <Drawer anchor="right" open={open} onClose={onClose} slotProps={{ paper: { className: 'evidence-drawer' } }}>
      <Box className="drawer-header">
        <Box><Typography className="eyebrow">VALIDATION EVIDENCE</Typography><Typography variant="h2">{title}</Typography>{subtitle && <Typography>{subtitle}</Typography>}</Box>
        <IconButton onClick={onClose} aria-label="关闭证据"><CloseRounded /></IconButton>
      </Box>
      <Box className="drawer-body">
        {chainEvidence && <section className="evidence-section">
          <Box className={`evidence-verdict ${chainEvidence.valid ? 'is-pass' : 'is-fail'}`}>
            {chainEvidence.valid ? <CheckCircleRounded /> : <ErrorRounded />}
            <Box><Typography component="strong">{chainEvidence.valid ? '离线链验证通过' : '离线链验证失败'}</Typography><Typography>{chainEvidence.message}</Typography></Box>
          </Box>
          <dl className="evidence-kv"><div><dt>验证时间</dt><dd>{formatDateTime(chainEvidence.verified_at)}</dd></div><div><dt>叶证书</dt><dd>{chainEvidence.leaf_subject}</dd></div><div><dt>根证书</dt><dd>{chainEvidence.root_subject}</dd></div></dl>
          <Typography variant="h3">信任路径</Typography>
          <ol className="trust-path">{chainEvidence.path_subjects?.map((subject, index) => <li key={`${subject}-${index}`}><RouteRounded fontSize="small" /><span>{subject}</span></li>)}</ol>
        </section>}
        {!!pathEvidence.length && <section className="evidence-section"><Typography variant="h3">时间点证据</Typography>{pathEvidence.map((point) => (
          <Box className="timepoint" key={point.at}>
            <Box className="timepoint-head"><Typography component="strong">{formatDateTime(point.at)}</Typography><span>激活锚点 {point.active_anchor_ids.join(', ') || '无'}</span></Box>
            {point.services.map((service) => <Box className="service-evidence" key={`${point.at}-${service.service_id}`}><span className={`reachability ${service.reachable ? 'is-pass' : 'is-fail'}`} /> <strong>{service.service_code}</strong><span>{service.reason}</span></Box>)}
          </Box>
        ))}</section>}
        {!chainEvidence && !pathEvidence.length && <Typography color="text.secondary">该记录没有可展示的逐路径证据。</Typography>}
      </Box>
    </Drawer>
  )
}

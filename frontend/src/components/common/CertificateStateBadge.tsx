import { Box } from '@mui/material'
import type { CertificateState } from '../../types/enums/certificate-state'

const labels: Record<CertificateState, string> = {
  valid: '有效',
  expiring: '临近到期',
  expired: '已过期',
  revoked: '已撤销',
}

export function CertificateStateBadge({ state }: { state: CertificateState }) {
  return <Box component="span" className={`state-badge certificate-${state}`}><span className="state-dot" />{labels[state]}</Box>
}


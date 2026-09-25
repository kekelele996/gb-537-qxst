import type { CertificateState } from './enums/certificate-state'

export interface TrustAnchor {
  id: number
  anchor_code: string
  subject_dn: string
  serial_number: string
  fingerprint_sha256: string
  not_before: string
  not_after: string
  key_algorithm: string
  certificate_state: CertificateState
  pem_redacted: string
  revoked_at?: string
  archived: boolean
  chain_count: number
  created_at: string
  updated_at: string
}

export interface ImportTrustAnchorInput {
  anchor_code: string
  certificate_pem: string
}


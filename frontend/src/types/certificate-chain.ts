import type { ChainState } from './enums/certificate-state'
import type { TrustAnchor } from './trust-anchor'

export interface CertificateReference {
  subject: string
  issuer: string
  serial_number: string
  fingerprint_sha256: string
  not_before: string
  not_after: string
  is_ca: boolean
}

export interface ChainValidationEvidence {
  valid: boolean
  verified_at: string
  leaf_subject: string
  root_subject: string
  path_subjects: string[]
  message: string
}

export interface CertificateChain {
  id: number
  chain_code: string
  trust_anchor_id: number
  trust_anchor?: TrustAnchor
  leaf_subject: string
  certificate_refs_json: CertificateReference[]
  chain_fingerprint: string
  valid_from: string
  valid_to: string
  validation_result: ChainValidationEvidence
  chain_state: ChainState
  source_checksum: string
  service_count: number
  created_at: string
  updated_at: string
}

export interface ImportCertificateChainInput {
  chain_code: string
  trust_anchor_id: number
  certificates_pem: string[]
}


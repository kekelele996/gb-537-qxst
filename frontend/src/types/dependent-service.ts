import type { CertificateChain } from './certificate-chain'

export type ServiceState = 'active' | 'inactive'
export type Criticality = 'low' | 'medium' | 'high' | 'critical'
export type Environment = 'development' | 'staging' | 'production' | 'disaster_recovery'
export type Protocol = 'tls' | 'mtls' | 'ldaps' | 'smtps' | 'kafka_tls'

export interface DependentService {
  id: number
  service_code: string
  name: string
  owner_team: string
  environment: Environment
  chain_id: number
  chain?: CertificateChain
  client_trust_refs_json: number[]
  protocol: Protocol
  criticality: Criticality
  dependency_edges_json: number[]
  service_state: ServiceState
  created_at: string
  updated_at: string
}

export interface DependentServiceInput {
  service_code?: string
  name: string
  owner_team: string
  environment: Environment
  chain_id: number
  client_trust_refs_json: number[]
  protocol: Protocol
  criticality: Criticality
  dependency_edges_json: number[]
}


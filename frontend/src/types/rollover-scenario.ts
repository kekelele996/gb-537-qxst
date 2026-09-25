import type { ScenarioState } from './enums/scenario-state'
import type { TrustAnchor } from './trust-anchor'

export interface AffectedService {
  id: number
  code: string
  state: string
  not_before?: string
  not_after?: string
  service_id?: number
  service_code?: string
  criticality?: string
  at?: string
  reason?: string
}

export interface BrokenPath {
  at: string
  service_codes: string[]
  reason: string
}

export interface ServiceEvidence {
  service_id: number
  service_code: string
  reachable: boolean
  selected_chain_id?: number
  selected_anchor_id?: number
  reason: string
}

export interface TimepointEvidence {
  at: string
  active_anchor_ids: number[]
  services: ServiceEvidence[]
}

export interface RolloverScenario {
  id: number
  name: string
  old_anchor_id: number
  new_anchor_id: number
  old_anchor?: TrustAnchor
  new_anchor?: TrustAnchor
  overlap_start: string
  overlap_end: string
  candidate_chain_ids: number[]
  algorithm_version: string
  input_hash: string
  simulation_time: string
  affected_services_json: AffectedService[]
  broken_paths_json: BrokenPath[]
  path_evidence_json: TimepointEvidence[]
  scenario_state: ScenarioState
  explanation: string
  created_by: number
  created_by_name: string
  verified_by?: number
  verified_by_name: string
  replay_verified: boolean
  duration_ms: number
  rollback_record: string
  created_at: string
  updated_at: string
}

export interface CreateRolloverScenarioInput {
  name: string
  old_anchor_id: number
  new_anchor_id: number
  overlap_start: string
  overlap_end: string
  candidate_chain_ids: number[]
  simulation_time: string
}


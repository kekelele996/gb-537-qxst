export type Role = 'admin' | 'pki_operator' | 'service_owner' | 'security_reviewer' | 'auditor'

export interface Actor {
  user_id: number
  username: string
  display_name: string
  team: string
  role: Role
}

export interface LoginResponse {
  access_token: string
  expires_at: string
  user: Actor
}


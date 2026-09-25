import type { Actor } from '../types/auth'
import { can } from './permissions'

function actor(role: Actor['role']): Actor {
  return { user_id: 1, username: role, display_name: role, team: 'PKI', role }
}

describe('RBAC visibility mapping', () => {
  it('keeps reviewer verification separate from scenario writes', () => {
    expect(can(actor('security_reviewer'), 'scenario.verify')).toBe(true)
    expect(can(actor('security_reviewer'), 'scenario.write')).toBe(false)
    expect(can(actor('pki_operator'), 'scenario.verify')).toBe(false)
    expect(can(actor('pki_operator'), 'scenario.run')).toBe(true)
  })

  it('limits service owners to dependency and scenario workflows', () => {
    expect(can(actor('service_owner'), 'dependency.write')).toBe(true)
    expect(can(actor('service_owner'), 'anchor.write')).toBe(false)
    expect(can(actor('service_owner'), 'audit.read')).toBe(false)
  })
})


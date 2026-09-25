import type { Actor, Role } from '../types/auth'

export type Permission = 'anchor.write' | 'chain.write' | 'dependency.write' | 'scenario.write' | 'scenario.run' | 'scenario.verify' | 'audit.read'

const permissions: Record<Role, Permission[]> = {
  admin: ['anchor.write', 'chain.write', 'dependency.write', 'scenario.write', 'scenario.run', 'scenario.verify', 'audit.read'],
  pki_operator: ['anchor.write', 'chain.write', 'dependency.write', 'scenario.write', 'scenario.run'],
  service_owner: ['dependency.write', 'scenario.write', 'scenario.run'],
  security_reviewer: ['scenario.verify', 'audit.read'],
  auditor: ['audit.read'],
}

export function can(user: Actor | null, permission: Permission) {
  return user ? permissions[user.role].includes(permission) : false
}

export const roleLabels: Record<Role, string> = {
  admin: '系统管理员',
  pki_operator: 'PKI 操作员',
  service_owner: '服务负责人',
  security_reviewer: '安全复核员',
  auditor: '审计员',
}


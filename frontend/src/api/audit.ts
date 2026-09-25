import type { AuditLog } from '../types/audit'
import type { Paginated } from '../types/common'
import { apiRequest } from './client'

export const auditApi = {
  list: (query = '') => apiRequest<Paginated<AuditLog>>(`/audit-logs${query}`),
}


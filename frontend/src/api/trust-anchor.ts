import type { Paginated } from '../types/common'
import type { ImportTrustAnchorInput, TrustAnchor } from '../types/trust-anchor'
import { apiRequest } from './client'

export const trustAnchorApi = {
  list: (query = '') => apiRequest<Paginated<TrustAnchor>>(`/trust-anchors${query}`),
  get: (id: number) => apiRequest<TrustAnchor>(`/trust-anchors/${id}`),
  import: (input: ImportTrustAnchorInput) => apiRequest<TrustAnchor>('/trust-anchors', { method: 'POST', body: JSON.stringify(input) }),
  lifecycle: (id: number, action: 'revoke' | 'archive' | 'restore') =>
    apiRequest<TrustAnchor>(`/trust-anchors/${id}/lifecycle`, { method: 'POST', body: JSON.stringify({ action }) }),
}


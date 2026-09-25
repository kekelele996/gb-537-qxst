import type { CertificateChain, ImportCertificateChainInput } from '../types/certificate-chain'
import type { Paginated } from '../types/common'
import type { ChainState } from '../types/enums/certificate-state'
import { apiRequest } from './client'

export const certificateChainApi = {
  list: (query = '') => apiRequest<Paginated<CertificateChain>>(`/certificate-chains${query}`),
  get: (id: number) => apiRequest<CertificateChain>(`/certificate-chains/${id}`),
  import: (input: ImportCertificateChainInput) => apiRequest<CertificateChain>('/certificate-chains', { method: 'POST', body: JSON.stringify(input) }),
  transition: (id: number, toState: ChainState) =>
    apiRequest<CertificateChain>(`/certificate-chains/${id}/transition`, { method: 'POST', body: JSON.stringify({ to_state: toState }) }),
  compare: (id: number, otherId: number) => apiRequest<Record<string, unknown>>(`/certificate-chains/${id}/compare/${otherId}`),
}


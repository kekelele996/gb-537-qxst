import type { Paginated } from '../types/common'
import type { DependentService, DependentServiceInput } from '../types/dependent-service'
import { apiRequest } from './client'

export const dependentServiceApi = {
  list: (query = '') => apiRequest<Paginated<DependentService>>(`/dependent-services${query}`),
  get: (id: number) => apiRequest<DependentService>(`/dependent-services/${id}`),
  create: (input: DependentServiceInput & { service_code: string }) => apiRequest<DependentService>('/dependent-services', { method: 'POST', body: JSON.stringify(input) }),
  update: (id: number, input: DependentServiceInput) => apiRequest<DependentService>(`/dependent-services/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
  deactivate: (id: number) => apiRequest<DependentService>(`/dependent-services/${id}/deactivate`, { method: 'POST' }),
}


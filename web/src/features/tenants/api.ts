import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

export type Tenant = {
  id: number
  slug: string
  name: string
  status: string
  created_by: number
  created_at: number
  updated_at: number
}

export type CreateTenantPayload = {
  slug: string
  name: string
}

export async function getTenants(): Promise<Tenant[]> {
  const response = await api.get('/api/tenants')
  return requireServerSuccess(response.data).data as Tenant[]
}

export async function createTenant(
  payload: CreateTenantPayload
): Promise<Tenant> {
  const response = await api.post('/api/tenants', payload)
  return requireServerSuccess(response.data).data as Tenant
}

import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'
import { searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'

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
  owner_user_id: number
}

export async function getTenants(): Promise<Tenant[]> {
  const response = await api.get('/api/tenants')
  return requireServerSuccess(response.data).data as Tenant[]
}

export async function promoteTenant(
  payload: CreateTenantPayload
): Promise<Tenant> {
  const response = await api.post('/api/tenants', payload)
  return requireServerSuccess(response.data).data as Tenant
}

export async function getTenantOwnerCandidates(): Promise<User[]> {
  const response = await searchUsers({ status: '1', page_size: 100 })
  if (!response.success || !response.data) return []
  return response.data.items
}

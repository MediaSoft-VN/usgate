import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

export type Tenant = {
  id: number
  slug: string
  partner_code: string
  name: string
  status: string
  created_by: number
  created_at: number
  updated_at: number
}

export type CreateTenantPayload = {
  slug: string
  partner_code?: string
  name: string
  owner_user_id: number
}

export type TenantMember = {
  id: number
  tenant_id: number
  user_id: number
  role_key: string
  status: string
}

export type TenantDomain = {
  id: number
  tenant_id: number
  hostname: string
  status: string
  verification_method: string
}

export type TenantDNSRecord = {
  cname_target: string
  a_record_ip: string
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

export async function selectTenant(tenantId: number): Promise<Tenant> {
  const response = await api.post('/api/tenant-context/select', {
    tenant_id: tenantId,
  })
  return requireServerSuccess(response.data).data as Tenant
}

export async function getTenantMembers(
  tenantId: number
): Promise<TenantMember[]> {
  const response = await api.get(`/api/tenants/${tenantId}/members`)
  return requireServerSuccess(response.data).data as TenantMember[]
}

export async function getTenantDomains(
  tenantId: number
): Promise<TenantDomain[]> {
  const response = await api.get(`/api/tenants/${tenantId}/domains`)
  return requireServerSuccess(response.data).data as TenantDomain[]
}

export async function getTenantDNSRecord(
  tenantId: number
): Promise<TenantDNSRecord> {
  const response = await api.get(`/api/tenants/${tenantId}/domains/dns-record`)
  return requireServerSuccess(response.data).data as TenantDNSRecord
}

export async function createTenantDomain(
  tenantId: number,
  hostname: string
): Promise<{ domain: TenantDomain; verification_token: string }> {
  const response = await api.post(`/api/tenants/${tenantId}/domains`, {
    hostname,
  })
  return requireServerSuccess(response.data).data as {
    domain: TenantDomain
    verification_token: string
  }
}

export async function verifyTenantDomain(
  tenantId: number,
  domainId: number,
  token: string
): Promise<void> {
  const response = await api.post(
    `/api/tenants/${tenantId}/domains/${domainId}/verify`,
    { token }
  )
  requireServerSuccess(response.data)
}

export async function updateTenantDomain(
  tenantId: number,
  domainId: number,
  hostname: string
): Promise<{ domain: TenantDomain; verification_token: string }> {
  const response = await api.patch(
    `/api/tenants/${tenantId}/domains/${domainId}`,
    { hostname }
  )
  return requireServerSuccess(response.data).data as {
    domain: TenantDomain
    verification_token: string
  }
}

export async function disableTenantDomain(
  tenantId: number,
  domainId: number
): Promise<void> {
  const response = await api.delete(
    `/api/tenants/${tenantId}/domains/${domainId}`
  )
  requireServerSuccess(response.data)
}

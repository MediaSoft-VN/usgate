import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import {
  createTenantDomain,
  getTenantDNSRecord,
  promoteTenant,
  selectTenant,
} from '../api'

describe('tenant management API', () => {
  afterEach(() => vi.restoreAllMocks())

  it('selects a tenant through the tenant context endpoint', async () => {
    const response = { id: 7, slug: 'acme', name: 'Acme', status: 'active' }
    const post = vi.spyOn(api, 'post').mockResolvedValue({ data: { success: true, data: response } })

    await expect(selectTenant(7)).resolves.toEqual(response)
    expect(post).toHaveBeenCalledWith('/api/tenant-context/select', { tenant_id: 7 })
  })

  it('promotes an enabled user through the admin tenant endpoint', async () => {
    const response = { id: 8, slug: 'acme', name: 'Acme', status: 'active' }
    const post = vi.spyOn(api, 'post').mockResolvedValue({ data: { success: true, data: response } })

    await expect(promoteTenant({ slug: 'acme', name: 'Acme', owner_user_id: 4 })).resolves.toEqual(response)
    expect(post).toHaveBeenCalledWith('/api/tenants', {
      slug: 'acme',
      name: 'Acme',
      owner_user_id: 4,
    })
  })

  it('surfaces a failed partner upgrade response', async () => {
    vi.spyOn(api, 'post').mockResolvedValue({
      data: { success: false, message: 'Partner already exists' },
    })

    await expect(
      promoteTenant({ slug: 'acme', name: 'Acme', owner_user_id: 4 })
    ).rejects.toThrow('Partner already exists')
  })

  it('uses the tenant-scoped domain endpoint', async () => {
    const post = vi.spyOn(api, 'post').mockResolvedValue({
      data: { success: true, data: { id: 3, tenant_id: 7, user_id: 4, role_key: 'member' } },
    })
    const remove = vi.spyOn(api, 'delete').mockResolvedValue({ data: { success: true } })

    await createTenantDomain(7, 'app.example.com')

    expect(post).toHaveBeenCalledWith('/api/tenants/7/domains', {
      hostname: 'app.example.com',
    })
    expect(remove).not.toHaveBeenCalled()
  })

  it('reads the tenant-scoped DNS record without exposing admin settings', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { type: 'CNAME', target: 'proxy.example.com' } },
    })

    await expect(getTenantDNSRecord(7)).resolves.toEqual({
      type: 'CNAME',
      target: 'proxy.example.com',
    })
    expect(get).toHaveBeenCalledWith('/api/tenants/7/domains/dns-record')
  })
})

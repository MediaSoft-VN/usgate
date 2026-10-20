import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import {
  createTenantDomain,
  disableTenantDomain,
  getTenantDNSRecord,
  promoteTenant,
  selectTenant,
  updateTenantDomain,
} from '../api'

describe('tenant management API', () => {
  afterEach(() => vi.restoreAllMocks())

  it('selects a tenant through the tenant context endpoint', async () => {
    const response = { id: 7, slug: 'acme', name: 'Acme', status: 'active' }
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true, data: response } })

    await expect(selectTenant(7)).resolves.toEqual(response)
    expect(post).toHaveBeenCalledWith('/api/tenant-context/select', {
      tenant_id: 7,
    })
  })

  it('promotes an enabled user through the admin tenant endpoint', async () => {
    const response = { id: 8, slug: 'acme', name: 'Acme', status: 'active' }
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true, data: response } })

    await expect(
      promoteTenant({ slug: 'acme', name: 'Acme', owner_user_id: 4 })
    ).resolves.toEqual(response)
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
      data: {
        success: true,
        data: { id: 3, tenant_id: 7, user_id: 4, role_key: 'member' },
      },
    })
    const remove = vi
      .spyOn(api, 'delete')
      .mockResolvedValue({ data: { success: true } })

    await createTenantDomain(7, 'app.example.com')

    expect(post).toHaveBeenCalledWith('/api/tenants/7/domains', {
      hostname: 'app.example.com',
    })
    expect(remove).not.toHaveBeenCalled()
  })

  it('reads the tenant-scoped DNS record without exposing admin settings', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          cname_target: 'proxy.example.com',
          a_record_ip: '203.0.113.10',
        },
      },
    })

    await expect(getTenantDNSRecord(7)).resolves.toEqual({
      cname_target: 'proxy.example.com',
      a_record_ip: '203.0.113.10',
    })
    expect(get).toHaveBeenCalledWith('/api/tenants/7/domains/dns-record')
  })

  it('updates a domain through the tenant-scoped endpoint', async () => {
    const patch = vi.spyOn(api, 'patch').mockResolvedValue({
      data: {
        success: true,
        data: { domain: { id: 3 }, verification_token: 'token' },
      },
    })

    await expect(updateTenantDomain(7, 3, 'new.example.com')).resolves.toEqual({
      domain: { id: 3 },
      verification_token: 'token',
    })
    expect(patch).toHaveBeenCalledWith('/api/tenants/7/domains/3', {
      hostname: 'new.example.com',
    })
  })

  it('deletes a domain through the tenant-scoped endpoint', async () => {
    const remove = vi.spyOn(api, 'delete').mockResolvedValue({
      data: { success: true },
    })

    await expect(disableTenantDomain(7, 3)).resolves.toBeUndefined()
    expect(remove).toHaveBeenCalledWith('/api/tenants/7/domains/3')
  })
})

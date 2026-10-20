import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import {
  createTenantDomain,
  promoteTenant,
  removeTenantMember,
  selectTenant,
  upsertTenantMember,
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

  it('uses the M1 member and domain endpoints with tenant-scoped paths', async () => {
    const post = vi.spyOn(api, 'post').mockResolvedValue({
      data: { success: true, data: { id: 3, tenant_id: 7, user_id: 4, role_key: 'member' } },
    })
    const remove = vi.spyOn(api, 'delete').mockResolvedValue({ data: { success: true } })

    await upsertTenantMember(7, { user_id: 4, role_key: 'member' })
    await createTenantDomain(7, 'app.example.com')
    await removeTenantMember(7, 4)

    expect(post).toHaveBeenNthCalledWith(1, '/api/tenants/7/members/invitations', {
      user_id: 4,
      role_key: 'member',
    })
    expect(post).toHaveBeenNthCalledWith(2, '/api/tenants/7/domains', {
      hostname: 'app.example.com',
    })
    expect(remove).toHaveBeenCalledWith('/api/tenants/7/members/4')
  })
})

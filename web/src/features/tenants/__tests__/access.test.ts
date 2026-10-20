import { describe, expect, it } from 'vitest'

import { canViewPartnerWorkspace, canViewTenantOverview } from '../access'

describe('tenant route access', () => {
  it('shows the partner workspace only to partner users', () => {
    expect(canViewPartnerWorkspace({ id: 1, username: 'partner', role: 1, is_partner: true })).toBe(true)
    expect(canViewPartnerWorkspace({ id: 2, username: 'user', role: 1 })).toBe(false)
  })

  it('keeps the tenant overview for platform admins and not partners', () => {
    expect(canViewTenantOverview({ id: 1, username: 'admin', role: 10 })).toBe(true)
    expect(canViewTenantOverview({ id: 2, username: 'partner', role: 10, is_partner: true })).toBe(false)
  })
})

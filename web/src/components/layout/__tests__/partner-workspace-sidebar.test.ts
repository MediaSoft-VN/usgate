import { describe, expect, it } from 'vitest'
import type { TFunction } from 'i18next'

import { PARTNER_WORKSPACE_VIEW } from '../config/partner-workspace.config'
import { resolveSidebarView } from '../lib/sidebar-view-registry'

describe('partner workspace sidebar', () => {
  it('resolves every partner workspace route to the contextual sidebar', () => {
    expect(resolveSidebarView('/partner-workspace/overview')?.id).toBe('partner-workspace')
    expect(resolveSidebarView('/partner-workspace/transactions')?.id).toBe('partner-workspace')
    expect(resolveSidebarView('/dashboard/overview')).toBeNull()
  })

  it('exposes separate links for each workspace area', () => {
    const groups = PARTNER_WORKSPACE_VIEW.getNavGroups(((key) => key) as TFunction)
    const urls = groups.flatMap((group) =>
      group.items.flatMap((item) => ('url' in item && item.url ? [item.url] : []))
    )

    expect(urls).toEqual([
      '/partner-workspace/overview',
      '/partner-workspace/users',
      '/partner-workspace/domains',
      '/partner-workspace/wallet',
      '/partner-workspace/transactions',
    ])
  })
})

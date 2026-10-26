import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'

import { canViewPartnerWorkspace } from '@/features/tenants/access'
import { getTenantContext } from '@/features/tenants/api'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/partner-workspace')({
  beforeLoad: async ({ location }) => {
    const { auth } = useAuthStore.getState()
    if (!canViewPartnerWorkspace(auth.user)) throw redirect({ to: '/403' })
    try {
      await getTenantContext()
    } catch {
      throw redirect({ to: '/403' })
    }
    // Partner owners/admins may manage their tenant from both the platform
    // domain and the tenant domain. Regular tenant members are not partners,
    // so canViewPartnerWorkspace still blocks them.
    if (location.pathname === '/partner-workspace') {
      throw redirect({ to: '/partner-workspace/overview' })
    }
  },
  component: Outlet,
})

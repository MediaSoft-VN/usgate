import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'

import { canViewPartnerWorkspace } from '@/features/tenants/access'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/partner-workspace')({
  beforeLoad: ({ location }) => {
    const { auth } = useAuthStore.getState()
    if (!canViewPartnerWorkspace(auth.user)) throw redirect({ to: '/403' })
    if (location.pathname === '/partner-workspace') {
      throw redirect({ to: '/partner-workspace/overview' })
    }
  },
  component: Outlet,
})

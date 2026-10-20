import { createFileRoute, redirect } from '@tanstack/react-router'

import { Tenants } from '@/features/tenants'
import { canViewTenantOverview } from '@/features/tenants/access'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/tenants/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!canViewTenantOverview(auth.user)) {
      throw redirect({ to: '/403' })
    }
  },
  component: Tenants,
})

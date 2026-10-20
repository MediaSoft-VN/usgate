import { createFileRoute, redirect } from '@tanstack/react-router'

import { PartnerWorkspace } from '@/features/tenants/partner-workspace'
import { canViewPartnerWorkspace } from '@/features/tenants/access'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/partner-workspace')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!canViewPartnerWorkspace(auth.user)) throw redirect({ to: '/403' })
  },
  component: PartnerWorkspace,
})

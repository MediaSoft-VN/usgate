import { createFileRoute } from '@tanstack/react-router'

import { PartnerWorkspace } from '@/features/tenants/partner-workspace'

export const Route = createFileRoute('/_authenticated/partner-workspace/users')({
  component: () => <PartnerWorkspace section='users' />,
})

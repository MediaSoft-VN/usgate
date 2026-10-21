import { createFileRoute } from '@tanstack/react-router'

import { PartnerWorkspace } from '@/features/tenants/partner-workspace'

export const Route = createFileRoute('/_authenticated/partner-workspace/wallet')({
  component: () => <PartnerWorkspace section='wallet' />,
})

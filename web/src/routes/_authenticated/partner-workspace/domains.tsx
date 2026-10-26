import { createFileRoute } from '@tanstack/react-router'

import { PartnerWorkspace } from '@/features/tenants/partner-workspace'

export const Route = createFileRoute('/_authenticated/partner-workspace/domains')({
  component: () => <PartnerWorkspace section='domains' />,
})

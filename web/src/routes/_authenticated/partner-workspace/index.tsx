import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/partner-workspace/')({
  beforeLoad: () => {
    throw redirect({ to: '/partner-workspace/overview' })
  },
})

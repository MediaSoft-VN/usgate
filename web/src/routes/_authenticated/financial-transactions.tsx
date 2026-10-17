import { createFileRoute, redirect } from '@tanstack/react-router'

import { FinancialTransactions } from '@/features/financial-transactions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/financial-transactions')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({ to: '/403' })
    }
  },
  component: FinancialTransactions,
})

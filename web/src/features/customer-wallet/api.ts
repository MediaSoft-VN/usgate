import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

export type FinancialWallet = {
  id: number
  owner_type: string
  owner_id: number
  currency: string
  available_minor: number
  reserved_minor: number
  status: string
  version: number
  created_at: number
  updated_at: number
}

export async function getFinancialWallets(ownerType: string) {
  const response = await api.get<{
    success: boolean
    data: { items: FinancialWallet[] }
  }>('/api/financial/wallets', { params: { owner_type: ownerType } })
  return requireServerSuccess(response.data).data.items
}

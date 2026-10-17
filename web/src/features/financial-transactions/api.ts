import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

export type FinancialTransaction = {
  id: number
  transaction_no: string
  wallet_account_id: number
  idempotency_key: string
  type: string
  status: string
  amount_minor: number
  currency: string
  provider: string
  external_reference: string
  created_at: number
  completed_at: number
}

export async function getFinancialTransactions(page: number, pageSize: number) {
  const response = await api.get<{
    success: boolean
    data: {
      items: FinancialTransaction[]
      total: number
      page: number
      page_size: number
    }
  }>('/api/financial/transactions', {
    params: { page, page_size: pageSize },
  })
  return requireServerSuccess(response.data).data
}

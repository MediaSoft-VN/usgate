import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

export type ProviderFunding = {
  id: number
  channel_id: number
  reference: string
  amount_minor: number
  currency: string
  source: 'manual' | 'sync'
  provider_reference: string
  note: string
  created_at: number
  created_by: string
}

export type ProviderFundingSummary = {
  currency: string
  amount_minor: number
}

export type ProviderFundingPage = {
  items: ProviderFunding[]
  total: number
  page: number
  page_size: number
  summary: ProviderFundingSummary[]
}

export async function getProviderFundings(
  page: number,
  pageSize: number,
) {
  const response = await api.get<{ success: boolean; data: ProviderFundingPage }>(
    '/api/financial/provider-funding',
    {
      params: {
        page,
        page_size: pageSize,
      },
    }
  )
  return requireServerSuccess(response.data).data
}

export async function createProviderFunding(input: {
  channel_id: number
  reference: string
  amount_minor: number
  currency: string
  source: 'manual'
}) {
  const response = await api.post('/api/financial/provider-funding', input)
  return requireServerSuccess(response.data).data
}

export type ProviderFundingSyncStatus = {
  running: boolean
  started_at: number
  completed_at: number
}

export async function getProviderFundingSyncStatus() {
  const response = await api.get<{
    success: boolean
    data: ProviderFundingSyncStatus
  }>('/api/financial/provider-funding/sync')
  return requireServerSuccess(response.data).data
}

export async function startProviderFundingSync() {
  const response = await api.post<{
    success: boolean
    data: Pick<ProviderFundingSyncStatus, 'running' | 'started_at'>
  }>('/api/financial/provider-funding/sync')
  return requireServerSuccess(response.data).data
}

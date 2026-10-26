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

export type ProviderFundingWalletSummary = {
  channel_id: number
  currency: string
  amount_minor: number
  allocated_minor: number
  available_minor: number
}

export type ProviderFundingChannelSummary = {
  channel_id: number
  channel_name: string
  currency: string
  amount_minor: number
}

export type ProviderFundingAllocation = {
  id: number
  user_id: number
  user_name: string
  user_display_name: string
  counterparty_name: string
  reference: string
  amount_minor: number
  currency: string
  quota: number
  exchange_rate: string
  created_at: number
  created_by: string
}

export type ProviderFundingPage = {
  items: ProviderFunding[]
  total: number
  page: number
  page_size: number
  wallets: ProviderFundingWalletSummary[]
  provider_wallets: ProviderFundingChannelSummary[]
  allocations: ProviderFundingAllocation[]
  cash_flows: ProviderCashFlow[]
  cash_flow_total: number
  report: ProviderCashFlowSummary[]
}

export type ProviderCashFlow = {
  id: number
  channel_id: number
  user_id: number
  direction: 'incoming' | 'outgoing'
  type: 'funding' | 'allocation'
  reference: string
  amount_minor: number
  currency: string
  source: string
  quota: number
  exchange_rate: string
  counterparty_name: string
  created_at: number
  created_by: string
}

export type ProviderCashFlowSummary = {
  currency: string
  incoming_minor: number
  outgoing_minor: number
  net_minor: number
  count: number
}

export async function getProviderFundings(
  page: number,
  pageSize: number,
  filters: {
    from_date?: string
    to_date?: string
    direction?: 'incoming' | 'outgoing'
  } = {}
) {
  const response = await api.get<{
    success: boolean
    data: ProviderFundingPage
  }>('/api/financial/provider-funding', {
    params: {
      page,
      page_size: pageSize,
      ...filters,
    },
  })
  return requireServerSuccess(response.data).data
}

export async function exportProviderFundings(
  filters: {
    from_date?: string
    to_date?: string
    direction?: 'incoming' | 'outgoing'
  } = {}
) {
  const response = await api.get<Blob>(
    '/api/financial/provider-funding/export',
    {
      params: filters,
      responseType: 'blob',
    }
  )
  return response.data
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

export async function allocateProviderFunding(input: {
  user_id: number
  reference: string
  amount_minor: number
  currency: string
}) {
  const response = await api.post(
    '/api/financial/provider-funding/allocate',
    input
  )
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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { afterEach, expect, it, vi } from 'vitest'

import { ProviderFunding } from '../index'

const providerFundingMocks = vi.hoisted(() => ({
  exportProviderFundings: vi.fn(),
  getProviderFundings: vi.fn(),
}))
providerFundingMocks.getProviderFundings.mockResolvedValue({
  items: [],
  total: 0,
  page: 1,
  page_size: 20,
  wallets: [],
  provider_wallets: [],
  allocations: [],
  cash_flows: [
    {
      id: 1,
      channel_id: 1,
      user_id: 0,
      direction: 'incoming',
      type: 'funding',
      reference: 'funding-1',
      amount_minor: 12500,
      currency: 'USD',
      source: 'manual',
      quota: 0,
      exchange_rate: '',
      counterparty_name: 'Provider One',
      created_at: 1,
      created_by: 'admin',
    },
  ],
  cash_flow_total: 1,
  report: [
    {
      currency: 'USD',
      incoming_minor: 12500,
      outgoing_minor: 2500,
      net_minor: 10000,
      count: 2,
    },
  ],
})
providerFundingMocks.exportProviderFundings.mockResolvedValue(new Blob(['csv']))

vi.mock('../api', () => ({
  exportProviderFundings: providerFundingMocks.exportProviderFundings,
  getProviderFundings: providerFundingMocks.getProviderFundings,
}))

vi.mock('../../channels/api', () => ({
  getChannels: vi.fn().mockResolvedValue({
    success: true,
    data: { items: [{ id: 1, name: 'Provider One', type: 'openai' }] },
  }),
}))

const i18n = createInstance()
await i18n.init({
  lng: 'en',
  resources: { en: { translation: {} } },
  initAsync: false,
})

afterEach(() => {
  cleanup()
  providerFundingMocks.getProviderFundings.mockClear()
  providerFundingMocks.exportProviderFundings.mockClear()
})

it('shows the filtered financial report and exports the same filter set', async () => {
  const user = userEvent.setup()
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ProviderFunding />
      </I18nextProvider>
    </QueryClientProvider>
  )

  expect(await screen.findByText('Financial report')).toBeInTheDocument()
  expect(screen.getByText('100')).toBeInTheDocument()

  await user.click(screen.getByRole('button', { name: 'Export' }))

  await waitFor(() =>
    expect(providerFundingMocks.exportProviderFundings).toHaveBeenCalledWith({})
  )
})

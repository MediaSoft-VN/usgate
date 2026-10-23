import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { afterEach, expect, it, vi } from 'vitest'

import { ProviderWallet } from '../index'

const updateProviderCredit = vi.hoisted(() => vi.fn())

vi.mock('../../provider-funding/api', () => ({
  createProviderFunding: vi.fn(),
  getProviderFundings: vi.fn().mockResolvedValue({
    wallets: [
      {
        currency: 'USD',
        amount_minor: 10000,
        allocated_minor: 2500,
        available_minor: 7500,
      },
    ],
    provider_wallets: [
      {
        channel_id: 1,
        channel_name: 'Provider One',
        currency: 'USD',
        amount_minor: 10000,
      },
    ],
  }),
  allocateProviderFunding: vi.fn(),
  getProviderFundingSyncStatus: vi.fn().mockResolvedValue({ running: false }),
  startProviderFundingSync: vi.fn(),
}))

vi.mock('../../users/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../users/api')>()
  return {
    ...actual,
    getUsers: vi.fn().mockResolvedValue({ data: { items: [] } }),
  }
})

vi.mock('../../channels/api', () => ({
  getChannels: vi.fn().mockResolvedValue({
    success: true,
    data: { items: [{ id: 1, name: 'Provider One', type: 'openai' }] },
  }),
  getProviderCredit: vi.fn().mockResolvedValue({
    channel_id: 1,
    balance: 100,
    threshold: 25,
    below_threshold: false,
    auto_disabled: false,
    status: 1,
    checked_at: 1,
  }),
  updateProviderCredit,
}))

const i18n = createInstance()
await i18n.init({
  lng: 'en',
  resources: { en: { translation: {} } },
  initAsync: false,
})

afterEach(() => {
  cleanup()
  updateProviderCredit.mockClear()
})

it('accepts whole-dollar credit allocations without a step mismatch', async () => {
  const user = userEvent.setup()
  const queryClient = new QueryClient()

  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ProviderWallet />
      </I18nextProvider>
    </QueryClientProvider>
  )

  await user.click(
    (await screen.findAllByRole('button', { name: 'Credit allocation' }))[0]
  )
  const amount = screen.getByRole('spinbutton')

  await user.type(amount, '120')

  expect(amount).toHaveAttribute('min', '0.01')
  expect(amount).toHaveAttribute('step', '0.01')
  expect(amount).toBeValid()
})

it('shows provider funding without using the channel upstream balance', async () => {
  const queryClient = new QueryClient()

  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ProviderWallet />
      </I18nextProvider>
    </QueryClientProvider>
  )

  expect(
    screen.getByRole('heading', { name: 'Provider total wallet' })
  ).toBeInTheDocument()
  expect(await screen.findByText('Provider One')).toBeInTheDocument()
  expect(screen.getAllByText('Available')).toHaveLength(1)
  expect(screen.getByText('Total funded')).toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: 'Sync credit' })).toHaveLength(1)
  expect(screen.getByText(/100 USD/)).toBeInTheDocument()
  expect(screen.getByText(/75 USD/)).toBeInTheDocument()
  expect(screen.queryByText(/42\.5 USD/)).not.toBeInTheDocument()
})

it('allows an administrator to update the provider credit threshold', async () => {
  const user = userEvent.setup()
  const queryClient = new QueryClient()

  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ProviderWallet />
      </I18nextProvider>
    </QueryClientProvider>
  )

  const threshold = await screen.findByLabelText('Provider credit threshold')
  await user.clear(threshold)
  await user.type(threshold, '30')
  await user.click(screen.getByRole('button', { name: 'Save' }))

  expect(updateProviderCredit).toHaveBeenCalledWith(1, 30)
})

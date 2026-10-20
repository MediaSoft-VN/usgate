import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { afterEach, expect, it, vi } from 'vitest'

import { ProviderWallet } from '../index'

vi.mock('../../provider-funding/api', () => ({
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
}))

vi.mock('../../users/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../users/api')>()
  return {
    ...actual,
    getUsers: vi.fn().mockResolvedValue({ data: { items: [] } }),
  }
})

const i18n = createInstance()
await i18n.init({
  lng: 'en',
  resources: { en: { translation: {} } },
  initAsync: false,
})

afterEach(() => {
  cleanup()
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

  expect(await screen.findByText('Provider One')).toBeInTheDocument()
  expect(screen.getAllByText('Available')).toHaveLength(1)
  expect(screen.getAllByText('Total funded')).toHaveLength(2)
  expect(screen.queryByText('Current Balance')).not.toBeInTheDocument()
  expect(screen.getAllByText(/100 USD/)).toHaveLength(2)
  expect(screen.queryByText(/42\.5 USD/)).not.toBeInTheDocument()
})

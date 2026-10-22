import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { UpstreamCostSync } from '../upstream-cost-sync'

const channels = [
  {
    id: 1,
    name: 'OpenAI channel',
    type: 1,
    type_name: 'OpenAI',
    supported: true,
    version: 7,
    costs: [],
  },
  {
    id: 2,
    name: 'Local Ollama',
    type: 4,
    type_name: 'Ollama',
    supported: false,
    unsupported_reason: 'manual',
    version: 3,
    costs: [],
  },
  {
    id: 3,
    name: 'Second OpenAI channel',
    type: 1,
    type_name: 'OpenAI',
    supported: true,
    version: 8,
    costs: [],
  },
]

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <UpstreamCostSync />
    </QueryClientProvider>
  )
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('upstream cost synchronization', () => {
  it('renders Ollama as manual and keeps it out of the supported select-all set', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: channels },
    })
    renderPage()
    expect(await screen.findByText('Ollama pricing is manual')).toBeVisible()
    expect(screen.getByRole('checkbox', { name: 'Select Local Ollama' })).toHaveAttribute('aria-disabled', 'true')
    expect(screen.getByRole('checkbox', { name: 'Select all supported channels' })).not.toBeChecked()
  })

  it('previews selected channels and applies only a fresh successful result', async () => {
    const post = vi.spyOn(api, 'post')
    post.mockResolvedValueOnce({
      data: {
        success: true,
        data: {
          results: [
            {
              channel_id: 1,
              source: 'models.dev',
              supported: true,
              version: 7,
              costs: [
                {
                  channel_id: 1,
                  model: 'gpt-test',
                  input_usd_per_million_micros: 1000000,
                  output_usd_per_million_micros: 2000000,
                  cache_read_usd_per_million_micros: 0,
                  cache_write_usd_per_million_micros: 0,
                  source: 'models.dev',
                  manual: false,
                  valid: true,
                  fetched_at: 1,
                },
              ],
            },
          ],
        },
      },
    })
    post.mockResolvedValueOnce({
      data: { success: true, data: { updated_channels: [1] } },
    })
    vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: channels } })
    renderPage()
    await userEvent.click(await screen.findByRole('checkbox', { name: 'Select OpenAI channel' }))
    await userEvent.click(screen.getByRole('button', { name: /Preview costs/i }))
    expect(await screen.findByText('Ready to apply')).toBeVisible()
    expect(screen.getByText(/gpt-test/)).toBeVisible()
    await userEvent.click(screen.getByRole('button', { name: /Apply costs/i }))
    await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
    expect(post.mock.calls[0]?.[0]).toBe('/api/upstream_costs/preview')
    expect(post.mock.calls[1]?.[0]).toBe('/api/upstream_costs/apply')
    expect(post.mock.calls[1]?.[1]).toEqual({
      items: [{ channel_id: 1, version: 7, costs: expect.any(Array) }],
    })
  })

  it('applies each channel separately when several previews are ready', async () => {
    const post = vi.spyOn(api, 'post')
    post.mockResolvedValueOnce({
      data: {
        success: true,
        data: {
          results: [
            { channel_id: 1, source: 'models.dev', supported: true, version: 7, costs: [] },
            { channel_id: 3, source: 'models.dev', supported: true, version: 8, costs: [] },
          ],
        },
      },
    })
    post.mockResolvedValue({
      data: { success: true, data: { updated_channels: [1, 3] } },
    })
    vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: channels } })
    renderPage()

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Select OpenAI channel' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'Select Second OpenAI channel' }))
    await userEvent.click(screen.getByRole('button', { name: /Preview costs/i }))
    await screen.findAllByText('Ready to apply')
    await userEvent.click(screen.getByRole('button', { name: /Apply costs/i }))

    await waitFor(() => expect(post).toHaveBeenCalledTimes(3))
    expect(post.mock.calls[1]?.[1]).toEqual({ items: [{ channel_id: 1, version: 7, costs: [] }] })
    expect(post.mock.calls[2]?.[1]).toEqual({ items: [{ channel_id: 3, version: 8, costs: [] }] })
  })
})

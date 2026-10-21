/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { PartnerWorkspace } from '../partner-workspace'

const tenant = {
  id: 7,
  slug: 'acme',
  partner_code: 'acme-partner',
  name: 'Acme',
  status: 'active',
  created_by: 1,
  created_at: 1,
  updated_at: 1,
}

const domain = {
  id: 11,
  tenant_id: tenant.id,
  hostname: 'app.example.com',
  status: 'pending',
  verification_method: 'dns',
}

afterEach(() => {
  vi.restoreAllMocks()
})

function renderWorkspace() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <PartnerWorkspace />
    </QueryClientProvider>
  )
  return queryClient
}

function mockTenantRequests() {
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/tenants') {
      return { data: { success: true, data: [tenant] } }
    }
    if (url === `/api/tenants/${tenant.id}/members`) {
      return { data: { success: true, data: [] } }
    }
    if (url === `/api/tenants/${tenant.id}/domains`) {
      return { data: { success: true, data: [domain] } }
    }
    return {
      data: {
        success: true,
        data: { cname_target: 'proxy.example.com', a_record_ip: '192.0.2.1' },
      },
    }
  })
  return get
}

test('clicking Verify opens the dialog and verifies the selected domain', async () => {
  const get = mockTenantRequests()
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true, data: null } })
  renderWorkspace()

  await userEvent.click(await screen.findByRole('button', { name: 'Verify' }))
  const dialog = await screen.findByRole('dialog', { name: 'Verify domain' })
  expect(dialog).toHaveTextContent(domain.hostname)

  await userEvent.type(
    within(dialog).getByRole('textbox'),
    'verification-token'
  )
  await userEvent.click(within(dialog).getByRole('button', { name: 'Verify' }))

  await waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      `/api/tenants/${tenant.id}/domains/${domain.id}/verify`,
      { token: 'verification-token' }
    )
  )
  await waitFor(() =>
    expect(
      get.mock.calls.filter(
        ([url]) => url === `/api/tenants/${tenant.id}/domains`
      ).length
    ).toBeGreaterThan(1)
  )
})

test('clicking Delete opens permanent deletion confirmation for the selected domain', async () => {
  const get = mockTenantRequests()
  const remove = vi
    .spyOn(api, 'delete')
    .mockResolvedValue({ data: { success: true, data: null } })
  renderWorkspace()

  await userEvent.click(await screen.findByRole('button', { name: 'Delete' }))
  const dialog = await screen.findByRole('alertdialog', {
    name: 'Delete domain',
  })
  expect(dialog).toHaveTextContent(
    'This permanently deletes the domain. It can be added again later.'
  )

  await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))

  await waitFor(() =>
    expect(remove).toHaveBeenCalledWith(
      `/api/tenants/${tenant.id}/domains/${domain.id}`
    )
  )
  await waitFor(() =>
    expect(
      get.mock.calls.filter(
        ([url]) => url === `/api/tenants/${tenant.id}/domains`
      ).length
    ).toBeGreaterThan(1)
  )
})

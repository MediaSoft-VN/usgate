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

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test } from 'vitest'

import { ProfileDropdown } from '@/components/profile-dropdown'
import { SidebarFooter, SidebarProvider } from '@/components/ui/sidebar'
import { ROLE } from '@/lib/roles'
import { STATUS_QUERY_KEY } from '@/lib/status-query'
import { useAuthStore } from '@/stores/auth-store'

afterEach(() => {
  useAuthStore.setState(useAuthStore.getInitialState(), true)
})

test('sidebar account menu opens upward from its footer and keeps account actions available', async () => {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'alice',
    role: ROLE.USER,
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client.setQueryData(STATUS_QUERY_KEY, {})
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => (
        <SidebarProvider>
          <SidebarFooter>
            <ProfileDropdown sidebar />
          </SidebarFooter>
        </SidebarProvider>
      ),
    }),
    history: createMemoryHistory({ initialEntries: ['/dashboard'] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )

  const footer = document.querySelector('[data-slot="sidebar-footer"]')
  expect(footer).not.toBeNull()
  const trigger = within(footer as HTMLElement).getByRole('button', {
    name: 'alice',
  })
  await userEvent.click(trigger)

  expect(await screen.findByRole('menu')).toHaveAttribute('data-side', 'top')
  expect(await screen.findByText('Profile')).toBeVisible()
  expect(screen.getByText('Sign out')).toBeVisible()
})

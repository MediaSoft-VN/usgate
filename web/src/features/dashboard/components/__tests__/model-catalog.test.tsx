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
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { ModelCatalog } from '../model-catalog'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

it('loads pricing API models and renders the model list on the dashboard route', async () => {
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/pricing') {
      return {
        data: {
          success: true,
          data: [
            {
              id: 1,
              model_name: 'catalog-regression-model',
              quota_type: 0,
              model_ratio: 1,
              completion_ratio: 1,
              enable_groups: ['default'],
              group_ratio: { default: 1 },
            },
          ],
          vendors: [],
          group_ratio: { default: 1 },
          usable_group: { default: { desc: '', ratio: 1 } },
          supported_endpoint: {},
          auto_groups: [],
        },
      }
    }
    if (url === '/api/status') {
      return { data: { success: true, data: {} } }
    }
    return { data: { success: true, data: {} } }
  })

  const rootRoute = createRootRoute()
  const dashboardRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: 'dashboard/$section',
    component: ModelCatalog,
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([dashboardRoute]),
    history: createMemoryHistory({
      initialEntries: ['/dashboard/model-catalog?view=table'],
    }),
  })
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  await router.load()

  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )

  expect(await screen.findByText('catalog-regression-model')).toBeVisible()
  expect(get).toHaveBeenCalledWith('/api/pricing')
})

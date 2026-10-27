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
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { Home } from '..'

vi.mock('@/components/layout', () => ({
  PublicLayout: (props: { children: import('react').ReactNode }) =>
    props.children,
}))

vi.mock('@/components/layout/components/footer', () => ({
  Footer: () => <footer />,
}))

vi.mock('@/context/theme-provider', () => ({
  useTheme: () => ({ resolvedTheme: 'light' }),
}))

vi.mock('@/features/home/hooks', () => ({
  useHomePageContent: () => ({ content: '', isLoaded: true, isUrl: false }),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: null }),
}))

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({ auth: { user: null } }),
}))

afterEach(() => {
  vi.clearAllMocks()
})

function renderHome() {
  const router = createRouter({
    routeTree: createRootRoute({ component: Home }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  return render(<RouterProvider router={router} />)
}

describe('public home landing page', () => {
  it('presents the hero, metrics, features, and steps before the footer', async () => {
    renderHome()

    expect(
      await screen.findByRole('heading', {
        name: /Unified API Gateway for.*Vast Range of AI Models/,
      })
    ).toBeVisible()
    expect(
      screen.getByRole('heading', {
        name: /Built for developers,\s*designed for scale/,
      })
    ).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Three steps to get started' })
    ).toBeVisible()
    expect(screen.getByRole('contentinfo')).toBeInTheDocument()
    expect(
      screen.queryByRole('heading', { name: /Ready to simplify/ })
    ).not.toBeInTheDocument()
    expect(
      screen.getAllByRole('button', { name: 'Get Started' })[0]
    ).toHaveAttribute('href', '/sign-up')
  })
})

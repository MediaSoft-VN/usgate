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
import { afterEach, describe, expect, it, vi } from 'vitest'

import { Route } from '../index'

const authStore = vi.hoisted(() => ({ user: null as { id: number } | null }))
const resolveAuthentication = vi.hoisted(() => vi.fn())

vi.mock('@/features/home', () => ({ Home: () => null }))
vi.mock('@/lib/auth-session', () => ({ resolveAuthentication }))
vi.mock('@/stores/auth-store', () => ({
  useAuthStore: { getState: () => ({ auth: authStore }) },
}))

afterEach(() => {
  authStore.user = null
  resolveAuthentication.mockReset()
})

describe('home route default redirect', () => {
  it('resolves the session then sends an anonymous visitor to sign-in', async () => {
    const beforeLoad = Route.options.beforeLoad

    await expect(beforeLoad?.({} as never)).rejects.toMatchObject({
      options: { to: '/sign-in', replace: true },
    })
    expect(resolveAuthentication).toHaveBeenCalledOnce()
  })

  it('resolves the session then sends an authenticated visitor to dashboard', async () => {
    authStore.user = { id: 1 }
    const beforeLoad = Route.options.beforeLoad

    await expect(beforeLoad?.({} as never)).rejects.toMatchObject({
      options: { to: '/dashboard', replace: true },
    })
    expect(resolveAuthentication).toHaveBeenCalledOnce()
  })
})

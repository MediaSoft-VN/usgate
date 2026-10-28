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
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { SePayIPNInfo } from '../sepay-ipn-info'
import { buildSePayIPNUrl } from '../sepay-ipn-url'

describe('SePay webhook setup information', () => {
  it('uses the configured callback base before the server or browser address', () => {
    expect(
      buildSePayIPNUrl(
        'https://callback.example.com///',
        'https://server.example.com',
        'https://browser.example.com'
      )
    ).toBe('https://callback.example.com/api/sepay/ipn')
    expect(buildSePayIPNUrl('', 'https://server.example.com/', '')).toBe(
      'https://server.example.com/api/sepay/ipn'
    )
    expect(buildSePayIPNUrl('', '', 'https://browser.example.com')).toBe(
      'https://browser.example.com/api/sepay/ipn'
    )
  })

  it('shows the shared endpoint and lets admins copy it', () => {
    const ipnUrl = 'https://gateway.example.com/api/sepay/ipn'
    render(<SePayIPNInfo ipnUrl={ipnUrl} />)

    expect(screen.getByText(ipnUrl)).toBeInTheDocument()
    expect(screen.getByText('IPN URL:')).toBeInTheDocument()
    expect(screen.getByText('SePay IPN callback URL')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: /SePay.*Open SePay website/ })
    ).toHaveAttribute('href', 'https://my.sepay.vn/login')
    expect(
      screen.getByRole('button', { name: 'Copy webhook URL' })
    ).toBeEnabled()
    expect(
      screen.queryByText(/Authenticate using the mode SECRET KEY/)
    ).not.toBeInTheDocument()
    expect(screen.queryByText(/Send application\/json/)).not.toBeInTheDocument()
  })
})

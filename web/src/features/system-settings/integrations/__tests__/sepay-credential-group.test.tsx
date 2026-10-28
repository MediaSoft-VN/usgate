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
import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { SePayCredentialGroup } from '../sepay-credential-group'

describe('SePay credential groups', () => {
  it('visually groups Sandbox and Live credentials under separate headings', () => {
    render(
      <>
        <SePayCredentialGroup mode='sandbox'>
          <label>MERCHANT ID</label>
          <label>SECRET KEY</label>
        </SePayCredentialGroup>
        <SePayCredentialGroup mode='live'>
          <label>MERCHANT ID</label>
          <label>SECRET KEY</label>
        </SePayCredentialGroup>
      </>
    )

    const sandboxGroup = screen.getByRole('region', {
      name: 'Sandbox credentials',
    })
    const liveGroup = screen.getByRole('region', { name: 'Live credentials' })

    expect(within(sandboxGroup).getByText('MERCHANT ID')).toBeInTheDocument()
    expect(within(sandboxGroup).getByText('SECRET KEY')).toBeInTheDocument()
    expect(within(liveGroup).getByText('MERCHANT ID')).toBeInTheDocument()
    expect(within(liveGroup).getByText('SECRET KEY')).toBeInTheDocument()
  })
})

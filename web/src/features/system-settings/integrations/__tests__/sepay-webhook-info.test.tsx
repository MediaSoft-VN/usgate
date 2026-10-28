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

import { SePayWebhookInfo } from '../sepay-webhook-info'
import { buildSePayWebhookUrl } from '../sepay-webhook-url'

describe('SePay webhook setup information', () => {
  it('uses the configured callback base before the server or browser address', () => {
    expect(
      buildSePayWebhookUrl(
        'https://callback.example.com///',
        'https://server.example.com',
        'https://browser.example.com'
      )
    ).toBe('https://callback.example.com/api/sepay/webhook')
    expect(buildSePayWebhookUrl('', 'https://server.example.com/', '')).toBe(
      'https://server.example.com/api/sepay/webhook'
    )
    expect(buildSePayWebhookUrl('', '', 'https://browser.example.com')).toBe(
      'https://browser.example.com/api/sepay/webhook'
    )
  })

  it('shows the endpoint, authentication options, and transfer matching rules', () => {
    const webhookUrl = 'https://gateway.example.com/api/sepay/webhook'
    render(<SePayWebhookInfo webhookUrl={webhookUrl} />)

    expect(screen.getByText(webhookUrl)).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Copy webhook URL' })
    ).toBeEnabled()
    expect(
      screen.getByText(
        'Authenticate with API Key (Authorization: Apikey <Webhook secret>) or HMAC-SHA256 using X-SePay-Timestamp and X-SePay-Signature.'
      )
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Send application/json. The transfer code or content must contain the order number, and the transfer amount must match exactly.'
      )
    ).toBeInTheDocument()
  })
})

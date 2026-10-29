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
import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { PAYMENT_TYPES } from '../constants'
import { requestPaymentAmount, usePayment } from './use-payment'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('payment amount routing', () => {
  test('uses the dedicated Waffo amount calculator', async () => {
    const calls: string[] = []
    const amount = await requestPaymentAmount(120, PAYMENT_TYPES.WAFFO, {
      regular: async () => {
        calls.push('regular')
        return { success: true, data: '1' }
      },
      sepay: async () => {
        calls.push('sepay')
        return { success: true, data: '260000' }
      },
      stripe: async () => {
        calls.push('stripe')
        return { success: true, data: '2' }
      },
      waffo: async (request) => {
        calls.push(`waffo:${request.amount}`)
        return { success: true, data: '18.75' }
      },
      waffoPancake: async () => {
        calls.push('pancake')
        return { success: true, data: '4' }
      },
    })

    expect(amount).toBe(18.75)
    expect(calls).toEqual(['waffo:120'])
  })

  test('uses the SePay VND amount calculator for the confirmation amount', async () => {
    const calls: string[] = []
    const amount = await requestPaymentAmount(5, PAYMENT_TYPES.SEPAY, {
      regular: async () => {
        calls.push('regular')
        return { success: true, data: '36.5' }
      },
      sepay: async (request) => {
        calls.push(`sepay:${request.amount}`)
        return { success: true, data: '130000' }
      },
      stripe: async () => ({ success: true, data: '2' }),
      waffo: async () => ({ success: true, data: '3' }),
      waffoPancake: async () => ({ success: true, data: '4' }),
    })

    expect(amount).toBe(130000)
    expect(calls).toEqual(['sepay:5'])
  })
})

test('posts SePay checkout fields in the documented order', async () => {
  let submittedFieldNames: string[] = []
  vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      message: 'success',
      data: {
        success_url: 'https://gateway.example/api/sepay/return',
        signature: 'signed',
        payment_method: 'BANK_TRANSFER',
        order_invoice_number: 'USR1SEPAY123',
        order_description: 'Top-up',
        operation: 'PURCHASE',
        currency: 'VND',
        merchant: 'merchant',
        order_amount: '50000',
        cancel_url: 'https://gateway.example/api/sepay/return',
        error_url: 'https://gateway.example/api/sepay/return',
        customer_id: '1',
      },
      url: 'https://pay-sandbox.sepay.vn/v1/checkout/init',
    },
  })
  vi.spyOn(HTMLFormElement.prototype, 'submit').mockImplementation(function (
    this: HTMLFormElement
  ) {
    submittedFieldNames = [...this.elements].map(
      (element) => (element as HTMLInputElement).name
    )
  })

  const { result } = renderHook(() => usePayment())
  let succeeded = false
  await act(async () => {
    succeeded = await result.current.processPayment(10, PAYMENT_TYPES.SEPAY)
  })

  expect(succeeded).toBe(true)
  expect(submittedFieldNames).toEqual([
    'order_amount',
    'merchant',
    'currency',
    'operation',
    'order_description',
    'order_invoice_number',
    'customer_id',
    'payment_method',
    'success_url',
    'error_url',
    'cancel_url',
    'signature',
  ])
})

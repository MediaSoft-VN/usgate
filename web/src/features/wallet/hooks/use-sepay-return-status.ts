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
import { useQuery } from '@tanstack/react-query'
import { useRef } from 'react'

import { getUserBillingHistory, isApiSuccess } from '../api'
import type { TopupStatus } from '../types'

const sePayStatusPollIntervalMs = 2_000
const sePayStatusPollTimeoutMs = 120_000
const sePayTopUpTradeNoPattern = /^USR\d+SEPAY\d+$/i

export function useSePayReturnStatus(
  tradeNo: string | undefined,
  enabled: boolean
) {
  const validTradeNo =
    tradeNo && sePayTopUpTradeNoPattern.test(tradeNo) ? tradeNo : undefined
  const pollingStartedAt = useRef({ tradeNo, timestamp: Date.now() })
  if (pollingStartedAt.current.tradeNo !== tradeNo) {
    pollingStartedAt.current = { tradeNo, timestamp: Date.now() }
  }

  return useQuery<TopupStatus | null>({
    queryKey: ['wallet', 'sepay-return-status', validTradeNo],
    queryFn: async () => {
      if (!validTradeNo) return null
      const response = await getUserBillingHistory(1, 10, validTradeNo)
      if (!isApiSuccess(response) || !response.data) return null
      return (
        response.data.items.find((item) => item.trade_no === validTradeNo)
          ?.status ?? null
      )
    },
    enabled: enabled && Boolean(validTradeNo),
    retry: false,
    refetchInterval: (query) => {
      const status = query.state.data
      if (status && status !== 'pending') return false
      if (
        Date.now() - pollingStartedAt.current.timestamp >=
        sePayStatusPollTimeoutMs
      ) {
        return false
      }
      return sePayStatusPollIntervalMs
    },
    refetchIntervalInBackground: false,
  })
}

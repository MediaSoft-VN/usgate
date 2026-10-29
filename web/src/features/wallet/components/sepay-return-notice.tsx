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
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

export type SePayReturnDisplayStatus =
  | 'pending'
  | 'success'
  | 'fail'
  | 'cancel'
  | 'expired'

interface SePayReturnNoticeProps {
  status: SePayReturnDisplayStatus
  tradeNo?: string
}

const statusCopy: Record<
  SePayReturnDisplayStatus,
  { title: string; description: string; destructive?: boolean }
> = {
  pending: {
    title: 'Payment processing',
    description:
      'Waiting for SePay to confirm this payment. Your wallet will update automatically.',
  },
  success: {
    title: 'Payment successful',
    description:
      'SePay confirmed the payment and your wallet has been updated.',
  },
  fail: {
    title: 'Payment failed',
    description:
      'SePay could not complete the payment. Check order history before retrying.',
    destructive: true,
  },
  cancel: {
    title: 'Payment canceled',
    description:
      'You canceled the SePay checkout. You can start another payment from the wallet.',
  },
  expired: {
    title: 'Payment expired',
    description: 'This payment order has expired.',
    destructive: true,
  },
}

export function SePayReturnNotice(props: SePayReturnNoticeProps) {
  const { t } = useTranslation()
  const copy = statusCopy[props.status]

  return (
    <Alert
      className='mb-4'
      variant={copy.destructive ? 'destructive' : 'default'}
      data-testid='sepay-return-notice'
    >
      <AlertTitle>{t(copy.title)}</AlertTitle>
      <AlertDescription>
        {t(copy.description)}
        {props.tradeNo && (
          <span className='mt-1 block'>
            {t('Order number: {{tradeNo}}', { tradeNo: props.tradeNo })}
          </span>
        )}
      </AlertDescription>
    </Alert>
  )
}

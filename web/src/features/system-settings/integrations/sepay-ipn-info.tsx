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

import { CopyButton } from '@/components/copy-button'
import { Alert, AlertDescription } from '@/components/ui/alert'

type SePayIPNInfoProps = {
  ipnUrl: string
}

export function SePayIPNInfo(props: SePayIPNInfoProps) {
  const { t } = useTranslation()

  return (
    <Alert className='border-teal-200 bg-teal-50 text-teal-950 dark:border-teal-900 dark:bg-teal-950 dark:text-teal-50'>
      <AlertDescription className='space-y-2 text-inherit'>
        <p className='font-medium'>{t('SePay IPN callback URL')}</p>
        <div className='flex min-w-0 flex-col items-start gap-1.5 sm:flex-row sm:items-center'>
          <span className='shrink-0 font-medium'>{t('IPN URL:')}</span>
          <span className='flex min-w-0 items-center gap-2'>
            <code className='min-w-0 rounded bg-teal-100 px-1.5 py-0.5 text-xs break-all dark:bg-teal-900'>
              {props.ipnUrl}
            </code>
            <CopyButton
              value={props.ipnUrl}
              size='icon'
              className='size-7'
              tooltip={t('Copy webhook URL')}
              aria-label={t('Copy webhook URL')}
            />
          </span>
        </div>
        <a
          href='https://my.sepay.vn/login'
          target='_blank'
          rel='noreferrer'
          className='inline-flex items-center gap-1.5 underline hover:no-underline'
        >
          SePay
          <span className='text-xs font-normal'>
            ({t('Open SePay website')})
          </span>
        </a>
      </AlertDescription>
    </Alert>
  )
}

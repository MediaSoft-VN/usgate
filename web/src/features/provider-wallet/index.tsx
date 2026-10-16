import { useQuery } from '@tanstack/react-query'
import { WalletCards } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { formatCurrencyUSD } from '@/lib/format'
import { createServerError } from '@/lib/server-error-message'

import { getChannels } from '../channels/api'

const PAGE_SIZE = 20

export function ProviderWallet() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const channelsQuery = useQuery({
    queryKey: ['provider-wallet', page],
    queryFn: async () => {
      const response = await getChannels({
        p: page,
        page_size: PAGE_SIZE,
        sort_by: 'id',
        sort_order: 'desc',
      })
      if (!response.success) {
        throw createServerError(response, t('Failed to load channels'))
      }
      return (
        response.data ?? { items: [], total: 0, page, page_size: PAGE_SIZE }
      )
    },
  })

  const data = channelsQuery.data
  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Provider')} {t('Wallet')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {channelsQuery.isLoading && <LoadingState />}
        {channelsQuery.isError && (
          <ErrorState onRetry={() => void channelsQuery.refetch()} />
        )}
        {data && data.items.length === 0 && (
          <EmptyState icon={WalletCards} title={t('No Channels Found')} />
        )}
        {data && data.items.length > 0 && (
          <div className='bg-card overflow-hidden rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Channel')}</TableHead>
                  <TableHead>{t('Provider')}</TableHead>
                  <TableHead>{t('Balance')}</TableHead>
                  <TableHead>{t('Used amount')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.items.map((channel) => (
                  <TableRow key={channel.id}>
                    <TableCell>
                      <div className='flex flex-col'>
                        <span>{channel.name}</span>
                        <span className='text-muted-foreground text-xs'>
                          #{channel.id}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>{channel.type}</TableCell>
                    <TableCell className='font-medium tabular-nums'>
                      {formatCurrencyUSD(channel.balance)}
                    </TableCell>
                    <TableCell className='tabular-nums'>
                      {formatQuotaWithCurrency(channel.used_quota, {
                        showSymbol: false,
                      })}
                    </TableCell>
                    <TableCell>
                      {channel.status === 1 ? t('Enabled') : t('Disabled')}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <div className='flex items-center justify-between border-t px-4 py-3'>
              <span className='text-muted-foreground text-sm'>
                {data.total} {t('Channel')}
              </span>
              <div className='flex items-center gap-2'>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page <= 1}
                  onClick={() => setPage((current) => current - 1)}
                >
                  {t('Previous')}
                </Button>
                <span className='text-muted-foreground text-sm'>
                  {page} / {totalPages}
                </span>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page >= totalPages}
                  onClick={() => setPage((current) => current + 1)}
                >
                  {t('Next')}
                </Button>
              </div>
            </div>
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

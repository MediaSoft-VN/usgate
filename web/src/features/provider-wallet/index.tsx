import { useQuery } from '@tanstack/react-query'
import { WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatNumber } from '@/lib/format'
import { createServerError } from '@/lib/server-error-message'

import { getChannels } from '../channels/api'
import { getProviderFundings } from '../provider-funding/api'

export function ProviderWallet() {
  const { t } = useTranslation()
  const channelsQuery = useQuery({
    queryKey: ['provider-wallet-channels'],
    queryFn: async () => {
      const response = await getChannels({ p: 1, page_size: 100, sort_by: 'id', sort_order: 'desc' })
      if (!response.success) throw createServerError(response, t('Failed to load'))
      return response.data?.items ?? []
    },
  })
  const walletQuery = useQuery({
    queryKey: ['provider-wallet'],
    queryFn: async () => (await getProviderFundings(1, 1)).wallets,
  })
  const channelNames = new Map(
    (channelsQuery.data ?? []).map((channel) => [channel.id, channel.name || String(channel.type)])
  )
  const wallets = walletQuery.data ?? []

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Provider total wallet')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {(channelsQuery.isLoading || walletQuery.isLoading) && <LoadingState />}
        {(channelsQuery.isError || walletQuery.isError) && (
          <ErrorState onRetry={() => { void channelsQuery.refetch(); void walletQuery.refetch() }} />
        )}
        {!channelsQuery.isLoading && !walletQuery.isLoading && wallets.length === 0 && (
          <EmptyState icon={WalletCards} title={t('No provider funding records')} />
        )}
        {wallets.length > 0 && (
          <div className='bg-card overflow-hidden rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Channel')}</TableHead>
                  <TableHead>{t('Currency')}</TableHead>
                  <TableHead>{t('Total funded')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {wallets.map((wallet) => (
                  <TableRow key={`${wallet.channel_id}-${wallet.currency}`}>
                    <TableCell>{channelNames.get(wallet.channel_id) ?? t('Unknown')}</TableCell>
                    <TableCell>{wallet.currency}</TableCell>
                    <TableCell className='tabular-nums'>{formatNumber(wallet.amount_minor / 100)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

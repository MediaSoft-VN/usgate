import { useQuery } from '@tanstack/react-query'
import { WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatNumber } from '@/lib/format'

import { getProviderFundings } from '../provider-funding/api'

export function ProviderWallet() {
  const { t } = useTranslation()
  const walletQuery = useQuery({
    queryKey: ['provider-wallet'],
    queryFn: async () => (await getProviderFundings(1, 1)).wallets,
  })
  const wallets = walletQuery.data ?? []

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Provider total wallet')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {walletQuery.isLoading && <LoadingState />}
        {walletQuery.isError && (
          <ErrorState onRetry={() => { void walletQuery.refetch() }} />
        )}
        {!walletQuery.isLoading && wallets.length === 0 && (
          <EmptyState icon={WalletCards} title={t('No provider funding records')} />
        )}
        {wallets.length > 0 && (
          <div className='bg-card overflow-hidden rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Currency')}</TableHead>
                  <TableHead>{t('Total funded')}</TableHead>
                  <TableHead>{t('Quota')}</TableHead>
                  <TableHead>{t('Available')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {wallets.map((wallet) => (
                  <TableRow key={wallet.currency}>
                    <TableCell>{wallet.currency}</TableCell>
                    <TableCell className='tabular-nums'>{formatNumber(wallet.amount_minor / 100)}</TableCell>
                    <TableCell className='tabular-nums'>{formatNumber(wallet.allocated_minor / 100)}</TableCell>
                    <TableCell className='tabular-nums'>{formatNumber(wallet.available_minor / 100)}</TableCell>
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

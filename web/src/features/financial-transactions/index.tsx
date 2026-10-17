import { useQuery } from '@tanstack/react-query'
import { ReceiptText } from 'lucide-react'
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
import { formatNumber } from '@/lib/format'

import { getFinancialTransactions } from './api'

const PAGE_SIZE = 20

export function FinancialTransactions() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const transactionsQuery = useQuery({
    queryKey: ['financial-transactions', page],
    queryFn: () => getFinancialTransactions(page, PAGE_SIZE),
  })
  const data = transactionsQuery.data
  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Financial transactions')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {transactionsQuery.isLoading && <LoadingState />}
        {transactionsQuery.isError && (
          <ErrorState onRetry={() => void transactionsQuery.refetch()} />
        )}
        {data && data.items.length === 0 && (
          <EmptyState icon={ReceiptText} title={t('No transactions found')} />
        )}
        {data && data.items.length > 0 && (
          <div className='bg-card overflow-hidden rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Transaction')}</TableHead>
                  <TableHead>{t('Type')}</TableHead>
                  <TableHead>{t('Amount')}</TableHead>
                  <TableHead>{t('Currency')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.items.map((transaction) => (
                  <TableRow key={transaction.id}>
                    <TableCell className='font-mono text-xs'>
                      {transaction.transaction_no}
                    </TableCell>
                    <TableCell>{transaction.type}</TableCell>
                    <TableCell className='tabular-nums'>
                      {formatNumber(transaction.amount_minor)}
                    </TableCell>
                    <TableCell>{transaction.currency}</TableCell>
                    <TableCell>{transaction.status}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <div className='flex items-center justify-between border-t px-4 py-3'>
              <span className='text-muted-foreground text-sm'>
                {data.total} {t('Transaction')}
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

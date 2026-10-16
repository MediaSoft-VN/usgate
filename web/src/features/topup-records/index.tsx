import { useQuery } from '@tanstack/react-query'
import { Download, ReceiptText } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge } from '@/components/status-badge'
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

import { getAllBillingHistory } from '../wallet/api'
import { formatTimestamp, getStatusConfig } from '../wallet/lib/billing'

const PAGE_SIZE = 20

export function TopupRecords() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const topupsQuery = useQuery({
    queryKey: ['topup-records', page],
    queryFn: async () => {
      const response = await getAllBillingHistory(page, PAGE_SIZE)
      if (!response.success) {
        throw createServerError(response, t('Failed to load top-up records'))
      }
      return response.data ?? { items: [], total: 0 }
    },
  })

  const data = topupsQuery.data
  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Top-up')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='sm'
          disabled={!data?.items.length}
          onClick={() => {
            if (!data?.items.length) return
            const escapeCsv = (value: unknown) =>
              `"${String(value ?? '').replaceAll('"', '""')}"`
            const rows = [
              [
                'ID',
                'User ID',
                'Amount',
                'Money',
                'Payment',
                'Created',
                'Status',
              ],
              ...data.items.map((record) => [
                record.id,
                record.user_id,
                record.amount,
                record.money,
                record.payment_method,
                formatTimestamp(record.create_time),
                record.status,
              ]),
            ]
            const csv = `\uFEFF${rows.map((row) => row.map(escapeCsv).join(',')).join('\n')}`
            const url = URL.createObjectURL(
              new Blob([csv], { type: 'text/csv;charset=utf-8' })
            )
            const link = document.createElement('a')
            link.href = url
            link.download = `topup-records-page-${page}.csv`
            link.click()
            URL.revokeObjectURL(url)
          }}
        >
          <Download className='size-4' aria-hidden='true' />
          {t('Export')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {topupsQuery.isLoading && <LoadingState />}
        {topupsQuery.isError && (
          <ErrorState onRetry={() => void topupsQuery.refetch()} />
        )}
        {data && data.items.length === 0 && (
          <EmptyState icon={ReceiptText} title={t('No Data')} />
        )}
        {data && data.items.length > 0 && (
          <div className='bg-card overflow-hidden rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('ID')}</TableHead>
                  <TableHead>{t('User ID')}</TableHead>
                  <TableHead>{t('Amount')}</TableHead>
                  <TableHead>{t('Payment')}</TableHead>
                  <TableHead>{t('Created')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.items.map((record) => {
                  const status = getStatusConfig(record.status)
                  return (
                    <TableRow key={record.id}>
                      <TableCell className='font-mono text-xs'>
                        {record.id}
                      </TableCell>
                      <TableCell>{record.user_id}</TableCell>
                      <TableCell>
                        <div className='flex flex-col tabular-nums'>
                          <span>{formatQuotaWithCurrency(record.amount)}</span>
                          <span className='text-muted-foreground text-xs'>
                            {formatCurrencyUSD(record.money)}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>{record.payment_method || '-'}</TableCell>
                      <TableCell>
                        {formatTimestamp(record.create_time)}
                      </TableCell>
                      <TableCell>
                        <StatusBadge
                          label={t(status.label)}
                          variant={status.variant}
                          copyable={false}
                        />
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
            <div className='flex items-center justify-between border-t px-4 py-3'>
              <span className='text-muted-foreground text-sm'>
                {data.total} {t('Top-up')}
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

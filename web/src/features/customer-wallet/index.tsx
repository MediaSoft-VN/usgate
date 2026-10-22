import { useQuery } from '@tanstack/react-query'
import { WalletCards } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { TimestampCell } from '@/components/activity-time-cell'
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
import { createServerError } from '@/lib/server-error-message'
import { formatNumber } from '@/lib/format'

import { getFinancialWallets } from './api'
import { getUsers } from '../users/api'
import { UserQuotaCell } from '../users/components/user-quota-cell'

const PAGE_SIZE = 20

export function CustomerWallet() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const usersQuery = useQuery({
    queryKey: ['customer-wallet', page],
    queryFn: async () => {
      const response = await getUsers({
        p: page,
        page_size: PAGE_SIZE,
        sort_by: 'id',
        sort_order: 'desc',
      })
      if (!response.success) {
        throw createServerError(response, t('Failed to load users'))
      }
      return (
        response.data ?? { items: [], total: 0, page, page_size: PAGE_SIZE }
      )
    },
  })
  const walletsQuery = useQuery({
    queryKey: ['financial-wallets', 'user'],
    queryFn: () => getFinancialWallets('user'),
  })

  const data = usersQuery.data
  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Wallet')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {usersQuery.isLoading && <LoadingState />}
        {usersQuery.isError && (
          <ErrorState onRetry={() => void usersQuery.refetch()} />
        )}
        {data && data.items.length === 0 && (
          <EmptyState icon={WalletCards} title={t('No Users Found')} />
        )}
        {data && data.items.length > 0 && (
          <div className='bg-card overflow-hidden rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Username')}</TableHead>
                  <TableHead>{t('Created At')}</TableHead>
                  <TableHead>{t('Group')}</TableHead>
                  <TableHead>{t('Quota')}</TableHead>
                  <TableHead>{t('Available balance')}</TableHead>
                  <TableHead>{t('Reserved balance')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.items.map((user) => {
                  const wallet = walletsQuery.data?.find(
                    (item) => item.owner_id === user.id
                  )
                  return (
                    <TableRow key={user.id}>
                    <TableCell>
                      <div className='flex flex-col'>
                        <span>{user.display_name || user.username}</span>
                        <span className='text-muted-foreground text-xs'>
                          {user.username}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <TimestampCell
                        timestamp={user.created_at ?? 0}
                        format='absolute'
                        justNowLabel={t('Just now')}
                      />
                    </TableCell>
                    <TableCell>{user.group}</TableCell>
                    <TableCell>
                      <UserQuotaCell
                        remaining={user.quota}
                        used={user.used_quota}
                      />
                    </TableCell>
                    <TableCell>
                      {wallet
                        ? `${wallet.currency} ${formatNumber(wallet.available_minor)}`
                        : '-'}
                    </TableCell>
                    <TableCell>
                      {wallet
                        ? `${wallet.currency} ${formatNumber(wallet.reserved_minor)}`
                        : '-'}
                    </TableCell>
                    <TableCell>
                      {user.status === 1 ? t('Enabled') : t('Disabled')}
                    </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
            <div className='flex items-center justify-between border-t px-4 py-3'>
              <span className='text-muted-foreground text-sm'>
                {data.total} {t('Users')}
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

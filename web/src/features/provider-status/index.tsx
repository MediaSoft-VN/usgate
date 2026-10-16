import { useQuery } from '@tanstack/react-query'
import { Activity, CheckCircle2, XCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge } from '@/components/status-badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { createServerError } from '@/lib/server-error-message'

import { getChannels } from '../channels/api'

function StatusSummaryCard(props: {
  title: string
  value: number
  icon: typeof Activity
}) {
  const Icon = props.icon
  return (
    <Card>
      <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
        <CardTitle className='text-muted-foreground text-sm font-medium'>
          {props.title}
        </CardTitle>
        <Icon className='text-muted-foreground size-4' aria-hidden='true' />
      </CardHeader>
      <CardContent>
        <div className='text-2xl font-bold tabular-nums'>{props.value}</div>
      </CardContent>
    </Card>
  )
}

export function ProviderStatus() {
  const { t } = useTranslation()
  const channelsQuery = useQuery({
    queryKey: ['provider-status'],
    queryFn: async () => {
      const response = await getChannels({
        p: 1,
        page_size: 100,
        sort_by: 'id',
        sort_order: 'desc',
      })
      if (!response.success) {
        throw createServerError(response, t('Failed to load channels'))
      }
      return response.data ?? { items: [], total: 0, page: 1, page_size: 100 }
    },
  })

  const channels = channelsQuery.data?.items ?? []
  const enabledCount = channels.filter((channel) => channel.status === 1).length
  const disabledCount = channels.length - enabledCount

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Provider')} {t('Status')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {channelsQuery.isLoading && <LoadingState />}
        {channelsQuery.isError && (
          <ErrorState onRetry={() => void channelsQuery.refetch()} />
        )}
        {channelsQuery.data && channels.length === 0 && (
          <EmptyState icon={Activity} title={t('No Channels Found')} />
        )}
        {channelsQuery.data && channels.length > 0 && (
          <div className='flex flex-col gap-4'>
            <div className='grid gap-4 sm:grid-cols-3'>
              <StatusSummaryCard
                title={t('Channels')}
                value={channelsQuery.data.total}
                icon={Activity}
              />
              <StatusSummaryCard
                title={t('Enabled')}
                value={enabledCount}
                icon={CheckCircle2}
              />
              <StatusSummaryCard
                title={t('Disabled')}
                value={disabledCount}
                icon={XCircle}
              />
            </div>
            <div className='bg-card overflow-hidden rounded-lg border'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Channel')}</TableHead>
                    <TableHead>{t('Provider')}</TableHead>
                    <TableHead>{t('Response Time')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {channels.map((channel) => (
                    <TableRow key={channel.id}>
                      <TableCell>{channel.name}</TableCell>
                      <TableCell>{channel.type}</TableCell>
                      <TableCell className='tabular-nums'>
                        {channel.response_time} ms
                      </TableCell>
                      <TableCell>
                        <StatusBadge
                          label={
                            channel.status === 1 ? t('Enabled') : t('Disabled')
                          }
                          variant={channel.status === 1 ? 'success' : 'danger'}
                          copyable={false}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

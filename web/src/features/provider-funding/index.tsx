import { useQuery } from '@tanstack/react-query'
import { WalletCards } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import dayjs from '@/lib/dayjs'
import { formatNumber, formatTimestampToDate } from '@/lib/format'
import { createServerError } from '@/lib/server-error-message'

import { getChannels } from '../channels/api'
import { CompactDateTimeRangePicker } from '../usage-logs/components/compact-date-time-range-picker'
import {
  getProviderFundings,
  type ProviderCashFlow,
  type ProviderFunding,
  type ProviderFundingAllocation,
} from './api'

const PAGE_SIZE = 20

type CashFlowFilters = {
  fromDate: string
  toDate: string
  direction: string
}

type CashFlowRow = {
  id: string
  direction: 'incoming' | 'outgoing'
  type: 'funding' | 'allocation'
  counterparty: string
  amountMinor: number
  currency: string
  reference: string
  createdBy: string
  createdAt: number
}

function formatSignedCashFlowAmount(
  amountMinor: number,
  direction: CashFlowRow['direction'],
  currency: string
) {
  const signedAmount =
    direction === 'incoming' ? Math.abs(amountMinor) : -Math.abs(amountMinor)
  const sign = signedAmount >= 0 ? '+' : '-'
  return `${sign}${formatNumber(Math.abs(signedAmount) / 100)} ${currency}`
}

function createCashFlowRows(
  fundings: ProviderFunding[],
  allocations: ProviderFundingAllocation[],
  channelNames: Map<number, string>,
  unknownLabel: string
): CashFlowRow[] {
  const fundingRows: CashFlowRow[] = fundings.map((funding) => ({
    id: `funding-${funding.id}`,
    direction: 'incoming',
    type: 'funding',
    counterparty: channelNames.get(funding.channel_id) ?? unknownLabel,
    amountMinor: funding.amount_minor,
    currency: funding.currency,
    reference: funding.reference,
    createdBy: funding.created_by,
    createdAt: funding.created_at,
  }))
  const allocationRows: CashFlowRow[] = allocations.map((allocation) => ({
    id: `allocation-${allocation.id}`,
    direction: 'outgoing',
    type: 'allocation',
    counterparty: allocation.counterparty_name || unknownLabel,
    amountMinor: allocation.amount_minor,
    currency: allocation.currency,
    reference: allocation.reference,
    createdBy: allocation.created_by,
    createdAt: allocation.created_at,
  }))

  return [...fundingRows, ...allocationRows].sort(
    (left, right) =>
      right.createdAt - left.createdAt || right.id.localeCompare(left.id)
  )
}

function mapProviderCashFlows(
  cashFlows: ProviderCashFlow[],
  channelNames: Map<number, string>,
  unknownLabel: string
): CashFlowRow[] {
  return cashFlows.map((flow) => ({
    id: String(flow.id),
    direction: flow.direction,
    type: flow.type,
    counterparty:
      flow.counterparty_name ||
      channelNames.get(flow.channel_id) ||
      unknownLabel,
    amountMinor: flow.amount_minor,
    currency: flow.currency,
    reference: flow.reference,
    createdBy: flow.created_by,
    createdAt: flow.created_at,
  }))
}

export function ProviderFunding() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')
  const [direction, setDirection] = useState('all')
  const [appliedFilters, setAppliedFilters] = useState<CashFlowFilters>({
    fromDate: '',
    toDate: '',
    direction: 'all',
  })
  const channelsQuery = useQuery({
    queryKey: ['provider-funding-channels'],
    queryFn: async () => {
      const response = await getChannels({
        p: 1,
        page_size: 100,
        sort_by: 'id',
        sort_order: 'desc',
      })
      if (!response.success) {
        throw createServerError(response, t('Failed to load'))
      }
      return response.data?.items ?? []
    },
  })
  const fundingsQuery = useQuery({
    queryKey: ['provider-funding', page, appliedFilters],
    queryFn: () =>
      getProviderFundings(page, PAGE_SIZE, {
        from_date: appliedFilters.fromDate || undefined,
        to_date: appliedFilters.toDate || undefined,
        direction:
          appliedFilters.direction === 'all'
            ? undefined
            : (appliedFilters.direction as 'incoming' | 'outgoing'),
      }),
  })
  const data = fundingsQuery.data
  const channelNames = new Map(
    (channelsQuery.data ?? []).map((channel) => [
      channel.id,
      String(channel.name || channel.type),
    ])
  )
  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const cashFlowRows = data?.cash_flows
    ? mapProviderCashFlows(data.cash_flows, channelNames, t('Unknown'))
    : createCashFlowRows(
        items,
        data?.allocations ?? [],
        channelNames,
        t('Unknown')
      )
  const directionLabels: Record<string, string> = {
    all: t('All'),
    incoming: t('Incoming'),
    outgoing: t('Outgoing'),
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Cash flow history')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {fundingsQuery.isLoading && <LoadingState />}
        {fundingsQuery.isError && (
          <ErrorState onRetry={() => void fundingsQuery.refetch()} />
        )}
        {data && (
          <div className='space-y-4'>
            <div className='bg-card grid gap-3 rounded-lg border p-4 lg:grid-cols-[2fr_1fr_1fr]'>
              <div className='grid w-full gap-1.5'>
                <Label>{t('Date Range')}</Label>
                <CompactDateTimeRangePicker
                  start={
                    fromDate
                      ? dayjs(`${fromDate}T00:00:00`).toDate()
                      : undefined
                  }
                  end={
                    toDate ? dayjs(`${toDate}T23:59:59`).toDate() : undefined
                  }
                  onChange={(range) => {
                    setFromDate(
                      range.start ? dayjs(range.start).format('YYYY-MM-DD') : ''
                    )
                    setToDate(
                      range.end ? dayjs(range.end).format('YYYY-MM-DD') : ''
                    )
                  }}
                />
              </div>
              <div className='grid w-full gap-1.5'>
                <Label htmlFor='cash-flow-direction'>{t('Direction')}</Label>
                <Select
                  value={direction}
                  onValueChange={(value) => {
                    setDirection(value ?? 'all')
                  }}
                >
                  <SelectTrigger id='cash-flow-direction'>
                    <SelectValue>
                      {directionLabels[direction] ?? directionLabels.all}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='all'>{t('All')}</SelectItem>
                    <SelectItem value='incoming'>{t('Incoming')}</SelectItem>
                    <SelectItem value='outgoing'>{t('Outgoing')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className='flex w-full items-end gap-2'>
                <Button
                  type='button'
                  className='flex-1'
                  onClick={() => {
                    setPage(1)
                    setAppliedFilters({
                      fromDate,
                      toDate,
                      direction,
                    })
                  }}
                >
                  {t('Search')}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  className='flex-1'
                  onClick={() => {
                    const emptyFilters = {
                      fromDate: '',
                      toDate: '',
                      direction: 'all',
                    }
                    setFromDate('')
                    setToDate('')
                    setDirection('all')
                    setPage(1)
                    setAppliedFilters(emptyFilters)
                  }}
                >
                  {t('Reset')}
                </Button>
              </div>
            </div>
            {cashFlowRows.length === 0 ? (
              <EmptyState
                icon={WalletCards}
                title={t('No provider funding records')}
              />
            ) : (
              <div className='bg-card overflow-hidden rounded-lg border'>
                <div className='overflow-x-auto'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Direction')}</TableHead>
                        <TableHead>{t('Type')}</TableHead>
                        <TableHead>{t('Counterparty')}</TableHead>
                        <TableHead>{t('Amount')}</TableHead>
                        <TableHead>{t('Currency')}</TableHead>
                        <TableHead>{t('Reference')}</TableHead>
                        <TableHead>{t('Time')}</TableHead>
                        <TableHead>{t('Created by')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {cashFlowRows.map((row) => (
                        <TableRow key={row.id}>
                          <TableCell
                            className={
                              row.direction === 'incoming'
                                ? 'text-emerald-600 dark:text-emerald-400'
                                : 'text-red-600 dark:text-red-400'
                            }
                          >
                            {row.direction === 'incoming'
                              ? t('Incoming')
                              : t('Outgoing')}
                          </TableCell>
                          <TableCell>
                            {row.type === 'funding'
                              ? t('Provider funding')
                              : t('Credit allocation')}
                          </TableCell>
                          <TableCell>{row.counterparty}</TableCell>
                          <TableCell
                            className={
                              row.direction === 'incoming'
                                ? 'font-medium text-emerald-600 tabular-nums dark:text-emerald-400'
                                : 'font-medium text-red-600 tabular-nums dark:text-red-400'
                            }
                          >
                            {formatSignedCashFlowAmount(
                              row.amountMinor,
                              row.direction,
                              row.currency
                            )}
                          </TableCell>
                          <TableCell>{row.currency}</TableCell>
                          <TableCell className='font-mono text-xs'>
                            {row.reference}
                          </TableCell>
                          <TableCell>
                            {formatTimestampToDate(row.createdAt)}
                          </TableCell>
                          <TableCell>{row.createdBy || t('Unknown')}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
                <div className='flex items-center justify-between border-t px-4 py-3'>
                  <span className='text-muted-foreground text-sm'>{total}</span>
                  <div className='flex items-center gap-2'>
                    <Button
                      variant='outline'
                      size='sm'
                      disabled={page <= 1}
                      onClick={() => setPage((value) => value - 1)}
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
                      onClick={() => setPage((value) => value + 1)}
                    >
                      {t('Next')}
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Info, WalletCards } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatNumber } from '@/lib/format'
import { createServerError } from '@/lib/server-error-message'

import { getChannels } from '../channels/api'
import { createProviderFunding, getProviderFundings } from './api'

const PAGE_SIZE = 20

export function ProviderFunding() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [channelID, setChannelID] = useState('')
  const [reference, setReference] = useState('')
  const [amountMinor, setAmountMinor] = useState('')
  const [currency, setCurrency] = useState('USD')
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
    queryKey: ['provider-funding', page],
    queryFn: () => getProviderFundings(page, PAGE_SIZE),
  })
  const createMutation = useMutation({
    mutationFn: () =>
      createProviderFunding({
        channel_id: Number(channelID),
        reference: reference.trim(),
        amount_minor: Number(amountMinor),
        currency: currency.trim(),
        source: 'manual',
      }),
    onSuccess: () => {
      toast.success(t('Provider funding recorded'))
      setReference('')
      setAmountMinor('')
      void queryClient.invalidateQueries({ queryKey: ['provider-funding'] })
    },
  })

  const data = fundingsQuery.data
  const channels = channelsQuery.data ?? []
  const items = data?.items ?? []
  const summary = data?.summary ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        <span className='inline-flex items-center gap-2'>
          {t('Provider funding')}
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger
                aria-label={t('Provider funding guide')}
                title={t('Provider funding guide')}
                className='text-primary hover:text-primary/80 inline-flex rounded-full focus-visible:ring-2 focus-visible:outline-none'
              >
                <Info className='size-5' strokeWidth={2.5} aria-hidden='true' />
              </TooltipTrigger>
              <TooltipContent className='max-w-sm whitespace-pre-line'>
                {t('Provider funding guide')}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </span>
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <form
          className='bg-card grid gap-3 rounded-lg border p-4 md:grid-cols-5'
          onSubmit={(event) => {
            event.preventDefault()
            if (!channelID || !reference.trim() || !amountMinor || !currency.trim()) return
            createMutation.mutate()
          }}
        >
          <div className='grid gap-2'>
            <Label htmlFor='provider-funding-channel' required>{t('Channel')}</Label>
            <Select value={channelID} onValueChange={(value) => setChannelID(value ?? '')}>
              <SelectTrigger id='provider-funding-channel' className='w-full'>
                <SelectValue placeholder={t('Select channel')} />
              </SelectTrigger>
              <SelectContent>
                {channels.map((channel) => (
                  <SelectItem key={channel.id} value={String(channel.id)}>
                    {channel.name || `#${channel.id}`} ({channel.type})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='provider-funding-reference' required>{t('Reference')}</Label>
            <Input id='provider-funding-reference' value={reference} onChange={(event) => setReference(event.target.value)} />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='provider-funding-amount' required>{t('Amount')}</Label>
            <Input id='provider-funding-amount' type='number' min={1} value={amountMinor} onChange={(event) => setAmountMinor(event.target.value)} />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='provider-funding-currency' required>{t('Currency')}</Label>
            <Input id='provider-funding-currency' value={currency} onChange={(event) => setCurrency(event.target.value.toUpperCase())} />
          </div>
          <div className='flex items-end'>
            <Button type='submit' disabled={createMutation.isPending}>{t('Record provider funding')}</Button>
          </div>
        </form>

        {fundingsQuery.isLoading && <LoadingState />}
        {fundingsQuery.isError && <ErrorState onRetry={() => void fundingsQuery.refetch()} />}
        {data && (
          <>
            <div className='grid gap-3 md:grid-cols-3'>
              {summary.map((item) => (
                <div className='bg-card rounded-lg border p-4' key={item.currency}>
                  <div className='text-muted-foreground text-sm'>{item.currency}</div>
                  <div className='text-2xl font-semibold tabular-nums'>{formatNumber(item.amount_minor)}</div>
                </div>
              ))}
            </div>
            {items.length === 0 ? (
              <EmptyState icon={WalletCards} title={t('No provider funding records')} />
            ) : (
              <div className='bg-card overflow-hidden rounded-lg border'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Channel')}</TableHead>
                      <TableHead>{t('Reference')}</TableHead>
                      <TableHead>{t('Amount')}</TableHead>
                      <TableHead>{t('Currency')}</TableHead>
                      <TableHead>{t('Source')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell>{item.channel_id}</TableCell>
                        <TableCell className='font-mono text-xs'>{item.reference}</TableCell>
                        <TableCell className='tabular-nums'>{formatNumber(item.amount_minor)}</TableCell>
                        <TableCell>{item.currency}</TableCell>
                        <TableCell>{item.source}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                <div className='flex items-center justify-between border-t px-4 py-3'>
                  <span className='text-muted-foreground text-sm'>{total}</span>
                  <div className='flex items-center gap-2'>
                    <Button variant='outline' size='sm' disabled={page <= 1} onClick={() => setPage((value) => value - 1)}>{t('Previous')}</Button>
                    <span className='text-muted-foreground text-sm'>{page} / {totalPages}</span>
                    <Button variant='outline' size='sm' disabled={page >= totalPages} onClick={() => setPage((value) => value + 1)}>{t('Next')}</Button>
                  </div>
                </div>
              </div>
            )}
          </>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

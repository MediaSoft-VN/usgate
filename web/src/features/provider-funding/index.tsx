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
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
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
import {
  createProviderFunding,
  getProviderPartners,
  getProviderFundingSyncStatus,
  getProviderFundings,
  startProviderFundingSync,
} from './api'

const PAGE_SIZE = 20

export function ProviderFunding() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [channelID, setChannelID] = useState('')
  const [reference, setReference] = useState('')
  const [amountMinor, setAmountMinor] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [fundingOpen, setFundingOpen] = useState(false)
  const partnersQuery = useQuery({
    queryKey: ['provider-partners'],
    queryFn: getProviderPartners,
  })
  const channelsQuery = useQuery({
    queryKey: ['provider-funding-channels'],
    queryFn: async () => {
      const response = await getChannels({ p: 1, page_size: 100, sort_by: 'id', sort_order: 'desc' })
      if (!response.success) throw createServerError(response, t('Failed to load'))
      return response.data?.items ?? []
    },
  })
  const syncQuery = useQuery({
    queryKey: ['provider-funding-sync-status'],
    queryFn: getProviderFundingSyncStatus,
    refetchInterval: (query) => (query.state.data?.running ? 2000 : false),
  })
  const syncMutation = useMutation({
    mutationFn: startProviderFundingSync,
    onSuccess: () => {
      toast.success(t('Provider balance sync started'))
      void syncQuery.refetch()
    },
  })
  const fundingsQuery = useQuery({
    queryKey: ['provider-funding', page],
    queryFn: () => getProviderFundings(page, PAGE_SIZE),
  })
  const selectedChannel = channelsQuery.data?.find((channel) => String(channel.id) === channelID)
  const createMutation = useMutation({
    mutationFn: () =>
      createProviderFunding({
        provider_partner_id: selectedChannel?.provider_partner_id ?? 0,
        channel_id: Number(channelID),
        reference: reference.trim(),
        amount_minor: Math.round(Number(amountMinor) * 100),
        currency: currency.trim(),
        source: 'manual',
      }),
    onSuccess: () => {
      toast.success(t('Provider funding recorded'))
      setReference('')
      setAmountMinor('')
      setFundingOpen(false)
      void queryClient.invalidateQueries({ queryKey: ['provider-funding'] })
    },
  })

  const data = fundingsQuery.data
  const partnerNames = new Map(
    (partnersQuery.data ?? []).map((partner) => [partner.id, partner.name])
  )
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
      <SectionPageLayout.Actions>
        <Dialog open={fundingOpen} onOpenChange={setFundingOpen}>
          <DialogTrigger render={<Button />}>{t('Record provider funding')}</DialogTrigger>
          <DialogContent className='sm:max-w-2xl'>
            <DialogHeader>
              <DialogTitle>{t('Record provider funding')}</DialogTitle>
            </DialogHeader>
            <form
              className='grid gap-3 md:grid-cols-2'
              onSubmit={(event) => {
                event.preventDefault()
                const amount = Number(amountMinor)
                if (
                  !selectedChannel?.provider_partner_id ||
                  !reference.trim() ||
                  !Number.isFinite(amount) ||
                  amount <= 0 ||
                  !Number.isSafeInteger(Math.round(amount * 100)) ||
                  !currency.trim()
                ) return
                createMutation.mutate()
              }}
            >
              <div className='grid gap-2'>
                <Label htmlFor='provider-funding-partner' required>{t('Provider partner')}</Label>
                <Select
                  value={channelID}
                  itemToStringValue={(value) => channelsQuery.data?.find((channel) => String(channel.id) === value)?.name ?? ''}
                  onValueChange={(value) => setChannelID(value ?? '')}
                >
                  <SelectTrigger id='provider-funding-partner' className='w-full'>
                    <SelectValue placeholder={t('Select provider partner')}>
                      {selectedChannel?.name || selectedChannel?.type}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {(channelsQuery.data ?? []).map((channel) => (
                      <SelectItem key={channel.id} value={String(channel.id)}>{channel.name || channel.type}</SelectItem>
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
              <p className='text-muted-foreground text-sm italic md:col-span-2'>{t('Provider funding guide')}</p>
              <DialogFooter className='md:col-span-2'>
                <Button type='submit' disabled={createMutation.isPending}>{t('Record provider funding')}</Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
        <Button
          disabled={syncMutation.isPending || syncQuery.data?.running === true}
          onClick={() => syncMutation.mutate()}
          variant='outline'
        >
          {syncQuery.data?.running
            ? t('Provider balance sync running')
            : t('Sync provider balances')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {fundingsQuery.isLoading && <LoadingState />}
        {fundingsQuery.isError && <ErrorState onRetry={() => void fundingsQuery.refetch()} />}
        {data && (
          <>
            <div className='mt-4 grid gap-3 md:grid-cols-3'>
              {summary.map((item) => (
                <div className='bg-card rounded-lg border p-4' key={item.currency}>
                  <div className='text-muted-foreground text-sm'>{item.currency}</div>
                  <div className='text-2xl font-semibold tabular-nums'>{formatNumber(item.amount_minor / 100)}</div>
                </div>
              ))}
            </div>
            {items.length === 0 ? (
              <EmptyState icon={WalletCards} title={t('No provider funding records')} />
            ) : (
              <div className='bg-card mt-4 overflow-hidden rounded-lg border'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Provider partner')}</TableHead>
                      <TableHead>{t('Reference')}</TableHead>
                      <TableHead>{t('Funding amount')}</TableHead>
                      <TableHead>{t('Currency')}</TableHead>
                      <TableHead>{t('Source')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell>{partnerNames.get(item.provider_partner_id) ?? t('Provider level')}</TableCell>
                        <TableCell className='font-mono text-xs'>{item.reference}</TableCell>
                        <TableCell className='tabular-nums'>{formatNumber(item.amount_minor / 100)}</TableCell>
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

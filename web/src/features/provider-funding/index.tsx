import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { WalletCards } from 'lucide-react'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatNumber } from '@/lib/format'

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
  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Provider funding')}</SectionPageLayout.Title>
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
            <Input id='provider-funding-channel' type='number' min={1} value={channelID} onChange={(event) => setChannelID(event.target.value)} />
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
              {data.summary.map((item) => (
                <div className='bg-card rounded-lg border p-4' key={item.currency}>
                  <div className='text-muted-foreground text-sm'>{item.currency}</div>
                  <div className='text-2xl font-semibold tabular-nums'>{formatNumber(item.amount_minor)}</div>
                </div>
              ))}
            </div>
            {data.items.length === 0 ? (
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
                    {data.items.map((item) => (
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
                  <span className='text-muted-foreground text-sm'>{data.total}</span>
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

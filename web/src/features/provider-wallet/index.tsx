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
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
import { formatNumber } from '@/lib/format'

import {
  allocateProviderFunding,
  getProviderFundings,
} from '../provider-funding/api'
import { getUsers } from '../users/api'
import type { User } from '../users/types'

export function ProviderWallet() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [allocationOpen, setAllocationOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [allocationReference, setAllocationReference] = useState('')
  const [allocationAmount, setAllocationAmount] = useState('')
  const [currency, setCurrency] = useState('USD')
  const walletQuery = useQuery({
    queryKey: ['provider-wallet'],
    queryFn: async () => await getProviderFundings(1, 1),
  })
  const usersQuery = useQuery({
    queryKey: ['provider-funding-users'],
    queryFn: async () =>
      (
        await getUsers({
          p: 1,
          page_size: 100,
          sort_by: 'username',
          sort_order: 'asc',
        })
      ).data?.items ?? [],
  })
  const userLabel = (user: { display_name: string; username: string }) =>
    user.display_name && user.display_name !== user.username
      ? `${user.display_name} (${user.username})`
      : user.username
  const allocationMutation = useMutation({
    mutationFn: () =>
      allocateProviderFunding({
        user_id: selectedUser?.id ?? 0,
        reference: allocationReference.trim(),
        amount_minor: Math.round(Number(allocationAmount) * 100),
        currency: currency.trim(),
      }),
    onSuccess: () => {
      toast.success(t('Success'))
      setAllocationReference('')
      setAllocationAmount('')
      setSelectedUser(null)
      setAllocationOpen(false)
      void queryClient.invalidateQueries({ queryKey: ['provider-wallet'] })
      void queryClient.invalidateQueries({ queryKey: ['provider-funding'] })
    },
  })
  const wallets = walletQuery.data?.wallets ?? []
  const providerWallets = walletQuery.data?.provider_wallets ?? []
  const channelFunding = new Map<number, typeof providerWallets>()
  for (const funding of providerWallets) {
    const summaries = channelFunding.get(funding.channel_id) ?? []
    summaries.push(funding)
    channelFunding.set(funding.channel_id, summaries)
  }
  const providerChannels = [
    ...new Map(
      providerWallets.map((wallet) => [
        wallet.channel_id,
        { id: wallet.channel_id, name: wallet.channel_name },
      ])
    ).values(),
  ]
  const hasWalletData = wallets.length > 0 || providerWallets.length > 0

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Provider total wallet')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Dialog open={allocationOpen} onOpenChange={setAllocationOpen}>
          <DialogTrigger render={<Button variant='outline' />}>
            {t('Credit allocation')}
          </DialogTrigger>
          <DialogContent className='sm:max-w-2xl'>
            <DialogHeader>
              <DialogTitle>{t('Credit allocation')}</DialogTitle>
            </DialogHeader>
            <form
              className='grid gap-3 md:grid-cols-2'
              onSubmit={(event) => {
                event.preventDefault()
                const amount = Number(allocationAmount)
                if (
                  !selectedUser ||
                  !allocationReference.trim() ||
                  !Number.isFinite(amount) ||
                  amount <= 0 ||
                  !Number.isSafeInteger(Math.round(amount * 100)) ||
                  !currency.trim()
                ) {
                  return
                }
                allocationMutation.mutate()
              }}
            >
              <div className='grid gap-2 md:col-span-2'>
                <Label htmlFor='provider-allocation-user' required>
                  {t('Username')}
                </Label>
                <Select
                  value={selectedUser}
                  itemToStringLabel={(user) => userLabel(user)}
                  itemToStringValue={(user) => String(user.id)}
                  onValueChange={(value) => setSelectedUser(value ?? null)}
                >
                  <SelectTrigger
                    id='provider-allocation-user'
                    className='w-full'
                  >
                    <SelectValue placeholder={t('Username')} />
                  </SelectTrigger>
                  <SelectContent>
                    {(usersQuery.data ?? []).map((user) => (
                      <SelectItem key={user.id} value={user}>
                        {userLabel(user)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='provider-allocation-reference' required>
                  {t('Reference')}
                </Label>
                <Input
                  id='provider-allocation-reference'
                  value={allocationReference}
                  onChange={(event) =>
                    setAllocationReference(event.target.value)
                  }
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='provider-allocation-amount' required>
                  {t('Amount')}
                </Label>
                <Input
                  id='provider-allocation-amount'
                  type='number'
                  min={0.01}
                  step={0.01}
                  value={allocationAmount}
                  onChange={(event) => setAllocationAmount(event.target.value)}
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='provider-allocation-currency' required>
                  {t('Currency')}
                </Label>
                <Input
                  id='provider-allocation-currency'
                  value={currency}
                  onChange={(event) =>
                    setCurrency(event.target.value.toUpperCase())
                  }
                />
              </div>
              <DialogFooter className='md:col-span-2'>
                <Button type='submit' disabled={allocationMutation.isPending}>
                  {t('Credit allocation')}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {walletQuery.isLoading && <LoadingState />}
        {walletQuery.isError && (
          <ErrorState
            onRetry={() => {
              void walletQuery.refetch()
            }}
          />
        )}
        {!walletQuery.isLoading && !hasWalletData && (
          <EmptyState
            icon={WalletCards}
            title={t('No provider funding records')}
          />
        )}
        {hasWalletData && (
          <div className='space-y-6'>
            {wallets.length > 0 && (
              <Card className='border-primary/30 bg-primary/5 shadow-sm'>
                <CardHeader className='border-b'>
                  <CardTitle className='text-primary text-xl'>
                    {t('Provider total wallet')}
                  </CardTitle>
                </CardHeader>
                <CardContent className='grid gap-4 pt-1 sm:grid-cols-3'>
                  {wallets.map((wallet) => (
                    <div key={wallet.currency} className='space-y-1'>
                      <p className='text-muted-foreground text-xs font-medium tracking-wide uppercase'>
                        {wallet.currency}
                      </p>
                      <div>
                        <p className='text-muted-foreground text-sm'>
                          {t('Total funded')}
                        </p>
                        <p className='text-2xl font-bold tabular-nums'>
                          {formatNumber(wallet.amount_minor / 100)}{' '}
                          {wallet.currency}
                        </p>
                      </div>
                      <div className='border-border/60 border-t pt-2'>
                        <p className='text-muted-foreground text-sm'>
                          {t('Available')}
                        </p>
                        <p className='text-2xl font-bold tabular-nums'>
                          {formatNumber(wallet.available_minor / 100)}{' '}
                          {wallet.currency}
                        </p>
                      </div>
                    </div>
                  ))}
                </CardContent>
              </Card>
            )}
            {providerChannels.length > 0 && (
              <section className='space-y-3'>
                <div>
                  <h3 className='text-lg font-semibold'>{t('Channels')}</h3>
                </div>
                <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3'>
                  {providerChannels.map((channel) => {
                    const funding = channelFunding.get(channel.id) ?? []
                    return (
                      <Card key={channel.id} size='sm' className='h-full'>
                        <CardHeader className='border-b'>
                          <CardTitle>
                            {channel.name || t('Unknown')}
                          </CardTitle>
                          <p className='text-muted-foreground text-sm'>
                            {t('Channel')} #{channel.id}
                          </p>
                        </CardHeader>
                        <CardContent className='grid gap-4 sm:grid-cols-2'>
                          {funding.length > 0 && (
                            <div>
                              <p className='text-muted-foreground text-sm'>
                                {t('Total funded')}
                              </p>
                              <div className='space-y-1'>
                                {funding.map((wallet) => (
                                  <p
                                    key={`${wallet.channel_id}-${wallet.currency}`}
                                    className='text-lg font-semibold tabular-nums'
                                  >
                                    {formatNumber(wallet.amount_minor / 100)}{' '}
                                    {wallet.currency}
                                  </p>
                                ))}
                              </div>
                            </div>
                          )}
                        </CardContent>
                      </Card>
                    )
                  })}
                </div>
              </section>
            )}
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

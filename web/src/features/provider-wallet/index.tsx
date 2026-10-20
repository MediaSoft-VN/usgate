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
  allocateProviderFunding,
  createProviderFunding,
  getProviderFundingSyncStatus,
  getProviderFundings,
  startProviderFundingSync,
} from '../provider-funding/api'
import { getUsers } from '../users/api'
import type { User } from '../users/types'

export function ProviderWallet() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [channelID, setChannelID] = useState('')
  const [reference, setReference] = useState('')
  const [amountMinor, setAmountMinor] = useState('')
  const [fundingOpen, setFundingOpen] = useState(false)
  const [allocationOpen, setAllocationOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [allocationReference, setAllocationReference] = useState('')
  const [allocationAmount, setAllocationAmount] = useState('')
  const [currency, setCurrency] = useState('USD')
  const syncQuery = useQuery({
    queryKey: ['provider-funding-sync-status'],
    queryFn: getProviderFundingSyncStatus,
    refetchInterval: (query) => (query.state.data?.running ? 2000 : false),
  })
  const syncMutation = useMutation({
    mutationFn: startProviderFundingSync,
    onSuccess: () => {
      toast.success(t('Provider credit sync started'))
      void syncQuery.refetch()
    },
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
  const selectedChannel = channelsQuery.data?.find(
    (channel) => String(channel.id) === channelID
  )
  const createMutation = useMutation({
    mutationFn: () =>
      createProviderFunding({
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
      void queryClient.invalidateQueries({ queryKey: ['provider-wallet'] })
      void queryClient.invalidateQueries({ queryKey: ['provider-funding'] })
    },
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
  const hasWalletData = wallets.length > 0 || providerWallets.length > 0

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        <span className='inline-flex items-center gap-2'>
          {t('Provider total wallet')}
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
          <DialogTrigger render={<Button />}>
            {t('Record credit')}
          </DialogTrigger>
          <DialogContent className='sm:max-w-2xl'>
            <DialogHeader>
              <DialogTitle>{t('Record credit')}</DialogTitle>
            </DialogHeader>
            <form
              className='grid gap-3 md:grid-cols-2'
              onSubmit={(event) => {
                event.preventDefault()
                const amount = Number(amountMinor)
                if (
                  !selectedChannel ||
                  !reference.trim() ||
                  !Number.isFinite(amount) ||
                  amount <= 0 ||
                  !Number.isSafeInteger(Math.round(amount * 100)) ||
                  !currency.trim()
                ) {
                  return
                }
                createMutation.mutate()
              }}
            >
              <div className='grid gap-2'>
                <Label htmlFor='provider-funding-channel' required>
                  {t('Channel')}
                </Label>
                <Select
                  value={channelID}
                  itemToStringValue={(value) =>
                    channelsQuery.data?.find(
                      (channel) => String(channel.id) === value
                    )?.name ?? ''
                  }
                  onValueChange={(value) => setChannelID(value ?? '')}
                >
                  <SelectTrigger
                    id='provider-funding-channel'
                    className='w-full'
                  >
                    <SelectValue placeholder={t('Select channel')}>
                      {selectedChannel?.name || selectedChannel?.type}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {(channelsQuery.data ?? []).map((channel) => (
                      <SelectItem key={channel.id} value={String(channel.id)}>
                        {channel.name || channel.type}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='provider-funding-reference' required>
                  {t('Reference')}
                </Label>
                <Input
                  id='provider-funding-reference'
                  value={reference}
                  onChange={(event) => setReference(event.target.value)}
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='provider-funding-amount' required>
                  {t('Amount')}
                </Label>
                <Input
                  id='provider-funding-amount'
                  type='number'
                  min={1}
                  value={amountMinor}
                  onChange={(event) => setAmountMinor(event.target.value)}
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='provider-funding-currency' required>
                  {t('Currency')}
                </Label>
                <Input
                  id='provider-funding-currency'
                  value={currency}
                  onChange={(event) =>
                    setCurrency(event.target.value.toUpperCase())
                  }
                />
              </div>
              <p className='text-muted-foreground text-sm italic md:col-span-2'>
                {t('Provider funding guide')}
              </p>
              <DialogFooter className='md:col-span-2'>
                <Button type='submit' disabled={createMutation.isPending}>
                  {t('Record credit')}
                </Button>
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
            ? t('Provider credit sync running')
            : t('Sync credit')}
        </Button>
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
                <CardContent className='grid gap-4 pt-4 sm:grid-cols-3'>
                  {wallets.map((wallet) => (
                    <div key={wallet.currency} className='space-y-1'>
                      <p className='text-muted-foreground text-xs font-medium tracking-wide uppercase'>
                        {wallet.currency}
                      </p>
                      <div>
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
            {providerWallets.length > 0 && (
              <section className='space-y-3'>
                <div>
                  <h3 className='text-lg font-semibold'>{t('Channels')}</h3>
                </div>
                <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3'>
                  {providerWallets.map((wallet) => {
                    return (
                      <Card
                        key={`${wallet.channel_id}-${wallet.currency}`}
                        size='sm'
                        className='h-full'
                      >
                        <CardHeader className='border-b'>
                          <CardTitle>
                            {wallet.channel_name || t('Unknown')}
                          </CardTitle>
                          <p className='text-muted-foreground text-sm'>
                            {t('Channel')} #{wallet.channel_id} ·{' '}
                            {wallet.currency}
                          </p>
                        </CardHeader>
                        <CardContent>
                          <p className='text-muted-foreground text-sm'>
                            {t('Total funded')}
                          </p>
                          <p className='text-2xl font-semibold tabular-nums'>
                            {formatNumber(wallet.amount_minor / 100)}{' '}
                            {wallet.currency}
                          </p>
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

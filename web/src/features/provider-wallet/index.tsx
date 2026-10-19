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
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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
import { formatNumber } from '@/lib/format'

import { getChannels } from '../channels/api'
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
  const channelsQuery = useQuery({
    queryKey: ['provider-wallet-channels'],
    queryFn: async () =>
      (await getChannels({ p: 1, page_size: 100, sort_by: 'id', sort_order: 'desc' })).data
        ?.items ?? [],
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
  const channelNames = new Map(
    (channelsQuery.data ?? []).map((channel) => [channel.id, channel.name || channel.type])
  )

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
        {!walletQuery.isLoading && wallets.length === 0 && (
          <EmptyState
            icon={WalletCards}
            title={t('No provider funding records')}
          />
        )}
        {wallets.length > 0 && (
          <div className='space-y-6'>
            {providerWallets.length > 0 && (
              <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3'>
                {providerWallets.map((wallet) => (
                  <Card key={`${wallet.channel_id}-${wallet.currency}`}>
                    <CardHeader>
                      <CardTitle>
                        {channelNames.get(wallet.channel_id) ?? t('Unknown')}
                      </CardTitle>
                      <p className='text-muted-foreground text-sm'>
                        {t('Currency')}: {wallet.currency}
                      </p>
                    </CardHeader>
                    <CardContent>
                      <p className='text-muted-foreground text-sm'>{t('Total funded')}</p>
                      <p className='text-2xl font-semibold tabular-nums'>
                        {formatNumber(wallet.amount_minor / 100)} {wallet.currency}
                      </p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
            <div className='bg-card overflow-hidden rounded-lg border'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Currency')}</TableHead>
                    <TableHead>{t('Total funded')}</TableHead>
                    <TableHead>{t('Credit allocation')}</TableHead>
                    <TableHead>{t('Available')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {wallets.map((wallet) => (
                    <TableRow key={wallet.currency}>
                      <TableCell>{wallet.currency}</TableCell>
                      <TableCell className='tabular-nums'>
                        {formatNumber(wallet.amount_minor / 100)}
                      </TableCell>
                      <TableCell className='tabular-nums'>
                        {formatNumber(wallet.allocated_minor / 100)}
                      </TableCell>
                      <TableCell className='tabular-nums'>
                        {formatNumber(wallet.available_minor / 100)}
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

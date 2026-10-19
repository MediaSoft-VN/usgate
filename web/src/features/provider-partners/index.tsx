import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { createServerError } from '@/lib/server-error-message'

import { getChannels } from '../channels/api'
import {
  assignChannelProviderPartner,
  createProviderPartner,
  getProviderPartners,
} from '../provider-funding/api'

export function ProviderPartners() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [partnerCode, setPartnerCode] = useState('')
  const [partnerName, setPartnerName] = useState('')
  const [providerPartnerID, setProviderPartnerID] = useState('')
  const [channelID, setChannelID] = useState('')
  const partnersQuery = useQuery({ queryKey: ['provider-partners'], queryFn: getProviderPartners })
  const channelsQuery = useQuery({
    queryKey: ['provider-partner-channels'],
    queryFn: async () => {
      const response = await getChannels({ p: 1, page_size: 100, sort_by: 'id', sort_order: 'desc' })
      if (!response.success) throw createServerError(response, t('Failed to load'))
      return response.data?.items ?? []
    },
  })
  const createMutation = useMutation({
    mutationFn: () => createProviderPartner({ code: partnerCode.trim(), name: partnerName.trim() }),
    onSuccess: (partner) => {
      toast.success(t('Provider partner created'))
      setPartnerCode('')
      setPartnerName('')
      setProviderPartnerID(String(partner.id))
      void queryClient.invalidateQueries({ queryKey: ['provider-partners'] })
    },
  })
  const assignMutation = useMutation({
    mutationFn: () => assignChannelProviderPartner(Number(channelID), Number(providerPartnerID)),
    onSuccess: () => {
      toast.success(t('Provider channel assigned'))
      setChannelID('')
      void channelsQuery.refetch()
    },
  })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Provider partners')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-4'>
          <section className='bg-card grid gap-3 rounded-lg border p-4 md:grid-cols-4'>
            <div className='md:col-span-2'>
              <div className='text-sm font-medium'>{t('Add provider partner')}</div>
              <div className='text-muted-foreground text-sm'>{t('Provider partner guide')}</div>
            </div>
            <Input
              aria-label={t('Provider partner code')}
              placeholder={t('Provider partner code')}
              value={partnerCode}
              onChange={(event) => setPartnerCode(event.target.value)}
            />
            <div className='flex gap-2'>
              <Input
                aria-label={t('Provider partner name')}
                placeholder={t('Provider partner name')}
                value={partnerName}
                onChange={(event) => setPartnerName(event.target.value)}
              />
              <Button
                disabled={createMutation.isPending || !partnerCode.trim() || !partnerName.trim()}
                onClick={() => createMutation.mutate()}
              >
                {t('Add')}
              </Button>
            </div>
          </section>

          <section className='bg-card grid gap-3 rounded-lg border p-4 md:grid-cols-3'>
            <div>
              <div className='text-sm font-medium'>{t('Assign channels to provider partner')}</div>
              <div className='text-muted-foreground text-sm'>{t('Provider channel assignment guide')}</div>
            </div>
            <Select value={providerPartnerID} onValueChange={(value) => setProviderPartnerID(value ?? '')}>
              <SelectTrigger className='w-full'>
                <SelectValue placeholder={t('Select provider partner')} />
              </SelectTrigger>
              <SelectContent>
                {(partnersQuery.data ?? []).map((partner) => (
                  <SelectItem key={partner.id} value={String(partner.id)}>{partner.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <div className='flex gap-2'>
              <Select value={channelID} onValueChange={(value) => setChannelID(value ?? '')}>
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('Select channel')} />
                </SelectTrigger>
                <SelectContent>
                  {(channelsQuery.data ?? []).map((channel) => (
                    <SelectItem key={channel.id} value={String(channel.id)}>{channel.name || channel.type}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button disabled={assignMutation.isPending || !channelID || !providerPartnerID} onClick={() => assignMutation.mutate()}>
                {t('Assign channel')}
              </Button>
            </div>
          </section>

          <section className='bg-card rounded-lg border p-4'>
            <h2 className='mb-3 text-sm font-medium'>{t('Provider partners')}</h2>
            <div className='grid gap-2'>
              {(partnersQuery.data ?? []).map((partner) => (
                <div className='flex items-center justify-between border-b py-2 last:border-0' key={partner.id}>
                  <span>{partner.name}</span>
                  <span className='text-muted-foreground text-sm'>{partner.code}</span>
                </div>
              ))}
            </div>
          </section>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

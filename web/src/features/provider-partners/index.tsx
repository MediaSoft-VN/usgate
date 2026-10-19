import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
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
  const [partnerName, setPartnerName] = useState('')
  const [imageURL, setImageURL] = useState('')
  const [websiteURL, setWebsiteURL] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [apiEndpoint, setApiEndpoint] = useState('')
  const [providerPartnerID, setProviderPartnerID] = useState('')
  const [channelID, setChannelID] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [assignOpen, setAssignOpen] = useState(false)
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
    mutationFn: () => createProviderPartner({
      name: partnerName.trim(),
      image_url: imageURL.trim(),
      website_url: websiteURL.trim(),
      api_key: apiKey,
      api_endpoint: apiEndpoint.trim(),
    }),
    onSuccess: (partner) => {
      toast.success(t('Provider partner created'))
      setPartnerName('')
      setImageURL('')
      setWebsiteURL('')
      setApiKey('')
      setApiEndpoint('')
      setProviderPartnerID(String(partner.id))
      setCreateOpen(false)
      void queryClient.invalidateQueries({ queryKey: ['provider-partners'] })
    },
  })
  const assignMutation = useMutation({
    mutationFn: () => assignChannelProviderPartner(Number(channelID), Number(providerPartnerID)),
    onSuccess: () => {
      toast.success(t('Provider channel assigned'))
      setChannelID('')
      setAssignOpen(false)
      void channelsQuery.refetch()
    },
  })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Provider partners')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-4'>
          <div className='flex flex-wrap gap-2'>
            <Dialog open={createOpen} onOpenChange={setCreateOpen}>
              <DialogTrigger render={<Button />}>{t('Add provider partner')}</DialogTrigger>
              <DialogContent className='sm:max-w-2xl'>
                <DialogHeader>
                  <DialogTitle>{t('Add provider partner')}</DialogTitle>
                  <DialogDescription>{t('Provider partner guide')}</DialogDescription>
                </DialogHeader>
                <div className='grid gap-3'>
                  <Input aria-label={t('Provider partner name')} placeholder={t('Provider partner name')} value={partnerName} onChange={(event) => setPartnerName(event.target.value)} />
                  <Input aria-label={t('Partner website')} placeholder={t('Partner website')} type='url' value={websiteURL} onChange={(event) => setWebsiteURL(event.target.value)} />
                  <Input aria-label={t('Partner image')} accept='image/png,image/jpeg' type='file' onChange={(event) => {
                    const file = event.target.files?.[0]
                    if (!file) return
                    if (file.size > 384 * 1024) {
                      toast.error(t('Provider image is too large'))
                      event.target.value = ''
                      return
                    }
                    const reader = new FileReader()
                    reader.addEventListener('load', () => setImageURL(typeof reader.result === 'string' ? reader.result : ''))
                    reader.readAsDataURL(file)
                  }} />
                  <Input aria-label={t('API endpoint')} placeholder={t('API endpoint')} type='url' value={apiEndpoint} onChange={(event) => setApiEndpoint(event.target.value)} />
                  <Input aria-label={t('API key')} placeholder={t('API key')} type='password' value={apiKey} onChange={(event) => setApiKey(event.target.value)} />
                </div>
                <DialogFooter>
                  <Button disabled={createMutation.isPending || !partnerName.trim() || !apiKey.trim() || !apiEndpoint.trim()} onClick={() => createMutation.mutate()}>
                    {t('Add')}
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
            <Dialog open={assignOpen} onOpenChange={setAssignOpen}>
              <DialogTrigger render={<Button variant='outline' />}>{t('Assign channels to provider partner')}</DialogTrigger>
              <DialogContent className='sm:max-w-2xl'>
                <DialogHeader>
                  <DialogTitle>{t('Assign channels to provider partner')}</DialogTitle>
                  <DialogDescription>{t('Provider channel assignment guide')}</DialogDescription>
                </DialogHeader>
                <div className='grid gap-3'>
                  <Select value={providerPartnerID} onValueChange={(value) => setProviderPartnerID(value ?? '')}>
                    <SelectTrigger className='w-full'><SelectValue placeholder={t('Select provider partner')} /></SelectTrigger>
                    <SelectContent>{(partnersQuery.data ?? []).map((partner) => <SelectItem key={partner.id} value={String(partner.id)}>{partner.name}</SelectItem>)}</SelectContent>
                  </Select>
                  <Select value={channelID} onValueChange={(value) => setChannelID(value ?? '')}>
                    <SelectTrigger className='w-full'><SelectValue placeholder={t('Select channel')} /></SelectTrigger>
                    <SelectContent>{(channelsQuery.data ?? []).map((channel) => <SelectItem key={channel.id} value={String(channel.id)}>{channel.name || channel.type}</SelectItem>)}</SelectContent>
                  </Select>
                </div>
                <DialogFooter>
                  <Button disabled={assignMutation.isPending || !channelID || !providerPartnerID} onClick={() => assignMutation.mutate()}>
                    {t('Assign channel')}
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <section className='bg-card rounded-lg border p-4'>
            <h2 className='mb-3 text-sm font-medium'>{t('Provider partners')}</h2>
            <div className='grid gap-2'>
              {(partnersQuery.data ?? []).map((partner) => (
                <div className='flex items-center justify-between border-b py-2 last:border-0' key={partner.id}>
                  <span>{partner.name}</span>
                  <span className='text-muted-foreground text-sm'>
                    {partner.code} · {partner.api_key_configured ? t('API key configured') : t('API key not configured')}
                  </span>
                </div>
              ))}
            </div>
          </section>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

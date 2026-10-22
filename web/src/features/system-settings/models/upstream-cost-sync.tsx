import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Check, RefreshCcw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { handleServerError } from '@/lib/handle-server-error'
import { requireServerSuccess } from '@/lib/server-error-message'

import {
  applyUpstreamCosts,
  getUpstreamCostChannels,
  previewUpstreamCosts,
} from '../api'
import type {
  ChannelModelCost,
  UpstreamCostChannel,
  UpstreamCostResult,
} from '../types'

const microsToUsd = (value: number) => value / 1_000_000

function costLabel(cost: ChannelModelCost) {
  return `${formatBillingCurrencyFromUSD(microsToUsd(cost.input_usd_per_million_micros))} / ${formatBillingCurrencyFromUSD(microsToUsd(cost.output_usd_per_million_micros))}`
}

function ResultStatus(props: { result?: UpstreamCostResult; channel: UpstreamCostChannel }) {
  const { t } = useTranslation()
  if (!props.channel.supported) {
    return <StatusBadge label={t('Manual')} variant='warning' size='sm' copyable={false} />
  }
  if (!props.result) {
    return <StatusBadge label={t('Not previewed')} variant='neutral' size='sm' copyable={false} />
  }
  if (props.result.error) {
    return <StatusBadge label={t('Failed')} variant='danger' size='sm' copyable={false} />
  }
  if (props.result.version !== props.channel.version) {
    return <StatusBadge label={t('Stale preview')} variant='warning' size='sm' copyable={false} />
  }
  return <StatusBadge label={t('Ready to apply')} variant='success' size='sm' copyable={false} />
}

export function UpstreamCostSync() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selected, setSelected] = useState<number[]>([])
  const [results, setResults] = useState<UpstreamCostResult[]>([])
  const [conflict, setConflict] = useState(false)
  const channelsQuery = useQuery({
    queryKey: ['upstream-cost-channels'],
    queryFn: async () => requireServerSuccess(await getUpstreamCostChannels()),
  })
  const channels = channelsQuery.data?.data ?? []
  const resultByChannel = useMemo(
    () => new Map(results.map((result) => [result.channel_id, result])),
    [results]
  )
  const preview = useMutation({
    mutationFn: () => previewUpstreamCosts(selected),
    onSuccess: (response) => {
      if (!response.success) {
        setConflict(true)
        return
      }
      setConflict(false)
      setResults(response.data.results)
      toast.success(t('Upstream cost preview completed'))
    },
    onError: (error) => handleServerError(error, t('Failed to preview upstream costs')),
  })
  const apply = useMutation({
    mutationFn: () => {
      const items = results
        .filter((result) => result.supported && !result.error && result.costs)
        .map((result) => ({
          channel_id: result.channel_id,
          version: result.version,
          costs: result.costs ?? [],
        }))
      return (async () => {
        let lastResponse: Awaited<ReturnType<typeof applyUpstreamCosts>> | undefined
        for (const item of items) {
          // Keep each request below the server body limit. A preview may contain
          // thousands of models across several channels, which otherwise makes
          // one combined apply request fail with a generic format error.
          lastResponse = await applyUpstreamCosts([item])
          if (!lastResponse.success) return lastResponse
        }
        return lastResponse ?? { success: true, message: '', data: { updated_channels: [] } }
      })()
    },
    onSuccess: async (response) => {
      if (!response.success) {
        setConflict(true)
        return
      }
      setConflict(false)
      toast.success(t('Upstream costs applied successfully'))
      setResults([])
      await queryClient.invalidateQueries({ queryKey: ['upstream-cost-channels'] })
    },
    onError: (error) => handleServerError(error, t('Failed to apply upstream costs')),
  })
  if (channelsQuery.isPending) return <div role='status'>{t('Loading...')}</div>
  if (channelsQuery.isError) {
    return <ErrorState title={t('Failed to load settings')} onRetry={() => void channelsQuery.refetch()} />
  }
  const selectable = channels.filter((channel) => channel.supported).map((channel) => channel.id)
  const allSelected = selectable.length > 0 && selectable.every((id) => selected.includes(id))
  const toggleAll = () => setSelected(allSelected ? [] : selectable)
  const canApply = results.some(
    (result) => result.supported && !result.error && result.version === channels.find((channel) => channel.id === result.channel_id)?.version
  )
  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div>
          <p className='text-muted-foreground text-sm'>{t('Provider costs are separate from customer sale pricing.')}</p>
          <p className='text-muted-foreground text-xs'>{t('Preview changes before applying them to channel cost data.')}</p>
        </div>
        <div className='flex gap-2'>
          <Button variant='outline' onClick={() => void channelsQuery.refetch()} disabled={preview.isPending || apply.isPending}>
            <RefreshCcw className='size-4' aria-hidden /> {t('Refresh')}
          </Button>
          <Button variant='outline' onClick={() => preview.mutate()} disabled={!selected.length || preview.isPending || apply.isPending}>
            <RefreshCcw className='size-4' aria-hidden /> {t('Preview costs')}
          </Button>
          <Button onClick={() => apply.mutate()} disabled={!canApply || preview.isPending || apply.isPending}>
            <Check className='size-4' aria-hidden /> {t('Apply costs')}
          </Button>
        </div>
      </div>
      {conflict && (
        <div role='alert' className='border-destructive/50 bg-destructive/10 text-destructive rounded-md border p-3 text-sm'>
          {t('The cost data changed while you were previewing. Refresh and preview again.')}
        </div>
      )}
      <Card>
        <CardHeader className='flex-row items-center justify-between space-y-0'>
          <CardTitle className='text-base'>{t('Upstream cost channels')}</CardTitle>
          <Checkbox checked={allSelected} onCheckedChange={toggleAll} aria-label={t('Select all supported channels')} />
        </CardHeader>
        <CardContent className='space-y-3'>
          {channels.map((channel) => {
            const result = resultByChannel.get(channel.id)
            const firstCost = result?.costs?.[0] ?? channel.costs[0]
            const isSelected = selected.includes(channel.id)
            return (
              <div key={channel.id} className='flex flex-wrap items-center gap-3 rounded-lg border p-3'>
                <Checkbox
                  checked={isSelected}
                  disabled={!channel.supported}
                  onCheckedChange={(checked) => setSelected((previous) => checked ? [...previous, channel.id] : previous.filter((id) => id !== channel.id))}
                  aria-label={t('Select {{name}}', { name: channel.name })}
                />
                <div className='min-w-40 flex-1'>
                  <div className='flex flex-wrap items-center gap-2 font-medium'>
                    {channel.name}
                    <StatusBadge label={channel.type_name} variant='neutral' size='sm' copyable={false} />
                  </div>
                  <div className='text-muted-foreground text-xs'>
                    {firstCost ? `${firstCost.model}: ${costLabel(firstCost)} / M` : t('No stored cost data')}
                  </div>
                  {result?.source && <div className='text-muted-foreground text-xs'>{t('Source')}: {result.source}</div>}
                </div>
                <ResultStatus result={result} channel={channel} />
                {!channel.supported && <span className='text-muted-foreground flex items-center gap-1 text-xs'><AlertTriangle className='size-3' aria-hidden />{t('Ollama pricing is manual')}</span>}
                {result?.error && <span role='alert' className='text-destructive max-w-md text-xs'>{result.error}</span>}
              </div>
            )
          })}
        </CardContent>
      </Card>
    </div>
  )
}

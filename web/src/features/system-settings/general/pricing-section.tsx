/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { handleServerError } from '@/lib/handle-server-error'
import { DEFAULT_CURRENCY_CONFIG } from '@/stores/system-config-store'

import { getExchangeRates, syncExchangeRates } from '../api'
import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

function isValidManualRates(value: string): boolean {
  try {
    const parsed: unknown = JSON.parse(value || '{}')
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return false
    }
    return Object.values(parsed as Record<string, unknown>).every((rate) => {
      const numeric = typeof rate === 'number' ? rate : Number(rate)
      return Number.isFinite(numeric) && numeric > 0
    })
  } catch {
    return false
  }
}

const createPricingSchema = (t: (key: string) => string) =>
  z
    .object({
      QuotaPerUnit: z.coerce.number().min(0, t('Value must be at least 0')),
      USDExchangeRate: z.coerce
        .number()
        .min(0.0001, t('Exchange rate must be greater than 0')),
      DisplayInCurrencyEnabled: z.boolean(),
      DisplayTokenStatEnabled: z.boolean(),
      general_setting: z.object({
        quota_display_type: z.enum(['USD', 'CNY', 'TOKENS', 'CUSTOM']),
        custom_currency_symbol: z.string().max(8).optional(),
        custom_currency_code: z.string().max(8).optional(),
        custom_currency_exchange_rate: z.coerce
          .number()
          .min(0.0001, t('Exchange rate must be greater than 0'))
          .optional(),
        exchange_rate_source: z.enum(['vietcombank', 'manual']),
        exchange_rate_type: z.enum(['cash', 'transfer', 'sell']),
        exchange_rate_sync_time: z.string().regex(/^([01]\d|2[0-3]):[0-5]\d$/),
        manual_exchange_rates: z.string(),
      }),
    })
    .superRefine((data, ctx) => {
      const displayType = data.general_setting.quota_display_type

      if (displayType === 'CUSTOM') {
        if (!data.general_setting.custom_currency_symbol?.trim()) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['general_setting', 'custom_currency_symbol'],
            message: t('Custom currency symbol is required'),
          })
        }

        if (!data.general_setting.custom_currency_code?.trim()) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['general_setting', 'custom_currency_code'],
            message: t('Custom currency code is required'),
          })
        }

        if (data.general_setting.custom_currency_exchange_rate == null) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['general_setting', 'custom_currency_exchange_rate'],
            message: t('Exchange rate is required'),
          })
        }
      }

      if (
        data.general_setting.exchange_rate_source === 'manual' &&
        !isValidManualRates(data.general_setting.manual_exchange_rates)
      ) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['general_setting', 'manual_exchange_rates'],
          message: t('Enter valid JSON rates, for example {"USD":"26100"}'),
        })
      }
    })

type PricingFormValues = z.infer<ReturnType<typeof createPricingSchema>>

type PricingSectionProps = {
  defaultValues: PricingFormValues
}

export function PricingSection({ defaultValues }: PricingSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const queryClient = useQueryClient()
  const [syncingRates, setSyncingRates] = useState(false)
  const ratesQuery = useQuery({
    queryKey: ['exchange-rates'],
    queryFn: getExchangeRates,
  })
  const rateRows = ratesQuery.data?.data?.rates ?? []

  let rateTableContent: ReactNode
  if (ratesQuery.isLoading) {
    rateTableContent = (
      <tr>
        <td colSpan={3} className='text-muted-foreground px-3 py-4 text-center'>
          {t('Loading')}
        </td>
      </tr>
    )
  } else if (rateRows.length === 0) {
    rateTableContent = (
      <tr>
        <td colSpan={3} className='text-muted-foreground px-3 py-4 text-center'>
          {t('No exchange rates yet')}
        </td>
      </tr>
    )
  } else {
    rateTableContent = rateRows.map((row) => (
      <tr key={row.quote_currency} className='border-b last:border-0'>
        <td className='px-3 py-2 font-medium'>{row.quote_currency}</td>
        <td className='px-3 py-2'>{row.rate}</td>
        <td className='px-3 py-2'>{row.effective_date || '—'}</td>
      </tr>
    ))
  }

  const pricingSchema = createPricingSchema(t)

  const { form, handleSubmit, handleReset, isDirty, isSubmitting } =
    useSettingsForm<PricingFormValues>({
      resolver: zodResolver(pricingSchema) as Resolver<
        PricingFormValues,
        unknown,
        PricingFormValues
      >,
      defaultValues,
      onSubmit: async (_data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          if (value === undefined || value === null) continue
          if (typeof value === 'object') continue

          let serialized: string | boolean = value as string | boolean

          if (typeof value === 'boolean') {
            serialized = String(value)
          } else if (typeof value === 'number') {
            serialized = Number.isFinite(value) ? String(value) : '0'
          }

          await updateOption.mutateAsync({
            key,
            value: serialized,
          })
        }
      },
    })

  const displayType = form.watch('general_setting.quota_display_type') ?? 'USD'
  const exchangeRateSource =
    form.watch('general_setting.exchange_rate_source') ?? 'vietcombank'
  const automaticRateSyncEnabled = exchangeRateSource === 'vietcombank'
  const displayInCurrencyEnabled = form.watch('DisplayInCurrencyEnabled')

  const handleSyncRates = async () => {
    setSyncingRates(true)
    try {
      await syncExchangeRates()
      await queryClient.invalidateQueries({ queryKey: ['exchange-rates'] })
      toast.success(t('Exchange rates updated successfully'))
    } catch (error) {
      handleServerError(error, t('Failed to update exchange rates'))
    } finally {
      setSyncingRates(false)
    }
  }
  const showTokensOnlyOption = displayType === 'TOKENS'
  const showQuotaPerUnit =
    displayType === 'TOKENS' ||
    defaultValues.QuotaPerUnit !== DEFAULT_CURRENCY_CONFIG.quotaPerUnit
  const showDisplayInCurrencyOption = displayInCurrencyEnabled === false

  return (
    <>
      <FormNavigationGuard when={isDirty} />

      <SettingsSection title={t('Pricing & Display')}>
        <Form {...form}>
          <SettingsForm onSubmit={handleSubmit}>
            <SettingsPageFormActions
              onSave={handleSubmit}
              onReset={handleReset}
              isSaving={updateOption.isPending || isSubmitting}
              isResetDisabled={!isDirty}
            />
            <FormDirtyIndicator isDirty={isDirty} />
            {showQuotaPerUnit && (
              <FormField
                control={form.control}
                name='QuotaPerUnit'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Quota Per Unit')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step='0.01'
                        value={field.value as number}
                        disabled
                        name={field.name}
                        onBlur={field.onBlur}
                        ref={field.ref}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Number of tokens per unit quota')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            <FormField
              control={form.control}
              name='general_setting.quota_display_type'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Display Mode')}</FormLabel>
                  <Select
                    items={[
                      { value: 'USD', label: t('USD') },
                      { value: 'CNY', label: t('CNY') },
                      { value: 'CUSTOM', label: t('Custom Currency') },
                      { value: 'TOKENS', label: t('Tokens Only') },
                    ]}
                    value={field.value}
                    onValueChange={field.onChange}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('Select display mode')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        <SelectItem value='USD'>{t('USD')}</SelectItem>
                        <SelectItem value='CNY'>{t('CNY')}</SelectItem>
                        <SelectItem value='CUSTOM'>
                          {t('Custom Currency')}
                        </SelectItem>
                        {showTokensOnlyOption && (
                          <SelectItem value='TOKENS'>
                            {t('Tokens Only')}
                          </SelectItem>
                        )}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t('Choose how quota values are shown to users')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {displayType !== 'TOKENS' && (
              <FormField
                control={form.control}
                name='USDExchangeRate'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {displayType === 'CNY'
                        ? t('CNY per USD')
                        : t('USD Exchange Rate')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step='0.01'
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Real exchange rate between USD and your payment gateway currency'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            {displayType === 'CUSTOM' && (
              <div className='grid gap-4 sm:grid-cols-3'>
                <FormField
                  control={form.control}
                  name='general_setting.custom_currency_code'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Custom Currency Code')}</FormLabel>
                      <FormControl>
                        <Input
                          type='text'
                          value={field.value ?? ''}
                          onChange={(e) =>
                            field.onChange(e.target.value.toUpperCase())
                          }
                          name={field.name}
                          onBlur={field.onBlur}
                          ref={field.ref}
                          maxLength={8}
                          placeholder={t('e.g. VND')}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('ISO code used by financial wallets')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='general_setting.custom_currency_symbol'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Custom Currency Symbol')}</FormLabel>
                      <FormControl>
                        <Input
                          type='text'
                          value={field.value ?? ''}
                          onChange={field.onChange}
                          name={field.name}
                          onBlur={field.onBlur}
                          ref={field.ref}
                          maxLength={8}
                          placeholder={t('e.g. ¥ or HK$')}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Prefix used when displaying prices')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='general_setting.custom_currency_exchange_rate'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Units per USD')}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          step='0.01'
                          value={field.value ?? ''}
                          onChange={(e) =>
                            field.onChange(
                              e.target.value === ''
                                ? undefined
                                : e.target.valueAsNumber
                            )
                          }
                          name={field.name}
                          onBlur={field.onBlur}
                          ref={field.ref}
                          placeholder={t('e.g. 8 means 1 USD = 8 units')}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Conversion rate from USD to your custom currency')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            )}

            {showDisplayInCurrencyOption && (
              <FormField
                control={form.control}
                name='DisplayInCurrencyEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Display in Currency')}</FormLabel>
                      <FormDescription>
                        {displayType === 'TOKENS'
                          ? t(
                              'Tokens-only mode will show raw quota values regardless of this toggle.'
                            )
                          : t('Show prices in currency instead of quota.')}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />
            )}

            <div className='space-y-4 rounded-lg border p-4'>
              <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                <div>
                  <h3 className='text-sm font-medium'>
                    {t('VND Exchange Rates')}
                  </h3>
                  <p className='text-muted-foreground text-sm'>
                    {t(
                      'Vietcombank daily rates or manually entered rates for SePay settlement'
                    )}
                  </p>
                </div>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={syncingRates}
                  onClick={handleSyncRates}
                >
                  {syncingRates ? t('Syncing...') : t('Sync Vietcombank rates')}
                </Button>
              </div>
              <div className='grid gap-4 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='general_setting.exchange_rate_source'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Rate Source')}</FormLabel>
                      <Select
                        items={[
                          {
                            value: 'vietcombank',
                            label: t('Vietcombank'),
                          },
                          { value: 'manual', label: t('Manual Rates') },
                        ]}
                        value={field.value}
                        onValueChange={field.onChange}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue
                              placeholder={t('Select rate source')}
                            />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            <SelectItem value='vietcombank'>
                              {t('Vietcombank')}
                            </SelectItem>
                            <SelectItem value='manual'>
                              {t('Manual Rates')}
                            </SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormDescription>
                        {t(
                          'Payments snapshot the active rate when an order is created'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='general_setting.exchange_rate_type'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Bank Rate Type')}</FormLabel>
                      <Select
                        items={[
                          { value: 'cash', label: t('Cash') },
                          { value: 'transfer', label: t('Transfer') },
                          { value: 'sell', label: t('Sell') },
                        ]}
                        value={field.value}
                        onValueChange={field.onChange}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder={t('Select rate type')} />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            <SelectItem value='cash'>{t('Cash')}</SelectItem>
                            <SelectItem value='transfer'>
                              {t('Transfer')}
                            </SelectItem>
                            <SelectItem value='sell'>{t('Sell')}</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormDescription>
                        {t('Sell is recommended for customer payments')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name='general_setting.exchange_rate_sync_time'
                render={({ field }) => (
                  <FormItem className='max-w-sm'>
                    <FormLabel>{t('Daily sync time')}</FormLabel>
                    <FormControl>
                      <Input
                        type='time'
                        step={60}
                        value={field.value || '10:00'}
                        onChange={field.onChange}
                        onBlur={field.onBlur}
                        name={field.name}
                        ref={field.ref}
                        disabled={!automaticRateSyncEnabled}
                      />
                    </FormControl>
                    <FormDescription>
                      {automaticRateSyncEnabled
                        ? t(
                            "Automatic VND rate sync runs at this time in the server's local timezone when Vietcombank is selected."
                          )
                        : t('Automatic sync is unavailable for manual rates.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {exchangeRateSource === 'manual' && (
                <FormField
                  control={form.control}
                  name='general_setting.manual_exchange_rates'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Manual Rates JSON')}</FormLabel>
                      <FormControl>
                        <Input
                          type='text'
                          value={field.value ?? '{}'}
                          onChange={field.onChange}
                          name={field.name}
                          onBlur={field.onBlur}
                          ref={field.ref}
                          placeholder={t(
                            'Enter valid JSON rates, for example {"USD":"26100"}'
                          )}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('VND per one unit of foreign currency')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
              <div>
                <h4 className='mb-2 text-sm font-medium'>
                  {t('Current VND rates')}
                </h4>
                <div className='overflow-x-auto rounded-md border'>
                  <table className='w-full text-sm'>
                    <thead>
                      <tr className='text-muted-foreground border-b text-left'>
                        <th className='px-3 py-2 font-medium'>
                          {t('Currency')}
                        </th>
                        <th className='px-3 py-2 font-medium'>
                          {t('Rate (VND)')}
                        </th>
                        <th className='px-3 py-2 font-medium'>
                          {t('Effective date')}
                        </th>
                      </tr>
                    </thead>
                    <tbody>{rateTableContent}</tbody>
                  </table>
                </div>
              </div>
            </div>

            <FormField
              control={form.control}
              name='DisplayTokenStatEnabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Display Token Statistics')}</FormLabel>
                    <FormDescription>
                      {t('Show token usage statistics in the UI')}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />
          </SettingsForm>
        </Form>
      </SettingsSection>
    </>
  )
}

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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  EmptyState,
  LoadingSkeleton,
  ModelCardGrid,
  ModelDetailsDrawer,
  PricingTable,
  SearchBar,
} from '@/features/pricing/components'
import { VIEW_MODES } from '@/features/pricing/constants'
import { useFilters } from '@/features/pricing/hooks/use-filters'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'

export function ModelCatalog() {
  const { t } = useTranslation()
  const [selectedModelName, setSelectedModelName] = useState<string | null>(
    null
  )
  const {
    models,
    groupRatio,
    usableGroup,
    endpointMap,
    autoGroups,
    isLoading,
    priceRate,
    usdExchangeRate,
  } = usePricingData()
  const {
    searchInput,
    setSearchInput,
    clearSearch,
    clearFilters,
    filteredModels,
    hasActiveFilters,
    groupFilter,
    tokenUnit,
    viewMode,
    showRechargePrice,
  } = useFilters(models)
  const selectedModel = useMemo(
    () => models.find((model) => model.model_name === selectedModelName),
    [models, selectedModelName]
  )
  let modelList = (
    <EmptyState
      searchQuery={searchInput}
      hasActiveFilters={hasActiveFilters}
      onClearFilters={() => {
        clearFilters()
        clearSearch()
      }}
    />
  )

  if (filteredModels.length > 0 && viewMode === VIEW_MODES.CARD) {
    modelList = (
      <ModelCardGrid
        models={filteredModels}
        onModelClick={setSelectedModelName}
        priceRate={priceRate}
        usdExchangeRate={usdExchangeRate}
        tokenUnit={tokenUnit}
        showRechargePrice={showRechargePrice}
        selectedGroup={groupFilter}
      />
    )
  } else if (filteredModels.length > 0) {
    modelList = (
      <PricingTable
        models={filteredModels}
        priceRate={priceRate}
        usdExchangeRate={usdExchangeRate}
        tokenUnit={tokenUnit}
        showRechargePrice={showRechargePrice}
        selectedGroup={groupFilter}
        onModelClick={setSelectedModelName}
      />
    )
  }

  if (isLoading) {
    return <LoadingSkeleton viewMode={viewMode} />
  }

  return (
    <div className='mx-auto w-full max-w-[1800px] space-y-4'>
      <header className='flex flex-wrap items-end justify-between gap-3'>
        <div>
          <h1 className='text-xl font-bold tracking-tight'>
            {t('Model Square')}
          </h1>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('This site currently has {{count}} models enabled', {
              count: models.length,
            })}
          </p>
        </div>
        <SearchBar
          value={searchInput}
          onChange={setSearchInput}
          onClear={clearSearch}
          placeholder={t('Search model name, provider, endpoint, or tag...')}
          className='w-full max-w-md'
        />
      </header>

      {modelList}

      {selectedModel && (
        <ModelDetailsDrawer
          open={Boolean(selectedModel)}
          onOpenChange={(open) => {
            if (!open) setSelectedModelName(null)
          }}
          model={selectedModel}
          groupRatio={groupRatio}
          usableGroup={usableGroup}
          endpointMap={
            endpointMap as Record<string, { path?: string; method?: string }>
          }
          autoGroups={autoGroups}
          priceRate={priceRate}
          usdExchangeRate={usdExchangeRate}
          tokenUnit={tokenUnit}
          showRechargePrice={showRechargePrice}
        />
      )}
    </div>
  )
}

import { useQuery } from '@tanstack/react-query'
import { Building2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

import { getTenants } from './api'

export function Tenants() {
  const { t } = useTranslation()
  const tenantsQuery = useQuery({ queryKey: ['tenants'], queryFn: getTenants })

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Tenant Management')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex min-h-0 flex-col gap-4'>
          {tenantsQuery.isLoading && <LoadingState />}
          {tenantsQuery.isError && <ErrorState onRetry={() => void tenantsQuery.refetch()} />}
          {tenantsQuery.data?.length === 0 && <EmptyState icon={Building2} title={t('No Tenants Found')} />}
           {tenantsQuery.data && tenantsQuery.data.length > 0 && <div className='overflow-hidden rounded-lg border'><Table><TableHeader><TableRow><TableHead>{t('Name')}</TableHead><TableHead>{t('Slug')}</TableHead><TableHead>{t('Owner')}</TableHead><TableHead>{t('Status')}</TableHead><TableHead>{t('ID')}</TableHead></TableRow></TableHeader><TableBody>{tenantsQuery.data.map((item) => <TableRow key={item.id}><TableCell>{item.name}</TableCell><TableCell className='font-mono text-xs'>{item.slug}</TableCell><TableCell>{item.created_by}</TableCell><TableCell>{item.status}</TableCell><TableCell>{item.id}</TableCell></TableRow>)}</TableBody></Table></div>}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, Plus } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { handleServerError } from '@/lib/handle-server-error'

import { createTenant, getTenants } from './api'

const tenantSchema = z.object({
  slug: z.string().trim().min(1).max(64),
  name: z.string().trim().min(1).max(128),
})

type TenantForm = z.infer<typeof tenantSchema>

export function Tenants() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const tenantsQuery = useQuery({
    queryKey: ['tenants'],
    queryFn: getTenants,
  })
  const form = useForm<TenantForm>({
    resolver: zodResolver(tenantSchema),
    defaultValues: { slug: '', name: '' },
  })
  const createMutation = useMutation({
    mutationFn: createTenant,
    onSuccess: async () => {
      form.reset()
      await queryClient.invalidateQueries({ queryKey: ['tenants'] })
    },
    onError: (error) => handleServerError(error, t('Failed to create tenant')),
  })

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Tenant Management')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex min-h-0 flex-col gap-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Plus className='size-4' aria-hidden='true' />
                {t('Create Tenant')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <form
                className='grid gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end'
                onSubmit={form.handleSubmit((values) =>
                  createMutation.mutate(values)
                )}
              >
                <label className='grid gap-1 text-sm'>
                  <span>{t('Tenant name')}</span>
                  <Input
                    aria-invalid={!!form.formState.errors.name}
                    {...form.register('name')}
                  />
                </label>
                <label className='grid gap-1 text-sm'>
                  <span>{t('Tenant slug')}</span>
                  <Input
                    aria-invalid={!!form.formState.errors.slug}
                    {...form.register('slug')}
                  />
                </label>
                <Button type='submit' disabled={createMutation.isPending}>
                  {createMutation.isPending ? t('Creating...') : t('Create')}
                </Button>
              </form>
            </CardContent>
          </Card>

          {tenantsQuery.isLoading && <LoadingState />}
          {tenantsQuery.isError && (
            <ErrorState onRetry={() => void tenantsQuery.refetch()} />
          )}
          {tenantsQuery.data?.length === 0 && (
            <EmptyState icon={Building2} title={t('No Tenants Found')} />
          )}
          {tenantsQuery.data && tenantsQuery.data.length > 0 && (
            <div className='bg-card overflow-hidden rounded-lg border'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Name')}</TableHead>
                    <TableHead>{t('Slug')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                    <TableHead>{t('ID')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tenantsQuery.data.map((tenant) => (
                    <TableRow key={tenant.id}>
                      <TableCell>{tenant.name}</TableCell>
                      <TableCell className='font-mono text-xs'>
                        {tenant.slug}
                      </TableCell>
                      <TableCell>{tenant.status}</TableCell>
                      <TableCell>{tenant.id}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

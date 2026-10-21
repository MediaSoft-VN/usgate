import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, Globe, Pencil, Plus, Trash2, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { handleServerError } from '@/lib/handle-server-error'

import {
  createTenantDomain,
  getTenantDomains,
  getTenantDNSRecord,
  getTenantMembers,
  getTenants,
  deleteTenantDomain,
  updateTenantDomain,
  verifyTenantDomain,
  type TenantDomain,
} from './api'

export function PartnerWorkspace() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedTenantId, setSelectedTenantId] = useState<number | null>(null)
  const [domainDialogOpen, setDomainDialogOpen] = useState(false)
  const [editDomain, setEditDomain] = useState<TenantDomain | null>(null)
  const [verifyDomain, setVerifyDomain] = useState<TenantDomain | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{
    kind: 'domain'
    id: number
  } | null>(null)
  const [hostname, setHostname] = useState('')

  const tenantsQuery = useQuery({ queryKey: ['tenants'], queryFn: getTenants })
  const tenant =
    tenantsQuery.data?.find((item) => item.id === selectedTenantId) ?? null
  const membersQuery = useQuery({
    queryKey: ['tenant-members', selectedTenantId],
    queryFn: () => getTenantMembers(selectedTenantId as number),
    enabled: selectedTenantId !== null,
  })
  const domainsQuery = useQuery({
    queryKey: ['tenant-domains', selectedTenantId],
    queryFn: () => getTenantDomains(selectedTenantId as number),
    enabled: selectedTenantId !== null,
  })
  const dnsRecordQuery = useQuery({
    queryKey: ['tenant-dns-record', selectedTenantId],
    queryFn: () => getTenantDNSRecord(selectedTenantId as number),
    enabled: selectedTenantId !== null,
  })

  const refreshTenant = async (key: 'tenant-members' | 'tenant-domains') => {
    await queryClient.invalidateQueries({ queryKey: [key, selectedTenantId] })
  }
  useEffect(() => {
    if (selectedTenantId === null && tenantsQuery.data?.[0]) {
      setSelectedTenantId(tenantsQuery.data[0].id)
    }
  }, [selectedTenantId, tenantsQuery.data])
  const domainMutation = useMutation({
    mutationFn: () => createTenantDomain(selectedTenantId as number, hostname),
    onSuccess: (result) => {
      setDomainDialogOpen(false)
      setHostname('')
       setVerifyDomain(result.domain)
      void refreshTenant('tenant-domains')
    },
    onError: (error) => handleServerError(error, t('Failed to add domain')),
  })
  const verifyMutation = useMutation({
    mutationFn: () =>
      verifyTenantDomain(
        selectedTenantId as number,
        verifyDomain?.id as number
      ),
    onSuccess: () => {
      setVerifyDomain(null)
      void refreshTenant('tenant-domains')
    },
    onError: (error) => handleServerError(error, t('Failed to verify domain')),
  })
  const updateMutation = useMutation({
    mutationFn: () =>
      updateTenantDomain(
        selectedTenantId as number,
        editDomain?.id as number,
        hostname
      ),
    onSuccess: (result) => {
      setDomainDialogOpen(false)
      setEditDomain(null)
      setHostname('')
       setVerifyDomain(result.domain)
      void refreshTenant('tenant-domains')
    },
    onError: (error) => handleServerError(error, t('Failed to update domain')),
  })
  const deleteMutation = useMutation({
    mutationFn: () =>
      deleteTenantDomain(
        selectedTenantId as number,
        deleteTarget?.id as number
      ),
    onSuccess: async () => {
      setDeleteTarget(null)
      await refreshTenant('tenant-domains')
    },
    onError: (error) => handleServerError(error, t('Failed to delete domain')),
  })

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Partner workspace')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex min-h-0 flex-col gap-4'>
          {tenant && (
            <Card>
              <CardHeader>
                <CardTitle className='flex items-center gap-2'>
                  <Building2 className='size-4' aria-hidden='true' />
                  {tenant.name}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className='text-muted-foreground text-sm'>{tenant.slug}</p>
              </CardContent>
            </Card>
          )}
          {tenantsQuery.isLoading && <LoadingState />}
          {tenantsQuery.isError && (
            <ErrorState onRetry={() => void tenantsQuery.refetch()} />
          )}
          {tenantsQuery.data?.length === 0 && (
            <EmptyState icon={Building2} title={t('No Tenants Found')} />
          )}
          {tenantsQuery.data && tenantsQuery.data.length > 0 && (
            <div className='overflow-hidden rounded-lg border'>
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
                  {tenantsQuery.data.map((item) => (
                    <TableRow
                      key={item.id}
                      data-state={
                        item.id === selectedTenantId ? 'selected' : undefined
                      }
                    >
                      <TableCell>{item.name}</TableCell>
                      <TableCell className='font-mono text-xs'>
                        {item.slug}
                      </TableCell>
                      <TableCell>{item.status}</TableCell>
                      <TableCell>{item.id}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
          {tenant && (
            <Tabs defaultValue='domains'>
              <TabsList aria-label={t('Tenant settings')}>
                <TabsTrigger value='members'>
                  <Users className='size-4' aria-hidden='true' />
                  {t('Members')}
                </TabsTrigger>
                <TabsTrigger value='domains'>
                  <Globe className='size-4' aria-hidden='true' />
                  {t('Domains')}
                </TabsTrigger>
              </TabsList>
              <TabsContent value='members'>
                <Card>
                  <CardHeader>
                    <CardTitle className='flex items-center justify-between'>
                      <span>{t('Members')}</span>
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    {membersQuery.isLoading ? (
                      <LoadingState />
                    ) : (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>{t('User')}</TableHead>
                            <TableHead>{t('Role')}</TableHead>
                            <TableHead>{t('Status')}</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {membersQuery.data?.map((member) => (
                            <TableRow key={member.id}>
                              <TableCell>{member.display_name || member.username}</TableCell>
                              <TableCell>{member.role_key}</TableCell>
                              <TableCell>{member.status}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>
              <TabsContent value='domains'>
                <Card>
                  <CardHeader>
                    <CardTitle className='flex items-center justify-between'>
                      <span>{t('Tenant domains')}</span>
                      <Dialog
                        open={domainDialogOpen}
                        onOpenChange={setDomainDialogOpen}
                      >
                        <DialogTrigger
                          render={<Button size='sm' />}
                          onClick={() => {
                            setEditDomain(null)
                            setHostname('')
                          }}
                        >
                          <Plus className='size-4' aria-hidden='true' />
                          {t('Add domain')}
                        </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>
                              {editDomain ? t('Edit domain') : t('Add domain')}
                            </DialogTitle>
                          </DialogHeader>
                          <label className='grid gap-1 text-sm'>
                            <Input
                              value={hostname}
                              onChange={(event) =>
                                setHostname(event.target.value)
                              }
                              placeholder='app.example.com'
                            />
                          </label>
                          <DialogFooter>
                            <Button
                              type='button'
                              onClick={() =>
                                editDomain
                                  ? updateMutation.mutate()
                                  : domainMutation.mutate()
                              }
                              disabled={
                                !hostname ||
                                domainMutation.isPending ||
                                updateMutation.isPending
                              }
                            >
                              {editDomain ? t('Save') : t('Add')}
                            </Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    {dnsRecordQuery.data && (
                      <div className='bg-muted mb-4 rounded-md p-3 text-sm'>
                        <p className='font-medium'>
                          {t('DNS records to configure')}
                        </p>
                        <div className='text-muted-foreground mt-1 grid gap-1'>
                          <p>
                            <span className='font-medium'>CNAME:</span>{' '}
                            <code className='font-mono'>
                              {dnsRecordQuery.data.cname_target}
                            </code>
                          </p>
                          <p>
                            <span className='font-medium'>A:</span>{' '}
                            <code className='font-mono'>
                              {dnsRecordQuery.data.a_record_ip}
                            </code>
                          </p>
                        </div>
                      </div>
                    )}
                    {domainsQuery.isLoading ? (
                      <LoadingState />
                    ) : (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>{t('Hostname')}</TableHead>
                            <TableHead>{t('Status')}</TableHead>
                            <TableHead className='text-end'>
                              {t('Actions')}
                            </TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {domainsQuery.data?.map((domain) => (
                            <TableRow key={domain.id}>
                              <TableCell className='font-mono text-xs'>
                                {domain.hostname}
                              </TableCell>
                              <TableCell>{domain.status}</TableCell>
                              <TableCell className='flex justify-end gap-2'>
                                <Button
                                  size='sm'
                                  variant='outline'
                                  onClick={() => {
                                    setEditDomain(domain)
                                    setHostname(domain.hostname)
                                    setDomainDialogOpen(true)
                                  }}
                                >
                                  <Pencil
                                    className='size-4'
                                    aria-hidden='true'
                                  />
                                  {t('Edit')}
                                </Button>
                                {domain.status === 'pending' && (
                                  <Button
                                    size='sm'
                                    variant='outline'
                                    onClick={() => setVerifyDomain(domain)}
                                  >
                                    {t('Verify')}
                                  </Button>
                                )}
                                <Button
                                  variant='destructive'
                                  size='sm'
                                  onClick={() =>
                                    setDeleteTarget({
                                      kind: 'domain',
                                      id: domain.id,
                                    })
                                  }
                                >
                                  <Trash2
                                    className='size-4'
                                    aria-hidden='true'
                                  />
                                  {t('Delete')}
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          )}
        </div>
        <Dialog
          open={verifyDomain !== null}
          onOpenChange={(open) => !open && setVerifyDomain(null)}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t('Verify domain')}</DialogTitle>
            </DialogHeader>
            <p className='text-muted-foreground text-sm'>
              {verifyDomain?.hostname}
            </p>
            <p className='text-muted-foreground text-sm'>
              {t('We will check the domain DNS records against the configured target.')}
            </p>
            <DialogFooter>
              <Button
                onClick={() => verifyMutation.mutate()}
                disabled={verifyMutation.isPending}
              >
                {t('Verify')}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <ConfirmDialog
          open={deleteTarget !== null}
          onOpenChange={(open) => !open && setDeleteTarget(null)}
          title={t('Delete domain')}
          desc={t(
            'This permanently deletes the domain. It can be added again later.'
          )}
          destructive
          handleConfirm={() => deleteMutation.mutate()}
          isLoading={deleteMutation.isPending}
          confirmText={t('Delete')}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

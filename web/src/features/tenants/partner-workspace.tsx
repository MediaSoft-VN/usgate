import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, Globe, Plus, Users } from 'lucide-react'
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
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
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
  getTenantMembers,
  getTenantMemberCandidates,
  getTenants,
  removeTenantDomain,
  removeTenantMember,
  updateTenantMember,
  upsertTenantMember,
  verifyTenantDomain,
  type TenantDomain,
} from './api'

const roles = ['owner', 'admin', 'member', 'viewer']

export function PartnerWorkspace() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedTenantId, setSelectedTenantId] = useState<number | null>(null)
  const [memberDialogOpen, setMemberDialogOpen] = useState(false)
  const [domainDialogOpen, setDomainDialogOpen] = useState(false)
  const [verifyDomain, setVerifyDomain] = useState<TenantDomain | null>(null)
  const [removeTarget, setRemoveTarget] = useState<{
    kind: 'member' | 'domain'
    id: number
  } | null>(null)
  const [memberUserId, setMemberUserId] = useState('')
  const [memberRole, setMemberRole] = useState('member')
  const [hostname, setHostname] = useState('')
  const [token, setToken] = useState('')
  const [verificationToken, setVerificationToken] = useState('')

  const tenantsQuery = useQuery({ queryKey: ['tenants'], queryFn: getTenants })
  const ownerCandidatesQuery = useQuery({
    queryKey: ['tenant-member-candidates', selectedTenantId],
    queryFn: () => getTenantMemberCandidates(selectedTenantId as number),
    enabled: selectedTenantId !== null,
  })
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

  const refreshTenant = async (key: 'tenant-members' | 'tenant-domains') => {
    await queryClient.invalidateQueries({ queryKey: [key, selectedTenantId] })
  }
  useEffect(() => {
    if (selectedTenantId === null && tenantsQuery.data?.[0]) {
      setSelectedTenantId(tenantsQuery.data[0].id)
    }
  }, [selectedTenantId, tenantsQuery.data])
  const memberMutation = useMutation({
    mutationFn: () =>
      upsertTenantMember(selectedTenantId as number, {
        user_id: Number(memberUserId),
        role_key: memberRole,
      }),
    onSuccess: async () => {
      setMemberDialogOpen(false)
      setMemberUserId('')
      await refreshTenant('tenant-members')
    },
    onError: (error) =>
      handleServerError(error, t('Failed to update tenant member')),
  })
  const roleMutation = useMutation({
    mutationFn: (values: { userId: number; role: string }) =>
      updateTenantMember(
        selectedTenantId as number,
        values.userId,
        values.role
      ),
    onSuccess: () => void refreshTenant('tenant-members'),
    onError: (error) =>
      handleServerError(error, t('Failed to update tenant member')),
  })
  const domainMutation = useMutation({
    mutationFn: () => createTenantDomain(selectedTenantId as number, hostname),
    onSuccess: (result) => {
      setDomainDialogOpen(false)
      setHostname('')
      setVerificationToken(result.verification_token)
      setToken(result.verification_token)
      setVerifyDomain(result.domain)
      void refreshTenant('tenant-domains')
    },
    onError: (error) => handleServerError(error, t('Failed to add domain')),
  })
  const verifyMutation = useMutation({
    mutationFn: () =>
      verifyTenantDomain(
        selectedTenantId as number,
        verifyDomain?.id as number,
        token
      ),
    onSuccess: () => {
      setVerifyDomain(null)
      setToken('')
      void refreshTenant('tenant-domains')
    },
    onError: (error) => handleServerError(error, t('Failed to verify domain')),
  })
  const removeMutation = useMutation({
    mutationFn: () =>
      removeTarget?.kind === 'member'
        ? removeTenantMember(selectedTenantId as number, removeTarget.id)
        : removeTenantDomain(
            selectedTenantId as number,
            removeTarget?.id as number
          ),
    onSuccess: async () => {
      setRemoveTarget(null)
      await refreshTenant(
        removeTarget?.kind === 'member' ? 'tenant-members' : 'tenant-domains'
      )
    },
    onError: (error) => handleServerError(error, t('Failed to remove item')),
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
            <Tabs defaultValue='members'>
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
                      <span>{t('Tenant members')}</span>
                      <Button
                        size='sm'
                        onClick={() => setMemberDialogOpen(true)}
                      >
                        <Plus className='size-4' aria-hidden='true' />
                        {t('Add member')}
                      </Button>
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
                            <TableHead className='text-end'>
                              {t('Actions')}
                            </TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {membersQuery.data?.map((member) => (
                            <TableRow key={member.id}>
                              <TableCell>
                                {ownerCandidatesQuery.data?.find(
                                  (user) => user.id === member.user_id
                                )?.display_name || `#${member.user_id}`}
                              </TableCell>
                              <TableCell>
                                <NativeSelect
                                  aria-label={t('Role for user {{id}}', {
                                    id: member.user_id,
                                  })}
                                  value={member.role_key}
                                  onChange={(event) =>
                                    roleMutation.mutate({
                                      userId: member.user_id,
                                      role: event.target.value,
                                    })
                                  }
                                >
                                  {roles.map((role) => (
                                    <NativeSelectOption key={role} value={role}>
                                      {role}
                                    </NativeSelectOption>
                                  ))}
                                </NativeSelect>
                              </TableCell>
                              <TableCell>{member.status}</TableCell>
                              <TableCell className='text-end'>
                                <Button
                                  variant='destructive'
                                  size='sm'
                                  onClick={() =>
                                    setRemoveTarget({
                                      kind: 'member',
                                      id: member.user_id,
                                    })
                                  }
                                >
                                  {t('Remove')}
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
              <TabsContent value='domains'>
                <Card>
                  <CardHeader>
                    <CardTitle className='flex items-center justify-between'>
                      <span>{t('Tenant domains')}</span>
                      <Button
                        size='sm'
                        onClick={() => setDomainDialogOpen(true)}
                      >
                        <Plus className='size-4' aria-hidden='true' />
                        {t('Add domain')}
                      </Button>
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
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
                                    setRemoveTarget({
                                      kind: 'domain',
                                      id: domain.id,
                                    })
                                  }
                                >
                                  {t('Remove')}
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
      </SectionPageLayout.Content>
      <Dialog open={memberDialogOpen} onOpenChange={setMemberDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Add member')}</DialogTitle>
          </DialogHeader>
          <div className='grid gap-3'>
            <label className='grid gap-1 text-sm'>
              <span>{t('User')}</span>
              <NativeSelect
                value={memberUserId}
                onChange={(event) => setMemberUserId(event.target.value)}
              >
                <NativeSelectOption value=''>
                  {t('Select an existing user')}
                </NativeSelectOption>
                {ownerCandidatesQuery.data?.map((user) => (
                  <NativeSelectOption key={user.id} value={user.id}>
                    {user.display_name || user.username} ({user.username})
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </label>
            <label className='grid gap-1 text-sm'>
              <span>{t('Role')}</span>
              <NativeSelect
                value={memberRole}
                onChange={(event) => setMemberRole(event.target.value)}
              >
                {roles.map((role) => (
                  <NativeSelectOption key={role} value={role}>
                    {role}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </label>
          </div>
          <DialogFooter>
            <Button
              onClick={() => memberMutation.mutate()}
              disabled={!memberUserId || memberMutation.isPending}
            >
              {t('Add')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={domainDialogOpen} onOpenChange={setDomainDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Add domain')}</DialogTitle>
          </DialogHeader>
          <label className='grid gap-1 text-sm'>
            <span>{t('Hostname')}</span>
            <Input
              value={hostname}
              onChange={(event) => setHostname(event.target.value)}
              placeholder='app.example.com'
            />
          </label>
          <DialogFooter>
            <Button
              onClick={() => domainMutation.mutate()}
              disabled={!hostname || domainMutation.isPending}
            >
              {t('Add')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
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
          {verificationToken && (
            <p className='text-sm'>
              <span>{t('Verification token')}: </span>
              <code className='bg-muted rounded px-1 break-all'>
                {verificationToken}
              </code>
            </p>
          )}
          <label className='grid gap-1 text-sm'>
            <span>{t('Verification token')}</span>
            <Input
              value={token}
              onChange={(event) => setToken(event.target.value)}
            />
          </label>
          <DialogFooter>
            <Button
              onClick={() => verifyMutation.mutate()}
              disabled={!token || verifyMutation.isPending}
            >
              {t('Verify')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        open={removeTarget !== null}
        onOpenChange={(open) => !open && setRemoveTarget(null)}
        title={t('Remove item')}
        desc={t('This action cannot be undone.')}
        destructive
        handleConfirm={() => removeMutation.mutate()}
        isLoading={removeMutation.isPending}
        confirmText={t('Remove')}
      />
    </SectionPageLayout>
  )
}

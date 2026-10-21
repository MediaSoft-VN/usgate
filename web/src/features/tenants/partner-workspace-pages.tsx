import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from '@tanstack/react-query'
import { Building2, Globe, Pencil, Plus, Trash2, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'

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
import { formatNumber, formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import {
  createTenantDomain,
  deleteTenantDomain,
  fundTenantMember,
  getTenantDNSRecord,
  getTenantDomains,
  getTenantFunding,
  getTenantMembers,
  getTenants,
  updateTenantDomain,
  verifyTenantDomain,
  type TenantDomain,
  type TenantFunding,
  type Tenant,
  type TenantDNSRecord,
  type TenantMember,
  type TenantWallet,
} from './api'

export type PartnerWorkspaceSection =
  | 'overview'
  | 'users'
  | 'domains'
  | 'wallet'
  | 'transactions'

type PartnerWorkspaceProps = {
  section?: PartnerWorkspaceSection
}

export function PartnerWorkspace(props: PartnerWorkspaceProps) {
  const { t } = useTranslation()
  const section = props.section ?? 'domains'
  const queryClient = useQueryClient()
  const [selectedTenantId, setSelectedTenantId] = useState<number | null>(null)
  const [domainDialogOpen, setDomainDialogOpen] = useState(false)
  const [editDomain, setEditDomain] = useState<TenantDomain | null>(null)
  const [verifyDomain, setVerifyDomain] = useState<TenantDomain | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null)
  const [hostname, setHostname] = useState('')
  const [fundingOpen, setFundingOpen] = useState(false)
  const [fundingMember, setFundingMember] = useState('')
  const [fundingAmount, setFundingAmount] = useState('')
  const [fundingReference, setFundingReference] = useState('')

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
    enabled: selectedTenantId !== null && section === 'domains',
  })
  const fundingQuery = useQuery({
    queryKey: ['tenant-funding', selectedTenantId],
    queryFn: () => getTenantFunding(selectedTenantId as number),
    enabled: selectedTenantId !== null,
  })

  useEffect(() => {
    if (selectedTenantId === null && tenantsQuery.data?.[0]) {
      setSelectedTenantId(tenantsQuery.data[0].id)
    }
  }, [selectedTenantId, tenantsQuery.data])

  const refreshDomains = () =>
    queryClient.invalidateQueries({ queryKey: ['tenant-domains', selectedTenantId] })

  const domainMutation = useMutation({
    mutationFn: () => createTenantDomain(selectedTenantId as number, hostname),
    onSuccess: (result) => {
      setDomainDialogOpen(false)
      setHostname('')
      setVerifyDomain(result.domain)
      void refreshDomains()
    },
    onError: (error) => handleServerError(error, t('Failed to add domain')),
  })
  const verifyMutation = useMutation({
    mutationFn: () =>
      verifyTenantDomain(selectedTenantId as number, verifyDomain?.id as number),
    onSuccess: () => {
      setVerifyDomain(null)
      void refreshDomains()
    },
    onError: (error) => handleServerError(error, t('Failed to verify domain')),
  })
  const updateMutation = useMutation({
    mutationFn: () =>
      updateTenantDomain(selectedTenantId as number, editDomain?.id as number, hostname),
    onSuccess: (result) => {
      setDomainDialogOpen(false)
      setEditDomain(null)
      setHostname('')
      setVerifyDomain(result.domain)
      void refreshDomains()
    },
    onError: (error) => handleServerError(error, t('Failed to update domain')),
  })
  const deleteMutation = useMutation({
    mutationFn: () => deleteTenantDomain(selectedTenantId as number, deleteTarget as number),
    onSuccess: async () => {
      setDeleteTarget(null)
      await refreshDomains()
    },
    onError: (error) => handleServerError(error, t('Failed to delete domain')),
  })
  const fundingMutation = useMutation({
    mutationFn: () =>
      fundTenantMember(selectedTenantId as number, {
        user_id: Number(fundingMember),
        reference: fundingReference.trim(),
        amount_minor: Math.round(Number(fundingAmount) * 100),
        currency: fundingQuery.data?.wallet.currency ?? 'USD',
      }),
    onSuccess: () => {
      setFundingOpen(false)
      setFundingMember('')
      setFundingAmount('')
      setFundingReference('')
      void fundingQuery.refetch()
      void membersQuery.refetch()
    },
    onError: (error) => handleServerError(error, t('Failed to fund member')),
  })

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Partner workspace')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex min-h-0 flex-col gap-4'>
          {section === 'overview' && (
            <OverviewSection tenant={tenant} tenantsQuery={tenantsQuery} t={t} />
          )}
          {section === 'users' && (
            <UsersSection membersQuery={membersQuery} t={t} />
          )}
          {section === 'domains' && (
            <DomainsSection
              t={t}
              tenant={tenant}
              domainsQuery={domainsQuery}
              dnsRecordQuery={dnsRecordQuery}
              domainDialogOpen={domainDialogOpen}
              setDomainDialogOpen={setDomainDialogOpen}
              editDomain={editDomain}
              setEditDomain={setEditDomain}
              hostname={hostname}
              setHostname={setHostname}
              domainMutation={domainMutation}
              updateMutation={updateMutation}
              setVerifyDomain={setVerifyDomain}
              setDeleteTarget={setDeleteTarget}
            />
          )}
          {(section === 'wallet' || section === 'transactions') && (
            <WalletSection
              section={section}
              t={t}
              fundingQuery={fundingQuery}
              membersQuery={membersQuery}
              fundingOpen={fundingOpen}
              setFundingOpen={setFundingOpen}
              fundingMember={fundingMember}
              setFundingMember={setFundingMember}
              fundingAmount={fundingAmount}
              setFundingAmount={setFundingAmount}
              fundingReference={fundingReference}
              setFundingReference={setFundingReference}
              fundingMutation={fundingMutation}
            />
          )}
        </div>
        <Dialog open={verifyDomain !== null} onOpenChange={(open) => !open && setVerifyDomain(null)}>
          <DialogContent>
            <DialogHeader><DialogTitle>{t('Verify domain')}</DialogTitle></DialogHeader>
            <p className='text-muted-foreground text-sm'>{verifyDomain?.hostname}</p>
            <p className='text-muted-foreground text-sm'>{t('We will check the domain DNS records against the configured target.')}</p>
            <DialogFooter><Button onClick={() => verifyMutation.mutate()} disabled={verifyMutation.isPending}>{t('Verify')}</Button></DialogFooter>
          </DialogContent>
        </Dialog>
        <ConfirmDialog
          open={deleteTarget !== null}
          onOpenChange={(open) => !open && setDeleteTarget(null)}
          title={t('Delete domain')}
          desc={t('This permanently deletes the domain. It can be added again later.')}
          destructive
          handleConfirm={() => deleteMutation.mutate()}
          isLoading={deleteMutation.isPending}
          confirmText={t('Delete')}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function OverviewSection(props: { tenant: Tenant | null; tenantsQuery: UseQueryResult<Tenant[], Error>; t: TFunction }) {
  return (
    <>
      {props.tenant && <Card><CardHeader><CardTitle className='flex items-center gap-2'><Building2 className='size-4' aria-hidden='true' />{props.tenant.name}</CardTitle></CardHeader><CardContent><p className='text-muted-foreground text-sm'>{props.tenant.slug}</p></CardContent></Card>}
      {props.tenantsQuery.isLoading && <LoadingState />}
      {props.tenantsQuery.isError && <ErrorState onRetry={() => void props.tenantsQuery.refetch()} />}
      {props.tenantsQuery.data?.length === 0 && <EmptyState icon={Building2} title={props.t('No Tenants Found')} />}
      {props.tenantsQuery.data && props.tenantsQuery.data.length > 0 && <div className='overflow-hidden rounded-lg border'><Table><TableHeader><TableRow><TableHead>{props.t('Name')}</TableHead><TableHead>{props.t('Slug')}</TableHead><TableHead>{props.t('Status')}</TableHead><TableHead>{props.t('ID')}</TableHead></TableRow></TableHeader><TableBody>{props.tenantsQuery.data.map((item) => <TableRow key={item.id}><TableCell>{item.name}</TableCell><TableCell className='font-mono text-xs'>{item.slug}</TableCell><TableCell>{item.status}</TableCell><TableCell>{item.id}</TableCell></TableRow>)}</TableBody></Table></div>}
    </>
  )
}

function UsersSection(props: { membersQuery: UseQueryResult<TenantMember[], Error>; t: TFunction }) {
  return <Card><CardHeader><CardTitle className='flex items-center gap-2'><Users className='size-4' aria-hidden='true' />{props.t('Users')}</CardTitle></CardHeader><CardContent>{props.membersQuery.isLoading ? <LoadingState /> : <Table><TableHeader><TableRow><TableHead>{props.t('User')}</TableHead><TableHead>{props.t('Role')}</TableHead><TableHead>{props.t('Status')}</TableHead><TableHead>{props.t('Joined at')}</TableHead></TableRow></TableHeader><TableBody>{(props.membersQuery.data ?? []).map((member) => <TableRow key={member.id}><TableCell>{member.display_name || member.username}</TableCell><TableCell>{member.role_key}</TableCell><TableCell>{member.status}</TableCell><TableCell>{formatTimestampToDate(member.created_at)}</TableCell></TableRow>)}</TableBody></Table>}</CardContent></Card>
}

type DomainsSectionProps = {
  t: TFunction
  tenant: Tenant | null
  domainsQuery: UseQueryResult<TenantDomain[], Error>
  dnsRecordQuery: UseQueryResult<TenantDNSRecord, Error>
  domainDialogOpen: boolean
  setDomainDialogOpen: (open: boolean) => void
  editDomain: TenantDomain | null
  setEditDomain: (domain: TenantDomain | null) => void
  hostname: string
  setHostname: (hostname: string) => void
  domainMutation: UseMutationResult<{ domain: TenantDomain }, Error, void, unknown>
  updateMutation: UseMutationResult<{ domain: TenantDomain }, Error, void, unknown>
  setVerifyDomain: (domain: TenantDomain) => void
  setDeleteTarget: (id: number) => void
}

function DomainsSection(props: DomainsSectionProps) {
  return <Card><CardHeader><CardTitle className='flex items-center justify-between'><span className='flex items-center gap-2'><Globe className='size-4' aria-hidden='true' />{props.t('Tenant domains')}</span><Dialog open={props.domainDialogOpen} onOpenChange={props.setDomainDialogOpen}><DialogTrigger render={<Button size='sm' />} onClick={() => { props.setEditDomain(null); props.setHostname('') }}><Plus className='size-4' aria-hidden='true' />{props.t('Add domain')}</DialogTrigger><DialogContent><DialogHeader><DialogTitle>{props.editDomain ? props.t('Edit domain') : props.t('Add domain')}</DialogTitle></DialogHeader><Input value={props.hostname} onChange={(event) => props.setHostname(event.target.value)} placeholder='app.example.com' /><DialogFooter><Button type='button' onClick={() => props.editDomain ? props.updateMutation.mutate() : props.domainMutation.mutate()} disabled={!props.hostname || props.domainMutation.isPending || props.updateMutation.isPending}>{props.editDomain ? props.t('Save') : props.t('Add')}</Button></DialogFooter></DialogContent></Dialog></CardTitle></CardHeader><CardContent>{props.dnsRecordQuery.data && <div className='bg-muted mb-4 rounded-md p-3 text-sm'><p className='font-medium'>{props.t('DNS records to configure')}</p><p className='text-muted-foreground mt-1'>CNAME: <code className='font-mono'>{props.dnsRecordQuery.data.cname_target}</code></p><p className='text-muted-foreground'>A: <code className='font-mono'>{props.dnsRecordQuery.data.a_record_ip}</code></p></div>}{props.domainsQuery.isLoading ? <LoadingState /> : <Table><TableHeader><TableRow><TableHead>{props.t('Hostname')}</TableHead><TableHead>{props.t('Status')}</TableHead><TableHead className='text-end'>{props.t('Actions')}</TableHead></TableRow></TableHeader><TableBody>{(props.domainsQuery.data ?? []).map((domain: TenantDomain) => <TableRow key={domain.id}><TableCell className='font-mono text-xs'>{domain.hostname}</TableCell><TableCell>{domain.status}</TableCell><TableCell className='flex justify-end gap-2'><Button size='sm' variant='outline' onClick={() => { props.setEditDomain(domain); props.setHostname(domain.hostname); props.setDomainDialogOpen(true) }}><Pencil className='size-4' aria-hidden='true' />{props.t('Edit')}</Button>{domain.status === 'pending' && <Button size='sm' variant='outline' onClick={() => props.setVerifyDomain(domain)}>{props.t('Verify')}</Button>}<Button variant='destructive' size='sm' onClick={() => props.setDeleteTarget(domain.id)}><Trash2 className='size-4' aria-hidden='true' />{props.t('Delete')}</Button></TableCell></TableRow>)}</TableBody></Table>}</CardContent></Card>
}

type WalletSectionProps = {
  section: 'wallet' | 'transactions'
  t: TFunction
  fundingQuery: UseQueryResult<{ wallet: TenantWallet; items: TenantFunding[] }, Error>
  membersQuery: UseQueryResult<TenantMember[], Error>
  fundingOpen: boolean
  setFundingOpen: (open: boolean) => void
  fundingMember: string
  setFundingMember: (member: string) => void
  fundingAmount: string
  setFundingAmount: (amount: string) => void
  fundingReference: string
  setFundingReference: (reference: string) => void
  fundingMutation: UseMutationResult<void, Error, void, unknown>
}

function WalletSection(props: WalletSectionProps) {
  const isWallet = props.section === 'wallet'
  return <Card><CardHeader><CardTitle className='flex items-center justify-between'><span>{isWallet ? props.t('Tenant wallet') : props.t('Funding transactions')}</span>{isWallet && <Dialog open={props.fundingOpen} onOpenChange={props.setFundingOpen}><DialogTrigger render={<Button size='sm' />}><Plus className='size-4' aria-hidden='true' />{props.t('Fund member')}</DialogTrigger><DialogContent><DialogHeader><DialogTitle>{props.t('Fund member')}</DialogTitle></DialogHeader><div className='grid gap-3'><label className='grid gap-1 text-sm'>{props.t('Member')}<Select value={props.fundingMember} onValueChange={(value) => props.setFundingMember(value ?? '')}><SelectTrigger className='w-full'><SelectValue placeholder={props.t('Select member')} /></SelectTrigger><SelectContent>{(props.membersQuery.data ?? []).filter((member: { status: string }) => member.status === 'active').map((member: { user_id: number; display_name: string; username: string }) => <SelectItem key={member.user_id} value={String(member.user_id)}>{member.display_name || member.username}</SelectItem>)}</SelectContent></Select></label><label className='grid gap-1 text-sm'>{props.t('Amount')} ({props.fundingQuery.data?.wallet.currency ?? 'USD'})<Input type='number' min='0.01' step='0.01' value={props.fundingAmount} onChange={(event) => props.setFundingAmount(event.target.value)} /></label><label className='grid gap-1 text-sm'>{props.t('Reference')}<Input value={props.fundingReference} onChange={(event) => props.setFundingReference(event.target.value)} /></label></div><DialogFooter><Button disabled={props.fundingMutation.isPending || !props.fundingMember || !props.fundingReference.trim() || Number(props.fundingAmount) <= 0} onClick={() => props.fundingMutation.mutate()}>{props.t('Fund member')}</Button></DialogFooter></DialogContent></Dialog>}</CardTitle></CardHeader><CardContent className='space-y-4'>{isWallet && <p className='text-2xl font-bold'>{formatNumber((props.fundingQuery.data?.wallet.available_minor ?? 0) / 100)} {props.fundingQuery.data?.wallet.currency ?? 'USD'} <span className='text-muted-foreground text-sm font-normal'>{props.t('Available')}</span></p>}<Table><TableHeader><TableRow><TableHead>{props.t('Member')}</TableHead><TableHead>{props.t('Amount')}</TableHead><TableHead>{props.t('Reference')}</TableHead></TableRow></TableHeader><TableBody>{(props.fundingQuery.data?.items ?? []).map((item: TenantFunding) => <TableRow key={item.id}><TableCell>{item.display_name || item.username}</TableCell><TableCell>{formatNumber(item.amount_minor / 100)} {item.currency}</TableCell><TableCell className='font-mono text-xs'>{item.reference}</TableCell></TableRow>)}</TableBody></Table></CardContent></Card>
}

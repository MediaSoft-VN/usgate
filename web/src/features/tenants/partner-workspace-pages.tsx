import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query'
import { Building2, CheckCircle2, CircleDollarSign, Globe, Pencil, Plus, Trash2, Users } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatNumber, formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { createTenantDomain, deleteTenantDomain, fundTenantMember, getTenantDNSRecord, getTenantDomains, getTenantFunding, getTenantMembers, getTenants, updateTenantDomain, verifyTenantDomain, type Tenant, type TenantDNSRecord, type TenantDomain, type TenantFunding, type TenantMember, type TenantWallet } from './api'

export type PartnerWorkspaceSection = 'overview' | 'users' | 'domains' | 'wallet' | 'transactions'
type PartnerWorkspaceProps = { section?: PartnerWorkspaceSection }

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
  const tenant = tenantsQuery.data?.find((item) => item.id === selectedTenantId) ?? null
  const membersQuery = useQuery({ queryKey: ['tenant-members', selectedTenantId], queryFn: () => getTenantMembers(selectedTenantId as number), enabled: selectedTenantId !== null })
  const domainsQuery = useQuery({ queryKey: ['tenant-domains', selectedTenantId], queryFn: () => getTenantDomains(selectedTenantId as number), enabled: selectedTenantId !== null })
  const dnsRecordQuery = useQuery({ queryKey: ['tenant-dns-record', selectedTenantId], queryFn: () => getTenantDNSRecord(selectedTenantId as number), enabled: selectedTenantId !== null && section === 'domains' })
  const fundingQuery = useQuery({ queryKey: ['tenant-funding', selectedTenantId], queryFn: () => getTenantFunding(selectedTenantId as number), enabled: selectedTenantId !== null })

  useEffect(() => { if (selectedTenantId === null && tenantsQuery.data?.[0]) setSelectedTenantId(tenantsQuery.data[0].id) }, [selectedTenantId, tenantsQuery.data])
  const refreshDomains = () => queryClient.invalidateQueries({ queryKey: ['tenant-domains', selectedTenantId] })
  const domainMutation = useMutation({ mutationFn: () => createTenantDomain(selectedTenantId as number, hostname), onSuccess: (result) => { setDomainDialogOpen(false); setHostname(''); setVerifyDomain(result.domain); void refreshDomains() }, onError: (error) => handleServerError(error, t('Failed to add domain')) })
  const verifyMutation = useMutation({ mutationFn: () => verifyTenantDomain(selectedTenantId as number, verifyDomain?.id as number), onSuccess: () => { setVerifyDomain(null); void refreshDomains() }, onError: (error) => handleServerError(error, t('Failed to verify domain')) })
  const updateMutation = useMutation({ mutationFn: () => updateTenantDomain(selectedTenantId as number, editDomain?.id as number, hostname), onSuccess: (result) => { setDomainDialogOpen(false); setEditDomain(null); setHostname(''); setVerifyDomain(result.domain); void refreshDomains() }, onError: (error) => handleServerError(error, t('Failed to update domain')) })
  const deleteMutation = useMutation({ mutationFn: () => deleteTenantDomain(selectedTenantId as number, deleteTarget as number), onSuccess: async () => { setDeleteTarget(null); await refreshDomains() }, onError: (error) => handleServerError(error, t('Failed to delete domain')) })
  const fundingMutation = useMutation({ mutationFn: () => fundTenantMember(selectedTenantId as number, { user_id: Number(fundingMember), reference: fundingReference.trim(), amount_minor: Math.round(Number(fundingAmount) * 100), currency: fundingQuery.data?.wallet.currency ?? 'USD' }), onSuccess: () => { setFundingOpen(false); setFundingMember(''); setFundingAmount(''); setFundingReference(''); void fundingQuery.refetch(); void membersQuery.refetch() }, onError: (error) => handleServerError(error, t('Failed to fund member')) })

  return <SectionPageLayout fixedContent stackActionsOnMobile>
    <SectionPageLayout.Title><span className='inline-flex items-center gap-2'><Building2 className='text-primary size-5' aria-hidden='true' />{t('Partner workspace')}</span></SectionPageLayout.Title>
    <SectionPageLayout.Actions>{tenant && <span className='text-muted-foreground max-w-full truncate text-sm'>{tenant.name}</span>}</SectionPageLayout.Actions>
    <SectionPageLayout.Content><div className='flex min-h-0 flex-col gap-6'>
      <WorkspaceIntro section={section} tenant={tenant} t={t} />
      {section === 'overview' && <OverviewSection tenant={tenant} tenantsQuery={tenantsQuery} membersQuery={membersQuery} domainsQuery={domainsQuery} fundingQuery={fundingQuery} t={t} />}
      {section === 'users' && <UsersSection membersQuery={membersQuery} t={t} />}
      {section === 'domains' && <DomainsSection t={t} domainsQuery={domainsQuery} dnsRecordQuery={dnsRecordQuery} domainDialogOpen={domainDialogOpen} setDomainDialogOpen={setDomainDialogOpen} editDomain={editDomain} setEditDomain={setEditDomain} hostname={hostname} setHostname={setHostname} domainMutation={domainMutation} updateMutation={updateMutation} setVerifyDomain={setVerifyDomain} setDeleteTarget={setDeleteTarget} />}
      {(section === 'wallet' || section === 'transactions') && <WalletSection section={section} t={t} fundingQuery={fundingQuery} membersQuery={membersQuery} fundingOpen={fundingOpen} setFundingOpen={setFundingOpen} fundingMember={fundingMember} setFundingMember={setFundingMember} fundingAmount={fundingAmount} setFundingAmount={setFundingAmount} fundingReference={fundingReference} setFundingReference={setFundingReference} fundingMutation={fundingMutation} />}
    </div>
      <Dialog open={verifyDomain !== null} onOpenChange={(open) => !open && setVerifyDomain(null)}><DialogContent><DialogHeader><DialogTitle>{t('Verify domain')}</DialogTitle></DialogHeader><p className='text-muted-foreground text-sm'>{verifyDomain?.hostname}</p><p className='text-muted-foreground text-sm'>{t('We will check the domain DNS records against the configured target.')}</p><DialogFooter><Button onClick={() => verifyMutation.mutate()} disabled={verifyMutation.isPending}>{t('Verify')}</Button></DialogFooter></DialogContent></Dialog>
      <ConfirmDialog open={deleteTarget !== null} onOpenChange={(open) => !open && setDeleteTarget(null)} title={t('Delete domain')} desc={t('This permanently deletes the domain. It can be added again later.')} destructive handleConfirm={() => deleteMutation.mutate()} isLoading={deleteMutation.isPending} confirmText={t('Delete')} />
    </SectionPageLayout.Content>
  </SectionPageLayout>
}

function WorkspaceIntro(props: { section: PartnerWorkspaceSection; tenant: Tenant | null; t: TFunction }) {
  const titles: Record<PartnerWorkspaceSection, string> = { overview: 'Overview', users: 'Users', domains: 'Domains', wallet: 'Wallet', transactions: 'Funding transactions' }
  return <div className='border-border/70 bg-card flex flex-col gap-2 rounded-xl border px-5 py-4 shadow-sm sm:flex-row sm:items-center sm:justify-between'><div><p className='text-muted-foreground text-sm'>{props.t('Partner workspace')}</p><h1 className='text-xl font-semibold tracking-tight'>{props.t(titles[props.section])}</h1></div>{props.tenant && <p className='text-muted-foreground text-sm'>{props.tenant.name}</p>}</div>
}

type OverviewProps = { tenant: Tenant | null; tenantsQuery: UseQueryResult<Tenant[], Error>; membersQuery: UseQueryResult<TenantMember[], Error>; domainsQuery: UseQueryResult<TenantDomain[], Error>; fundingQuery: UseQueryResult<{ wallet: TenantWallet; items: TenantFunding[] }, Error>; t: TFunction }
function OverviewSection(props: OverviewProps) {
  if (props.tenantsQuery.isLoading) return <LoadingState />
  if (props.tenantsQuery.isError) return <ErrorState onRetry={() => void props.tenantsQuery.refetch()} />
  if (!props.tenant) return <EmptyState icon={Building2} title={props.t('No Tenants Found')} />
  const stats = [{ label: props.t('Users'), value: props.membersQuery.data?.length ?? 0, icon: Users }, { label: props.t('Domains'), value: props.domainsQuery.data?.length ?? 0, icon: Globe }, { label: props.t('Available'), value: `${formatNumber((props.fundingQuery.data?.wallet.available_minor ?? 0) / 100)} ${props.fundingQuery.data?.wallet.currency ?? 'USD'}`, icon: CircleDollarSign }]
  return <div className='space-y-6'><div className='grid gap-4 sm:grid-cols-3'>{stats.map((stat) => <Card key={stat.label}><CardContent className='flex items-center gap-4 p-5'><div className='bg-primary/10 text-primary rounded-lg p-2.5'><stat.icon className='size-5' aria-hidden='true' /></div><div><p className='text-muted-foreground text-sm'>{stat.label}</p><p className='text-2xl font-semibold'>{stat.value}</p></div></CardContent></Card>)}</div><Card><CardHeader><CardTitle>{props.t('Workspace at a glance')}</CardTitle></CardHeader><CardContent><p className='text-muted-foreground text-sm'>{props.t('Manage members, custom domains, and funding from one place.')}</p></CardContent></Card></div>
}

function UsersSection(props: { membersQuery: UseQueryResult<TenantMember[], Error>; t: TFunction }) {
  let content: ReactNode
  if (props.membersQuery.isLoading) content = <LoadingState />
  else if (props.membersQuery.isError) content = <ErrorState onRetry={() => void props.membersQuery.refetch()} />
  else if ((props.membersQuery.data ?? []).length === 0) content = <EmptyState icon={Users} title={props.t('No users found')} />
  else content = <div className='overflow-x-auto'><Table><TableHeader><TableRow><TableHead>{props.t('Account name')}</TableHead><TableHead>{props.t('Role')}</TableHead><TableHead>{props.t('Status')}</TableHead><TableHead>{props.t('Joined at')}</TableHead></TableRow></TableHeader><TableBody>{(props.membersQuery.data ?? []).map((member) => <TableRow key={member.id}><TableCell><div className='font-medium'>{member.display_name || member.username}</div><div className='text-muted-foreground text-xs'>{member.username}</div></TableCell><TableCell>{member.role_key}</TableCell><TableCell>{member.status}</TableCell><TableCell>{formatTimestampToDate(member.created_at)}</TableCell></TableRow>)}</TableBody></Table></div>
  return <Card><CardHeader><CardTitle className='flex items-center gap-2'><Users className='text-primary size-5' aria-hidden='true' />{props.t('Users')}</CardTitle></CardHeader><CardContent>{content}</CardContent></Card>
}

type DomainsProps = { t: TFunction; domainsQuery: UseQueryResult<TenantDomain[], Error>; dnsRecordQuery: UseQueryResult<TenantDNSRecord, Error>; domainDialogOpen: boolean; setDomainDialogOpen: (open: boolean) => void; editDomain: TenantDomain | null; setEditDomain: (domain: TenantDomain | null) => void; hostname: string; setHostname: (hostname: string) => void; domainMutation: UseMutationResult<{ domain: TenantDomain }, Error, void, unknown>; updateMutation: UseMutationResult<{ domain: TenantDomain }, Error, void, unknown>; setVerifyDomain: (domain: TenantDomain) => void; setDeleteTarget: (id: number) => void }
function DomainsSection(props: DomainsProps) {
  const action = props.editDomain ? props.updateMutation : props.domainMutation
  return <div className='grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(280px,360px)]'><Card className='min-w-0'><CardHeader className='flex flex-row items-center justify-between gap-3'><CardTitle className='flex items-center gap-2'><Globe className='text-primary size-5' aria-hidden='true' />{props.t('Domains')}</CardTitle><Dialog open={props.domainDialogOpen} onOpenChange={props.setDomainDialogOpen}><DialogTrigger render={<Button />} onClick={() => { props.setEditDomain(null); props.setHostname('') }}><Plus className='size-4' aria-hidden='true' />{props.t('Add domain')}</DialogTrigger><DialogContent><DialogHeader><DialogTitle>{props.editDomain ? props.t('Edit domain') : props.t('Add domain')}</DialogTitle></DialogHeader><Label htmlFor='partner-domain-hostname'>{props.t('Hostname')}</Label><Input id='partner-domain-hostname' value={props.hostname} onChange={(event) => props.setHostname(event.target.value)} placeholder='app.example.com' /><DialogFooter><Button onClick={() => action.mutate()} disabled={!props.hostname.trim() || action.isPending}>{props.editDomain ? props.t('Save') : props.t('Add')}</Button></DialogFooter></DialogContent></Dialog></CardHeader><CardContent>{props.domainsQuery.isLoading ? <LoadingState /> : <div className='overflow-x-auto'><Table><TableHeader><TableRow><TableHead>{props.t('Hostname')}</TableHead><TableHead>{props.t('Status')}</TableHead><TableHead className='text-end'>{props.t('Actions')}</TableHead></TableRow></TableHeader><TableBody>{(props.domainsQuery.data ?? []).map((domain) => <TableRow key={domain.id}><TableCell className='font-mono text-sm'>{domain.hostname}</TableCell><TableCell><span className='inline-flex items-center gap-1.5'><span className={`${domain.status === 'verified' ? 'bg-emerald-500' : 'bg-amber-500'} size-2 rounded-full`} aria-hidden='true' />{domain.status}</span></TableCell><TableCell><div className='flex justify-end gap-2'><Button size='sm' variant='outline' onClick={() => { props.setEditDomain(domain); props.setHostname(domain.hostname); props.setDomainDialogOpen(true) }}><Pencil className='size-4' aria-hidden='true' />{props.t('Edit')}</Button>{domain.status === 'pending' && <Button size='sm' variant='outline' onClick={() => props.setVerifyDomain(domain)}>{props.t('Verify')}</Button>}<Button variant='destructive' size='sm' onClick={() => props.setDeleteTarget(domain.id)}><Trash2 className='size-4' aria-hidden='true' />{props.t('Delete')}</Button></div></TableCell></TableRow>)}</TableBody></Table></div>}</CardContent></Card><DNSInstructions record={props.dnsRecordQuery.data} t={props.t} /></div>
}
function DNSInstructions(props: { record?: TenantDNSRecord; t: TFunction }) { return <Card className='h-fit'><CardHeader><CardTitle>{props.t('DNS records to configure')}</CardTitle></CardHeader><CardContent className='space-y-4 text-sm'><p className='text-muted-foreground'>{props.t('Point your domain to the following records, then verify it.')}</p>{props.record ? <div className='space-y-3'><div><p className='text-muted-foreground'>CNAME</p><code className='bg-muted block overflow-x-auto rounded-md p-2'>{props.record.cname_target}</code></div><div><p className='text-muted-foreground'>A</p><code className='bg-muted block overflow-x-auto rounded-md p-2'>{props.record.a_record_ip}</code></div></div> : <LoadingState />}</CardContent></Card> }

type WalletProps = { section: 'wallet' | 'transactions'; t: TFunction; fundingQuery: UseQueryResult<{ wallet: TenantWallet; items: TenantFunding[] }, Error>; membersQuery: UseQueryResult<TenantMember[], Error>; fundingOpen: boolean; setFundingOpen: (open: boolean) => void; fundingMember: string; setFundingMember: (member: string) => void; fundingAmount: string; setFundingAmount: (amount: string) => void; fundingReference: string; setFundingReference: (reference: string) => void; fundingMutation: UseMutationResult<void, Error, void, unknown> }
function WalletSection(props: WalletProps) {
  const isWallet = props.section === 'wallet'
  const wallet = props.fundingQuery.data?.wallet
  return <div className='space-y-6'>{isWallet && <div className='grid gap-4 sm:grid-cols-2'><Card className='bg-primary text-primary-foreground'><CardContent className='p-6'><p className='text-primary-foreground/70 text-sm'>{props.t('Partner owner platform wallet balance')}</p><p className='mt-2 text-3xl font-semibold'>{formatNumber((wallet?.available_minor ?? 0) / 100)} <span className='text-lg'>{wallet?.currency ?? 'USD'}</span></p><p className='text-primary-foreground/70 mt-2 text-xs'>{props.t('Funding is debited from the partner owner and credited to the member quota.')}</p></CardContent></Card><Card><CardContent className='flex h-full items-center gap-3 p-6'><CheckCircle2 className='text-emerald-500 size-6' aria-hidden='true' /><div><p className='font-medium'>{props.t('Funding transactions')}</p><p className='text-muted-foreground text-sm'>{props.t('Review the latest member funding activity.')}</p></div></CardContent></Card></div>}<Card><CardHeader className='flex flex-row items-center justify-between'><CardTitle className='flex items-center gap-2'><CircleDollarSign className='text-primary size-5' aria-hidden='true' />{isWallet ? props.t('Funding history') : props.t('Funding transactions')}</CardTitle>{isWallet && <FundingDialog props={props} />}</CardHeader><CardContent>{(props.fundingQuery.data?.items ?? []).length === 0 ? <EmptyState icon={CircleDollarSign} title={props.t('No funding transactions')} /> : <div className='overflow-x-auto'><Table><TableHeader><TableRow><TableHead>{props.t('Member')}</TableHead><TableHead>{props.t('Amount')}</TableHead><TableHead>{props.t('Reference')}</TableHead><TableHead>{props.t('Date')}</TableHead></TableRow></TableHeader><TableBody>{(props.fundingQuery.data?.items ?? []).map((item) => <TableRow key={item.id}><TableCell>{item.display_name || item.username}</TableCell><TableCell>{formatNumber(item.amount_minor / 100)} {item.currency}</TableCell><TableCell className='font-mono text-xs'>{item.reference}</TableCell><TableCell>{formatTimestampToDate(item.created_at)}</TableCell></TableRow>)}</TableBody></Table></div>}</CardContent></Card></div>
}
 function FundingDialog({ props }: { props: WalletProps }) { return <Dialog open={props.fundingOpen} onOpenChange={props.setFundingOpen}><DialogTrigger render={<Button />}><Plus className='size-4' aria-hidden='true' />{props.t('Fund member')}</DialogTrigger><DialogContent><DialogHeader><DialogTitle>{props.t('Fund member')}</DialogTitle></DialogHeader><div className='grid gap-4'><div className='grid gap-2'><Label htmlFor='partner-funding-member'>{props.t('Member')}</Label><Select value={props.fundingMember} onValueChange={(value) => props.setFundingMember(value ?? '')}><SelectTrigger id='partner-funding-member' className='w-full'><SelectValue placeholder={props.t('Select member')} /></SelectTrigger><SelectContent>{(props.membersQuery.data ?? []).filter((member) => member.status === 'active' && member.role_key !== 'owner').map((member) => <SelectItem key={member.user_id} value={String(member.user_id)}>{member.display_name || member.username}</SelectItem>)}</SelectContent></Select></div><div className='grid gap-2'><Label htmlFor='partner-funding-amount'>{props.t('Amount')}</Label><Input id='partner-funding-amount' type='number' min='0.01' step='0.01' value={props.fundingAmount} onChange={(event) => props.setFundingAmount(event.target.value)} /></div><div className='grid gap-2'><Label htmlFor='partner-funding-reference'>{props.t('Reference')}</Label><Input id='partner-funding-reference' value={props.fundingReference} onChange={(event) => props.setFundingReference(event.target.value)} /></div></div><DialogFooter><Button disabled={props.fundingMutation.isPending || !props.fundingMember || !props.fundingReference.trim() || Number(props.fundingAmount) <= 0} onClick={() => props.fundingMutation.mutate()}>{props.t('Fund member')}</Button></DialogFooter></DialogContent></Dialog> }

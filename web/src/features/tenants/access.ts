import type { AuthUser } from '@/stores/auth-store'

export function canViewTenantOverview(user: AuthUser | null): boolean {
  return user !== null && user.role >= 10 && user.is_partner !== true
}

export function canViewPartnerWorkspace(user: AuthUser | null): boolean {
  return user?.is_partner === true
}

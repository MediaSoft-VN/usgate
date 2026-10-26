import type { TFunction } from 'i18next'
import { Globe, LayoutDashboard, ReceiptText, Users, Wallet } from 'lucide-react'

import type { NavGroup, SidebarView } from '../types'

function getPartnerWorkspaceNavGroups(t: TFunction): NavGroup[] {
  return [
    {
      id: 'partner-workspace',
      title: t('Partner workspace'),
      items: [
        { title: t('Overview'), url: '/partner-workspace/overview', icon: LayoutDashboard },
        { title: t('Users'), url: '/partner-workspace/users', icon: Users },
        { title: t('Domains'), url: '/partner-workspace/domains', icon: Globe },
        { title: t('Wallet'), url: '/partner-workspace/wallet', icon: Wallet },
        { title: t('Funding transactions'), url: '/partner-workspace/transactions', icon: ReceiptText },
      ],
    },
  ]
}

export const PARTNER_WORKSPACE_VIEW: SidebarView = {
  id: 'partner-workspace',
  pathPattern: /^\/partner-workspace(\/|$)/,
  parent: {
    to: '/dashboard/overview',
    label: 'Back to Dashboard',
  },
  getNavGroups: getPartnerWorkspaceNavGroups,
}

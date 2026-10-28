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
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { SettingsCard } from '../components/settings-card'

type SePayCredentialGroupProps = {
  mode: 'sandbox' | 'live'
  children: ReactNode
}

export function SePayCredentialGroup(props: SePayCredentialGroupProps) {
  const { t } = useTranslation()
  const isSandbox = props.mode === 'sandbox'
  const title = isSandbox ? t('Sandbox credentials') : t('Live credentials')
  const highlightClassName = isSandbox
    ? 'border-sky-300 bg-sky-50/60 dark:border-sky-900 dark:bg-sky-950/30'
    : 'border-emerald-300 bg-emerald-50/60 dark:border-emerald-900 dark:bg-emerald-950/30'

  return (
    <section aria-label={title}>
      <SettingsCard
        title={title}
        className={`${highlightClassName} shadow-none`}
      >
        <div className='grid gap-4'>{props.children}</div>
      </SettingsCard>
    </section>
  )
}

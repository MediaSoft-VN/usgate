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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()

  return (
    <div className='relative grid h-svh max-w-none overflow-hidden bg-background lg:grid-cols-[minmax(0,1fr)_minmax(420px,560px)]'>
      <div
        aria-hidden='true'
        className='relative hidden overflow-hidden border-e border-border/60 bg-[linear-gradient(145deg,oklch(0.22_0.08_260),oklch(0.34_0.14_250)_48%,oklch(0.26_0.1_190))] lg:block'
      >
        <img
          src='https://www.usgate.ai/assets/hero-banner-v4.png'
          alt=''
          aria-hidden='true'
          className='absolute inset-0 size-full object-cover opacity-90'
          decoding='async'
          fetchPriority='high'
        />
        <div className='absolute inset-0 bg-slate-950/35' />
        <div className='absolute -top-40 -start-32 size-[32rem] rounded-full bg-primary/30 blur-3xl' />
        <div className='absolute -end-40 bottom-0 size-[28rem] rounded-full bg-chart-2/25 blur-3xl' />
        <div className='absolute inset-0 bg-[linear-gradient(120deg,transparent_0%,oklch(1_0_0/0.08)_48%,transparent_100%)]' />
        <div className='absolute inset-x-16 top-1/2 h-px bg-white/20' />
        <div className='absolute inset-y-24 start-1/2 w-px bg-white/15' />
      </div>
      <Link
        to='/'
        className='absolute top-4 left-4 z-10 flex items-center gap-2 transition-opacity hover:opacity-80 sm:top-8 sm:left-8'
      >
        <div className='relative h-8 w-8'>
          {loading ? (
            <Skeleton className='absolute inset-0 rounded-full' />
          ) : (
            <img
              src={logo}
              alt={t('Logo')}
              className='h-8 w-8 rounded-full object-cover'
            />
          )}
        </div>
        {loading ? (
          <Skeleton className='h-6 w-24' />
        ) : (
          <h1 className='text-xl font-medium'>{systemName}</h1>
        )}
      </Link>
      <div className='container flex items-center bg-background/80 pt-16 backdrop-blur-xl sm:pt-0'>
        <div className='mx-auto flex w-full flex-col justify-center space-y-2 px-4 py-8 sm:w-[480px] sm:p-8'>
          {children}
        </div>
      </div>
    </div>
  )
}

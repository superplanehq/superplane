import clsx from 'clsx'
import type React from 'react'

const sizes = {
  xs: 'sm:max-w-xs',
  sm: 'sm:max-w-sm',
  md: 'sm:max-w-md',
  lg: 'sm:max-w-lg',
  xl: 'sm:max-w-xl',
  '2xl': 'sm:max-w-2xl',
  '3xl': 'sm:max-w-3xl',
  '4xl': 'sm:max-w-4xl',
  '5xl': 'sm:max-w-5xl',
}

export function Dialog({
  size = 'lg',
  className,
  children,
  open,
  onClose,
  ...props
}: {
  size?: keyof typeof sizes
  className?: string
  children: React.ReactNode
  open: boolean
  onClose: () => void
} & React.ComponentPropsWithoutRef<'div'>) {
  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="fixed inset-0 bg-zinc-950/25 dark:bg-zinc-950/50"
        onClick={onClose}
      />
      <div
        className={clsx(
          className,
          sizes[size],
          'relative w-full min-w-0 rounded-2xl bg-white p-8 shadow-lg ring-1 ring-zinc-950/10 dark:bg-zinc-900 dark:ring-white/10',
          'overflow-y-auto max-h-[100vh]'
        )}
        {...props}
      >
        {children}
      </div>
    </div>
  )
}

export function DialogTitle({
  className,
  ...props
}: React.ComponentPropsWithoutRef<'h2'>) {
  return (
    <h2
      {...props}
      className={clsx(className, 'text-lg/6 font-semibold text-balance text-zinc-950 sm:text-base/6 dark:text-white')}
    />
  )
}

export function DialogDescription({
  className,
  ...props
}: React.ComponentPropsWithoutRef<'div'>) {
  return <div {...props} className={clsx(className, 'mt-2 text-pretty text-zinc-600 dark:text-zinc-400')} />
}

export function DialogBody({ className, ...props }: React.ComponentPropsWithoutRef<'div'>) {
  return <div {...props} className={clsx(className, 'mt-6')} />
}

export function DialogActions({ className, ...props }: React.ComponentPropsWithoutRef<'div'>) {
  return (
    <div
      {...props}
      className={clsx(
        className,
        'mt-8 flex flex-col-reverse items-center justify-end gap-3 *:w-full sm:flex-row sm:*:w-auto'
      )}
    />
  )
}
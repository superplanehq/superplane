import clsx from 'clsx'
import type React from 'react'

export function CheckboxGroup({ className, ...props }: React.ComponentPropsWithoutRef<'div'>) {
  return (
    <div
      data-slot="control"
      {...props}
      className={clsx(
        className,
        'space-y-3',
        'has-data-[slot=description]:space-y-6 has-data-[slot=description]:**:data-[slot=label]:font-medium'
      )}
    />
  )
}

export function CheckboxField({
  className,
  onClick,
  ...props
}: React.ComponentPropsWithoutRef<'div'> & {
  onClick?: () => void
}) {
  return (
    <div
      data-slot="field"
      {...props}
      onClick={onClick}
      className={clsx(
        className,
        'grid grid-cols-[1.125rem_1fr] gap-x-4 gap-y-1 sm:grid-cols-[1rem_1fr] cursor-pointer',
        '*:data-[slot=control]:col-start-1 *:data-[slot=control]:row-start-1 *:data-[slot=control]:mt-0.75 sm:*:data-[slot=control]:mt-1',
        '*:data-[slot=label]:col-start-2 *:data-[slot=label]:row-start-1',
        '*:data-[slot=description]:col-start-2 *:data-[slot=description]:row-start-2',
        'has-data-[slot=description]:**:data-[slot=label]:font-medium'
      )}
    />
  )
}

export function Checkbox({
  className,
  checked,
  onChange,
  ...props
}: {
  className?: string
  checked?: boolean
  onChange?: (checked: boolean) => void
} & Omit<React.ComponentPropsWithoutRef<'input'>, 'type' | 'checked' | 'onChange'>) {
  return (
    <span
      data-slot="control"
      className={clsx(className, 'group inline-flex focus:outline-hidden')}
    >
      <span 
        className={clsx([
          'relative isolate flex size-4.5 items-center justify-center rounded-[0.3125rem] sm:size-4 cursor-pointer',
          'before:absolute before:inset-0 before:-z-10 before:rounded-[calc(0.3125rem-1px)] before:bg-white before:shadow-sm',
          'dark:before:hidden',
          'dark:bg-white/5',
          'border border-zinc-950/15 hover:border-zinc-950/30',
          'dark:border-white/15 dark:hover:border-white/30',
          'after:absolute after:inset-0 after:rounded-[calc(0.3125rem-1px)] after:shadow-[inset_0_1px_theme(colors.white/15%)]',
          'dark:after:-inset-px dark:after:hidden dark:after:rounded-[0.3125rem]',
          'focus:outline-2 focus:outline-offset-2 focus:outline-blue-500',
          checked && [
            'before:bg-zinc-900 border-transparent',
            'dark:bg-zinc-600 dark:border-white/5',
            'dark:after:block'
          ]
        ])}
        onClick={() => {
          onChange?.(!checked)
        }}
      >
        <input
          {...props}
          type="checkbox"
          checked={checked}
          onChange={(e) => {
            onChange?.(e.target.checked)
          }}
          className="absolute inset-0 opacity-0 cursor-pointer z-10"
        />
        <svg
          className={clsx(
            'size-4 stroke-white opacity-0 sm:h-3.5 sm:w-3.5',
            checked && 'opacity-100'
          )}
          viewBox="0 0 14 14"
          fill="none"
        >
          <path
            d="M3 8L6 11L11 3.5"
            strokeWidth={2}
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </span>
    </span>
  )
}
'use client'

import * as Headless from '@headlessui/react'
import clsx from 'clsx'
import { useState, useRef, useEffect } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'

export function MultiCombobox<T extends { id: string }>({
  options,
  displayValue,
  filter,
  anchor = 'bottom',
  className,
  placeholder,
  autoFocus,
  'aria-label': ariaLabel,
  showButton = false,
  children,
  value = [],
  onChange,
  onRemove,
  allowCustomValues = false,
  createCustomValue,
  validateValue,
  validateInput,
  ...props
}: {
  options: T[]
  displayValue: (value: T) => string
  filter?: (value: T, query: string) => boolean
  className?: string
  placeholder?: string
  autoFocus?: boolean
  'aria-label'?: string
  showButton?: boolean
  children: (value: T, isSelected: boolean) => React.ReactElement
  value?: T[]
  onChange?: (values: T[]) => void
  onRemove?: (value: T) => void
  allowCustomValues?: boolean
  createCustomValue?: (query: string) => T
  validateValue?: (value: T) => boolean
  validateInput?: (input: string) => boolean
} & Omit<Headless.ComboboxProps<T, false>, 'as' | 'multiple' | 'children' | 'value' | 'onChange' | 'defaultValue' | 'by' | 'virtual'> & { anchor?: 'top' | 'bottom' }) {
  const [query, setQuery] = useState('')
  const [isOpen, setIsOpen] = useState(false)
  const [justAddedTag, setJustAddedTag] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => {
        document.removeEventListener('mousedown', handleClickOutside)
      }
    }
  }, [isOpen])

  const filteredOptions = query === ''
    ? options.filter(option => !value.some(selected => selected.id === option.id))
    : options
        .filter(option => !value.some(selected => selected.id === option.id))
        .filter((option) =>
          filter ? filter(option, query) : displayValue(option)?.toLowerCase().includes(query.toLowerCase())
        )

  const customEmailSuggestion = allowCustomValues &&
    query.trim() !== '' &&
    validateInput && validateInput(query.trim()) &&
    createCustomValue &&
    !options.some(option => displayValue(option).toLowerCase() === query.toLowerCase()) &&
    !value.some(selected => displayValue(selected).toLowerCase() === query.toLowerCase())
    ? [createCustomValue(query.trim())]
    : []

  const allOptions = [...filteredOptions, ...customEmailSuggestion]
  const hasMatches = allOptions.length > 0
  const canCreateCustomValue = allowCustomValues &&
    query.trim() !== '' &&
    !hasMatches &&
    customEmailSuggestion.length === 0 &&
    createCustomValue

  const handleSelect = (selectedOption: T) => {
    if (!selectedOption) return

    const newValues = [...value, selectedOption]
    onChange?.(newValues)
    setQuery('')
    setJustAddedTag(true)
    setIsOpen(false)
    setTimeout(() => {
      inputRef.current?.focus()
    }, 0)
  }

  const handleCreateCustomValue = () => {
    if (!canCreateCustomValue) return

    if (validateInput && !validateInput(query.trim())) {
      return
    }

    const customValue = createCustomValue!(query.trim())
    const newValues = [...value, customValue]
    onChange?.(newValues)
    setQuery('')
    setJustAddedTag(true)
    setIsOpen(false)
    setTimeout(() => {
      inputRef.current?.focus()
    }, 0)
  }

  const handleRemove = (option: T) => {
    const newValues = value.filter(v => v.id !== option.id)
    onChange?.(newValues)
    onRemove?.(option)
  }

  const handleInputClick = () => {
    setIsOpen(true)
  }

  const handleInputFocus = () => {
    if (value.length === 0 || !justAddedTag) {
      setIsOpen(true)
    }
    setJustAddedTag(false)
  }

  const handleInputBlur = () => {
    setTimeout(() => {
      if (canCreateCustomValue && query.trim() !== '') {
        if (!validateInput || validateInput(query.trim())) {
          handleCreateCustomValue()
        }
      }
    }, 100)
  }

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Backspace' && query === '' && value.length > 0) {
      handleRemove(value[value.length - 1])
    }

    if (event.key === 'Enter' && canCreateCustomValue) {
      event.preventDefault()
      if (!validateInput || validateInput(query.trim())) {
        handleCreateCustomValue()
      }
    }
  }

  return (
    <div ref={containerRef} className='w-full relative'>
      <Headless.Combobox
        {...props}
        multiple={false}
        value={null}
        onChange={(selectedOption: T | null) => {
          if (selectedOption) {
            handleSelect(selectedOption)
          }
        }}
        onClose={() => {setQuery(''); setIsOpen(false)}}
        immediate
      >
        <span
          data-slot="control"
          className={clsx([
            className,
          'relative block w-full',
          'before:absolute before:inset-px before:rounded-[calc(var(--radius-lg)-1px)] before:bg-white before:shadow-sm',
          'dark:before:hidden',
          'after:pointer-events-none after:absolute after:inset-0 after:rounded-lg after:ring-transparent after:ring-inset sm:focus-within:after:ring-2 sm:focus-within:after:ring-blue-500',
          'has-data-disabled:opacity-50 has-data-disabled:before:bg-zinc-950/5 has-data-disabled:before:shadow-none',
          'has-data-invalid:before:shadow-red-500/10',
        ])}
      >
        <div
          className={clsx([
            'relative flex flex-wrap items-center gap-1 w-full appearance-none rounded-lg py-[calc(--spacing(2.5)-1px)] sm:py-[calc(--spacing(1.5)-1px)]',
            'pr-[calc(--spacing(10)-1px)] pl-[calc(--spacing(3.5)-1px)] sm:pr-[calc(--spacing(9)-1px)] sm:pl-[calc(--spacing(3)-1px)]',
            'text-base/6 text-zinc-950 sm:text-sm/6 dark:text-white',
            'border border-zinc-950/10 hover:border-zinc-950/20 dark:border-white/10 dark:hover:border-white/20',
            'bg-transparent dark:bg-white/5',
            'focus-within:border-blue-500 dark:focus-within:border-blue-400',
            'data-invalid:border-red-500 data-invalid:hover:border-red-500 dark:data-invalid:border-red-500 dark:data-invalid:hover:border-red-500',
            'data-disabled:border-zinc-950/20 dark:data-disabled:border-white/15 dark:data-disabled:bg-white/2.5',
            'dark:scheme-dark',
            'cursor-text',
          ])}
          onClick={handleInputClick}
        >
          {value.filter(option => option && option.id).map((option) => {
            const isValid = validateValue ? validateValue(option) : true
            return (
              <span
                key={option.id}
                className={clsx(
                  'inline-flex items-center gap-1 px-1 rounded-md text-xs',
                  isValid ? (
                    'bg-zinc-50 text-zinc-700 border border-zinc-200 dark:bg-zinc-800/20 dark:text-zinc-300 dark:border-zinc-800'
                  ) : (
                    'bg-red-50 text-red-700 border border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
                  )
                )}
              >
                {children(option, true)}
                {!isValid && (
                  <MaterialSymbol name="warning" size="sm" className="text-red-500 dark:text-red-400" />
                )}
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleRemove(option)
                  }}
                  className="ml-1 hover:bg-zinc-50 dark:hover:bg-zinc-800/30 rounded transition-colors"
                >
                  <MaterialSymbol name="close" size="sm" />
                </button>
              </span>
            )
          })}

          <Headless.ComboboxInput
            ref={inputRef}
            autoFocus={autoFocus}
            data-slot="control"
            aria-label={ariaLabel}
            value={query}
            displayValue={() => ''}
            onChange={(event) => {
              setQuery(event.target.value)
              if (event.target.value.length > 0) {
                setIsOpen(true)
              }
            }}
            onKeyDown={handleKeyDown}
            onFocus={handleInputFocus}
            onBlur={handleInputBlur}
            onClick={handleInputClick}
            placeholder={value.length === 0 ? placeholder : ''}
            className={clsx([
              'flex-grow-1 min-w-[120px] border-none outline-none bg-transparent',
              'text-base/6 text-zinc-950 placeholder:text-zinc-500 sm:text-sm/6 dark:text-white dark:placeholder:text-zinc-400',
            ])}
          />
        </div>
      {showButton && (
         <Headless.ComboboxButton className="group absolute inset-y-0 right-0 flex items-center px-2">
          <MaterialSymbol name="expand_more" size="sm" />
        </Headless.ComboboxButton>
      )}

      </span>

      {isOpen && (query === '' || hasMatches) && (
        <Headless.ComboboxOptions
          static
          className={clsx(
            'absolute top-full left-0 right-0 z-10 mt-1',
            'scroll-py-1 rounded-xl p-1 select-none empty:invisible w-full',
            'outline outline-transparent focus:outline-hidden',
            'max-h-60 overflow-y-auto overscroll-contain',
            'bg-white dark:bg-zinc-800',
            'shadow-lg ring-1 ring-zinc-950/10 dark:ring-white/10',
            'transition-opacity duration-100 ease-in'
          )}
        >
          {allOptions.filter(option => option && option.id).map((option) => (
            <MultiComboboxOption key={option.id} value={option}>
              {children(option, false)}
            </MultiComboboxOption>
          ))}
        </Headless.ComboboxOptions>
      )}
      </Headless.Combobox>
    </div>
  )
}

export function MultiComboboxOption<T>({
  children,
  className,
  ...props
}: { className?: string; children?: React.ReactNode } & Omit<
  Headless.ComboboxOptionProps<'div', T>,
  'as' | 'className'
>) {
  const sharedClasses = clsx(
    'flex min-w-0 items-center',
    '*:data-[slot=icon]:size-5 *:data-[slot=icon]:shrink-0 sm:*:data-[slot=icon]:size-4',
    '*:data-[slot=icon]:text-zinc-500 group-data-focus/option:*:data-[slot=icon]:text-white dark:*:data-[slot=icon]:text-zinc-400',
    'forced-colors:*:data-[slot=icon]:text-[CanvasText] forced-colors:group-data-focus/option:*:data-[slot=icon]:text-[Canvas]',
    '*:data-[slot=avatar]:-mx-0.5 *:data-[slot=avatar]:size-6 sm:*:data-[slot=avatar]:size-5'
  )

  return (
    <Headless.ComboboxOption
      {...props}
      className={clsx(
        'group/option grid w-full cursor-default grid-cols-[1fr_--spacing(5)] items-baseline gap-x-2 rounded-lg py-2.5 pr-2 pl-3.5 sm:grid-cols-[1fr_--spacing(4)] sm:py-1.5 sm:pr-2 sm:pl-3',
        'text-base/6 text-zinc-950 sm:text-sm/6 dark:text-white forced-colors:text-[CanvasText]',
        'outline-hidden data-focus:bg-blue-500 data-focus:text-white',
        'forced-color-adjust-none forced-colors:data-focus:bg-[Highlight] forced-colors:data-focus:text-[HighlightText]',
        'data-disabled:opacity-50'
      )}
    >
      <span className={clsx(className, sharedClasses)}>{children}</span>
    </Headless.ComboboxOption>
  )
}

export function MultiComboboxLabel({ className, ...props }: React.ComponentPropsWithoutRef<'span'>) {
  return <span {...props} className={clsx(className, 'ml-2.5 truncate first:ml-0 sm:ml-2 sm:first:ml-0')} />
}

export function MultiComboboxDescription({ className, children, ...props }: React.ComponentPropsWithoutRef<'span'>) {
  return (
    <span
      {...props}
      className={clsx(
        className,
        'flex flex-1 overflow-hidden text-zinc-500 group-data-focus/option:text-white before:w-2 before:min-w-0 before:shrink dark:text-zinc-400'
      )}
    >
      <span className="flex-1 truncate">{children}</span>
    </span>
  )
}
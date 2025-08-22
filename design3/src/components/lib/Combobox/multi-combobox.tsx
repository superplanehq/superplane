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

  // Handle clicking outside to close dropdown
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

  // Add custom email suggestion if query is a valid email and allowCustomValues is true
  const customEmailSuggestion = allowCustomValues && 
    query.trim() !== '' && 
    validateInput && validateInput(query.trim()) && // Use proper email validation
    createCustomValue &&
    !options.some(option => displayValue(option).toLowerCase() === query.toLowerCase()) &&
    !value.some(selected => displayValue(selected).toLowerCase() === query.toLowerCase())
    ? [createCustomValue(query.trim())]
    : []

  const allOptions = [...filteredOptions, ...customEmailSuggestion]
  const hasMatches = allOptions.length > 0
  // Allow custom value creation for non-email inputs (like names) when no matches and no email suggestion
  const canCreateCustomValue = allowCustomValues && 
    query.trim() !== '' && 
    !hasMatches && 
    customEmailSuggestion.length === 0 && 
    createCustomValue

  const handleSelect = (selectedOption: T) => {
    console.log('handleSelect called with:', selectedOption)
    console.log('current value:', value)
    if (!selectedOption) return
    
    const newValues = [...value, selectedOption]
    console.log('new values:', newValues)
    onChange?.(newValues)
    setQuery('')
    setJustAddedTag(true)
    setIsOpen(false)
    // Keep input focused
    setTimeout(() => {
      inputRef.current?.focus()
    }, 0)
  }

  const handleCreateCustomValue = () => {
    if (!canCreateCustomValue) return
    
    // Validate input before creating custom value
    if (validateInput && !validateInput(query.trim())) {
      // Keep the text in input for user to correct
      return
    }
    
    const customValue = createCustomValue!(query.trim())
    const newValues = [...value, customValue]
    onChange?.(newValues)
    setQuery('')
    setJustAddedTag(true)
    setIsOpen(false)
    // Keep input focused
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
    // Show list immediately if no items selected, or if we haven't just added a tag
    if (value.length === 0 || !justAddedTag) {
      setIsOpen(true)
    }
    // Reset the flag after focus
    setJustAddedTag(false)
  }

  const handleInputBlur = () => {
    // Small delay to allow click events to register before blur
    setTimeout(() => {
      if (canCreateCustomValue && query.trim() !== '') {
        // Only create custom value on blur if input is valid
        if (!validateInput || validateInput(query.trim())) {
          handleCreateCustomValue()
        }
        // If invalid, keep text in input (don't call handleCreateCustomValue)
      }
    }, 100)
  }

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Backspace' && query === '' && value.length > 0) {
      handleRemove(value[value.length - 1])
    }
    
    if (event.key === 'Enter' && canCreateCustomValue) {
      event.preventDefault()
      // Only create if input passes validation
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
          // Basic layout
          'relative block w-full',
          // Background color + shadow applied to inset pseudo element, so shadow blends with border in light mode
          'before:absolute before:inset-px before:rounded-[calc(var(--radius-lg)-1px)] before:bg-white before:shadow-sm',
          // Background color is moved to control and shadow is removed in dark mode so hide `before` pseudo
          'dark:before:hidden',
          // Focus ring
          'after:pointer-events-none after:absolute after:inset-0 after:rounded-lg after:ring-transparent after:ring-inset sm:focus-within:after:ring-2 sm:focus-within:after:ring-blue-500',
          // Disabled state
          'has-data-disabled:opacity-50 has-data-disabled:before:bg-zinc-950/5 has-data-disabled:before:shadow-none',
          // Invalid state
          'has-data-invalid:before:shadow-red-500/10',
        ])}
      >
        <div
          className={clsx([
            // Basic layout
            'relative flex flex-wrap items-center gap-1 w-full appearance-none rounded-lg py-[calc(--spacing(2.5)-1px)] sm:py-[calc(--spacing(1.5)-1px)]',
            // Horizontal padding
            'pr-[calc(--spacing(10)-1px)] pl-[calc(--spacing(3.5)-1px)] sm:pr-[calc(--spacing(9)-1px)] sm:pl-[calc(--spacing(3)-1px)]',
            // Typography
            'text-base/6 text-zinc-950 sm:text-sm/6 dark:text-white',
            // Border
            'border border-zinc-950/10 hover:border-zinc-950/20 dark:border-white/10 dark:hover:border-white/20',
            // Background color
            'bg-transparent dark:bg-white/5',
            // Focus styles
            'focus-within:border-blue-500 dark:focus-within:border-blue-400',
            // Invalid state
            'data-invalid:border-red-500 data-invalid:hover:border-red-500 dark:data-invalid:border-red-500 dark:data-invalid:hover:border-red-500',
            // Disabled state
            'data-disabled:border-zinc-950/20 dark:data-disabled:border-white/15 dark:data-disabled:bg-white/2.5',
            // System icons
            'dark:scheme-dark',
            // Cursor
            'cursor-text',
          ])}
          onClick={handleInputClick}
        >
          {/* Selected Tags */}
          {value.filter(option => option && option.id).map((option) => {
            const isValid = validateValue ? validateValue(option) : true
            return (
              <span
                key={option.id}
                className={clsx(
                  'inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs',
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
                  className="ml-1 hover:bg-zinc-50 dark:hover:bg-zinc-800/30 rounded p-0.5 transition-colors"
                >
                  <MaterialSymbol name="close" size="sm" />
                </button>
              </span>
            )
          })}

          {/* Input */}
          <Headless.ComboboxInput
            ref={inputRef}
            autoFocus={autoFocus}
            data-slot="control"
            aria-label={ariaLabel}
            value={query}
            displayValue={() => ''}
            onChange={(event) => {
              setQuery(event.target.value)
              // Show dropdown when user starts typing
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
              // Basic layout
              'flex-grow-1 min-w-[120px] border-none outline-none bg-transparent',
              // Typography
              'text-base/6 text-zinc-950 placeholder:text-zinc-500 sm:text-sm/6 dark:text-white dark:placeholder:text-zinc-400',
            ])}
          />
        </div>

        <Headless.ComboboxButton className="group absolute inset-y-0 right-0 flex items-center px-2">
          <svg
            className="size-5 stroke-zinc-500 group-data-disabled:stroke-zinc-600 group-data-hover:stroke-zinc-700 sm:size-4 dark:stroke-zinc-400 dark:group-data-hover:stroke-zinc-300 forced-colors:stroke-[CanvasText]"
            viewBox="0 0 16 16"
            aria-hidden="true"
            fill="none"
          >
            <path d="M5.75 10.75L8 13L10.25 10.75" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
            <path d="M10.25 5.25L8 3L5.75 5.25" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </Headless.ComboboxButton>
      </span>

      {isOpen && (query === '' || hasMatches) && (
        <Headless.ComboboxOptions
          static
          className={clsx(
            // Positioning - absolute to parent container, full width
            'absolute top-full left-0 right-0 z-10 mt-1',
            // Base styles
            'scroll-py-1 rounded-xl p-1 select-none empty:invisible w-full',
            // Invisible border that is only visible in `forced-colors` mode for accessibility purposes
            'outline outline-transparent focus:outline-hidden',
            // Handle scrolling when menu won't fit in viewport
            'max-h-60 overflow-y-auto overscroll-contain',
            // Popover background
            'bg-white dark:bg-zinc-800',
            // Shadows
            'shadow-lg ring-1 ring-zinc-950/10 dark:ring-white/10',
            // Transitions
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
  let sharedClasses = clsx(
    // Base
    'flex min-w-0 items-center',
    // Icons
    '*:data-[slot=icon]:size-5 *:data-[slot=icon]:shrink-0 sm:*:data-[slot=icon]:size-4',
    '*:data-[slot=icon]:text-zinc-500 group-data-focus/option:*:data-[slot=icon]:text-white dark:*:data-[slot=icon]:text-zinc-400',
    'forced-colors:*:data-[slot=icon]:text-[CanvasText] forced-colors:group-data-focus/option:*:data-[slot=icon]:text-[Canvas]',
    // Avatars
    '*:data-[slot=avatar]:-mx-0.5 *:data-[slot=avatar]:size-6 sm:*:data-[slot=avatar]:size-5'
  )

  return (
    <Headless.ComboboxOption
      {...props}
      className={clsx(
        // Basic layout
        'group/option grid w-full cursor-default grid-cols-[1fr_--spacing(5)] items-baseline gap-x-2 rounded-lg py-2.5 pr-2 pl-3.5 sm:grid-cols-[1fr_--spacing(4)] sm:py-1.5 sm:pr-2 sm:pl-3',
        // Typography
        'text-base/6 text-zinc-950 sm:text-sm/6 dark:text-white forced-colors:text-[CanvasText]',
        // Focus
        'outline-hidden data-focus:bg-blue-500 data-focus:text-white',
        // Forced colors mode
        'forced-color-adjust-none forced-colors:data-focus:bg-[Highlight] forced-colors:data-focus:text-[HighlightText]',
        // Disabled
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
import { useState, useRef, useEffect } from 'react'
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol'
import { twMerge } from 'tailwind-merge'

export interface TaggedInputOption {
  id: string
  label: string
  value: string
  description?: string
}

export interface TaggedInputProps {
  value: string
  onChange: (value: string) => void
  options: TaggedInputOption[]
  placeholder?: string
  className?: string
  error?: boolean
  disabled?: boolean
  variant?: 'default' | 'inline' // New prop to control display style
}

interface ParsedSegment {
  type: 'text' | 'tag'
  content: string
  originalValue?: string // For tags, the original ${{ }} value
}

let timeoutId: NodeJS.Timeout | undefined

export function TaggedInput({
  value = '',
  onChange,
  options = [],
  placeholder = 'Enter value...',
  className,
  error = false,
  disabled = false,
  variant = 'default'
}: TaggedInputProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [cursorPosition, setCursorPosition] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const parseValue = (inputValue: string): ParsedSegment[] => {
    const segments: ParsedSegment[] = []
    const regex = /\$\{\{\s*([^}]+)\s*\}\}/g
    let lastIndex = 0
    let match

    while ((match = regex.exec(inputValue)) !== null) {

      if (match.index > lastIndex) {
        const textContent = inputValue.slice(lastIndex, match.index)
        if (textContent) {
          segments.push({
            type: 'text',
            content: textContent
          })
        }
      }


      segments.push({
        type: 'tag',
        content: match[1].trim(),
        originalValue: match[0]
      })

      lastIndex = regex.lastIndex
    }


    if (lastIndex < inputValue.length) {
      const remainingText = inputValue.slice(lastIndex)
      if (remainingText) {
        segments.push({
          type: 'text',
          content: remainingText
        })
      }
    }

    return segments
  }

  const segments = parseValue(value)

  const filteredOptions = query === ''
    ? options
    : options.filter(option =>
      option.label.toLowerCase().includes(query.toLowerCase()) ||
      option.value.toLowerCase().includes(query.toLowerCase())
    )

  const isTypingExpression = (newValue: string) => {
    const beforeCursor = newValue.slice(0, cursorPosition + 1)
    const cursorChar = beforeCursor.slice(-1)
    const lastOpenBrace = beforeCursor.lastIndexOf('${')
    const lastCloseBrace = beforeCursor.lastIndexOf('}}')
    return lastOpenBrace > lastCloseBrace && lastOpenBrace !== -1 || cursorChar === '$';
  }

  const getCurrentExpression = () => {
    const beforeCursor = value.slice(0, cursorPosition)
    const lastOpenBrace = beforeCursor.lastIndexOf('${{')
    if (lastOpenBrace === -1) return ''
    return beforeCursor.slice(lastOpenBrace + 3).trim()
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value
    const newCursorPosition = e.target.selectionStart || 0

    onChange(newValue)
    setCursorPosition(newCursorPosition)
    setQuery(getCurrentExpression())


    if (isTypingExpression(newValue) || variant === 'inline') {
      if (timeoutId) {
        clearTimeout(timeoutId)
      }
      setIsOpen(true)
    } else {
      setIsOpen(false)
    }
  }

  const handleOptionSelect = (option: TaggedInputOption) => {
    const beforeCursor = value.slice(0, cursorPosition)
    const afterCursor = value.slice(cursorPosition)
    const lastOpenBrace = beforeCursor.lastIndexOf('${{')

    let newValue: string
    let newCursorPos: number

    if (lastOpenBrace !== -1 && lastOpenBrace > beforeCursor.lastIndexOf('}}')) {

      newValue = beforeCursor.slice(0, lastOpenBrace) + option.value + afterCursor
      newCursorPos = lastOpenBrace + option.value.length
    } else {

      newValue = beforeCursor + option.value + afterCursor
      newCursorPos = cursorPosition + option.value.length
    }

    onChange(newValue)
    setIsOpen(false)
    setQuery('')


    setTimeout(() => {
      if (inputRef.current) {
        inputRef.current.setSelectionRange(newCursorPos, newCursorPos)
        setCursorPosition(newCursorPos)
        inputRef.current.focus()
      }
    }, 0)
  }

  const showInputOptions = () => {

    if (timeoutId) {
      clearTimeout(timeoutId)
    }


    setIsOpen(true)
    setQuery('') // Show all options
    if (inputRef.current) {
      inputRef.current.focus()
    }
  }

  const handleInputFocus = () => {
    if (timeoutId) {
      clearTimeout(timeoutId)
    }


    if (variant === 'inline' && value.endsWith('$')) {
      setIsOpen(true)
      setQuery('')
    }
  }

  const handleInputBlur = () => {
    if (timeoutId) {
      clearTimeout(timeoutId)
    }
    timeoutId = setTimeout(() => {
      setIsOpen(false)
    }, 200)
  }

  const removeTag = (tagIndex: number) => {
    const tagSegments = segments.filter(s => s.type === 'tag')
    if (tagIndex < tagSegments.length) {
      const tagToRemove = tagSegments[tagIndex]
      const newValue = value.replace(tagToRemove.originalValue || '', '')
      onChange(newValue)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      setIsOpen(false)
      setQuery('')
    } else if (e.key === 'Enter' && isOpen && filteredOptions.length > 0) {
      e.preventDefault()
      handleOptionSelect(filteredOptions[0])
    } else if (e.key === 'ArrowDown' && isOpen) {
      e.preventDefault()

    }
  }

  const handleSelectionChange = () => {
    if (inputRef.current) {
      setCursorPosition(inputRef.current.selectionStart || 0 - value.indexOf('${{'))
    }
  }

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        !inputRef.current?.contains(event.target as Node)
      ) {
        setIsOpen(false)
        setQuery('')
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  return (
    <div className="relative">
      {/* Tag Display (shows parsed segments as visual tags) - only for default variant */}
      {variant === 'default' && segments.length > 0 && segments.some(s => s.type === 'tag') && (
        <div className="flex flex-wrap gap-1 mb-2 p-2 bg-zinc-50 dark:bg-zinc-800/50 border border-zinc-200 dark:border-zinc-700 rounded">
          {segments.map((segment, index) => {
            if (segment.type === 'tag') {
              const tagIndex = segments.slice(0, index + 1).filter(s => s.type === 'tag').length - 1
              return (
                <div
                  key={index}
                  className="inline-flex items-center bg-blue-100 dark:bg-blue-800/30 text-blue-800 dark:text-blue-200 px-2 py-1 rounded text-xs font-medium border border-blue-200 dark:border-blue-700"
                >
                  <span className="font-mono">{segment.originalValue}</span>
                  <button
                    type="button"
                    onClick={() => removeTag(tagIndex)}
                    className="ml-1 text-blue-600 dark:text-blue-300 hover:text-blue-800 dark:hover:text-blue-100"
                  >
                    <MaterialSymbol name="close" size="sm" />
                  </button>
                </div>
              )
            } else if (segment.content.trim()) {
              return (
                <span key={index} className="inline-flex items-center bg-zinc-200 dark:bg-zinc-700 text-zinc-700 dark:text-zinc-300 px-2 py-1 rounded text-xs">
                  "{segment.content}"
                </span>
              )
            }
            return null
          })}
        </div>
      )}

      {/* Input Field */}
      <div className="relative flex items-center">
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onSelect={handleSelectionChange}
          onClick={handleSelectionChange}
          onFocus={handleInputFocus}
          onBlur={handleInputBlur}
          placeholder={placeholder}
          disabled={disabled}
          className={twMerge(
            'w-full px-3 py-2 bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100',
            'border rounded-md focus:outline-none focus:ring-2 font-mono',
            error
              ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
              : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500',
            disabled && 'opacity-50 cursor-not-allowed',

            variant === 'inline' && segments.some(s => s.type === 'tag') && 'text-transparent caret-zinc-900 dark:caret-zinc-100 relative',

            value ? 'pr-10' : 'pr-8', // More space when both buttons are visible
            className,
            'text-xs'
          )}
        />

        {/* Show Input Options Button */}
        <button
          type="button"
          onClick={showInputOptions}
          disabled={disabled}
          className={twMerge(
            'absolute text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-200 z-20',
            value ? 'right-6  ' : 'right-2'
          )}
          title="Insert input reference"
        >
          <MaterialSymbol name="input" size="sm" />
        </button>

        {/* Clear Button */}
        {value && (
          <button
            type="button"
            onClick={() => onChange('')}
            disabled={disabled}
            className="absolute right-2 text-zinc-400 hover:text-zinc-600 dark:text-zinc-500 dark:hover:text-zinc-300 z-20"
            title="Clear"
          >
            <MaterialSymbol name="clear" size="sm" />
          </button>
        )}
      </div>

      {/* Inline Tag Overlay - only for inline variant */}
      {variant === 'inline' && segments.length > 0 && segments.some(s => s.type === 'tag') && (
        <div className={twMerge(
          'absolute inset-0 flex items-center py-2 pointer-events-none z-10 bg-transparent overflow-hidden',
          value ? 'px-3 pr-10' : 'px-3 pr-8' // Match input padding
        )}>
          <div className="flex items-center flex-nowrap gap-0 text-xs overflow-hidden font-mono max-w-full">
            {segments.map((segment, index) => {
              if (segment.type === 'tag') {
                return (
                  <span
                    key={index}
                    className="inline-flex items-center bg-blue-100 dark:bg-blue-800/30 text-blue-700 dark:text-blue-200 px-1 py-0.5 rounded text-xs font-medium border border-blue-200 dark:border-blue-600 flex-shrink-0 max-w-fit"
                    style={{
                      width: `${Math.min((segment.originalValue?.length || 0) * 0.6, 20)}em`,
                      minWidth: 'fit-content'
                    }}
                  >
                    <span className="truncate">{segment.originalValue}</span>
                  </span>
                )
              } else if (segment.content.trim()) {
                return (
                  <span
                    key={index}
                    className="text-zinc-700 dark:text-zinc-300 text-xs whitespace-nowrap truncate flex-shrink"
                    style={{
                      width: `${Math.min(segment.content.length * 0.6, 10)}em`,
                      maxWidth: `${Math.min(segment.content.length * 0.6, 10)}em`
                    }}
                  >
                    {segment.content}
                  </span>
                )
              }
              return null
            })}
          </div>
        </div>
      )}

      {/* Autocomplete Dropdown */}
      {isOpen && (
        <div
          ref={dropdownRef}
          className="absolute z-50 mt-1 min-w-full w-max max-w-md max-h-60 overflow-auto bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded shadow-lg"
        >
          {filteredOptions.length === 0 ? (
            <div className="px-3 py-2 text-xs text-zinc-500 dark:text-zinc-400">
              {options.length === 0 ? 'No inputs available' : 'No matching options found'}
            </div>
          ) : (
            filteredOptions.map((option) => (
              <div
                key={option.id}
                className="px-3 py-2 cursor-pointer hover:bg-blue-500 hover:text-white text-xs"
                onClick={() => handleOptionSelect(option)}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium">{option.label}</div>
                    {option.description && (
                      <div className="text-xs opacity-70">{option.description}</div>
                    )}
                  </div>
                  <div className="text-xs font-mono opacity-70 ml-2">{option.value}</div>
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  )
}
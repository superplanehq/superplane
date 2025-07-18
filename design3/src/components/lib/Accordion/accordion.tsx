import { useState } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import clsx from 'clsx'

export interface AccordionItem {
  id: string
  title: string | React.ReactNode
  content: React.ReactNode
  defaultOpen?: boolean
  disabled?: boolean
}

export interface AccordionProps {
  items: AccordionItem[]
  multiple?: boolean
  className?: string
}

export function Accordion({ items, multiple = false, className }: AccordionProps) {
  const [openItems, setOpenItems] = useState<Set<string>>(
    new Set(items.filter(item => item.defaultOpen).map(item => item.id))
  )

  const toggleItem = (itemId: string) => {
    const newOpenItems = new Set(openItems)
    
    if (newOpenItems.has(itemId)) {
      newOpenItems.delete(itemId)
    } else {
      if (!multiple) {
        newOpenItems.clear()
      }
      newOpenItems.add(itemId)
    }
    
    setOpenItems(newOpenItems)
  }

  return (
    <div className={clsx('space-y-2', className)}>
      {items.map((item) => (
        <AccordionItem
          key={item.id}
          item={item}
          isOpen={openItems.has(item.id)}
          onToggle={() => toggleItem(item.id)}
        />
      ))}
    </div>
  )
}

interface AccordionItemProps {
  item: AccordionItem
  isOpen: boolean
  onToggle: () => void
}

function AccordionItem({ item, isOpen, onToggle }: AccordionItemProps) {
  return (
    <div className="border-b border-zinc-200 dark:border-zinc-700  overflow-hidden">
      <button
        type="button"
        className={clsx(
          'w-full flex items-center justify-between p-3 text-left transition-colors',
          
          {
            'cursor-not-allowed opacity-50': item.disabled,
            'cursor-pointer': !item.disabled
          }
        )}
        onClick={onToggle}
        disabled={item.disabled}
        aria-expanded={isOpen}
      >
        <span className={clsx("flex-auto zinc-900 dark:white text-sm", isOpen ? 'font-bold text-zinc-700 dark:text-zinc-200 hover:text-zinc-600 dark:hover:text-zinc-100' : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-400')}>
          {item.title}
        </span>
        <MaterialSymbol
          name={isOpen ? 'expand_less' : 'expand_more'}
          className={clsx(
            'text-zinc-500 dark:text-zinc-400',
          )}
        />
      </button>
      
      {isOpen && (
        <div className="p-3 pt-0">
          {item.content}
        </div>
      )}
    </div>
  )
}

// Controlled version for more complex use cases
export interface ControlledAccordionProps {
  items: AccordionItem[]
  openItems: string[]
  onToggle: (itemId: string) => void
  multiple?: boolean
  className?: string
}

export function ControlledAccordion({ 
  items, 
  openItems, 
  onToggle, 
  multiple = false, 
  className 
}: ControlledAccordionProps) {
  const openItemsSet = new Set(openItems)

  const handleToggle = (itemId: string) => {
    onToggle(itemId)
  }

  return (
    <div className={clsx(className)}>
      {items.map((item) => (
        <AccordionItem
          key={item.id}
          item={item}
          isOpen={openItemsSet.has(item.id)}
          onToggle={() => handleToggle(item.id)}
        />
      ))}
    </div>
  )
}
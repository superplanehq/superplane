import React, { useState, useCallback } from 'react'
import clsx from 'clsx'

export interface Tab {
  id: string
  label: string
  icon?: React.ReactNode
  count?: number
  disabled?: boolean
}

export interface TabsProps {
  tabs: Tab[]
  defaultTab?: string
  onTabChange?: (tabId: string) => void
  className?: string
  variant?: 'default' | 'pills' | 'underline'
}

export interface TabsState {
  active: string
  tabs: Tab[]
}

// Hook to manage tab state
export function useTabs(defaultTab: string, tabs: Tab[]): TabsState & { setActive: (tabId: string) => void } {
  const [active, setActive] = useState(defaultTab)

  const setActiveTab = useCallback((tabId: string) => {
    setActive(tabId)
  }, [])

  return {
    active,
    tabs,
    setActive: setActiveTab,
  }
}

// Main Tabs component
export function Tabs({ 
  tabs, 
  defaultTab, 
  onTabChange, 
  className,
  variant = 'default'
}: TabsProps) {
  const [activeTab, setActiveTab] = useState(defaultTab || tabs[0]?.id || '')

  const handleTabClick = useCallback((tabId: string) => {
    if (tabs.find(tab => tab.id === tabId)?.disabled) return
    
    setActiveTab(tabId)
    onTabChange?.(tabId)
  }, [tabs, onTabChange])

  const baseClasses = clsx(
    'w-full',
    {
      'border-b border-zinc-200 dark:border-zinc-700': variant === 'default' || variant === 'underline',
    },
    className
  )

  const navClasses = clsx(
    'flex',
    {
      'gap-1 p-1 bg-zinc-100 dark:bg-zinc-800 rounded-lg': variant === 'pills',
      'gap-6': variant === 'default',
      'gap-4': variant === 'underline',
    }
  )

  return (
    <div className={baseClasses}>
      <nav className={navClasses}>
        {tabs.map((tab) => (
          <TabItem 
            key={tab.id} 
            tab={tab} 
            activeTab={activeTab} 
            onClick={handleTabClick}
            variant={variant}
          />
        ))}
      </nav>
    </div>
  )
}

// Individual tab item component
function TabItem({ 
  tab, 
  activeTab, 
  onClick, 
  variant 
}: { 
  tab: Tab
  activeTab: string
  onClick: (tabId: string) => void
  variant: 'default' | 'pills' | 'underline'
}) {
  const isActive = activeTab === tab.id
  const isDisabled = tab.disabled

  const handleClick = useCallback(() => {
    if (!isDisabled) {
      onClick(tab.id)
    }
  }, [tab.id, onClick, isDisabled])

  const buttonClasses = clsx(
    'relative flex items-center gap-2 font-medium text-sm transition-all duration-200 ease-in-out focus:outline-hidden',
    {
      // Default variant
      'px-1 py-3 border-b-2 border-transparent': variant === 'default',
      'text-blue-600 border-blue-500 dark:text-blue-400': variant === 'default' && isActive,
      'text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300': variant === 'default' && !isActive && !isDisabled,
      
      // Pills variant
      'px-3 py-2 rounded-md': variant === 'pills',
      'bg-white text-zinc-900 shadow-sm dark:bg-zinc-700 dark:text-white': variant === 'pills' && isActive,
      'text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white': variant === 'pills' && !isActive && !isDisabled,
      
      // Underline variant
      'px-3 py-2 relative': variant === 'underline',
      'text-blue-600 dark:text-blue-400': variant === 'underline' && isActive,
      'text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300': variant === 'underline' && !isActive && !isDisabled,
      
      // Disabled state
      'opacity-50 cursor-not-allowed': isDisabled,
      'cursor-pointer': !isDisabled,
    }
  )

  return (
    <button
      type="button"
      className={buttonClasses}
      onClick={handleClick}
      disabled={isDisabled}
      data-testid={`tab-${tab.label.toLowerCase()}`}
    >
      {tab.icon && (
        <span className="flex-shrink-0 w-4 h-4">
          {tab.icon}
        </span>
      )}
      <span className="leading-none whitespace-nowrap">
        {tab.label}
      </span>
      <TabCountBadge count={tab.count} variant={variant} />
      
      {variant === 'underline' && (
        <TabUnderline isActive={isActive} />
      )}
    </button>
  )
}

// Underline indicator for underline variant
function TabUnderline({ isActive }: { isActive: boolean }) {
  const underlineClasses = clsx(
    'absolute inset-x-0 bottom-0 h-0.5 bg-blue-500 transition-all duration-200 ease-in-out',
    {
      'scale-x-100': isActive,
      'scale-x-0': !isActive,
    }
  )

  return <div className={underlineClasses} />
}

// Count badge component
function TabCountBadge({ count, variant }: { count?: number; variant: string }) {
  if (!count || count <= 0) {
    return null
  }

  const badgeClasses = clsx(
    'inline-flex items-center justify-center text-xs font-medium rounded-full min-w-[1.25rem] h-5 px-1.5',
    {
      'bg-zinc-100 text-zinc-600 dark:bg-zinc-700 dark:text-zinc-300': variant === 'default' || variant === 'underline',
      'bg-zinc-200 text-zinc-700 dark:bg-zinc-600 dark:text-zinc-200': variant === 'pills',
    }
  )

  return (
    <span className={badgeClasses}>
      {count > 99 ? '99+' : count}
    </span>
  )
}

// Controlled tabs component for more complex use cases
export function ControlledTabs({ 
  tabs, 
  activeTab, 
  onTabChange, 
  className,
  variant = 'default'
}: {
  tabs: Tab[]
  activeTab: string
  onTabChange: (tabId: string) => void
  className?: string
  variant?: 'default' | 'pills' | 'underline'
}) {
  const handleTabClick = useCallback((tabId: string) => {
    if (tabs.find(tab => tab.id === tabId)?.disabled) return
    onTabChange(tabId)
  }, [tabs, onTabChange])

  const baseClasses = clsx(
    'w-full',
    {
      'border-b border-zinc-200 dark:border-zinc-700': variant === 'default' || variant === 'underline',
    },
    className
  )

  const navClasses = clsx(
    'flex',
    {
      'gap-1 p-1 bg-zinc-100 dark:bg-zinc-800 rounded-lg': variant === 'pills',
      'gap-6': variant === 'default',
      'gap-4': variant === 'underline',
    }
  )

  return (
    <div className={baseClasses}>
      <nav className={navClasses}>
        {tabs.map((tab) => (
          <TabItem 
            key={tab.id} 
            tab={tab} 
            activeTab={activeTab} 
            onClick={handleTabClick}
            variant={variant}
          />
        ))}
      </nav>
    </div>
  )
}
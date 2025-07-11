import React from 'react'
import { Link } from '../Link/link'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem
} from '../Dropdown/dropdown'
import clsx from 'clsx'
import { Button } from '../Button/button'

export interface BreadcrumbDropdownItem {
  label: string
  href?: string
  onClick?: () => void
}

export interface BreadcrumbItem {
  label: string
  href?: string
  icon?: string
  current?: boolean
  onClick?: () => void
  dropdown?: BreadcrumbDropdownItem[]
  starred?: boolean
  onStarToggle?: (starred: boolean) => void
}

export interface BreadcrumbsProps {
  items: BreadcrumbItem[]
  className?: string
  separator?: '/' | '>' | '•'
  showDivider?: boolean
}

export function Breadcrumbs({
  items,
  className,
  separator = '/',
  showDivider = true
}: BreadcrumbsProps) {
  if (!items.length) return null

  return (
    <nav 
      className={clsx(
        'flex items-center space-x-2 text-sm',
        className
      )}
      aria-label="Breadcrumb"
    >
      {/* Divider line */}
      {showDivider && (
        <div className="h-5 w-px bg-zinc-300 dark:bg-zinc-600 mr-4" />
      )}
      
      {items.map((item, index) => (
        <React.Fragment key={index}>
          <div className="flex items-center">
            {item.current ? (
              // Current page (not clickable)
              <span 
                className="text-zinc-900 dark:text-zinc-100 font-medium flex items-center"
                aria-current="page"
              >
                {item.icon && (
                  <MaterialSymbol 
                    name={item.icon} 
                    className="text-zinc-700 dark:text-zinc-300 mr-1" 
                    size="sm" 
                  />
                )}
                {item.label}
              </span>
            ) : item.href ? (
              // Clickable link
              <Link
                href={item.href}
                className="text-blue-700 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-100 transition-colors flex items-center"
              >
                {item.icon && (
                  <MaterialSymbol 
                    name={item.icon} 
                    className="text-blue-700 dark:text-blue-400 mr-1" 
                    size="sm" 
                  />
                )}
                {item.label}
              </Link>
            ) : item.onClick ? (
              // Clickable button
              <button
                onClick={item.onClick}
                className="text-blue-700 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-100 transition-colors flex items-center"
              >
                {item.icon && (
                  <MaterialSymbol 
                    name={item.icon} 
                    className="text-blue-700 dark:text-blue-400 mr-1" 
                    size="sm" 
                  />
                )}
                {item.label}
              </button>
            ) : (
              // Non-clickable item
              <span className="text-zinc-600 dark:text-zinc-400 flex items-center">
                {item.icon && (
                  <MaterialSymbol 
                    name={item.icon} 
                    className="text-zinc-600 dark:text-zinc-400 mr-1" 
                    size="sm" 
                  />
                )}
                {item.label}
              </span>
            )}
            {/* Dropdown for breadcrumb item */}
            {item.dropdown && item.dropdown.length > 0 && (
              <Dropdown>
                <DropdownButton plain className="p-0 w-7 !leading-none">
                  <MaterialSymbol name="expand_more" size="lg" className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300" />
                </DropdownButton>
                <DropdownMenu>
                  {item.dropdown.map((dropdownItem, dropdownIndex) => (
                    <DropdownItem 
                      key={dropdownIndex}
                      onClick={dropdownItem.onClick}
                      href={dropdownItem.href}
                    >
                      {dropdownItem.label}
                    </DropdownItem>
                  ))}
                </DropdownMenu>
              </Dropdown>
            )}
            {/* Star toggle button */}
            {item.onStarToggle && (
              <Button
                onClick={() => item.onStarToggle!(!item.starred)}
                aria-label={item.starred ? 'Remove from favorites' : 'Add to favorites'}
                className="w-7 flex-items-center !p-0 !leading-none"
                plain
              >
                <MaterialSymbol 
                  name="star"
                  fill={item.starred ? 1 : 0}
                  size="lg" 
                  className={item.starred 
                    ? "text-yellow-500 hover:text-yellow-600 dark:text-yellow-400 dark:hover:text-yellow-300" 
                    : "text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                  }
                />
              </Button>
            )}
          </div>
          
          {/* Separator */}
          {index < items.length - 1 && (
            <span className="text-zinc-400" aria-hidden="true">
              {separator}
            </span>
          )}
        </React.Fragment>
      ))}
    </nav>
  )
}
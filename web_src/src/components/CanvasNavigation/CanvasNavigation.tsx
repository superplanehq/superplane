import { Link } from 'react-router-dom'
import * as Headless from '@headlessui/react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import {
  Dropdown,
  DropdownMenu,
  DropdownItem
} from '../Dropdown/dropdown'
import { ControlledTabs, type Tab } from '../Tabs/tabs'
import { useOrganizationCanvases } from '../../hooks/useOrganizationData'

export interface CanvasNavigationProps {
  canvasId: string
  canvasName: string
  activeView: 'editor' | 'settings'
  onViewChange: (view: 'editor' | 'settings') => void
  organizationId: string
}

export function CanvasNavigation({
  canvasName,
  activeView,
  onViewChange,
  organizationId
}: CanvasNavigationProps) {
  const { data: canvasesData = [] } = useOrganizationCanvases(organizationId)

  const navigationTabs: Tab[] = [
    {
      id: 'editor',
      label: 'Preview',
    },
    {
      id: 'settings',
      label: 'Settings',
    }
  ]

  return (
    <nav className="flex items-center bg-zinc-200 dark:bg-zinc-950 border-b border-zinc-200 dark:border-zinc-800">
      {/* Back Button */}
      <div className='flex border-r border-zinc-400 dark:border-zinc-600 dark:bg-zinc-900'>
        <Link
          to={`/${organizationId}`}
          className='px-3 py-1 hover:bg-zinc-300 dark:hover:bg-zinc-800 text-zinc-950 dark:text-white'
        >
          <MaterialSymbol size='lg' weight={400} name="arrow_back" />
        </Link>
      </div>

      {/* Canvas Dropdown */}
      <div className='flex px-2 hover:bg-zinc-300 dark:hover:bg-zinc-800'>
        <Dropdown>
          <Headless.MenuButton
            className="flex items-center gap-3 rounded-xl border border-transparent p-1 data-active:border-zinc-200 data-hover:border-zinc-200 dark:data-active:border-zinc-700 dark:data-hover:border-zinc-700"
            aria-label="Canvas options"
          >
            <span className="block text-left">
              <span className="block text-md font-bold text-zinc-950 dark:text-white">
                {canvasName}
              </span>
            </span>
            <MaterialSymbol className='text-zinc-950 dark:text-white' size='lg' weight={400} name="expand_more" />
          </Headless.MenuButton>
          <DropdownMenu className="min-w-(--button-width) z-50">
            {canvasesData.map((canvas) => (
              <DropdownItem
                key={canvas.metadata?.id}
                href={`/${organizationId}/canvas/${canvas.metadata?.id}#${activeView}`}
              >
                {canvas.metadata?.name}
              </DropdownItem>
            ))}
          </DropdownMenu>
        </Dropdown>
      </div>


      {/* Navigation Tabs */}
      <div className="flex items-center h-full">
        <ControlledTabs
          tabs={navigationTabs}
          activeTab={activeView}
          variant='underline'
          onTabChange={(tabId) => onViewChange(tabId as 'editor' | 'settings')}
        />
      </div>
    </nav>
  )
}
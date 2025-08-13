import { Link } from 'react-router-dom'
import * as Headless from '@headlessui/react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import {
  Dropdown,
  DropdownMenu,
  DropdownItem,
  DropdownButton
} from '../Dropdown/dropdown'
import { ControlledTabs, type Tab } from '../Tabs/tabs'
import { useOrganizationCanvases } from '../../hooks/useOrganizationData'
import { CanvasSecrets } from '../SettingsPage/components/CanvasSecrets'
import { CanvasIntegrations } from '../SettingsPage/components/CanvasIntegrations'
import { CanvasMembers } from '../SettingsPage/components/CanvasMembers'
import { CanvasDelete } from '../SettingsPage/components/CanvasDelete'

export type CanvasView = 'editor' | 'integrations' | 'members' | 'secrets' | 'delete'

export interface CanvasNavigationProps {
  canvasName: string
  activeView: CanvasView
  onViewChange: (view: CanvasView) => void
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
      id: 'integrations',
      label: 'Integrations',
    },
    {
      id: 'members',
      label: 'Members',
    },
    {
      id: 'secrets',
      label: 'Secrets',
    }
  ]

  return (
    <nav className="flex items-center bg-zinc-200 dark:bg-zinc-950 border-b border-zinc-200 dark:border-zinc-800 h-[2.7rem]">
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
      <div className="flex items-center justify-between w-full h-full">
        <ControlledTabs
          tabs={navigationTabs}
          activeTab={activeView === 'delete' ? 'secrets' : activeView}
          variant='dark-underline'
          onTabChange={(tabId) => onViewChange(tabId as CanvasView)}
        />

        {/* More Actions Dropdown */}
        <div className="flex items-center">
          <Dropdown>
            <DropdownButton plain className="p-1 hover:bg-zinc-200 dark:hover:bg-zinc-700 rounded">
              <MaterialSymbol name="more_vert" size="md" className="text-black dark:text-zinc-400" />
            </DropdownButton>
            <DropdownMenu>
              <DropdownItem onClick={() => onViewChange('delete')}>
                Delete
              </DropdownItem>
            </DropdownMenu>
          </Dropdown>
        </div>
      </div>
    </nav>
  )
}

export function CanvasNavigationContent({ canvasId, activeView, organizationId }: {
  canvasId: string
  activeView: CanvasView
  organizationId: string
}) {

  switch (activeView) {
    case 'secrets':
      return <CanvasSecrets canvasId={canvasId} organizationId={organizationId} />
    case 'integrations':
      return <CanvasIntegrations canvasId={canvasId} organizationId={organizationId} />
    case 'members':
      return <CanvasMembers canvasId={canvasId} organizationId={organizationId} />
    case 'delete':
      return <CanvasDelete canvasId={canvasId} organizationId={organizationId} />
    default:
      return null
  }
}
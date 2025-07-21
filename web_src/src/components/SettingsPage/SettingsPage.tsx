import { useState } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Sidebar, SidebarBody, SidebarItem, SidebarLabel } from '../Sidebar/sidebar'
import { CanvasMembers, CanvasSecrets, CanvasIntegrations, CanvasDelete } from './components'
import { useParams } from 'react-router-dom'

interface SettingsPageProps {
  organizationId: string
}

interface Tab {
  id: string
  label: string
  icon: React.ReactNode
}

type TabType = 'members' | 'secrets' | 'integrations' | 'delete';

export function SettingsPage({ organizationId }: SettingsPageProps) {
  // Get canvas ID from URL params
  const { canvasId } = useParams<{ canvasId: string }>()
  // Mock canvas name - in real app this would come from canvas data
  const canvasName = 'Sample Canvas'
  const [activeTab, setActiveTab] = useState<TabType>('members')

  const tabs: Tab[] = [
    {
      id: 'members',
      label: 'Members',
      icon: <MaterialSymbol name="person" size="sm" />,
    },
    {
      id: 'secrets',
      label: 'Secrets',
      icon: <MaterialSymbol name="key" size="sm" />,
    },
    {
      id: 'integrations',
      label: 'Integrations',
      icon: <MaterialSymbol name="integration_instructions" size="sm" />,
    },
    {
      id: 'delete',
      label: 'Delete',
      icon: <MaterialSymbol name="delete" size="sm" />,
    },
  ]

  return (
    <div className="flex h-full bg-gray-50 dark:bg-zinc-900">
      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Settings Content */}
        <main className="flex-1 flex">
          {/* Sidebar Navigation */}
          <Sidebar className='w-64 bg-white dark:bg-zinc-950 border-r border-zinc-200 dark:border-zinc-800'>
            <SidebarBody>
              {tabs.map((tab) => (
                <SidebarItem
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as TabType)}
                  className={`${activeTab === tab.id ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`}
                >
                  <div className={`flex items-center gap-3 px-3 py-2 ${activeTab === tab.id ? 'font-semibold' : 'font-normal'}`}>
                    {tab.icon}
                    <SidebarLabel>{tab.label}</SidebarLabel>
                  </div>
                </SidebarItem>
              ))}
            </SidebarBody>
          </Sidebar>

          {/* Main Content Area */}
          <div className="flex-1 p-6">
            <div className="max-w-5xl mx-auto">
              {/* Render appropriate component based on active tab */}
              {activeTab === 'members' && (
                <CanvasMembers canvasId={canvasId!} organizationId={organizationId} />
              )}

              {activeTab === 'secrets' && (
                <CanvasSecrets canvasId={canvasId!} organizationId={organizationId} />
              )}

              {activeTab === 'integrations' && (
                <CanvasIntegrations canvasId={canvasId!} organizationId={organizationId} />
              )}

              {activeTab === 'delete' && (
                <CanvasDelete
                  canvasId={canvasId!}
                  canvasName={canvasName}
                  organizationId={organizationId}
                />
              )}
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
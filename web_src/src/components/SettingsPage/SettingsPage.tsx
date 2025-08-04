import { useState, useEffect } from 'react'
import { Sidebar, SidebarBody, SidebarItem, SidebarLabel } from '../Sidebar/sidebar'
import { CanvasMembers, CanvasSecrets, CanvasIntegrations, CanvasDelete } from './components'
import { useParams } from 'react-router-dom'

interface SettingsPageProps {
  organizationId: string
}

interface Tab {
  id: string
  label: string
}

export type TabType = 'members' | 'secrets' | 'integrations' | 'delete';

export function SettingsPage({ organizationId }: SettingsPageProps) {
  const { canvasId } = useParams<{ canvasId: string }>()
  const [activeTab, setActiveTab] = useState<TabType>('members')

  const getActiveTabFromUrl = (): TabType => {
    const hash = window.location.hash
    const urlParams = new URLSearchParams(hash.split('?')[1] || '')
    const tab = urlParams.get('tab') as TabType
    return tab && ['members', 'secrets', 'integrations', 'delete'].includes(tab) ? tab : 'members'
  }

  const updateActiveTab = (tab: TabType) => {
    const hash = window.location.hash
    const [hashPath] = hash.split('?')
    const newHash = `${hashPath}?tab=${tab}`
    window.location.hash = newHash
    setActiveTab(tab)
  }

  useEffect(() => {
    const initialTab = getActiveTabFromUrl()
    setActiveTab(initialTab)

    const handleHashChange = () => {
      const newTab = getActiveTabFromUrl()
      setActiveTab(newTab)
    }

    window.addEventListener('hashchange', handleHashChange)
    return () => window.removeEventListener('hashchange', handleHashChange)
  }, [])

  const tabs: Tab[] = [
    {
      id: 'members',
      label: 'Members',
    },
    {
      id: 'secrets',
      label: 'Secrets',
    },
    {
      id: 'integrations',
      label: 'Integrations',
    },
    {
      id: 'delete',
      label: 'Delete',
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
                  onClick={() => updateActiveTab(tab.id as TabType)}
                  className={`${activeTab === tab.id ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`}
                >
                  <div className={`flex items-center gap-3 px-3 ${activeTab === tab.id ? 'font-semibold' : 'font-normal'}`}>
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
                <CanvasIntegrations canvasId={canvasId!} updateActiveTab={updateActiveTab} />
              )}

              {activeTab === 'delete' && (
                <CanvasDelete
                  canvasId={canvasId!}
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
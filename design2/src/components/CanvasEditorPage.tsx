import { useState, useCallback } from 'react'
import {
  ReactFlow,
  MiniMap,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  type OnConnect,
  type Node,
  type Edge,
  BackgroundVariant,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Button } from './lib/Button/button'
import { Avatar } from './lib/Avatar/avatar'
import { Text } from './lib/Text/text'
import { Heading } from './lib/Heading/heading'
import { NavigationVertical, type User, type Organization, type NavigationLink } from './lib/Navigation/navigation-vertical'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'

interface CanvasEditorPageProps {
  canvasId: string
  onSignOut?: () => void
  navigationLinks?: NavigationLink[]
  onLinkClick?: (linkId: string) => void
  onBack?: () => void
}

// Sample workflow nodes
const initialNodes: Node[] = [
  {
    id: '1',
    type: 'input',
    data: { label: 'Start' },
    position: { x: 250, y: 25 },
    style: {
      background: '#3b82f6',
      color: 'white',
      border: '1px solid #2563eb',
      borderRadius: '8px',
      padding: '10px'
    }
  },
  {
    id: '2',
    data: { label: 'Process Data' },
    position: { x: 100, y: 125 },
    style: {
      background: '#10b981',
      color: 'white', 
      border: '1px solid #059669',
      borderRadius: '8px',
      padding: '10px'
    }
  },
  {
    id: '3',
    data: { label: 'Validate Input' },
    position: { x: 400, y: 125 },
    style: {
      background: '#f59e0b',
      color: 'white',
      border: '1px solid #d97706',
      borderRadius: '8px',
      padding: '10px'
    }
  },
  {
    id: '4',
    data: { label: 'Send Notification' },
    position: { x: 250, y: 250 },
    style: {
      background: '#8b5cf6',
      color: 'white',
      border: '1px solid #7c3aed',
      borderRadius: '8px',
      padding: '10px'
    }
  },
  {
    id: '5',
    type: 'output',
    data: { label: 'Complete' },
    position: { x: 250, y: 375 },
    style: {
      background: '#ef4444',
      color: 'white',
      border: '1px solid #dc2626',
      borderRadius: '8px',
      padding: '10px'
    }
  }
]

// Sample workflow edges
const initialEdges: Edge[] = [
  { id: 'e1-2', source: '1', target: '2', animated: true },
  { id: 'e1-3', source: '1', target: '3', animated: true },
  { id: 'e2-4', source: '2', target: '4', animated: true },
  { id: 'e3-4', source: '3', target: '4', animated: true },
  { id: 'e4-5', source: '4', target: '5', animated: true }
]

export function CanvasEditorPage({ 
  canvasId, 
  onSignOut, 
  navigationLinks = [], 
  onLinkClick,
  onBack
}: CanvasEditorPageProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)
  const [showMiniMap, setShowMiniMap] = useState(true)

  const onConnect: OnConnect = useCallback(
    (params) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  )

  // Mock user and organization data
  const currentUser: User = {
    id: '1',
    name: 'John Doe',
    email: 'john@superplane.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  const currentOrganization: Organization = {
    id: '1',
    name: 'Development Team',
    plan: 'Pro Plan',
    initials: 'DT',
  }

  // Navigation handlers
  const handleHelpClick = () => {
    console.log('Opening help documentation...')
  }

  const handleUserMenuAction = (action: 'profile' | 'settings' | 'signout') => {
    switch (action) {
      case 'profile':
        console.log('Navigating to user profile...')
        break
      case 'settings':
        console.log('Opening account settings...')
        break
      case 'signout':
        onSignOut?.()
        break
    }
  }

  const handleOrganizationMenuAction = (action: 'settings' | 'billing' | 'members') => {
    console.log(`Organization action: ${action}`)
  }

  const handleLinkClick = (linkId: string) => {
    if (onLinkClick) {
      onLinkClick(linkId)
    } else {
      console.log(`Navigation link clicked: ${linkId}`)
    }
  }

  const handleSave = () => {
    console.log('Saving canvas...', { nodes, edges })
  }

  const handleExport = () => {
    console.log('Exporting canvas...')
  }

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900 flex">
      {/* Vertical Navigation */}
      <NavigationVertical
        user={currentUser}
        organization={currentOrganization}
        showOrganization={false}
        links={navigationLinks}
        onHelpClick={handleHelpClick}
        onUserMenuAction={handleUserMenuAction}
        onOrganizationMenuAction={handleOrganizationMenuAction}
        onLinkClick={handleLinkClick}
      />

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <header className="bg-white dark:bg-zinc-800 border-b border-zinc-200 dark:border-zinc-700 px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Button
                plain
                onClick={onBack}
                className="text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300"
              >
                <MaterialSymbol name="arrow_back" className="mr-2" />
                Back to Canvases
              </Button>
              <div className="h-6 w-px bg-zinc-200 dark:bg-zinc-700" />
              <div>
                <Heading level={1} className="!text-xl mb-1">Canvas Editor</Heading>
                <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                  Canvas ID: {canvasId}
                </Text>
              </div>
            </div>
            
            <div className="flex items-center space-x-3">
              <Button
                plain
                onClick={() => setShowMiniMap(!showMiniMap)}
                className="text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300"
              >
                <MaterialSymbol name={showMiniMap ? 'visibility_off' : 'visibility'} className="mr-2" />
                {showMiniMap ? 'Hide' : 'Show'} Minimap
              </Button>
              <Button color="zinc" onClick={handleExport}>
                <MaterialSymbol name="download" className="mr-2" />
                Export
              </Button>
              <Button color="blue" onClick={handleSave}>
                <MaterialSymbol name="save" className="mr-2" />
                Save
              </Button>
            </div>
          </div>
        </header>

        {/* React Flow Canvas */}
        <div className="flex-1 relative">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            fitView
            className="bg-zinc-50 dark:bg-zinc-900"
          >
            <Controls className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg" />
            <Background 
              variant={BackgroundVariant.Dots} 
              gap={12} 
              size={1} 
              className="bg-zinc-50 dark:bg-zinc-900"
            />
            {showMiniMap && (
              <MiniMap 
                className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg"
                nodeColor={(node) => {
                  if (node.type === 'input') return '#3b82f6'
                  if (node.type === 'output') return '#ef4444'
                  return '#10b981'
                }}
              />
            )}
          </ReactFlow>
        </div>
      </div>
    </div>
  )
}
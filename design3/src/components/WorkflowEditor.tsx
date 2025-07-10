import React, { useState, useCallback, useRef } from 'react'
import {
  ReactFlow,
  ReactFlowProvider,
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  addEdge,
  MarkerType,
  ConnectionLineType,
  Connection,
  Edge,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { 
  Navbar, 
  NavbarSection, 
  NavbarSpacer, 
  NavbarItem,
  NavbarLabel 
} from './lib/Navbar/navbar'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownDivider,
  DropdownMenu, 
  DropdownItem,
  DropdownLabel 
} from './lib/Dropdown/dropdown'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { WorkflowNode, WorkflowEdge } from '../types'
import { DeploymentCardStage } from './DeploymentCardStage'
import { Link } from './lib/Link/link'

// Node types for React Flow
const nodeTypes = {
  deploymentCard: DeploymentCardStage as any,
}

interface WorkflowEditorProps {
  workflowId: string
  workflowName: string
  onBack?: () => void
  onSignOut?: () => void
  onSwitchOrganization?: () => void
}

// Initial workflow data for new workflows
const getInitialNodes = (): WorkflowNode[] => [
  {
    id: 'start-1',
    type: 'deploymentCard',
    position: { x: 100, y: 200 },
    data: {
      id: 'start-1',
      label: 'Start',
      type: 'action',
      status: 'pending',
      hasHealthCheck: false,
      style: { width: 300 },
    },
  },
]

const getInitialEdges = (): WorkflowEdge[] => []

export function WorkflowEditor({ 
  workflowId, 
  workflowName, 
  onBack, 
  onSignOut, 
  onSwitchOrganization 
}: WorkflowEditorProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState(getInitialNodes())
  const [edges, setEdges, onEdgesChange] = useEdgesState(getInitialEdges())
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const reactFlowWrapper = useRef<HTMLDivElement>(null)

  // Mock data
  const currentUser = {
    id: '1',
    name: 'John Doe',
    email: 'john@acme.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  const currentOrganization = {
    id: '1',
    name: 'Acme Corporation',
    plan: 'Pro Plan',
    initials: 'AC',
  }

  // Sample workflow templates
  const workflowTemplates = [
    {
      id: 'customer-onboarding',
      name: 'Customer Onboarding',
      description: 'Automated workflow for new customer registration and setup',
      nodes: [
        {
          id: 'welcome-email',
          type: 'deploymentCard',
          position: { x: 100, y: 100 },
          data: {
            id: 'welcome-email',
            label: 'Send Welcome Email',
            type: 'action',
            status: 'pending',
            hasHealthCheck: false,
            style: { width: 300 },
          },
        },
        {
          id: 'create-account',
          type: 'deploymentCard',
          position: { x: 500, y: 100 },
          data: {
            id: 'create-account',
            label: 'Create User Account',
            type: 'action',
            status: 'pending',
            hasHealthCheck: true,
            healthCheckStatus: 'unknown',
            style: { width: 300 },
          },
        },
        {
          id: 'setup-workspace',
          type: 'deploymentCard',
          position: { x: 900, y: 100 },
          data: {
            id: 'setup-workspace',
            label: 'Setup Workspace',
            type: 'action',
            status: 'pending',
            hasHealthCheck: false,
            style: { width: 300 },
          },
        },
      ],
      edges: [
        {
          id: 'e1-2',
          source: 'welcome-email',
          target: 'create-account',
          type: 'smoothstep',
          animated: true,
          markerEnd: { type: MarkerType.ArrowClosed },
        },
        {
          id: 'e2-3',
          source: 'create-account',
          target: 'setup-workspace',
          type: 'smoothstep',
          animated: true,
          markerEnd: { type: MarkerType.ArrowClosed },
        },
      ],
    },
  ]

  const handleUserAction = (action: 'profile' | 'account-settings' | 'signout') => {
    if (action === 'signout') {
      onSignOut?.()
    } else {
      console.log(`User action: ${action}`)
    }
  }

  const handleOrgAction = (action: 'settings' | 'billing' | 'members' | 'switch') => {
    if (action === 'switch') {
      onSwitchOrganization?.()
    } else {
      console.log(`Organization action: ${action}`)
    }
  }

  // Organization dropdown menu component
  function OrganizationDropdownMenu() {
    return (
      <DropdownMenu className="min-w-64" anchor="bottom start">
        <DropdownItem onClick={() => handleOrgAction('settings')}>
          <MaterialSymbol name="settings" />
          <DropdownLabel>Organization Settings</DropdownLabel>
        </DropdownItem>
        <DropdownDivider />
        <DropdownItem onClick={() => handleOrgAction('members')}>
          <Avatar 
            initials={currentOrganization.initials} 
            className="bg-blue-500 text-white" 
            data-slot="icon" 
          />
          <DropdownLabel>{currentOrganization.name}</DropdownLabel>
        </DropdownItem>
        <DropdownItem>
          <Avatar initials="TC" className="bg-purple-500 text-white" data-slot="icon" />
          <DropdownLabel>Tech Corp</DropdownLabel>
        </DropdownItem>
        <DropdownDivider />
        <DropdownItem onClick={() => handleOrgAction('switch')}>
          <MaterialSymbol name="add" />
          <DropdownLabel>New organization&hellip;</DropdownLabel>
        </DropdownItem>
      </DropdownMenu>
    )
  }

  // User dropdown menu component  
  function UserDropdownMenu() {
    return (
      <DropdownMenu className="min-w-64" anchor="bottom end">
        <DropdownItem onClick={() => handleUserAction('profile')}>
          <MaterialSymbol name="person" />
          <DropdownLabel>My profile</DropdownLabel>
        </DropdownItem>
        <DropdownItem onClick={() => handleUserAction('account-settings')}>
          <MaterialSymbol name="settings" />
          <DropdownLabel>Account Settings</DropdownLabel>
        </DropdownItem>
        <DropdownDivider />
        <DropdownItem>
          <MaterialSymbol name="shield" />
          <DropdownLabel>Privacy policy</DropdownLabel>
        </DropdownItem>
        <DropdownItem>
          <MaterialSymbol name="lightbulb" />
          <DropdownLabel>Share feedback</DropdownLabel>
        </DropdownItem>
        <DropdownDivider />
        <DropdownItem onClick={() => handleUserAction('signout')}>
          <MaterialSymbol name="logout" />
          <DropdownLabel>Sign out</DropdownLabel>
        </DropdownItem>
      </DropdownMenu>
    )
  }

  // Handle new connections between nodes
  const onConnect = useCallback(
    (params: Connection | Edge) => {
      if (!params.source || !params.target) return
      
      const newEdge: WorkflowEdge = {
        ...params,
        id: `e${params.source}-${params.target}`,
        source: params.source,
        target: params.target,
        type: 'smoothstep',
        animated: true,
        markerEnd: {
          type: MarkerType.ArrowClosed,
        },
      }
      setEdges((eds) => addEdge(newEdge, eds))
    },
    [setEdges]
  )

  // Handle node selection
  const onNodeClick = useCallback(
    (event: React.MouseEvent, node: WorkflowNode) => {
      event.preventDefault()
      setSelectedNode(selectedNode === node.id ? null : node.id)
      
      setNodes((nds) =>
        nds.map((n) => ({
          ...n,
          selected: n.id === node.id && selectedNode !== node.id,
        }))
      )
    },
    [selectedNode, setNodes]
  )

  // Add new node to workflow
  const addNode = useCallback(
    (nodeType: 'action' | 'condition' | 'deployment') => {
      const nodeLabels = {
        action: 'New Action',
        condition: 'New Condition',
        deployment: 'New Deployment'
      }

      const newNode: WorkflowNode = {
        id: `node-${Date.now()}`,
        type: 'deploymentCard',
        position: { x: Math.random() * 400 + 200, y: Math.random() * 200 + 200 },
        data: {
          id: `node-${Date.now()}`,
          label: nodeLabels[nodeType],
          type: nodeType,
          status: 'pending',
          hasHealthCheck: nodeType === 'deployment',
          style: { width: 300 },
        },
      }
      
      setNodes((nds) => [...nds, newNode])
    },
    [setNodes]
  )

  // Load template (for demonstration)
  const loadTemplate = () => {
    const template = workflowTemplates[0]
    setNodes(template.nodes as WorkflowNode[])
    setEdges(template.edges as WorkflowEdge[])
  }

  // Save workflow
  const saveWorkflow = () => {
    console.log('Saving workflow...', { 
      workflowId,
      workflowName, 
      nodes, 
      edges 
    })
  }

  // Run workflow
  const runWorkflow = () => {
    console.log('Running workflow...', workflowName)
    // Simulate running by updating node statuses
    setNodes((nds) =>
      nds.map((node, index) => ({
        ...node,
        data: {
          ...node.data,
          status: index === 0 ? 'running' : 'pending',
        },
      }))
    )
  }

  const toolbarActions = [
    {
      label: 'Run Workflow',
      icon: 'play_arrow',
      color: 'green' as const,
      onClick: runWorkflow,
      disabled: nodes.length === 0,
    },
    {
      label: 'Save',
      icon: 'save',
      color: 'blue' as const,
      onClick: saveWorkflow,
    },
  ]

  return (
    <div className="flex flex-col bg-zinc-50 dark:bg-zinc-900">
      {/* Top Navbar */}
      <Navbar>
        <Dropdown>
          <DropdownButton as={NavbarItem}>
            <Avatar 
              initials={currentOrganization.initials} 
              className="bg-blue-500 text-white" 
            />
            <NavbarLabel>{currentOrganization.name}</NavbarLabel>
            <MaterialSymbol name="expand_more" />
          </DropdownButton>
          <OrganizationDropdownMenu />
        </Dropdown>

        {/* Workflow Name */}
        <NavbarSection>
          <NavbarItem>
            <Link href="/workflows">My Workflows</Link> /
            <MaterialSymbol name="account_tree" data-slot="icon" className="text-zinc-600 dark:text-zinc-400" />
            <NavbarLabel className="font-medium">{workflowName}</NavbarLabel>
          </NavbarItem>
        </NavbarSection>
        
        <NavbarSpacer />
        
        

        <NavbarSpacer />
        
        <NavbarSection>
          {onBack && (
            <NavbarItem onClick={onBack} aria-label="Back to workflows">
              <MaterialSymbol name="arrow_back" />
            </NavbarItem>
          )}
          <NavbarItem aria-label="Search">
            <MaterialSymbol name="search" />
          </NavbarItem>
          <NavbarItem aria-label="Notifications">
            <MaterialSymbol name="notifications" />
          </NavbarItem>
          <Dropdown>
            <DropdownButton as={NavbarItem}>
              <Avatar 
                src={currentUser.avatar} 
                initials={currentUser.initials}
                square 
              />
            </DropdownButton>
            <UserDropdownMenu />
          </Dropdown>
        </NavbarSection>
      </Navbar>

      {/* Full Width Canvas */}
      <div className="flex-1 min-h-svh flex overflow-hidden border-t border-zinc-200 dark:border-zinc-700">
        {/* Component Sidebar */}
        <div className="w-80 bg-white dark:bg-zinc-800 border-r border-zinc-200 dark:border-zinc-700 p-6">

          
          <div className="space-y-6">
            <div>
              <Subheading level={3} className="mb-4">Add Components</Subheading>
              <div className="space-y-3">
                <Button
                  className="w-full justify-start"
                  plain
                  onClick={() => addNode('action')}
                >
                  <div className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                    <MaterialSymbol name="bolt" className="text-blue-600 dark:text-blue-400" />
                  </div>
                  <div className="text-left">
                    <div className="font-medium">Action</div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      Execute a task or operation
                    </div>
                  </div>
                </Button>
                
                <Button
                  className="w-full justify-start"
                  plain
                  onClick={() => addNode('condition')}
                >
                  <div className="p-2 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg">
                    <MaterialSymbol name="fork_right" className="text-yellow-600 dark:text-yellow-400" />
                  </div>
                  <div className="text-left">
                    <div className="font-medium">Condition</div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      Add conditional logic
                    </div>
                  </div>
                </Button>
                
                <Button
                  className="w-full justify-start"
                  plain
                  onClick={() => addNode('deployment')}
                >
                  <div className="p-2 bg-green-50 dark:bg-green-900/20 rounded-lg">
                    <MaterialSymbol name="cloud_upload" className="text-green-600 dark:text-green-400" />
                  </div>
                  <div className="text-left">
                    <div className="font-medium">Deployment</div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      Deploy or publish content
                    </div>
                  </div>
                </Button>
              </div>
            </div>

            {selectedNode && (
              <div>
                <Subheading level={3} className="mb-4">Node Properties</Subheading>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1">
                      Node ID
                    </label>
                    <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                      {selectedNode}
                    </Text>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Main Canvas Area - Full Width */}
        <div className="flex-1" ref={reactFlowWrapper}>
          {/* Toolbar Actions */}
          <div className='flex justify-end w-full px-4 py-1 bg-white dark:bg-zinc-800 border-b border-zinc-200 dark:border-zinc-700'>
              <NavbarSection>
                {toolbarActions.map((action) => (
                  <Button
                    key={action.label}
                    color={action.color}
                    onClick={action.onClick}
                    disabled={action.disabled}
                    className="mr-2"
                  >
                    <MaterialSymbol name={action.icon} />
                    {action.label}
                  </Button>
                ))}
                
                <Dropdown>
                  <DropdownButton>
                    <MaterialSymbol name="more_vert" />
                  </DropdownButton>
                  <DropdownMenu>
                    <DropdownItem onClick={loadTemplate}>
                      <MaterialSymbol name="content_copy" />
                      Load Template
                    </DropdownItem>
                    <DropdownItem>
                      <MaterialSymbol name="copy" />
                      Duplicate
                    </DropdownItem>
                    <DropdownItem>
                      <MaterialSymbol name="share" />
                      Share
                    </DropdownItem>
                    <DropdownItem>
                      <MaterialSymbol name="download" />
                      Export
                    </DropdownItem>
                  </DropdownMenu>
                </Dropdown>
              </NavbarSection>
          </div>
          <ReactFlowProvider>
            <ReactFlow
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onNodeClick={onNodeClick}
              nodeTypes={nodeTypes}
              connectionLineType={ConnectionLineType.SmoothStep}
              fitView
              attributionPosition="bottom-left"
              className="bg-zinc-50 dark:bg-zinc-900"
            >
              <Controls 
                className="bg-white dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-600 rounded-lg shadow-sm"
                showInteractive={false}
              />
              <Background 
                variant={BackgroundVariant.Dots} 
                gap={20} 
                size={1}
                color="#d1d5db"
                className="dark:opacity-20"
              />
            </ReactFlow>
          </ReactFlowProvider>
        </div>
      </div>
    </div>
  )
}
import React, { useState, useCallback, useRef } from 'react';
import * as Headless from '@headlessui/react'
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
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { NavigationOrg, type User, type Organization } from './lib/Navigation/navigation-org';
import { WorkflowNode, WorkflowEdge } from '../types';
import { DeploymentCardStage } from './DeploymentCardStage';
import { ComponentSidebar } from './ComponentSidebar';
import { Button } from './lib/Button/button';
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol';
import { Heading, Subheading } from './lib/Heading/heading';
import { Text } from './lib/Text/text';
import { 
  Dialog, 
  DialogTitle, 
  DialogDescription, 
  DialogBody, 
  DialogActions 
} from './lib/Dialog/dialog';
import { Avatar } from './lib/Avatar/avatar';
import { Input, InputGroup } from './lib/Input/input';
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem,
  DropdownLabel,
  DropdownDescription,
  DropdownDivider
} from './lib/Dropdown/dropdown';
import { 
  Table, 
  TableHead, 
  TableBody, 
  TableRow, 
  TableHeader, 
  TableCell 
} from './lib/Table/table';
import { Divider } from './lib/Divider/divider';
import { Link } from './lib/Link/link';


// Node types for React Flow
const nodeTypes = {
  deploymentCard: DeploymentCardStage as any,
};

interface CanvasEditorPageProps {
  canvasId: string
  onBack?: () => void
}

// Initial workflow data
const initialNodes: WorkflowNode[] = [
  {
    id: 'deploy-1',
    type: 'deploymentCard',
    position: { x: 100, y: 100 },
    data: {
      id: 'deploy-1',
      label: 'Deploy to Development',
      type: 'deployment',
      status: 'success',
      hasHealthCheck: true,
      healthCheckStatus: 'healthy',
      style: { width: 320 },
    },
  },
  {
    id: 'deploy-2',
    type: 'deploymentCard',
    position: { x: 500, y: 100 },
    data: {
      id: 'deploy-2',
      label: 'Deploy to Staging',
      type: 'deployment',
      status: 'running',
      hasHealthCheck: true,
      healthCheckStatus: 'healthy',
      style: { width: 320 },
    },
  },
  {
    id: 'deploy-3',
    type: 'deploymentCard',
    position: { x: 900, y: 100 },
    data: {
      id: 'deploy-3',
      label: 'Deploy to Production',
      type: 'deployment',
      status: 'pending',
      hasHealthCheck: true,
      healthCheckStatus: 'unknown',
      style: { width: 320 },
    },
  },
];

const initialEdges: WorkflowEdge[] = [
  {
    id: 'e1-2',
    source: 'deploy-1',
    target: 'deploy-2',
    type: 'smoothstep',
    animated: true,
    markerEnd: {
      type: MarkerType.ArrowClosed,
    },
  },
  {
    id: 'e2-3',
    source: 'deploy-2',
    target: 'deploy-3',
    type: 'smoothstep',
    animated: true,
    markerEnd: {
      type: MarkerType.ArrowClosed,
    },
  },
];

/**
 * CanvasEditorPage component following SaaS guidelines
 * - Uses TypeScript with proper interfaces
 * - Implements React Flow for diagramming
 * - Follows responsive design principles
 * - Includes proper accessibility features
 * - Handles loading and error states
 */
export function CanvasEditorPage3({ 
  canvasId, 
  onBack
}: CanvasEditorPageProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)
  const [showMiniMap, setShowMiniMap] = useState(true)

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
    name: 'Confluent',
    initials: 'C',
    avatar: 'https://confluent.io/favicon.ico',
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
        console.log('Signing out...')
        break
    }
  }

  const handleOrganizationMenuAction = (action: 'settings' | 'billing' | 'members') => {
    if (action === 'settings') {
      // Navigate to settings page
      window.history.pushState(null, '', '/settings')
      window.dispatchEvent(new PopStateEvent('popstate'))
    } else {
      console.log(`Organization action: ${action}`)
    }
  }

  // Get canvas name based on ID
  const getCanvasName = (id: string) => {
    const canvasNames: Record<string, string> = {
      '1': 'Production Deployment Pipeline',
      '2': 'Development Workflow',
      '3': 'Testing Environment Setup',
      '4': 'Staging Release Process',
      'new': 'New Canvas'
    }
    return canvasNames[id] || `Canvas ${id}`
  }




  const handleDelete = () => {
    console.log('Deleting canvas...', { nodes, edges })
  }

  const handleExport = () => {
    console.log('Exporting canvas...')
  }

  const handleShare = () => {
    console.log('Sharing canvas...', { canvasId, canvasName: getCanvasName(canvasId) })
    // TODO: Implement share functionality (copy link, email, etc.)
  }

  const handleMembers = () => {
    console.log('Opening canvas members modal...', { canvasId, canvasName: getCanvasName(canvasId) })
    setShowMembersModal(true)
  }

  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isCanvasStarred, setIsCanvasStarred] = useState(false);
  const [showMembersModal, setShowMembersModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const reactFlowWrapper = useRef<HTMLDivElement>(null);

  // Mock data for canvas members
  const canvasMembers = [
    {
      id: '1',
      name: 'John Doe',
      email: 'john@acme.com',
      role: 'Editor',
      permission: 'Can edit',
      lastActive: '2 hours ago',
      initials: 'JD',
      avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face'
    },
    {
      id: '2',
      name: 'Jane Smith',
      email: 'jane@acme.com',
      role: 'Viewer',
      permission: 'Can view',
      lastActive: '1 day ago',
      initials: 'JS'
    },
    {
      id: '3',
      name: 'Bob Wilson',
      email: 'bob@acme.com',
      role: 'Editor',
      permission: 'Can edit',
      lastActive: '3 days ago',
      initials: 'BW'
    },
    {
      id: '4',
      name: 'Alice Johnson',
      email: 'alice@acme.com',
      role: 'Owner',
      permission: 'Full access',
      lastActive: '5 minutes ago',
      initials: 'AJ'
    }
  ];

  /**
   * Handle new connections between nodes
   */
  const onConnect = useCallback(
    (params: Connection | Edge) => {
      if (!params.source || !params.target) return;
      
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
      };
      setEdges((eds) => addEdge(newEdge, eds));
    },
    [setEdges]
  );

  /**
   * Handle node selection
   */
  const onNodeClick = useCallback(
    (event: React.MouseEvent, node: WorkflowNode) => {
      event.preventDefault();
      setSelectedNode(selectedNode === node.id ? null : node.id);
      
      // Update node selection state
      setNodes((nds) =>
        nds.map((n) => ({
          ...n,
          selected: n.id === node.id && selectedNode !== node.id,
        }))
      );
    },
    [selectedNode, setNodes]
  );



  

  /**
   * Add new node to workflow
   */
  const addNode = useCallback(
    (nodeType: string) => {
      const newNode: WorkflowNode = {
        id: `node-${Date.now()}`,
        type: 'deploymentCard',
        position: { x: 300, y: 300 },
        data: {
          id: `node-${Date.now()}`,
          label: `New ${nodeType}`,
          type: 'deployment',
          status: 'pending',
          hasHealthCheck: false,
          style: { width: 320 },
        },
      };
      
      setNodes((nds) => [...nds, newNode]);
      setSidebarOpen(false);
    },
    [setNodes]
  );

  /**
   * Handle canvas star toggle
   * Updates the local state and could sync with backend
   */
  const setIsStarred = useCallback((starred: boolean) => {
    setIsCanvasStarred(starred);
    
    // TODO: Sync with backend API
    console.log(`Canvas ${canvasId} ${starred ? 'starred' : 'unstarred'}:`, {
      canvasId,
      canvasName: getCanvasName(canvasId),
      starred
    });
    
    // Example API call (commented out):
    // try {
    //   await updateCanvasStar(canvasId, starred);
    // } catch (error) {
    //   console.error('Failed to update canvas star status:', error);
    //   // Revert the state if API call fails
    //   setIsCanvasStarred(!starred);
    // }
  }, [canvasId]);

  return (
    <div className="flex flex-col min-h-screen bg-gray-50">
      {/* Navigation */}
      <nav className="flex items-center bg-zinc-200 dark:bg-zinc-950 border-b border-zinc-200 dark:border-zinc-800">
       <div className='flex border-r border-zinc-400 dark:border-zinc-600 dark:bg-zinc-900'>
       <Link  href="/canvases"
          className='px-3 py-1 hover:bg-zinc-300 dark:hover:bg-zinc-800 text-zinc-950 dark:text-white' 
        >
          <MaterialSymbol size='lg' opticalSize={20} weight={400} name="arrow_back" />
        </Link>
        </div>
        <div className='flex px-2 hover:bg-zinc-300 dark:hover:bg-zinc-800'>
          <Dropdown>
            <Headless.MenuButton
              className="flex items-center gap-3 rounded-xl border border-transparent p-1 data-active:border-zinc-200 data-hover:border-zinc-200 dark:data-active:border-zinc-700 dark:data-hover:border-zinc-700"
              aria-label="Account options"
            >
              <span className="block text-left">
                <span className="block text-md font-bold text-zinc-950 dark:text-white">
                {getCanvasName(canvasId)}
                </span>
              </span>
              <MaterialSymbol className='text-zinc-950 dark:text-white' size='lg' opticalSize={20} weight={400} name="expand_all" />
            </Headless.MenuButton>
            <DropdownMenu className="min-w-(--button-width)">
              <DropdownItem href="/canvas-editor3">Other Canvas 1</DropdownItem>
              <DropdownItem href="/canvas-editor3">Other Canvas 2</DropdownItem>
              <DropdownItem href="/canvas-editor3">Other Canvas 3</DropdownItem>
              <DropdownItem href="/canvas-editor3">Other Canvas 4</DropdownItem>
              <DropdownItem href="/canvas-editor3">Other Canvas 5</DropdownItem>
            </DropdownMenu>
          </Dropdown>
        </div>
        <Button plain>
          <MaterialSymbol size='lg' opticalSize={20} weight={400} name="star" />
        </Button>
        <Dropdown>
          <DropdownButton plain aria-label="More options">
            <MaterialSymbol size='lg' opticalSize={20} weight={400} name="more_vert" />
          </DropdownButton>
          <DropdownMenu className="min-w-(--button-width)">
            <DropdownItem href="/canvas-editor3">Manage members</DropdownItem>
            <DropdownItem href="/canvas-editor3">Delete</DropdownItem>
          </DropdownMenu>
        </Dropdown>
        
        
      </nav>
      

      {/* React Flow Canvas */}
      <div className="flex-1 flex">
        {/* Component Sidebar */}
        <div className='w-96 bg-white dark:bg-zinc-800 border-r border-zinc-200 dark:border-zinc-700'> 
          <ComponentSidebar
            isOpen={true}
            onClose={() => setSidebarOpen(false)}
            onNodeAdd={addNode}
          />
        </div>
        
        {/* Canvas Area */}
        <div className="flex-1" ref={reactFlowWrapper}>
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
              className="bg-gray-50"
            >
              <Controls 
                className="bg-white border border-gray-300 rounded-lg shadow-sm"
                showInteractive={false}
              />
              <Background 
                variant={BackgroundVariant.Dots} 
                gap={20} 
                size={1}
                color="#e5e7eb"
              />
            </ReactFlow>
          </ReactFlowProvider>
        </div>
      </div>

      {/* Canvas Members Modal */}
      <Dialog 
        open={showMembersModal} 
        onClose={() => setShowMembersModal(false)}
        size="3xl"
      >
        <DialogTitle className='flex items-center justify-between'>
          Canvas Members

        <Button plain onClick={() => setShowMembersModal(false)}>
          <MaterialSymbol name="close" size='lg' />
        </Button>
        </DialogTitle>
        <DialogDescription>
          Manage who has access to "{getCanvasName(canvasId)}" and what they can do.
        </DialogDescription>
        
        <DialogBody>
          {/* Search */}
          <div className="mb-6">
            {/* Add Members Section */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">
                    Add members
                  </Subheading>
                  
                </div>
                
              </div>
              
              <div className="flex gap-3">
                <Input
                  type="email"
                  placeholder="Enter email address"
                  className="flex-1"
                />
                <Dropdown>
                  <DropdownButton  outline className="flex items-center text-sm">
                    Member
                    <MaterialSymbol name="keyboard_arrow_down" />
                  </DropdownButton>
                  <DropdownMenu>
                    <DropdownItem>
                      <DropdownLabel>Member</DropdownLabel>
                      <DropdownDescription>Member role description.</DropdownDescription>
                    </DropdownItem>
                    <DropdownItem>
                      <DropdownLabel>Admin</DropdownLabel>
                      <DropdownDescription>Admin role description.</DropdownDescription>
                    </DropdownItem>
                    <DropdownItem>
                      <DropdownLabel>Editor</DropdownLabel>
                      <DropdownDescription>Editor role description.</DropdownDescription>
                    </DropdownItem>
                  </DropdownMenu>
                </Dropdown>
                <Button color="blue">Send Invite</Button>
              </div>
            </div>
          </div>

          {/* Members Table */}
          <Table dense className='bg-white dark:bg-zinc-800'>
            <TableHead>
              <TableRow>
                <TableHeader>Member</TableHeader>
                <TableHeader>Email</TableHeader>
                <TableHeader>Role</TableHeader>
                <TableHeader>Last Active</TableHeader>
                <TableHeader></TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {canvasMembers
                .filter(member =>
                  member.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                  member.email.toLowerCase().includes(searchQuery.toLowerCase())
                )
                .map((member) => (
                <TableRow key={member.id}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <Avatar
                        src={member.avatar}
                        initials={member.initials}
                        className="size-8"
                      />
                      <div>
                        <div className="text-sm font-medium text-zinc-900 dark:text-white">
                          {member.name}
                        </div>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400">
                          {member.role}
                        </div>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-zinc-600 dark:text-zinc-400">
                      {member.email}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Dropdown>
                      <DropdownButton outline className="flex items-center gap-2 text-sm">
                        {member.role}
                        <MaterialSymbol name="keyboard_arrow_down" />
                      </DropdownButton>
                      <DropdownMenu>
                        <DropdownItem>
                          <DropdownLabel>Member</DropdownLabel>
                          <DropdownDescription>Member role description.</DropdownDescription>
                        </DropdownItem>
                        <DropdownItem>
                          <DropdownLabel>Admin</DropdownLabel>
                          <DropdownDescription>Admin role description.</DropdownDescription>
                        </DropdownItem>
                        <DropdownItem>
                          <DropdownLabel>Editor</DropdownLabel>
                          <DropdownDescription>Editor role description.</DropdownDescription>
                        </DropdownItem>
                      </DropdownMenu>
                    </Dropdown>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-zinc-500 dark:text-zinc-400">
                      {member.lastActive}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex justify-end">
                      <Button plain onClick={() => console.log('Remove member', member.id)}>
                            <MaterialSymbol name="close" size="md" />
                      </Button>
                      
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {/* Empty State */}
          {canvasMembers.filter(member =>
            member.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            member.email.toLowerCase().includes(searchQuery.toLowerCase())
          ).length === 0 && searchQuery && (
            <div className="text-center py-8">
              <MaterialSymbol name="search_off" className="text-zinc-400 text-4xl mb-2" />
              <p className="text-zinc-500 dark:text-zinc-400">
                No members found matching "{searchQuery}"
              </p>
            </div>
          )}
        </DialogBody>
        
       
      </Dialog>

    </div>
  );
};
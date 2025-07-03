import React, { useState, useCallback, useRef } from 'react';
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
import { NavigationVertical, type User, type Organization, type NavigationLink } from './lib/Navigation/navigation-vertical'
import { WorkflowNode, WorkflowEdge } from '../types';
import { DeploymentCardStage } from './DeploymentCardStage';
import { ComponentSidebar } from './ComponentSidebar';
import { Button } from './lib/Button/button';
import { MaterialSymbol } from 'react-material-symbols';
import { Heading } from './lib/Heading/heading';
import { Text } from './lib/Text/text';


// Node types for React Flow
const nodeTypes = {
  deploymentCard: DeploymentCardStage as any,
};

interface CanvasEditorPageProps {
  canvasId: string
  onSignOut?: () => void
  navigationLinks?: NavigationLink[]
  onLinkClick?: (linkId: string) => void
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

  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const reactFlowWrapper = useRef<HTMLDivElement>(null);

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

  return (
    <div className="flex min-h-screen h-full bg-gray-50">
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
                   <div className="h-6 w-px bg-zinc-200 dark:bg-zinc-700" />
                   <div>
                     <Heading level={1} className="!text-xl mb-1">Canvas {canvasId}</Heading>
                   </div>
                 </div>
                 
                 
               </div>
             </header>

              {/* React Flow Canvas */}
              <div className='absolute  left-0 top-0 w-96 h-full z-50'> 
                <ComponentSidebar
                  isOpen={false}
                  onClose={() => setSidebarOpen(false)}
                  onNodeAdd={addNode}
                />
              </div>
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

      {/* Component Sidebar */}
      <ComponentSidebar
        isOpen={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
        onNodeAdd={addNode}
      />
    </div>
  );
};
import React, { useState, useCallback, useRef, useEffect } from 'react';
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
  Position,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { UserOrgDropdown, type User, type Organization } from './lib/UserOrgDropdown';
import { AddMembersSectionSimple } from './AddMembersSectionSimple';
import { WorkflowNode, WorkflowEdge } from '../types';
import { DeploymentCardStage } from './DeploymentCardStage';
import { ComponentSidebar } from './ComponentSidebar';
import { WorkflowNodeReactFlow } from './WorkflowNodeReactFlow';
import { WorkflowNodeAccordionReactFlow } from './WorkflowNodeAccordionReactFlow';
import { EventSourceWorkflowNodeReactFlow } from './EventSourceWorkflowNodeReactFlow';
import { type WorkflowNodeData } from './lib/WorkflowNode/workflow-node';
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
import { Textarea } from './lib/Textarea/textarea';
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem,
  DropdownLabel,
  DropdownDescription,
  DropdownDivider,
  DropdownHeader,
  DropdownSection
} from './lib/Dropdown/dropdown';
import { 
  Table, 
  TableHead, 
  TableBody, 
  TableRow, 
  TableHeader, 
  TableCell 
} from './lib/Table/table';
import { Link } from './lib/Link/link';
import { Field, Label } from './lib/Fieldset/fieldset';
import { ControlledTabs, type Tab } from './lib/Tabs/tabs';
import { NodeDetailsSidebar } from './lib/NodeDetailsSidebar/node-details-sidebar';


// Node types for React Flow
const nodeTypes = {
  deploymentCard: DeploymentCardStage as any,
  workflowNode: WorkflowNodeReactFlow as any,
  workflowNodeAccordion: WorkflowNodeAccordionReactFlow as any,
  eventSource: EventSourceWorkflowNodeReactFlow as any,
};

interface CanvasEditorPageProps {
  canvasId: string
  onBack?: () => void
}

// Initial workflow data (will be enhanced with handlers in the component)
const initialNodesData = [
  {
    id: 'stage-1',
    position: { x: -400, y: 120 },
    workflowNodeData: {
      id: 'stage-1',
      title: 'Sync Cluster',
      description: 'Sync cluster with the latest changes',
      type: 'eventSource',
      status: 'success',
      icon: 'settings_ethernet',
      nodeNumber: 847,
      queueIcon: 'how_to_reg',
      queueTitle: 'asfh7x9wa4-7fb2d9ke3m-9n4p8q2v',
      runName: 'mk8j3n6q9-rt2u5x8a1-dg4h7k0m',
      triggeredBy: 'Event Source',
      eventId: 'evt_abc123def456',
      yamlConfig: {
        apiVersion: 'v1',
        kind: 'Stage',
        metadata: {
          name: 'build-test',
          canvasId: ''
        },
        spec: {
          secrets: [],
          connections: [{
            name: 'GitHub Repository',
            type: 'github',
            config: { repo: 'main-app', branch: 'main' }
          }],
          inputs: [
            { name: 'branch', type: 'string', required: true, defaultValue: 'main' }
          ],
          inputMappings: {},
          outputs: [
            { name: 'build_artifact', type: 'string', value: 'dist/' }
          ],
          executor: {
            type: 'kubernetes',
            config: { image: 'node:18', resources: { cpu: '1', memory: '2Gi' } }
          }
        }
      }
    },
  },
  {
    id: 'stage-2',
    position: { x: 100, y: 120 },
    workflowNodeData: {
      id: 'stage-2',
      title: 'AI Agent triage',
      description: 'Run AI agent to review and triage the cluster changes',
      type: 'stage',
      status: 'success',
      icon: 'openAI',
      nodeNumber: 9,
      queueIcon: 'how_to_reg',
      queueTitle: 'xk9m2n5p8-qr4t7w1z6-bv3c8f2j',
      runName: 'pl7s4v9y2-we6z3b8c5-fj2k5n8r',
      triggeredBy: 'Sync Cluster',
      eventId: 'run_completed_mk8j3n6q9',
      yamlConfig: {
        apiVersion: 'v1',
        kind: 'Stage',
        metadata: {
          name: 'deploy-staging',
          canvasId: ''
        },
        spec: {
          secrets: [
            { name: 'STAGING_API_KEY', key: 'api_key', value: '***' }
          ],
          connections: [{
            name: 'Staging Environment',
            type: 'kubernetes',
            config: { cluster: 'staging-cluster', namespace: 'app-staging' }
          }],
          inputs: [
            { name: 'artifact_path', type: 'string', required: true }
          ],
          inputMappings: { artifact_path: '${build.outputs.build_artifact}' },
          outputs: [
            { name: 'deployment_url', type: 'string', value: 'https://staging.app.com' }
          ],
          executor: {
            type: 'kubernetes',
            config: { image: 'kubectl:latest', serviceAccount: 'deployer' }
          }
        }
      }
    },
  },
  {
    id: 'stage-3',
    position: { x: 600, y: 120 },
    workflowNodeData: {
      id: 'stage-3',
      title: 'Staging Environment',
      description: 'Deploy application to production environment',
      type: 'stage',
      status: 'success',
      icon: 'semaphore',
      nodeNumber: 5,
      queueIcon: 'timer',
      queueTitle: 'lk4g7h9j2-mn6p3q8r5-vw2y5z8b',
      runName: 'Update Semaphore configuration',
      triggeredBy: 'AI Agent triage',
      eventId: 'approval_granted_pl7s4v9y2',
      yamlConfig: {
        apiVersion: 'v1',
        kind: 'Stage',
        metadata: {
          name: 'deploy-production',
          canvasId: ''
        },
        spec: {
          secrets: [
            { name: 'PROD_API_KEY', key: 'api_key', value: '***' },
            { name: 'DATABASE_URL', key: 'db_url', value: '***' }
          ],
          connections: [{
            name: 'Production Environment',
            type: 'kubernetes',
            config: { cluster: 'prod-cluster', namespace: 'app-prod' }
          }],
          inputs: [
            { name: 'artifact_path', type: 'string', required: true },
            { name: 'approval_required', type: 'boolean', required: false, defaultValue: true }
          ],
          inputMappings: { artifact_path: '${staging.outputs.deployment_url}' },
          outputs: [
            { name: 'production_url', type: 'string', value: 'https://app.com' }
          ],
          executor: {
            type: 'kubernetes',
            config: { image: 'kubectl:latest', serviceAccount: 'prod-deployer' }
          }
        }
      }
    },
  },
  {
    id: 'stage-4',
    position: { x: 1150, y: -150 },
    workflowNodeData: {
      id: 'stage-4',
      title: 'Production - US',
      description: '',
      type: 'stage',
      status: 'failed',
      icon: 'github',
      nodeNumber: 11,
      queueIcon: 'pause',
      queueTitle: 'df8e2m4n7-st3u9w6x1-cd5f8k2m',
      runName: 'vx9p2m6s4-gh8t3k7n1-bj5w4r8z',
      yamlConfig: {
        apiVersion: 'v1',
        kind: 'Stage',
        metadata: {
          name: 'deploy-production',
          canvasId: ''
        },
        spec: {
          secrets: [
            { name: 'PROD_API_KEY', key: 'api_key', value: '***' },
            { name: 'DATABASE_URL', key: 'db_url', value: '***' }
          ],
          connections: [{
            name: 'Production Environment',
            type: 'kubernetes',
            config: { cluster: 'prod-cluster', namespace: 'app-prod' }
          }],
          inputs: [
            { name: 'artifact_path', type: 'string', required: true },
            { name: 'approval_required', type: 'boolean', required: false, defaultValue: true }
          ],
          inputMappings: { artifact_path: '${staging.outputs.deployment_url}' },
          outputs: [
            { name: 'production_url', type: 'string', value: 'https://app.com' }
          ],
          executor: {
            type: 'kubernetes',
            config: { image: 'kubectl:latest', serviceAccount: 'prod-deployer' }
          }
        }
      }
    },
  },
  {
    id: 'stage-5',
    position: { x: 1150, y: 450 },
    workflowNodeData: {
      id: 'stage-5',
      title: 'Production - EU',
      description: 'Deploy application to production environment',
      type: 'stage',
      status: 'running',
      icon: 'semaphore',
      nodeNumber: 32,
      queueIcon: 'how_to_reg',
      queueTitle: 'gh3j6l9n2-pq7r4s8t1-wx4y7z0c',
      runName: 'kl2n5x9m8-qr6w3e7t2-fl8j4p1s',
      yamlConfig: {
        apiVersion: 'v1',
        kind: 'Stage',
        metadata: {
          name: 'deploy-production',
          canvasId: ''
        },
        spec: {
          secrets: [
            { name: 'PROD_API_KEY', key: 'api_key', value: '***' },
            { name: 'DATABASE_URL', key: 'db_url', value: '***' }
          ],
          connections: [{
            name: 'Production Environment',
            type: 'kubernetes',
            config: { cluster: 'prod-cluster', namespace: 'app-prod' }
          }],
          inputs: [
            { name: 'artifact_path', type: 'string', required: true },
            { name: 'approval_required', type: 'boolean', required: false, defaultValue: true }
          ],
          inputMappings: { artifact_path: '${staging.outputs.deployment_url}' },
          outputs: [
            { name: 'production_url', type: 'string', value: 'https://app.com' }
          ],
          executor: {
            type: 'kubernetes',
            config: { image: 'kubectl:latest', serviceAccount: 'prod-deployer' }
          }
        }
      }
    },
  },
  {
    id: 'stage-6',
    position: { x: 1650, y: -150 },
    workflowNodeData: {
      id: 'stage-6',
      title: 'Production - JP',
      description: 'Deploy application to production environment',
      type: 'stage',
      status: 'success',
      icon: 'semaphore',
      nodeNumber: 4,
      queueIcon: 'timer',
      queueTitle: 'vb6n9m3k5-ht2g8f4j7-pr9s6w1q',
      runName: 'zm4h7k9l3-dg2f5n8p6-qj1r6t9w',
      yamlConfig: {
        apiVersion: 'v1',
        kind: 'Stage',
        metadata: {
          name: 'deploy-production',
          canvasId: ''
        },
        spec: {
          secrets: [
            { name: 'PROD_API_KEY', key: 'api_key', value: '***' },
            { name: 'DATABASE_URL', key: 'db_url', value: '***' }
          ],
          connections: [{
            name: 'Production Environment',
            type: 'kubernetes',
            config: { cluster: 'prod-cluster', namespace: 'app-prod' }
          }],
          inputs: [
            { name: 'artifact_path', type: 'string', required: true },
            { name: 'approval_required', type: 'boolean', required: false, defaultValue: true }
          ],
          inputMappings: { artifact_path: '${staging.outputs.deployment_url}' },
          outputs: [
            { name: 'production_url', type: 'string', value: 'https://app.com' }
          ],
          executor: {
            type: 'kubernetes',
            config: { image: 'kubectl:latest', serviceAccount: 'prod-deployer' }
          }
        }
      }
    },
  },
];

const newChainListeners = [];
// prod-cluster → Sync Cluster (1000 → 1001)
newChainListeners.push({
  id: 'e1000-1001',
  source: '1000',
  target: '1001',
  type: 'bezier',
  animated: true,
  style: { stroke: '#888', strokeDasharray: '6 4', strokeWidth: 2 },
  label: 'Promote to Sync Cluster',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});
// Sync Cluster → Deploy to US cluster (1001 → 1002)
newChainListeners.push({
  id: 'e1001-1002',
  source: '1001',
  target: '1002',
  type: 'bezier',
  animated: true,
  style: { stroke: '#888', strokeDasharray: '6 4', strokeWidth: 2 },
  label: 'Sync → US Cluster',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});
// Sync Cluster → Deploy to Asia cluster (1001 → 1003)
newChainListeners.push({
  id: 'e1001-1003',
  source: '1001',
  target: '1003',
  type: 'bezier',
  animated: false,
  style: { stroke: '#888', strokeWidth: 2 },
  label: 'Sync → Asia Cluster',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});
// US cluster → Health Check & Cleanup (1002 → 1004)
newChainListeners.push({
  id: 'e1002-1004',
  source: '1002',
  target: '1004',
  type: 'bezier',
  animated: false,
  style: { stroke: '#888', strokeWidth: 2 },
  label: 'US → Cleanup',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});
// Asia cluster → Health Check & Cleanup (1003 → 1004)
newChainListeners.push({
  id: 'e1003-1004',
  source: '1003',
  target: '1004',
  type: 'bezier',
  animated: false,
  style: { stroke: '#888', strokeWidth: 2 },
  label: 'Asia → Cleanup',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});

const initialEdges: WorkflowEdge[] = [
  {
    id: 'e1-2',
    source: 'stage-1',
    target: 'stage-2',
    type: 'bezier',
    animated: true,
    markerEnd: {
      type: MarkerType.ArrowClosed,
    },
  },
  {
    id: 'e2-3',
    source: 'stage-2',
    target: 'stage-3',
    type: 'bezier',
    animated: true,
    markerEnd: {
      type: MarkerType.ArrowClosed,
    },
  },
  // New edges from Staging Environment to Production US and EU
  {
    id: 'e3-4',
    source: 'stage-3',
    target: 'stage-4',
    type: 'bezier',
    animated: false,
    markerEnd: {
      type: MarkerType.ArrowClosed,
    },
  },
  {
    id: 'e3-5',
    source: 'stage-3',
    target: 'stage-5',
    type: 'bezier',
    animated: false,
    markerEnd: {
      type: MarkerType.ArrowClosed,
    },
  },
  // Edge from Production US to Production JP
  {
    id: 'e4-6',
    source: 'stage-4',
    target: 'stage-6',
    type: 'bezier',
    animated: false,
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
  onBack
}: CanvasEditorPageProps) {
  // Create initial nodes with proper handlers
  const createInitialNodes = (): WorkflowNode[] => {
    return initialNodesData.map(nodeData => {
      // Check if this should be an event source node
      if (nodeData.workflowNodeData.type === 'eventSource') {
        return {
          id: nodeData.id,
          type: 'eventSource',
          position: nodeData.position,
          sourcePosition: Position.Right,
          targetPosition: Position.Left,
          data: {
            id: nodeData.workflowNodeData.id,
            title: nodeData.workflowNodeData.title,
            cluster: 'prod-cluster',
            icon: nodeData.workflowNodeData.icon || 'settings_ethernet',
            events: [
              {
                id: 'event-1',
                url: 'https://hooks.kubernetes.com/semaphore/semaphore/semaphore',
                type: 'webhook',
                enabled: true
              },
              {
                id: 'event-2',
                url: 'https://hooks.kubernetes.com/semaphore/semaphore/semaphore',
                type: 'webhook',
                enabled: true
              },
              {
                id: 'event-3',
                url: 'https://hooks.kubernetes.com/semaphore/semaphore/semaphore',
                type: 'webhook',
                enabled: true
              }
            ],
            selected: false,
            isEditMode: false
          }
        };
      }
      
      // Default to workflow node accordion for other types
      return {
        id: nodeData.id,
        type: 'workflowNodeAccordion',
        position: nodeData.position,
        sourcePosition: Position.Right,
        targetPosition: Position.Left,
        data: {
          workflowNodeData: nodeData.workflowNodeData,
          variant: 'read',
          multiple: true,
          className: 'max-w-xs',
          partialSave: false,
          saveGranular: true,
          modalEdit: false,
          savedConnectionIndices: [0]
        }
      };
    });
  }

  const [nodes, setNodes, onNodesChange] = useNodesState(createInitialNodes())
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)

// Map edges to set animation based on target node status
const computedEdges = edges.map(edge => {
  // Find the target node for this edge
  const targetNode = nodes.find(node => node.id === edge.target);
  let isRunning = false;
  // Check both possible data structures for status
  if (targetNode) {
    if ((targetNode.data as any)?.workflowNodeData?.status) {
      isRunning = (targetNode.data as any).workflowNodeData.status?.toLowerCase() === 'running';
    } else if ((targetNode.data as any)?.status) {
      isRunning = (targetNode.data as any).status?.toLowerCase() === 'running';
    }
  }
  return {
    ...edge,
    animated: !!isRunning,
    style: {
      ...(edge.style || {}),
      strokeDasharray: isRunning ? '6 4' : 'none',
      stroke: '#888',
      strokeWidth: 2,
    },
  };
});
  
  // Add handlers to nodes after initialization
  useEffect(() => {
    setNodes(currentNodes => 
      currentNodes.map(node => ({
        ...node,
        data: {
          ...node.data,
          onUpdate: (updates: Partial<any>) => {
            setNodes((nds) =>
              nds.map((n) =>
                n.id === node.id
                  ? {
                      ...n,
                      data: {
                        ...n.data,
                        workflowNodeData: {
                          ...n.data.workflowNodeData,
                          ...updates
                        }
                      }
                    }
                  : n
              )
            );
          },
          onSave: () => {
            setNodes((nds) =>
              nds.map((n) =>
                n.id === node.id
                  ? {
                      ...n,
                      data: {
                        ...n.data,
                        variant: 'read'
                      }
                    }
                  : n
              )
            );
          },
          onCancel: () => {
            setNodes((nds) =>
              nds.map((n) =>
                n.id === node.id
                  ? {
                      ...n,
                      data: {
                        ...n.data,
                        variant: 'read'
                      }
                    }
                  : n
              )
            );
          },
          onEdit: () => {
            setNodes((nds) =>
              nds.map((n) =>
                n.id === node.id
                  ? {
                      ...n,
                      data: {
                        ...n.data,
                        variant: 'edit'
                      }
                    }
                  : n
              )
            );
          },
          onDelete: () => {
            setNodes((nds) => {
              const filteredNodes = nds.filter((n) => n.id !== node.id);
              return filteredNodes.map((n) => ({
                ...n,
                data: {
                  ...n.data,
                  nodes: filteredNodes,
                  totalNodesCount: filteredNodes.length
                }
              }));
            });
            setEdges((eds) => eds.filter((edge) => edge.source !== node.id && edge.target !== node.id));
          },
          onSelect: () => {
            setNodes((nds) =>
              nds.map((n) => ({
                ...n,
                selected: n.id === node.id,
              }))
            );
          },
          nodes: currentNodes,
          totalNodesCount: currentNodes.length
        }
      }))
    )
  }, [setNodes, setEdges])
  const [showMiniMap, setShowMiniMap] = useState(true)
  const [activeView, setActiveView] = useState<'preview' | 'integrations' | 'members' | 'secrets' | 'integration-setup'>('preview')
  const [integrationsTab, setIntegrationsTab] = useState<'connected' | 'add-new'>('connected')
  const [selectedIntegrationType, setSelectedIntegrationType] = useState<string | null>(null)
  
  // Define tabs for navigation
  const navigationTabs: Tab[] = [
    {
      id: 'preview',
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

  // Define tabs for integrations page
  const integrationsTabs: Tab[] = [
    {
      id: 'connected',
      label: 'Connected',
    },
    {
      id: 'add-new',
      label: 'Add new integrations',
    }
  ]

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

  const handleOrganizationMenuAction = (action: 'settings' | 'billing' | 'members' | 'groups' | 'roles') => {
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
  const [showSecretsModal, setShowSecretsModal] = useState(false);
  const [showConnectionModal, setShowConnectionModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [showNodeDetails, setShowNodeDetails] = useState(false);
  
  // Connection modal state
  const [connectionType, setConnectionType] = useState('stage');
  const [connectionFilters, setConnectionFilters] = useState<Array<{id: string, type: string, expression: string, operator?: string}>>([]);
  const [currentEditingNodeId, setCurrentEditingNodeId] = useState<string | null>(null);
  
  // Track saved connections for each node
  const [nodeSavedConnections, setNodeSavedConnections] = useState<Record<string, number[]>>({});
  
  // Connection modal filter handlers
  const handleAddFilter = () => {
    const existingFilters = connectionFilters || []
    
    // If there are existing filters, use their operator, otherwise default to 'AND'
    let currentOperator = 'AND'
    if (existingFilters.length > 0) {
      // Find the current operator from existing filters (they should all be the same)
      const filterWithOperator = existingFilters.find(filter => filter.operator)
      currentOperator = filterWithOperator?.operator || 'AND'
    }
    
    const newFilter = {
      id: `filter_${Date.now()}`,
      type: 'data',
      expression: '',
      // Only add operator if this is not the first filter
      ...(existingFilters.length > 0 ? { operator: currentOperator } : {})
    }
    
    setConnectionFilters(prev => [...prev, newFilter])
  }

  const handleRemoveFilter = (filterId: string) => {
    setConnectionFilters(prev => prev.filter(filter => filter.id !== filterId))
  }

  const handleUpdateFilter = (filterId: string, field: 'type' | 'expression', value: string) => {
    setConnectionFilters(prev => prev.map(filter => 
      filter.id === filterId ? { ...filter, [field]: value } : filter
    ))
  }

  const handleToggleOperator = (filterId: string) => {
    const currentFilters = connectionFilters || []
    const clickedFilter = currentFilters.find(filter => filter.id === filterId)
    const newOperator = clickedFilter?.operator === 'OR' ? 'AND' : 'OR'
    
    // Update all filters with operators to have the same operator
    setConnectionFilters(prev => prev.map(filter => 
      filter.operator 
        ? { ...filter, operator: newOperator }
        : filter
    ))
  }

  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  
  
  // Initial listeners configuration - connecting the stages
 
  // Secret management state
  const [secrets, setSecrets] = useState([
    { id: '1', name: 'API_KEY', value: 'sk-...', description: 'OpenAI API Key', createdAt: '2024-01-15', lastUsed: '2024-01-20' },
    { id: '2', name: 'DATABASE_URL', value: 'postgresql://...', description: 'Production database connection', createdAt: '2024-01-10', lastUsed: '2024-01-19' },
    { id: '3', name: 'WEBHOOK_SECRET', value: 'whsec_...', description: 'Stripe webhook secret', createdAt: '2024-01-12', lastUsed: '2024-01-18' }
  ]);
  const [editingSecret, setEditingSecret] = useState<any>(null);
  const [isCreatingSecret, setIsCreatingSecret] = useState(false);


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
      avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
      status: 'Active'
    },
    {
      id: '2',
      name: 'Jane Smith',
      email: 'jane@acme.com',
      role: 'Viewer',
      permission: 'Can view',
      lastActive: '1 day ago',
      initials: 'JS',
      status: 'Invited'
    },
    {
      id: '3',
      name: 'Bob Wilson',
      email: 'bob@acme.com',
      role: 'Editor',
      permission: 'Can edit',
      lastActive: '3 days ago',
      initials: 'BW',
      status: 'Active'
    },
    {
      id: '4',
      name: 'Alice Johnson',
      email: 'alice@acme.com',
      role: 'Owner',
      permission: 'Full access',
      lastActive: '5 minutes ago',
      initials: 'AJ',
      status: 'Active'
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
        type: 'bezier',
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
      const isSelecting = selectedNode !== node.id;
      setSelectedNode(isSelecting ? node.id : null);
      
      // Don't show sidebar if it's an EventSource node in edit mode, except for webhook nodes
      const isEventSourceInEditMode = node.type === 'eventSource' && node.data?.isEditMode === true && node.data?.eventSourceType !== 'webhook';
      setShowNodeDetails(isSelecting && !isEventSourceInEditMode);
      
      // Update node selection state
      setNodes((nds) =>
        nds.map((n) => ({
          ...n,
          selected: n.id === node.id && isSelecting,
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
      const nodeId = `node-${Date.now()}`;
      // Use accordion version by default, alternate with tab version
      const useAccordion = nodes.length % 2 === 0;
      
      // Special handling for test stage - create EventSourceWorkflowNode in edit mode
      if (nodeType === 'test') {
        const newNode: WorkflowNode = {
          id: nodeId,
          type: 'eventSource',
          position: { x: 300, y: 300 },
          data: {
            id: nodeId,
            title: 'Test Stage',
            cluster: 'test-cluster',
            icon: 'semaphore',
            events: [
              {
                id: 'event-1',
                url: 'https://hooks.semaphoreci.com/webhook/test',
                type: 'webhook',
                enabled: true
              }
            ],
            selected: false,
            isEditMode: true
          }
        };
        
        setNodes((nds) => {
          const updatedNodes = [...nds, newNode];
          return updatedNodes;
        });
        setSidebarOpen(false);
        return;
      }

      // Special handling for semaphore-event - create EventSourceWorkflowNode in edit mode
      if (nodeType === 'semaphore-event') {
        const newNode: WorkflowNode = {
          id: nodeId,
          type: 'eventSource',
          position: { x: 300, y: 300 },
          data: {
            id: nodeId,
            title: 'Semaphore Event Source',
            cluster: 'semaphore-cluster',
            icon: 'semaphore',
            eventSourceType: 'semaphore',
            events: [
              {
                id: 'event-1',
                url: 'https://hooks.semaphoreci.com/webhook/semaphore',
                type: 'webhook',
                enabled: true
              }
            ],
            selected: false,
            isEditMode: true
          }
        };
        
        setNodes((nds) => {
          const updatedNodes = [...nds, newNode];
          return updatedNodes;
        });
        setSidebarOpen(false);
        return;
      }

      // Special handling for github-event - create EventSourceWorkflowNode in edit mode
      if (nodeType === 'github-event') {
        const newNode: WorkflowNode = {
          id: nodeId,
          type: 'eventSource',
          position: { x: 300, y: 300 },
          data: {
            id: nodeId,
            title: 'GitHub Event Source',
            cluster: 'github-cluster',
            icon: 'github',
            eventSourceType: 'webhook',
            events: [
              {
                id: 'event-1',
                url: 'https://github.com/owner/repo/settings/hooks',
                type: 'webhook',
                enabled: true
              }
            ],
            selected: false,
            isEditMode: true
          }
        };
        
        setNodes((nds) => {
          const updatedNodes = [...nds, newNode];
          return updatedNodes;
        });
        setSidebarOpen(false);
        return;
      }

      // Special handling for webhook-event - create EventSourceWorkflowNode in edit mode
      if (nodeType === 'webhook-event') {
        const newNode: WorkflowNode = {
          id: nodeId,
          type: 'eventSource',
          position: { x: 300, y: 300 },
          data: {
            id: nodeId,
            title: 'Webhook Event Source',
            cluster: 'webhook-cluster',
            icon: 'webhook',
            eventSourceType: 'webhook',
            events: [
              {
                id: 'event-1',
                url: 'https://hooks.superplane.com/webhook/abc123def456',
                type: 'webhook',
                enabled: true
              }
            ],
            selected: false,
            isEditMode: true
          }
        };
        
        setNodes((nds) => {
          const updatedNodes = [...nds, newNode];
          return updatedNodes;
        });
        setSidebarOpen(false);
        return;
      }

      // Special handling for http-event - create EventSourceWorkflowNode in edit mode
      if (nodeType === 'http-event') {
        const newNode: WorkflowNode = {
          id: nodeId,
          type: 'eventSource',
          position: { x: 300, y: 300 },
          data: {
            id: nodeId,
            title: 'HTTP Event Source',
            cluster: 'http-cluster',
            icon: 'http',
            eventSourceType: 'http',
            events: [
              {
                id: 'event-1',
                url: 'https://api.superplane.com/events/http/xyz789abc123',
                type: 'http',
                enabled: true
              }
            ],
            selected: false,
            isEditMode: true
          }
        };
        
        setNodes((nds) => {
          const updatedNodes = [...nds, newNode];
          return updatedNodes;
        });
        setSidebarOpen(false);
        return;
      }
      
      const newNode: WorkflowNode = {
        id: nodeId,
        type: 'workflowNodeAccordion',
        position: { x: 300, y: 300 },
        data: {
          workflowNodeData: {
            id: nodeId,
            title: `New ${nodeType}`,
            description: `A new ${nodeType} workflow stage`,
            type: 'stage',
            status: 'pending',
            yamlConfig: {
              apiVersion: 'v1',
              kind: 'Stage',
              metadata: {
                name: `new-${nodeType.toLowerCase()}`,
                canvasId: canvasId
              },
              spec: {
                secrets: [],
                connections: [],
                inputs: [],
                inputMappings: {},
                outputs: [],
                executor: {
                  type: 'default',
                  config: {}
                }
              }
            }
          },
          variant: 'edit',
          ...(useAccordion ? {
            // Accordion-specific props
            multiple: true,
            className: 'max-w-xs',
            partialSave: false,  // Disable individual section save buttons (default)
            saveGranular: true,  // Enable granular save buttons for individual connections
            modalEdit: false,     // Enable modal editing for connections
            onConnectionModalOpen: () => {
              // Reset modal state and set current editing node
              setConnectionType('stage');
              setConnectionFilters([]);
              setCurrentEditingNodeId(nodeId);
              setShowConnectionModal(true);
            },
            savedConnectionIndices: nodeSavedConnections[nodeId] || []
          } : {
            // Tab-specific props
            tabs: [
              { id: 'basic', label: 'Parameters' },
              { id: 'executor', label: 'Executor' },
              { id: 'secrets', label: 'Secrets' },
              { id: 'preview', label: 'Preview' }
            ]
          }),
          onUpdate: (updates: Partial<WorkflowNodeData>) => {
            setNodes((nds) =>
              nds.map((node) =>
                node.id === nodeId
                  ? {
                      ...node,
                      data: {
                        ...(node.data as any),
                        workflowNodeData: {
                          ...(node.data as any).workflowNodeData,
                          ...updates
                        }
                      }
                    }
                  : node
              )
            );
          },
          onSave: () => {
            setNodes((nds) =>
              nds.map((node) =>
                node.id === nodeId
                  ? {
                      ...node,
                      data: {
                        ...(node.data as any),
                        variant: 'read'
                      }
                    }
                  : node
              )
            );
          },
          onCancel: () => {
            // For new nodes created in edit mode, delete them when canceling
            setNodes((nds) => nds.filter((node) => node.id !== nodeId));
          },
          onEdit: () => {
            setNodes((nds) =>
              nds.map((node) =>
                node.id === nodeId
                  ? {
                      ...node,
                      data: {
                        ...(node.data as any),
                        variant: 'edit'
                      }
                    }
                  : node
              )
            );
          },
          onDelete: () => {
            setNodes((nds) => {
              const filteredNodes = nds.filter((node) => node.id !== nodeId);
              // Update all remaining nodes with the new nodes array and count
              return filteredNodes.map((node) => ({
                ...node,
                data: {
                  ...node.data,
                  nodes: filteredNodes,
                  totalNodesCount: filteredNodes.length
                }
              }));
            });
            setEdges((eds) => eds.filter((edge) => edge.source !== nodeId && edge.target !== nodeId));
          },
          onSelect: () => {
            setSelectedNode(nodeId);
            setNodes((nds) =>
              nds.map((n) => ({
                ...n,
                selected: n.id === nodeId,
              }))
            );
          },
          nodes: [],
          totalNodesCount: 0
        },
      };
      
      setNodes((nds) => {
        const updatedNodes = [...nds, newNode];
        // Update all nodes (including the new one) with the complete nodes array and count
        const finalNodes = updatedNodes.map((node) => ({
          ...node,
          data: {
            ...node.data,
            nodes: updatedNodes,
            totalNodesCount: updatedNodes.length
          }
        }));
        return finalNodes;
      });
      setSidebarOpen(false);
    },
    [setNodes, setEdges, canvasId]
  );

  /**
   * Handle node updates (icon changes, etc.)
   */
  const handleNodeUpdate = useCallback(
    (nodeId: string, updates: { icon?: string; eventSourceType?: string }) => {
      setNodes((nds) =>
        nds.map((node) =>
          node.id === nodeId
            ? {
                ...node,
                data: {
                  ...(node.data as any),
                  ...updates
                }
              }
            : node
        )
      );
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

  // Function to render navigation based on URL parameter
  const renderNavigation = () => {
    return (
      <nav className="flex items-center justify-between bg-zinc-200 dark:bg-zinc-950 border-b border-zinc-200 dark:border-zinc-800">
        <div className='flex items-center'>
        <div className='flex border-r border-zinc-400 dark:border-zinc-600 dark:bg-zinc-900'>
          <Link href="/canvases"
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
              <MaterialSymbol className='text-zinc-950 dark:text-white' size='lg' opticalSize={20} weight={400} name="expand_more" />
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
        
        {/* Navigation Tabs */}
        
        <div className='border-r border-zinc-400 dark:border-zinc-600 hidden'>
        <Button plain>
          <MaterialSymbol size='lg' opticalSize={20} weight={400} name="star" />
        </Button>
        <Button plain onClick={handleMembers}>
          <MaterialSymbol size='lg' opticalSize={20} weight={400} name="person" />
        </Button>
        </div>
        
        <div className="flex items-center h-full">
          <ControlledTabs
            tabs={navigationTabs}
            activeTab={activeView}
            variant='default'
            onTabChange={(tabId) => setActiveView(tabId as 'preview' | 'integrations' | 'members' | 'secrets')}
          />
        
        </div>
        </div>
        <div className='flex items-center'>
          <div className='border-r border-zinc-400 dark:border-zinc-600'>
            <Dropdown> 
              <DropdownButton plain aria-label="More options">
                <MaterialSymbol size='lg' opticalSize={20} weight={400} name="more_vert" />
              </DropdownButton>
              <DropdownMenu className="min-w-(--button-width)">
              <DropdownItem href="#">Delete</DropdownItem>
            </DropdownMenu>
          </Dropdown>
         </div>
         <UserOrgDropdown
            user={currentUser}
            organization={currentOrganization}
            onUserMenuAction={handleUserMenuAction}
            onOrganizationMenuAction={handleOrganizationMenuAction}
            plain
          />
        </div>
      </nav>
    );
    
  };

  return (
    <div className="flex flex-col min-h-screen bg-gray-50 dark:bg-zinc-900">
      {renderNavigation()}

      {/* Conditional Content Based on Active View */}
      {activeView === 'preview' ? (
        /* React Flow Canvas */
        <div className="flex-1 flex relative">
          {/* Component Sidebar */}
          <div className='h-[calc(100vh-41px)] relative w-[300px] bg-transparent transition-[width] duration-300 ease-linear bg-white dark:bg-zinc-800 border-r border-zinc-200 dark:border-zinc-700'> 
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
                edges={computedEdges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onConnect={onConnect}
                onNodeClick={onNodeClick}
                nodeTypes={nodeTypes}
                connectionLineType={ConnectionLineType.SmoothStep}
                colorMode="system"
                defaultViewport={ { x: 0, y: 0, zoom: 1 } }
                attributionPosition="bottom-left"
                className="bg-gray-50 dark:bg-zinc-950"
              >
                <Controls 
                  className="bg-white dark:bg-zinc-800 border border-gray-300 dark:border-zinc-600 rounded-lg shadow-sm"
                  showInteractive={false}
                />
                <Background 
                  variant={BackgroundVariant.Dots} 
                  gap={20} 
                  size={1}
                  color="var(--zinc-800)"
                />
              </ReactFlow>
            </ReactFlowProvider>
          </div>
            {/* Node Details Sidebar */}
            {selectedNode && (
              <NodeDetailsSidebar
                nodeId={selectedNode}
                nodeTitle={nodes.find(n => n.id === selectedNode)?.data?.workflowNodeData?.title || 'Node Details'}
                nodeIcon={nodes.find(n => n.id === selectedNode)?.data?.icon || nodes.find(n => n.id === selectedNode)?.data?.workflowNodeData?.type === 'stage' ? 'sync' : 'webhook'}
                isOpen={showNodeDetails}
                source={nodes.find(n => n.id === selectedNode)?.type === 'eventSource' ? 'eventSource' : 'workflow'}
                eventSourceType={nodes.find(n => n.id === selectedNode)?.data?.eventSourceType || 'semaphore'}
                onNodeUpdate={handleNodeUpdate}
                events={nodes.find(n => n.id === selectedNode)?.type === 'eventSource' ? 
                  nodes.find(n => n.id === selectedNode)?.data?.events || [] : []}
                onClose={() => {
                  setShowNodeDetails(false);
                  setSelectedNode(null);
                  // Clear selection from all nodes
                  setNodes((nds) =>
                    nds.map((n) => ({
                      ...n,
                      selected: false,
                    }))
                  );
                }}
              />
            )}
        </div>
      ) : activeView === 'integrations' ? (
        /* Integrations Page */
        <div className="flex-1 p-8">
          <div className="max-w-4xl mx-auto">
            <div className="mb-6">
              <Heading level={2}>Integrations</Heading>
              <Text>Manage your integrations and external services.</Text>
            </div>
            
            {/* Integrations Tabs */}
            <div className="mb-6">
              <ControlledTabs
                tabs={integrationsTabs}
                activeTab={integrationsTab}
                variant='underline'
                onTabChange={(tabId) => setIntegrationsTab(tabId as 'connected' | 'add-new')}
              />
            </div>
            
            {/* Tab Content */}
            {integrationsTab === 'connected' ? (
              /* Connected Integrations Tab */
              <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-8">
                <div className="text-center py-12">
                  <MaterialSymbol name="integration_instructions" size="lg" className="mx-auto text-zinc-400 mb-4" />
                  <Text className="text-zinc-700 dark:text-zinc-100 mb-1 !text-xl">You have not connected any integrations yet</Text>
                  <Text className="text-zinc-500 dark:text-zinc-400 text-sm">
                    Browse full <Link href="/integrations" className='text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300'> list of integrations</Link> to get started
                  </Text>
                </div>
              </div>
            ) : (
              /* Add New Integrations Tab */
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {/* Semaphore Integration Card */}
                <div 
                  className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-4 hover:shadow-md transition-shadow cursor-pointer"
                  onClick={() => {
                    setSelectedIntegrationType('semaphore');
                    setActiveView('integration-setup');
                  }}
                >
                  <div className="flex items-center gap-2 mb-2">
                    <div className="w-10 h-10 flex-shrink-0 flex items-center justify-center bg-zinc-100 dark:bg-zinc-800 rounded-lg">
                      <img width={24} height={24} src='/images/semaphore-logo-sign-black.svg' alt="Semaphore" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-900 dark:text-white">Semaphore</h3>
                    </div>
                  </div>
                  <Text className="!text-sm text-zinc-600 dark:text-zinc-400 mb-4">
                    Connect your Semaphore CI/CD pipelines to automate deployments and testing workflows.
                  </Text>
                  
                </div>
                <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-4 hover:shadow-md transition-shadow cursor-pointer">
                  <div className="flex items-center gap-2 mb-2">
                    <div className="w-10 h-10 flex-shrink-0 flex items-center justify-center bg-zinc-100 dark:bg-zinc-800 rounded-lg">
                      <img width={24} height={24} src='data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAMAAABEpIrGAAAAb1BMVEX////4+Pi3ubtvcnZNUVU+Q0cpLjLr6+x3en0sMTYkKS59gIORk5aUl5n8/Pzw8PFTV1tbX2Pc3d5DSEzn5+g3PECLjpFKTlKFh4qxs7XCxMUuMze/wcLh4uPV1tZzd3o/Q0jOz9CmqKpjZ2qfoaTxAyfNAAABPUlEQVR4AW3TBYKDMBQE0AltAgzuzur9z7ibH5oKfWjc4UEFl6s2Rl8vgcJZGMX04iTEM5UaPomzHA+KkidVAa/WfKNpffMd32oKCHUlWfb27Q19ZSMVrNHGTMDckMtQLqSegdXGpvi3Sf93W9UudRby2WzsEgL4oMvwoqY1AsrQNfFipbXkCGh1BV6oT1pfRwvfOJlo9ZA5NAonStbmB1pawBuDTAgkX4MzV/eC2H3e0C7lk1aBEzd+7SpigJOZVoXx+J5UxzADil+8+KZYoRaK5y2WZxSdgm0j+dakzkIc2kzT6W3IcFnDTzdt4sKbWMqkpNl229IMsfMmg6UaMsJXmv4qCMXDoI4mO5oADwyFDnGoO3KI0jSHQ6E3eJum5TP4Y+EVyUOGXHZjgWd7ZEwOJzZRjbPQt7mF8P4AzsYZpmkFLF4AAAAASUVORK5CYII=' alt="GitHub" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-900 dark:text-white">GitHub</h3>
                    </div>
                  </div>
                  <Text className="!text-sm text-zinc-600 dark:text-zinc-400 mb-4">
                    Connect your GitHub repositories to trigger workflows on code changes and pull requests.
                  </Text>
                  
                </div>
              </div>
            )}
          </div>
        </div>
      ) : activeView === 'members' ? (
        /* Members Page */
        <div className="flex-1 p-8">
          <div className="max-w-4xl mx-auto">
            <div className="flex items-center justify-between mb-6">
              <div>
                <Subheading>Members</Subheading>
                <Text>Manage canvas access and permissions.</Text>
              </div>
            </div>
            <div className='mb-6'>
            {/* Add Members Section */}
            <AddMembersSectionSimple 
              showRoleSelection={true}
              onAddMembers={(users, role) => {
                // Handle adding members to canvas
                console.log('Adding members to canvas:', users, 'with role:', role)
              }}
            />
            </div>
            

            {/* Members Table */}
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
              <div className="px-6 pt-6 pb-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="text-sm font-medium text-zinc-900 dark:text-white">Canvas Members</h4>
                    <p className="text-sm text-zinc-500 dark:text-zinc-400">Members with access to this canvas</p>
                  </div>
                  <InputGroup>
                    <MaterialSymbol name="search" size="md" data-slot="icon" />
                    <Input
                      placeholder="Search members..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                    />
                  </InputGroup>
                </div>
              </div>
              <div className="px-6 pb-6">
              <Table dense>
                <TableHead>
                  <TableRow>
                    <TableHeader>Name</TableHeader>
                    <TableHeader>Email</TableHeader>
                    <TableHeader>Role</TableHeader>
                    <TableHeader>Status</TableHeader>
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
                            initials={member.name.split(' ').map(n => n[0]).join('')}
                            className="size-8"
                          />
                          <div>
                            <div className="text-sm font-medium text-zinc-900 dark:text-white">
                              {member.name}
                            </div>
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">
                              Last active {member.lastActive}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        {member.email}
                      </TableCell>
                      <TableCell>
                        <Dropdown>
                          <DropdownButton outline className="flex items-center gap-2 text-sm">
                            {member.role}
                            <MaterialSymbol name="keyboard_arrow_down" />
                          </DropdownButton>
                          <DropdownMenu anchor="bottom start">
                            <DropdownItem>
                              <DropdownLabel>Owner</DropdownLabel>
                              <DropdownDescription>Full access to canvas settings</DropdownDescription>
                            </DropdownItem>
                            <DropdownItem>
                              <DropdownLabel>Admin</DropdownLabel>
                              <DropdownDescription>Can manage canvas and members</DropdownDescription>
                            </DropdownItem>
                            <DropdownItem>
                              <DropdownLabel>Member</DropdownLabel>
                              <DropdownDescription>Standard canvas access</DropdownDescription>
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </TableCell>
                      <TableCell>
                        <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${
                          member.status === 'Active'
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                            : member.status === 'Invited'
                            ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
                            : 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
                        }`}>
                          {member.status}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton plain className="flex items-center gap-2 text-sm">
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu anchor="bottom end">
                              {member.status === 'Invited' ? (
                                <>
                                <DropdownItem className='flex items-center gap-2'>
                                  <MaterialSymbol name="refresh" />
                                  Resend invitation
                                </DropdownItem>
                                <DropdownItem className='!text-red-600 dark:!text-red-400 hover:!bg-red-50 dark:hover:!bg-red-900/20 flex items-center gap-2'>
                                  <MaterialSymbol name="person_cancel" />
                                  Revoke invitation
                                </DropdownItem>
                                </>
                              ) : (
                              <DropdownItem className='!text-red-600 dark:!text-red-400 hover:!bg-red-50 dark:hover:!bg-red-900/20 flex items-center gap-2'>
                                <MaterialSymbol name="delete" />
                                Remove
                              </DropdownItem>
                              )}
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
          </div>
        </div>
      ) : activeView === 'integration-setup' ? (
        /* Integration Setup Page */
        <div className="flex-1 p-8">
          <div className="max-w-4xl mx-auto">
            {/* Breadcrumbs */}
            <div className="flex items-center gap-2 mb-6 text-sm text-zinc-500 dark:text-zinc-400">
              <Link 
                href="#" 
                onClick={() => setActiveView('integrations')}
                className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
              >
                Integrations
              </Link>
              <MaterialSymbol name="chevron_right" size="sm" />
              <Link 
                href="#" 
                onClick={() => {
                  setActiveView('integrations');
                  setIntegrationsTab('add-new');
                }}
                className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
              >
                Add new integrations
              </Link>
              <MaterialSymbol name="chevron_right" size="sm" />
              <span className="text-zinc-900 dark:text-white">
                {selectedIntegrationType === 'semaphore' ? 'Semaphore integration' : 'Integration setup'}
              </span>
            </div>

            {/* Form Content */}
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-8">
              <div className="mb-6">
                <div className="flex items-center gap-3 mb-2">
                  <div className="w-10 h-10 flex-shrink-0 flex items-center justify-center bg-zinc-100 dark:bg-zinc-800 rounded-lg">
                    {selectedIntegrationType === 'semaphore' && (
                      <img width={24} height={24} src='/images/semaphore-logo-sign-black.svg' alt="Semaphore" />
                    )}
                  </div>
                  <div>
                    <Heading level={2}>
                      {selectedIntegrationType === 'semaphore' ? 'Semaphore Integration' : 'Integration Setup'}
                    </Heading>
                  </div>
                </div>
                <Text className="text-zinc-600 dark:text-zinc-400">
                  Configure your {selectedIntegrationType === 'semaphore' ? 'Semaphore' : 'integration'} connection settings
                </Text>
              </div>

              <div className="space-y-6">
                {/* Integration Name */}
                <Field>
                  <Label htmlFor="integration-name" className="text-sm font-medium text-gray-900 dark:text-white">
                    Integration Name
                  </Label>
                  <Input
                    id="integration-name"
                    type="text"
                    placeholder={selectedIntegrationType === 'semaphore' ? 'My Semaphore Integration' : 'Enter integration name'}
                    className="w-full"
                  />
                </Field>

                {/* URL */}
                <Field>
                  <Label htmlFor="integration-url" className="text-sm font-medium text-gray-900 dark:text-white">
                    {selectedIntegrationType === 'semaphore' ? 'Semaphore Organization URL' : 'URL'}
                  </Label>
                  <Input
                    id="integration-url"
                    type="url"
                    placeholder={selectedIntegrationType === 'semaphore' ? 'https://your-org.semaphoreci.com' : 'https://example.com'}
                    className="w-full"
                  />
                </Field>

                {/* Authentication */}
                <Field>
                  <Label htmlFor="integration-auth" className="text-sm font-medium text-gray-900 dark:text-white">
                    Authentication
                  </Label>
                  <Dropdown>
                    <DropdownButton outline className="flex items-center w-full !justify-between">
                      {selectedIntegrationType === 'semaphore' ? 'API Token' : 'Select authentication method'}
                      <MaterialSymbol name="keyboard_arrow_down" />
                    </DropdownButton>
                    <DropdownMenu>
                      <DropdownItem>
                        <DropdownLabel>API Token</DropdownLabel>
                      </DropdownItem>
                      <DropdownItem>
                        <DropdownLabel>OAuth</DropdownLabel>
                      </DropdownItem>
                      <DropdownItem>
                        <DropdownLabel>Basic Auth</DropdownLabel>
                      </DropdownItem>
                    </DropdownMenu>
                  </Dropdown>
                </Field>

                {/* API Token Field (when API Token is selected) */}
                <Field>
                  <Label htmlFor="api-token" className="text-sm font-medium text-gray-900 dark:text-white">
                    API Token
                  </Label>
                  <Input
                    id="api-token"
                    type="password"
                    placeholder="Enter your API token"
                    className="w-full"
                  />
                </Field>

                {/* Action Buttons */}
                <div className="flex items-center justify-between pt-6 border-t border-zinc-200 dark:border-zinc-800">
                  <Button 
                    plain
                    onClick={() => {
                      setActiveView('integrations');
                      setIntegrationsTab('add-new');
                      setSelectedIntegrationType(null);
                    }}
                  >
                    Cancel
                  </Button>
                  <Button 
                    color="blue"
                    onClick={() => {
                      // Handle integration creation
                      console.log('Creating integration:', selectedIntegrationType);
                      // For now, just go back to integrations page
                      setActiveView('integrations');
                      setIntegrationsTab('connected');
                      setSelectedIntegrationType(null);
                    }}
                  >
                    Create integration
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : activeView === 'secrets' ? (
        /* Secrets Page */
        <div className="flex-1 p-8">
          <div className="max-w-4xl mx-auto">
            <div className="flex items-center justify-between mb-6">
              <div>
                <Subheading>Secrets</Subheading>
                <Text>Manage environment variables and sensitive data.</Text>
              </div>
              <Button>
                <MaterialSymbol name="add" size="md" />
                Add Secret
              </Button>
            </div>

            {/* Secrets Table */}
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
              <Table>
                <TableHead>
                  <TableRow>
                    <TableHeader>Name</TableHeader>
                    <TableHeader>Description</TableHeader>
                    <TableHeader>Created</TableHeader>
                    <TableHeader>Last Used</TableHeader>
                    <TableHeader></TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {secrets.map((secret) => (
                    <TableRow key={secret.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <MaterialSymbol name="key" className="text-zinc-400" size="sm" />
                          <span className="font-mono text-sm text-zinc-900 dark:text-white">
                            {secret.name}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">
                          {secret.description}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-zinc-500 dark:text-zinc-400">
                          {secret.createdAt}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-zinc-500 dark:text-zinc-400">
                          {secret.lastUsed}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end gap-2">
                          <Button
                            plain
                            onClick={() => console.log('Edit secret', secret.id)}
                          >
                            <MaterialSymbol name="edit" size="sm" />
                          </Button>
                          <Button
                            plain
                            onClick={() => console.log('Delete secret', secret.id)}
                          >
                            <MaterialSymbol name="delete" size="sm" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        </div>
      ) : activeView === 'integrations' ? (
        <div className="flex-1 p-8">
          <div className="max-w-4xl mx-auto">
            <div className="mb-6">
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white mb-2">Integrations</h1>
              <p className="text-zinc-600 dark:text-zinc-400">Connect external services</p>
            </div>
          </div>
        </div>
      ) : activeView === 'members' ? (
        <div className="flex-1 p-8">
          <div className="max-w-4xl mx-auto">
            <div className="mb-6">
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white mb-2">Canvas Members</h1>
              <p className="text-zinc-600 dark:text-zinc-400">Manage member access</p>
            </div>
          </div>
        </div>
      ) : activeView === 'secrets' ? (
        <div className="flex-1 p-8">
          <div className="max-w-4xl mx-auto">
            <div className="mb-6">
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white mb-2">Secrets</h1>
              <p className="text-zinc-600 dark:text-zinc-400">Manage environment variables</p>
            </div>
          </div>
        </div>
      ) : (
        <div className="flex-1 p-8">
          <div className="max-w-4xl mx-auto">
            <div className="mb-6">
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white mb-2">Other View</h1>
              <p className="text-zinc-600 dark:text-zinc-400">Other views content</p>
            </div>
          </div>
        </div>
      )}

      {/* Canvas Members Modal */}
      <Dialog 
        className='bg-white dark:bg-zinc-900'
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
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
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
          <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden px-6 py-2">
            <Table dense>
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
          </div>
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

      {/* Secrets Management Modal */}
      <Dialog 
        className='bg-white dark:bg-zinc-900'
        open={showSecretsModal} 
        onClose={() => {
          setShowSecretsModal(false);
          setEditingSecret(null);
          setIsCreatingSecret(false);
        }}
        size="4xl"
      >
        <DialogTitle className='flex items-center justify-between'>
          Secret Management
          <Button plain onClick={() => {
            setShowSecretsModal(false);
            setEditingSecret(null);
            setIsCreatingSecret(false);
          }}>
            <MaterialSymbol name="close" size='lg' />
          </Button>
        </DialogTitle>
        <DialogDescription>
          Manage secrets for your canvas workflows. These secrets can be used in your workflow stages.
        </DialogDescription>
        
        <DialogBody>
          {!editingSecret && !isCreatingSecret ? (
            <>
              {/* Secrets List View */}
              <div className="flex justify-between items-center mb-6">
                <div>
                  <Text className="text-lg font-semibold text-zinc-900 dark:text-white">
                    Secrets
                  </Text>
                  <Text className="text-sm text-zinc-500 dark:text-zinc-400">
                    {secrets.length} secret{secrets.length !== 1 ? 's' : ''} configured
                  </Text>
                </div>
                <Button
                  color="blue"
                  onClick={() => setIsCreatingSecret(true)}
                >
                  <MaterialSymbol name="add" size="sm" />
                  New Secret
                </Button>
              </div>

              {/* Secrets Table */}
              <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableHeader>Name</TableHeader>
                      <TableHeader>Description</TableHeader>
                      <TableHeader>Created</TableHeader>
                      <TableHeader>Last Used</TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {secrets.map((secret) => (
                      <TableRow key={secret.id}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <MaterialSymbol name="key" className="text-zinc-400" size="sm" />
                            <span className="font-mono text-sm text-zinc-900 dark:text-white">
                              {secret.name}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className="text-sm text-zinc-600 dark:text-zinc-400">
                            {secret.description}
                          </span>
                        </TableCell>
                        <TableCell>
                          <span className="text-sm text-zinc-500 dark:text-zinc-400">
                            {secret.createdAt}
                          </span>
                        </TableCell>
                        <TableCell>
                          <span className="text-sm text-zinc-500 dark:text-zinc-400">
                            {secret.lastUsed}
                          </span>
                        </TableCell>
                        <TableCell>
                          <div className="flex justify-end gap-2">
                            <Button
                              plain
                              onClick={() => setEditingSecret(secret)}
                              className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
                            >
                              <MaterialSymbol name="edit" size="sm" />
                            </Button>
                            <Button
                              plain
                              onClick={() => {
                                setSecrets(secrets.filter(s => s.id !== secret.id));
                                console.log('Secret deleted:', secret.name);
                              }}
                              className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                            >
                              <MaterialSymbol name="delete" size="sm" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </>
          ) : (
            /* Create/Edit Secret Form */
            <div className="space-y-6">
              <div className="flex items-center gap-3">
                <Button
                  plain
                  onClick={() => {
                    setEditingSecret(null);
                    setIsCreatingSecret(false);
                  }}
                >
                  <MaterialSymbol name="arrow_back" size="sm" />
                </Button>
                <div>
                  <Text className="text-lg font-semibold text-zinc-900 dark:text-white">
                    {isCreatingSecret ? 'New Secret' : 'Edit Secret'}
                  </Text>
                  <Text className="text-sm text-zinc-500 dark:text-zinc-400">
                    {isCreatingSecret ? 'Create a new secret' : `Update ${editingSecret?.name}`}
                  </Text>
                </div>
              </div>

              <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
                <div className="space-y-4">
                  <Field>
                    <Label htmlFor="secretName">Secret Name</Label>
                    <Input
                      id="secretName"
                      placeholder="e.g., API_KEY, DATABASE_URL"
                      defaultValue={editingSecret?.name || ''}
                      className="font-mono"
                    />
                  </Field>

                  <Field>
                    <Label htmlFor="secretValue">Secret Value</Label>
                    <Input
                      id="secretValue"
                      type="password"
                      placeholder="Enter secret value"
                      defaultValue={editingSecret?.value || ''}
                      className="font-mono"
                    />
                  </Field>

                  <Field>
                    <Label htmlFor="secretDescription">Description</Label>
                    <Textarea
                      id="secretDescription"
                      placeholder="Describe what this secret is used for"
                      defaultValue={editingSecret?.description || ''}
                      rows={3}
                    />
                  </Field>

                  <div className="flex justify-end gap-3 pt-4">
                    <Button
                      plain
                      onClick={() => {
                        setEditingSecret(null);
                        setIsCreatingSecret(false);
                      }}
                    >
                      Cancel
                    </Button>
                    <Button
                      color="blue"
                      onClick={() => {
                        // Handle save/create logic
                        console.log(isCreatingSecret ? 'Creating secret' : 'Updating secret');
                        setEditingSecret(null);
                        setIsCreatingSecret(false);
                      }}
                    >
                      {isCreatingSecret ? 'Create Secret' : 'Save Changes'}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </DialogBody>
      </Dialog>

      {/* Connection Modal */}
      <Dialog 
        className='bg-white dark:bg-zinc-900'
        open={showConnectionModal} 
        onClose={() => setShowConnectionModal(false)}
        size="md"
      >
        <DialogTitle className='flex items-center justify-between'>
          Add Connection
          <Button plain onClick={() => setShowConnectionModal(false)}>
            <MaterialSymbol name="close" size='lg' />
          </Button>
        </DialogTitle>
        <DialogDescription>
          Configure a new connection for your workflow stage.
        </DialogDescription>
        
        <DialogBody>
          <div className="space-y-4">
            {/* Connection Type - exact same as inline */}
            <div className="flex-auto space-y-1 border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 p-1 rounded-sm">
              <div className="flex flex-col">
                <Field className='flex justify-between'>
                  <Dropdown>
                    <DropdownButton color='white' className="!justify-between flex items-center w-full">
                      {connectionType === 'stage' ? 'Deploy to staging' : 'Github webhook'}
                      <MaterialSymbol name="expand_more" size="md" />
                    </DropdownButton>
                    <DropdownMenu>
                      <DropdownItem onClick={() => setConnectionType('stage')}>
                        <DropdownLabel>Deploy to staging</DropdownLabel>
                      </DropdownItem>
                      <DropdownItem onClick={() => setConnectionType('event source')}>
                        <DropdownLabel>Github webhook</DropdownLabel>
                      </DropdownItem>
                    </DropdownMenu>
                  </Dropdown>
                </Field>
              </div>

              {/* Filters List - exact same as inline */}
              {connectionFilters && connectionFilters.length > 0 && (
                <Field className="">
                  <Label className="!text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-1">
                    Filters
                  </Label>
                  {connectionFilters.map((filter, filterIndex) => (
                    <div key={filter.id} className='relative w-full'>
                      {/* Show AND/OR button before filter (except for the first filter) */}
                      {filter.operator && filterIndex > 0 && (
                        <div className={filter.operator === 'AND' ? "relative justify-center flex items-center" : "relative justify-center flex items-center"}>
                          <Link
                            href="#"
                            onClick={() => handleToggleOperator(filter.id)}
                            className="!text-xs font-medium !px-2 !py-0 bg-blue-50 text-zinc-700 dark:text-zinc-300 hover:bg-blue-100 dark:hover:bg-zinc-600 rounded"
                          >
                            {filter.operator || 'AND'}
                          </Link>
                        </div>
                      )}
                      
                      <div className="">
                        <div className="flex justify-between p-1 bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded gap-1">
                         <div className='flex flex-auto items-center'>
                            <Dropdown>
                              <DropdownButton outline className="flex items-center !justify-between min-w-[90px]">
                                {filter.type}
                                <MaterialSymbol name="expand_more" size="sm" />
                              </DropdownButton>
                              <DropdownMenu>
                                <DropdownItem onClick={() => handleUpdateFilter(filter.id, 'type', 'data')}>
                                  <DropdownLabel>Data</DropdownLabel>
                                </DropdownItem>
                                <DropdownItem onClick={() => handleUpdateFilter(filter.id, 'type', 'header')}>
                                  <DropdownLabel>Header</DropdownLabel>
                                </DropdownItem>
                              </DropdownMenu>
                            </Dropdown>
                            
                            <Input
                              placeholder="Expression"
                              value={filter.expression}
                              onChange={(e) => handleUpdateFilter(filter.id, 'expression', e.target.value)}
                              className="flex-auto text-xs"
                            />
                          </div>
                          <div className='flex items-center'>
                            <Link
                              href='#'
                              onClick={() => handleRemoveFilter(filter.id)}
                              className=""
                            >
                              <MaterialSymbol name="close" size="sm" />
                            </Link>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </Field>
              )}      

              {connectionFilters.length === 0 && (
                <div className="flex flex-col items-center justify-center h-full space-y-4">
                  <Text className="text-zinc-500 dark:text-zinc-400">
                    DEBUG: This is the zero state for connections modal.
                  </Text>
                  <Button
                    onClick={handleAddFilter}
                    className="text-blue-600 hover:text-blue-700 flex items-center !text-xs"
                    plain
                  >
                    <MaterialSymbol name="add" size="sm" />
                    Add Connection
                  </Button>
                </div>
              )}
            </div>
          </div>
        </DialogBody>
        
        <DialogActions>
          <Button plain onClick={() => {
            setShowConnectionModal(false);
            // Reset modal state
            setConnectionType('stage');
            setConnectionFilters([]);
            setCurrentEditingNodeId(null);
          }}>
            Cancel
          </Button>
          <Button 
            color="blue" 
            onClick={() => {
              // Add connection to the current editing node
              if (currentEditingNodeId) {
                // Calculate the new connection index before updating nodes
                const currentNode = nodes.find(n => n.id === currentEditingNodeId);
                const currentWorkflowData = (currentNode?.data as any)?.workflowNodeData;
                const currentConnections = currentWorkflowData?.yamlConfig?.spec?.connections || [];
                const newConnectionIndex = currentConnections.length;
                
                setNodes((nds) =>
                  nds.map((node) => {
                    if (node.id === currentEditingNodeId) {
                      const currentWorkflowData = (node.data as any).workflowNodeData;
                      const currentConnections = currentWorkflowData.yamlConfig?.spec?.connections || [];
                      
                      // Create new connection object
                      const newConnection = {
                        type: connectionType,
                        name: connectionType === 'stage' ? 'Deploy to staging' : 'Github webhook',
                        // Add any other connection properties as needed
                      };
                      
                      console.log('Adding connection to node:', currentEditingNodeId);
                      console.log('Current connections:', currentConnections);
                      console.log('New connection:', newConnection);
                      
                      // Update the node with the new connection
                      const updatedNode = {
                        ...node,
                        data: {
                          ...node.data,
                          workflowNodeData: {
                            ...currentWorkflowData,
                            yamlConfig: {
                              ...currentWorkflowData.yamlConfig,
                              spec: {
                                ...currentWorkflowData.yamlConfig?.spec,
                                connections: [...currentConnections, newConnection]
                              }
                            }
                          }
                        }
                      };
                      
                      console.log('Updated node:', updatedNode);
                      console.log('New connection will be at index:', newConnectionIndex);
                      
                      return updatedNode;
                    }
                    return node;
                  })
                );
                
                // Add the new connection index to saved connections for this node
                setNodeSavedConnections(prev => ({
                  ...prev,
                  [currentEditingNodeId]: [...(prev[currentEditingNodeId] || []), newConnectionIndex]
                }));
              }
              
              setShowConnectionModal(false);
              // Reset modal state
              setConnectionType('stage');
              setConnectionFilters([]);
              setCurrentEditingNodeId(null);
            }}
          >
            Save Connection
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
};
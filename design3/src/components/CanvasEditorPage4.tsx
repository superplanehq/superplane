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
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { NavigationOrg, type User, type Organization } from './lib/Navigation/navigation-org';
import { WorkflowNode, WorkflowEdge } from '../types';
import { DeploymentCardStage } from './DeploymentCardStage';
import { ComponentSidebar } from './ComponentSidebar';
import { WorkflowNodeReactFlow, type WorkflowNodeReactFlowData } from './WorkflowNodeReactFlow';
import { WorkflowNodeAccordionReactFlow, type WorkflowNodeAccordionReactFlowData } from './WorkflowNodeAccordionReactFlow';
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
  DropdownDescription
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
import { ControlledTabs } from './lib/Tabs/tabs';
import { Divider } from './lib/Divider/divider';


// Node types for React Flow
const nodeTypes = {
  deploymentCard: DeploymentCardStage as any,
  workflowNode: WorkflowNodeReactFlow as any,
  workflowNodeAccordion: WorkflowNodeAccordionReactFlow as any,
};

interface CanvasEditorPage4Props {
  canvasId: string
  onBack?: () => void
}

// Initial workflow data
const initialNodes: WorkflowNode[] = [
 
];

const initialEdges: WorkflowEdge[] = [

];

/**
 * CanvasEditorPage4 component following SaaS guidelines
 * - Uses TypeScript with proper interfaces
 * - Implements React Flow for diagramming
 * - Follows responsive design principles
 * - Includes proper accessibility features
 * - Handles loading and error states
 */
export function CanvasEditorPage4({ 
  canvasId, 
  onBack
}: CanvasEditorPage4Props) {
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
  const [rightSidebarOpen, setRightSidebarOpen] = useState(false);
  const [editingNodeId, setEditingNodeId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState('connections');
  const [isCanvasStarred, setIsCanvasStarred] = useState(false);
  const [showMembersModal, setShowMembersModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [navigationParam, setNavigationParam] = useState<string>('dropdown'); // default to center
  const [nodeEditorMode, setNodeEditorMode] = useState<'modal' | 'sidebar'>('sidebar'); // default to sidebar
  const [showNodeEditorModal, setShowNodeEditorModal] = useState(false);
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  
  // Node editing state - this would normally come from the selected node
  const [editingNodeData, setEditingNodeData] = useState<WorkflowNodeData | null>(null);
  const [yamlConfig, setYamlConfig] = useState<any>(null);
  const [connectionFilters, setConnectionFilters] = useState<Record<number, Array<{id: string, type: string, expression: string, operator?: string}>>>({}); 
  const [inputMappings, setInputMappings] = useState<Record<string, Array<{id: string, connection: string, value: string}>>>({}); 
  const [showGitHubModal, setShowGitHubModal] = useState(false);
  const [isGitHubConnected, setIsGitHubConnected] = useState(false);
  const [githubProjects, setGithubProjects] = useState<Array<{id: string, name: string, url: string}>>([]);
  const [selectedGitHubProject, setSelectedGitHubProject] = useState<string>('');

  // Get navigation and nodeEditor parameters from URL
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const navParam = urlParams.get('nav') || 'left';
    const nodeEditorParam = urlParams.get('nodeEditor') as 'modal' | 'sidebar' || 'sidebar';
    
    setNavigationParam(navParam);
    setNodeEditorMode(nodeEditorParam);
  }, []);
  
  // Load node data when editing a node
  useEffect(() => {
    if (editingNodeId) {
      const node = nodes.find(n => n.id === editingNodeId);
      if (node && node.data) {
        const nodeData = (node.data as any).workflowNodeData as WorkflowNodeData;
        setEditingNodeData(nodeData);
        setYamlConfig(nodeData.yamlConfig || {
          apiVersion: 'v1',
          kind: 'Stage',
          metadata: {
            name: nodeData.title.toLowerCase().replace(/\s+/g, '-'),
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
        });
      }
    } else {
      setEditingNodeData(null);
      setYamlConfig(null);
    }
  }, [editingNodeId, nodes, canvasId]);

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
      
      // Open editor based on mode
      setEditingNodeId(node.id);
      if (nodeEditorMode === 'modal') {
        setShowNodeEditorModal(true);
      } else {
        setRightSidebarOpen(true);
      }
    },
    [selectedNode, setNodes, nodeEditorMode]
  );



  

  /**
   * Add new node to workflow
   */
  const addNode = useCallback(
    (nodeType: string) => {
      const nodeId = `node-${Date.now()}`;
      console.log('Adding node, current nodes count:', nodes.length);
      // Use accordion version by default, alternate with tab version
      const useAccordion = true; // Always use accordion for testing
      
      const newNode: WorkflowNode = {
        id: nodeId,
        type: useAccordion ? 'workflowNodeAccordion' : 'workflowNode',
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
          variant: 'read',
          ...(useAccordion ? {
            // Accordion-specific props
            multiple: true,
            className: 'max-w-xs',
            partialSave: true  // Enable individual section save buttons for Canvas 4
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
        console.log('Adding new node, updated nodes count:', updatedNodes.length);
        // Update all nodes (including the new one) with the complete nodes array and count
        const finalNodes = updatedNodes.map((node) => {
          console.log('Updating node data for node:', node.id);
          const updatedNodeData = {
            ...node,
            data: {
              ...node.data,
              nodes: updatedNodes,
              totalNodesCount: updatedNodes.length
            }
          };
          console.log('Updated node data:', updatedNodeData.data);
          return updatedNodeData;
        });
        console.log('Final nodes with updated data:', finalNodes);
        console.log('Sample node data:', finalNodes[0]?.data);
        return finalNodes;
      });
      setSidebarOpen(false);
      
      // Open editor based on mode
      setEditingNodeId(nodeId);
      if (nodeEditorMode === 'modal') {
        // Show the node for a millisecond before opening the modal
        setTimeout(() => {
          setShowNodeEditorModal(true);
        }, 150);
      } else {
        setRightSidebarOpen(true);
      }
    },
    [setNodes, setEdges, canvasId, nodeEditorMode]
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
  
  // Tab content handlers (extracted from accordion)
  const handleAddConnection = () => {
    if (!yamlConfig) return;
    setYamlConfig(prev => ({
      ...prev,
      spec: {
        ...prev.spec,
        connections: [...(prev.spec.connections || []), { 
          name: '', 
          type: 'stage', 
          config: {} 
        }]
      }
    }));
  };

  const handleAddFilter = (connectionIndex: number) => {
    const existingFilters = connectionFilters[connectionIndex] || [];
    let currentOperator = 'AND';
    if (existingFilters.length > 0) {
      const filterWithOperator = existingFilters.find(filter => filter.operator);
      currentOperator = filterWithOperator?.operator || 'AND';
    }
    
    const newFilter = {
      id: `filter_${Date.now()}`,
      type: 'Data',
      expression: '',
      operator: existingFilters.length > 0 ? currentOperator : undefined
    };
    
    setConnectionFilters(prev => ({
      ...prev,
      [connectionIndex]: [...(prev[connectionIndex] || []), newFilter]
    }));
  };

  const handleRemoveFilter = (connectionIndex: number, filterId: string) => {
    setConnectionFilters(prev => ({
      ...prev,
      [connectionIndex]: (prev[connectionIndex] || []).filter(filter => filter.id !== filterId)
    }));
  };

  const handleUpdateFilter = (connectionIndex: number, filterId: string, field: 'type' | 'expression', value: string) => {
    setConnectionFilters(prev => ({
      ...prev,
      [connectionIndex]: (prev[connectionIndex] || []).map(filter => 
        filter.id === filterId ? { ...filter, [field]: value } : filter
      )
    }));
  };

  const handleToggleOperator = (connectionIndex: number, filterId: string) => {
    setConnectionFilters(prev => {
      const currentFilters = prev[connectionIndex] || [];
      const clickedFilter = currentFilters.find(filter => filter.id === filterId);
      const newOperator = clickedFilter?.operator === 'OR' ? 'AND' : 'OR';
      
      return {
        ...prev,
        [connectionIndex]: currentFilters.map(filter => 
          filter.operator 
            ? { ...filter, operator: newOperator }
            : filter
        )
      };
    });
  };

  const handleAddInputMapping = (inputId: string) => {
    const newMapping = {
      id: `mapping_${Date.now()}`,
      connection: '',
      value: ''
    };
    
    setInputMappings(prev => ({
      ...prev,
      [inputId]: [...(prev[inputId] || []), newMapping]
    }));
  };

  const handleRemoveInputMapping = (inputId: string, mappingId: string) => {
    setInputMappings(prev => ({
      ...prev,
      [inputId]: (prev[inputId] || []).filter(mapping => mapping.id !== mappingId)
    }));
  };

  const handleUpdateInputMapping = (inputId: string, mappingId: string, field: 'connection' | 'value', value: string) => {
    setInputMappings(prev => ({
      ...prev,
      [inputId]: (prev[inputId] || []).map(mapping => 
        mapping.id === mappingId ? { ...mapping, [field]: value } : mapping
      )
    }));
  };

  const handleExecutorTypeChange = (type: string) => {
    if (!yamlConfig) return;
    setYamlConfig(prev => ({
      ...prev,
      spec: {
        ...prev.spec,
        executor: {
          type,
          config: {}
        }
      }
    }));
    
    if (type !== 'github') {
      setIsGitHubConnected(false);
      setGithubProjects([]);
      setSelectedGitHubProject('');
    }
  };

  const handleConnectGitHub = () => {
    setShowGitHubModal(true);
  };

  const handleGitHubLogin = () => {
    setTimeout(() => {
      setIsGitHubConnected(true);
      setGithubProjects([
        { id: '1', name: 'my-awesome-project', url: 'https://github.com/user/my-awesome-project' },
        { id: '2', name: 'react-components', url: 'https://github.com/user/react-components' },
        { id: '3', name: 'api-service', url: 'https://github.com/user/api-service' },
        { id: '4', name: 'frontend-app', url: 'https://github.com/user/frontend-app' },
        { id: '5', name: 'backend-service', url: 'https://github.com/user/backend-service' }
      ]);
      setShowGitHubModal(false);
    }, 1000);
  };

  const handleGitHubProjectSelect = (projectId: string) => {
    setSelectedGitHubProject(projectId);
    const selectedProject = githubProjects.find(p => p.id === projectId);
    if (selectedProject && yamlConfig) {
      setYamlConfig(prev => ({
        ...prev,
        spec: {
          ...prev.spec,
          executor: {
            type: 'github',
            config: {
              project: selectedProject.name,
              url: selectedProject.url
            }
          }
        }
      }));
    }
  };

  // Tab definitions
  const tabs = [
    { id: 'connections', label: 'Connections' },
    { id: 'inputs', label: 'Inputs' },
    { id: 'executor', label: 'Executor' },
    { id: 'outputs', label: 'Outputs' }  ];

  // Render tab content
  const renderTabContent = () => {
    if (!yamlConfig) return null;

    switch (activeTab) {
      case 'connections':
        return (
          <div className="space-y-4">
            <div className="flex justify-end items-center mb-3">
              <Button
                onClick={handleAddConnection}
                className="text-blue-600 hover:text-blue-700 flex items-center !text-xs"
                plain
              >
                <MaterialSymbol name="add" size="sm" />
                Add Connection
              </Button>
            </div>

            {/* Connections List */}
            <div className="space-y-3">
              {yamlConfig.spec.connections?.map((connection, index) => (
                <div key={index} className="border p-3 border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 rounded-lg">
                  <div className="space-y-3">
                    <div className="flex flex-col">
                      <Field className='flex justify-between'>
                        <Dropdown>
                          <DropdownButton outline className="justify-between">
                            {connection.type === 'stage' ? 'Stage' : connection.type === 'event source' ? 'Event Source' : connection.type || 'Select Type'}
                            <MaterialSymbol name="expand_more" size="sm" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem onClick={() => {
                              const newConnections = [...(yamlConfig.spec.connections || [])]
                              newConnections[index] = { ...connection, type: 'stage' }
                              setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                            }}>
                              <DropdownLabel>Stage</DropdownLabel>
                            </DropdownItem>
                            <DropdownItem onClick={() => {
                              const newConnections = [...(yamlConfig.spec.connections || [])]
                              newConnections[index] = { ...connection, type: 'event source' }
                              setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                            }}>
                              <DropdownLabel>Event Source</DropdownLabel>
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                        <Button
                          plain
                          onClick={() => {
                            const newConnections = yamlConfig.spec.connections?.filter((_, i) => i !== index) || []
                            setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                          }}
                          className="text-red-600 hover:text-red-700 ml-3"
                        >
                          <MaterialSymbol name="delete" size="sm" />
                        </Button>
                      </Field>
                      <Field className="flex-1 mt-2">
                        <Input
                          placeholder="Connection name"
                          value={connection.name}
                          onChange={(e) => {
                            const newConnections = [...(yamlConfig.spec.connections || [])]
                            newConnections[index] = { ...connection, name: e.target.value }
                            setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                          }}
                          className="w-full"
                        />
                      </Field>
                    </div>
                    {/* Filters List */}
                    {connectionFilters[index] && connectionFilters[index].length > 0 && (
                      <Field className="">
                        <Label className="text-xs font-medium text-zinc-600 dark:text-zinc-400">
                          Filters
                        </Label>
                        {connectionFilters[index].map((filter, filterIndex) => (
                          <div key={filter.id} className='relative'>
                            {filter.operator && filterIndex > 0 && (
                              <div className={filter.operator === 'AND' ? "absolute top-0 transform -translate-y-1/2 left-0 right-0 text-center" : "relative justify-center flex items-center"}>
                                <Button
                                  onClick={() => handleToggleOperator(index, filter.id)}
                                  className="!text-xs !px-2 !py-0 bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-300 hover:bg-zinc-200 dark:hover:bg-zinc-600 rounded"
                                  plain
                                >
                                  {filter.operator || 'AND'}
                                </Button>
                              </div>
                            )}
                            
                            <div className="flex justify-between items-center">
                              <div className="flex items-center p-2 bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded gap-1">
                                <Dropdown>
                                  <DropdownButton outline className="flex items-center min-w-[80px]">
                                    {filter.type}
                                    <MaterialSymbol name="expand_more" size="sm" />
                                  </DropdownButton>
                                  <DropdownMenu>
                                    <DropdownItem onClick={() => handleUpdateFilter(index, filter.id, 'type', 'data')}>
                                      <DropdownLabel>Data</DropdownLabel>
                                    </DropdownItem>
                                    <DropdownItem onClick={() => handleUpdateFilter(index, filter.id, 'type', 'header')}>
                                      <DropdownLabel>Header</DropdownLabel>
                                    </DropdownItem>
                                  </DropdownMenu>
                                </Dropdown>
                                
                                <Input
                                  placeholder="Expression"
                                  value={filter.expression}
                                  onChange={(e) => handleUpdateFilter(index, filter.id, 'expression', e.target.value)}
                                  className="flex-1 text-xs"
                                />
                              </div>
                              <Button
                                plain
                                onClick={() => handleRemoveFilter(index, filter.id)}
                                className="!p-1"
                              >
                                <MaterialSymbol name="close" size="sm" />
                              </Button>
                            </div>
                          </div>
                        ))}
                      </Field>
                    )}      
                    {/* Define Filters Button */}
                    <div className="mt-3 flex justify-end">
                      <Button
                        onClick={() => handleAddFilter(index)}
                        className="flex items-center !text-xs"
                        plain
                      >
                        <MaterialSymbol name="add" size="sm" />
                        Add Filter
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
              {(!yamlConfig.spec.connections || yamlConfig.spec.connections?.length === 0) && (
                <div className="flex justify-center items-center h-full">
                  <Text className="text-zinc-500 dark:text-zinc-400">
                    No connections added
                  </Text>
                </div>
              )}
            </div>
          </div>
        );

      case 'inputs':
        return (
          <div className="space-y-6">
            <div>
              <div className="flex justify-between items-center mb-3">
                <Field>
                  <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                    Inputs
                  </Label>
                </Field>
                <Button
                  onClick={() => setYamlConfig(prev => ({
                    ...prev,
                    spec: {
                      ...prev.spec,
                      inputs: [...(prev.spec.inputs || []), { name: '', type: 'string', required: false }]
                    }
                  }))}
                  className="text-blue-600 hover:text-blue-700"
                  plain
                >
                  <MaterialSymbol name="add" size="sm" />
                  Add Input
                </Button>
              </div>
              <div className="space-y-4">
                {yamlConfig.spec.inputs?.map((input, index) => {
                  const inputId = `input_${index}`
                  return (
                    <div key={index} className="p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-700">
                      <div className="flex justify-between items-start mb-3">
                        <div className="flex-1 space-y-3">
                          <Field>
                            <div className='flex items-center justify-between'>
                              <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                                New input
                              </Label>
                              <Button
                                plain
                                onClick={() => {
                                  const newInputs = yamlConfig.spec.inputs?.filter((_, i) => i !== index) || []
                                  setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                                  setInputMappings(prev => {
                                    const newMappings = { ...prev }
                                    delete newMappings[inputId]
                                    return newMappings
                                  })
                                }}
                                className="text-red-600 hover:text-red-700 ml-3"
                              >
                                <MaterialSymbol name="delete" size="sm" />
                              </Button>
                            </div>
                            <Input
                              placeholder="Name"
                              value={input.name}
                              onChange={(e) => {
                                const newInputs = [...(yamlConfig.spec.inputs || [])]
                                newInputs[index] = { ...input, name: e.target.value }
                                setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                              }}
                              className="w-full"
                            />
                          </Field>

                          <Field>
                            <Textarea
                              placeholder="Description"
                              value={input.description || ''}
                              onChange={(e) => {
                                const newInputs = [...(yamlConfig.spec.inputs || [])]
                                newInputs[index] = { ...input, description: e.target.value }
                                setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                              }}
                              rows={2}
                              className="w-full"
                            />
                          </Field>

                          <Field>
                            {inputMappings[inputId] && inputMappings[inputId].length > 0 && (
                              <div className="space-y-2 mb-3">
                                {inputMappings[inputId].map((mapping) => (
                                  <div key={mapping.id} className="flex items-center gap-2">
                                    <Dropdown>
                                      <DropdownButton outline className="min-w-[100px] flex items-center justify-between text-xs">
                                        {mapping.connection || 'Connection'}
                                        <MaterialSymbol name="expand_more" size="sm" />
                                      </DropdownButton>
                                      <DropdownMenu>
                                        {yamlConfig.spec.connections?.map((connection, connIndex) => (
                                          <DropdownItem 
                                            key={connIndex}
                                            onClick={() => handleUpdateInputMapping(inputId, mapping.id, 'connection', connection.name)}
                                          >
                                            <DropdownLabel>{connection.name || `Connection ${connIndex + 1}`}</DropdownLabel>
                                          </DropdownItem>
                                        ))}
                                      </DropdownMenu>
                                    </Dropdown>
                                    
                                    <Input
                                      placeholder="Value"
                                      value={mapping.value}
                                      onChange={(e) => handleUpdateInputMapping(inputId, mapping.id, 'value', e.target.value)}
                                      className="flex-1 text-xs"
                                    />
                                    
                                    <Button
                                      plain
                                      onClick={() => handleRemoveInputMapping(inputId, mapping.id)}
                                      className="text-red-600 hover:text-red-700"
                                    >
                                      <MaterialSymbol name="close" size="sm" />
                                    </Button>
                                  </div>
                                ))}
                              </div>
                            )}

                            <Button
                              onClick={() => handleAddInputMapping(inputId)}
                              className="text-blue-600 hover:text-blue-700 text-xs"
                              plain
                            >
                              <MaterialSymbol name="add" size="sm" />
                              Add Mapping
                            </Button>
                          </Field>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        );

      case 'executor':
        return (
          <div className="space-y-4">
            <Field>
              <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                Executor Type
              </Label>
              <Dropdown>
                <DropdownButton outline className="w-full flex items-center justify-between">
                  <span>{yamlConfig.spec.executor?.type || 'Select executor type'}</span>
                  <MaterialSymbol name="expand_more" size="sm" />
                </DropdownButton>
                <DropdownMenu>
                  <DropdownItem onClick={() => handleExecutorTypeChange('semaphore')}>
                    <DropdownLabel>Semaphore</DropdownLabel>
                  </DropdownItem>
                  <DropdownItem onClick={() => handleExecutorTypeChange('github')}>
                    <DropdownLabel>GitHub</DropdownLabel>
                  </DropdownItem>
                </DropdownMenu>
              </Dropdown>
            </Field>
            
            {yamlConfig.spec.executor?.type === 'github' && (
              <Field>
                <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  GitHub Integration
                </Label>
                {!isGitHubConnected ? (
                  <div className="space-y-2">
                    <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                      Connect with GitHub to proceed
                    </Text>
                    <Button
                      onClick={handleConnectGitHub}
                      className="w-full flex items-center justify-center gap-2"
                      color="blue"
                    >
                      <MaterialSymbol name="link" size="sm" />
                      Connect with GitHub
                    </Button>
                  </div>
                ) : (
                  <div className="space-y-2">
                    <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                      Select a GitHub project:
                    </Text>
                    <Dropdown>
                      <DropdownButton outline className="w-full flex items-center justify-between">
                        <span>
                          {selectedGitHubProject ? 
                            githubProjects.find(p => p.id === selectedGitHubProject)?.name : 
                            'Select a project'
                          }
                        </span>
                        <MaterialSymbol name="expand_more" size="sm" />
                      </DropdownButton>
                      <DropdownMenu>
                        {githubProjects.map((project) => (
                          <DropdownItem 
                            key={project.id} 
                            onClick={() => handleGitHubProjectSelect(project.id)}
                          >
                            <DropdownLabel>{project.name}</DropdownLabel>
                          </DropdownItem>
                        ))}
                      </DropdownMenu>
                    </Dropdown>
                  </div>
                )}
              </Field>
            )}
            
            {yamlConfig.spec.executor?.type === 'semaphore' && (
              <Field>
                <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  Configuration (JSON)
                </Label>
                <Textarea
                  value={JSON.stringify(yamlConfig.spec.executor?.config || {}, null, 2)}
                  onChange={(e) => {
                    try {
                      const config = JSON.parse(e.target.value)
                      setYamlConfig(prev => ({ 
                        ...prev, 
                        spec: { 
                          ...prev.spec, 
                          executor: { 
                            type: 'semaphore',
                            config 
                          }
                        }
                      }))
                    } catch (err) {
                      // Invalid JSON, don't update
                    }
                  }}
                  placeholder="{}"
                  rows={6}
                  className="w-full font-mono text-sm"
                />
              </Field>
            )}
          </div>
        );

      case 'outputs':
        return (
          <div className="space-y-4">
            <div className="flex justify-between items-center mb-3">
              <Field>
                <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  Outputs
                </Label>
              </Field>
              <Button
                onClick={() => setYamlConfig(prev => ({
                  ...prev,
                  spec: {
                    ...prev.spec,
                    outputs: [...(prev.spec.outputs || []), { name: '', type: 'string', value: '', description: '', required: false }]
                  }
                }))}
                className="text-blue-600 hover:text-blue-700"
                plain
              >
                <MaterialSymbol name="add" size="sm" />
                Add Output
              </Button>
            </div>
            <div className="space-y-3">
              {yamlConfig.spec.outputs?.map((output, index) => (
                <div key={index} className="relative p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-700">
                  <div className="absolute top-2 right-2">
                    <Button
                      plain
                      onClick={() => {
                        const newOutputs = yamlConfig.spec.outputs?.filter((_, i) => i !== index) || []
                        setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, outputs: newOutputs } }))
                      }}
                      className="text-red-600 hover:text-red-700"
                    >
                      <MaterialSymbol name="close" size="sm" />
                    </Button>
                  </div>
                  
                  <div className="space-y-3 pr-8">
                    <Field>
                      <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                        Name
                      </Label>
                      <Input
                        placeholder="Output name"
                        value={output.name}
                        onChange={(e) => {
                          const newOutputs = [...(yamlConfig.spec.outputs || [])]
                          newOutputs[index] = { ...output, name: e.target.value }
                          setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, outputs: newOutputs } }))
                        }}
                      />
                    </Field>
                    
                    <Field>
                      <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                        Description
                      </Label>
                      <Textarea
                        placeholder="Output description"
                        value={output.description || ''}
                        onChange={(e) => {
                          const newOutputs = [...(yamlConfig.spec.outputs || [])]
                          newOutputs[index] = { ...output, description: e.target.value }
                          setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, outputs: newOutputs } }))
                        }}
                        rows={2}
                      />
                    </Field>
                    
                    <Field>
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id={`output-required-${index}`}
                          checked={output.required || false}
                          onChange={(e) => {
                            const newOutputs = [...(yamlConfig.spec.outputs || [])]
                            newOutputs[index] = { ...output, required: e.target.checked }
                            setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, outputs: newOutputs } }))
                          }}
                          className="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                        />
                        <Label htmlFor={`output-required-${index}`} className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                          Is Required
                        </Label>
                      </div>
                    </Field>
                  </div>
                </div>
              ))}
            </div>
          </div>
        );

      case 'advanced':
        return (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  API Version
                </Label>
                <Input
                  type="text"
                  value={yamlConfig.apiVersion}
                  onChange={(e) => setYamlConfig(prev => ({ ...prev, apiVersion: e.target.value }))}
                  placeholder="v1"
                  className="w-full"
                />
              </Field>
              <Field>
                <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  Kind
                </Label>
                <Input
                  type="text"
                  value={yamlConfig.kind}
                  onChange={(e) => setYamlConfig(prev => ({ ...prev, kind: e.target.value }))}
                  placeholder="Stage"
                  className="w-full"
                />
              </Field>
            </div>
            <Field>
              <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                Name
              </Label>
              <Input
                type="text"
                value={yamlConfig.metadata.name}
                onChange={(e) => setYamlConfig(prev => ({ 
                  ...prev, 
                  metadata: { ...prev.metadata, name: e.target.value }
                }))}
                placeholder="deploy-to-staging"
                className="w-full"
              />
            </Field>
            <Field>
              <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                Canvas ID
              </Label>
              <Input
                type="text"
                value={yamlConfig.metadata.canvasId}
                onChange={(e) => setYamlConfig(prev => ({ 
                  ...prev, 
                  metadata: { ...prev.metadata, canvasId: e.target.value }
                }))}
                placeholder="c2181c55-64ac-41ba-8925-0eaf0357b3f6"
                className="w-full"
              />
            </Field>
          </div>
        );

      default:
        return null;
    }
  };

  // Render node editor content (shared between sidebar and modal)
  const renderNodeEditorContent = () => {
    if (!editingNodeId || !editingNodeData) {
      return (
        <div className="text-sm text-zinc-500 dark:text-zinc-500">
          No node selected
        </div>
      );
    }

    return (
      <div className="flex-1 flex flex-col">
        {/* Tabs */}
        <div className="border-b border-zinc-200 dark:border-zinc-700 px-4">
          <ControlledTabs
            tabs={tabs}
            activeTab={activeTab}
            onTabChange={setActiveTab}
            variant="underline"
          />
        </div>
        
        {/* Tab Content */}
        <div className="flex-1 p-4 overflow-y-auto">
          {renderTabContent()}
        </div>
      </div>
    );
  };

  // Function to render navigation based on URL parameter
  const renderNavigation = () => {
    switch (navigationParam) {
      case 'left':
        return (
          <NavigationOrg
            user={currentUser}
            organization={currentOrganization}
            breadcrumbsVariant='centered'
            breadcrumbs={[
              {
                label: "Canvases",
                icon: "automation",
                href: "/canvases",
                current: false
              },
              {
                label: getCanvasName(canvasId),
                current: true,
                starred: isCanvasStarred,
                dropdown: [
                  {
                    label: "Manage Members",
                    onClick: handleMembers
                  },
                  {
                    label: "Delete",
                    onClick: handleDelete
                  }
                ],
                onStarToggle: setIsStarred
              }
            ]}
            onHelpClick={handleHelpClick}
            onUserMenuAction={handleUserMenuAction}
            onOrganizationMenuAction={handleOrganizationMenuAction}
          />
        );
      case 'dropdown':
        return (
          <nav className="flex items-center bg-zinc-200 dark:bg-zinc-950 border-b border-zinc-200 dark:border-zinc-800">
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
                  <MaterialSymbol className='text-zinc-950 dark:text-white' size='lg' opticalSize={20} weight={400} name="expand_all" />
                </Headless.MenuButton>
                <DropdownMenu className="min-w-(--button-width)">
                  <DropdownItem href="/canvas4/1">Other Canvas 1</DropdownItem>
                  <DropdownItem href="/canvas4/2">Other Canvas 2</DropdownItem>
                  <DropdownItem href="/canvas4/3">Other Canvas 3</DropdownItem>
                  <DropdownItem href="/canvas4/4">Other Canvas 4</DropdownItem>
                  <DropdownItem href="/canvas4/5">Other Canvas 5</DropdownItem>
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
                <DropdownItem href="/canvas4/1">Manage members</DropdownItem>
                <DropdownItem href="/canvas4/1">Delete</DropdownItem>
              </DropdownMenu>
            </Dropdown>
          </nav>
        );
      default:
        return (
          <NavigationOrg
            user={currentUser}
            organization={currentOrganization}
            breadcrumbs={[
              {
                label: "Canvases",
                icon: "automation",
                href: "/canvases",
                current: false
              },
              {
                label: getCanvasName(canvasId),
                current: true,
                starred: isCanvasStarred,
                dropdown: [
                  {
                    label: "Manage Members",
                    onClick: handleMembers
                  },
                  {
                    label: "Delete",
                    onClick: handleDelete
                  }
                ],
                onStarToggle: setIsStarred
              }
            ]}
            onHelpClick={handleHelpClick}
            onUserMenuAction={handleUserMenuAction}
            onOrganizationMenuAction={handleOrganizationMenuAction}
          />
        );
    }
  };

  return (
    <div className="flex flex-col min-h-screen bg-gray-50">
      {renderNavigation()}

      {/* React Flow Canvas */}
      <div className="flex-1 flex">
        {/* Component Sidebar */}
        <div className='w-80 bg-white dark:bg-zinc-800 border-r border-zinc-200 dark:border-zinc-700'> 
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
        
        {/* Right Sidebar for Node Editing */}
        {rightSidebarOpen && nodeEditorMode === 'sidebar' && (
          <div className="bg-white dark:bg-zinc-800 border-l border-zinc-200 dark:border-zinc-700 flex flex-col">
            {/* Sidebar Header */}
            <div className="p-4 border-b border-zinc-200 dark:border-zinc-700 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">
                Edit Node
              </h2>
              <Button
                plain
                onClick={() => {
                  setRightSidebarOpen(false);
                  setEditingNodeId(null);
                }}
                className="text-zinc-600 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300"
              >
                <MaterialSymbol name="close" size="lg" />
              </Button>
            </div>
            
            {/* Sidebar Content */}
            <div className="flex-1 flex flex-col">
              {renderNodeEditorContent()}
            </div>
          </div>
        )}
      </div>

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

      {/* Node Editor Modal */}
      <Dialog 
        className='bg-white dark:bg-zinc-900'
        open={showNodeEditorModal && nodeEditorMode === 'modal'} 
        onClose={() => {
          setShowNodeEditorModal(false);
          setEditingNodeId(null);
        }}
        size="2xl"
      >
        <DialogTitle className='flex items-center justify-between'>
          <div className="flex items-center gap-2">
            <MaterialSymbol name="edit" size="lg" />
            <span>Edit Node</span>
          </div>
          <Button 
            plain 
            onClick={() => {
              setShowNodeEditorModal(false);
              setEditingNodeId(null);
            }}
          >
            <MaterialSymbol name="close" size='lg' />
          </Button>
        </DialogTitle>
        <DialogDescription>
          Edit the configuration for the selected workflow node.
        </DialogDescription>
        
        <DialogBody className="p-0">
          <div className="h-[600px] flex flex-col">
            {renderNodeEditorContent()}
          </div>
        </DialogBody>
      </Dialog>

      {/* GitHub Authentication Modal */}
      <Dialog open={showGitHubModal} onClose={() => setShowGitHubModal(false)}>
        <DialogTitle>Connect to GitHub</DialogTitle>
        <DialogDescription>
          Connect your GitHub account to access your repositories and enable GitHub-based execution.
        </DialogDescription>
        <DialogBody>
          <div className="flex flex-col items-center space-y-4 py-6">
            <div className="flex items-center justify-center w-16 h-16 bg-gray-100 dark:bg-gray-800 rounded-full">
              <MaterialSymbol name="account_circle" size="lg" className="text-gray-600 dark:text-gray-400" />
            </div>
            <div className="text-center">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                GitHub Authentication
              </h3>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Click the button below to authenticate with GitHub and access your repositories.
              </p>
            </div>
          </div>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={() => setShowGitHubModal(false)}>
            Cancel
          </Button>
          <Button onClick={handleGitHubLogin} color="blue" className="flex items-center gap-2">
            <MaterialSymbol name="login" size="sm" />
            Login with GitHub
          </Button>
        </DialogActions>
      </Dialog>

    </div>
  );
};
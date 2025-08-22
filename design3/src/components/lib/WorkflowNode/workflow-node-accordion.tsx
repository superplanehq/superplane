import { useState, useEffect, useCallback, useRef } from 'react'
import { Handle, Position } from '@xyflow/react';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { getStatusConfig } from '../../../utils/status-config'
import { Button } from '../Button/button'
import { Input } from '../Input/input'
import { Textarea } from '../Textarea/textarea'
import { Field, Label } from '../Fieldset/fieldset'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem,
  DropdownLabel
} from '../Dropdown/dropdown'
import { 
  Dialog, 
  DialogTitle,
  DialogBody
} from '../Dialog/dialog'
import { ControlledAccordion, type AccordionItem } from '../Accordion/accordion'
import { type WorkflowNodeData, type WorkflowNodeProps } from './workflow-node'
import clsx from 'clsx'
import { Subheading } from '../Heading/heading'
import { Divider } from '../Divider/divider'
import { Text } from '../Text/text'
import { Link } from '../Link/link'
import Tippy from '@tippyjs/react'
import { Badge, BadgeButton } from '../Badge/badge'
import { ControlledTabs } from '../Tabs/tabs';

export type { WorkflowNodeData } from './workflow-node'

export interface WorkflowNodeAccordionProps extends Omit<WorkflowNodeProps, 'tabs'> {
  sections?: AccordionItem[]
  multiple?: boolean
  partialSave?: boolean
  saveGranular?: boolean
  onSelect?: () => void
  nodes?: any[]
  totalNodesCount?: number
  savedConnectionIndices?: number[]
}

export function WorkflowNodeAccordion({
  data,
  variant = 'read',
  selected,
  className,
  sections: customSections,
  multiple = true,
  partialSave = false,
  saveGranular = false,
  onUpdate,
  onDelete,
  onEdit,
  onSave,
  onCancel,
  onSelect,
  nodes = [],
  totalNodesCount = 0,
  savedConnectionIndices = []
}: WorkflowNodeAccordionProps) {
  const [editedTitle, setEditedTitle] = useState(data.title)
  const [editedDescription, setEditedDescription] = useState(data.description || '')
  const [editedType, setEditedType] = useState(data.type)
  
  // YAML configuration state
  const [yamlConfig, setYamlConfig] = useState(data.yamlConfig || {
    apiVersion: 'v1',
    kind: 'Stage',
    metadata: {
      name: data.title.toLowerCase().replace(/\s+/g, '-'),
      canvasId: ''
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
  })

  // Sync internal yamlConfig with prop changes
  useEffect(() => {
    if (data.yamlConfig) {
      setYamlConfig(data.yamlConfig);
    }
  }, [data.yamlConfig])

  // Sync savedConnections with prop changes
  useEffect(() => {
    setSavedConnections(new Set(savedConnectionIndices));
  }, [savedConnectionIndices])

  // Sync temp values when data changes
  useEffect(() => {
    setTempTitle(data.title);
    setTempDescription(data.description || '');
  }, [data.title, data.description])

  // Helper function to mark a section as modified
  const markSectionModified = useCallback((sectionId: string) => {
    setModifiedSections(prev => new Set([...prev, sectionId]));
  }, []);

  // Helper function to clear modified status for a section
  const clearSectionModified = useCallback((sectionId: string) => {
    setModifiedSections(prev => {
      const newSet = new Set(prev);
      newSet.delete(sectionId);
      return newSet;
    });
  }, []);

  // Track connections changes from external sources (like modal saves)
  const prevConnectionsRef = useRef<number>(data.yamlConfig?.spec?.connections?.length || 0);
  
  useEffect(() => {
    const currentConnectionsCount = data.yamlConfig?.spec?.connections?.length || 0;
    const prevConnectionsCount = prevConnectionsRef.current;
    
    // If connections were added externally, mark section as modified
    if (currentConnectionsCount > prevConnectionsCount) {
      markSectionModified('connections');
    }
    
    // Update the reference
    prevConnectionsRef.current = currentConnectionsCount;
  }, [data.yamlConfig?.spec?.connections, markSectionModified])

  
  // Component to render the orange modification indicator
  const ModificationIndicator = ({ sectionId }: { sectionId: string }) => {
    if (!modifiedSections.has(sectionId)) return null;
    
    return (
      
      <div className="flex items-center gap-2 ml-2">
        <div className="w-1.5 h-1.5 bg-orange-500 rounded-full" />
        <Link href="#" onClick={() => clearSectionModified(sectionId)} className='leading-none'> 
          <MaterialSymbol 
            name="undo" 
            size="sm" 
            className="text-gray-600 hover:text-gray-800 cursor-pointer"
          />
        </Link>
      </div>
    );
  }

  // Component to render the orange dot for modified fields
  const FieldModificationIndicator = ({ field }: { field: 'title' | 'description' }) => {
    if (!modifiedFields.has(field)) return null;
    
    return (
      <div className="w-1.5 h-1.5 bg-orange-500 rounded-full ml-2" />
    );
  }

  // Inline editing handlers
  const handleStartEdit = (field: 'title' | 'description') => {
    setEditingField(field);
    if (field === 'title') {
      setTempTitle(data.title);
    } else {
      setTempDescription(data.description || '');
    }
  }

  const handleSaveInlineEdit = (field: 'title' | 'description') => {
    const hasChanged = field === 'title' ? tempTitle !== data.title : tempDescription !== (data.description || '');
    
    if (hasChanged) {
      // Mark field as modified
      setModifiedFields(prev => new Set([...prev, field]));
      
      // Update the data
      if (field === 'title') {
        onUpdate?.({ title: tempTitle });
      } else {
        onUpdate?.({ description: tempDescription });
      }
    }
    
    setEditingField(null);
  }

  const handleCancelInlineEdit = () => {
    setTempTitle(data.title);
    setTempDescription(data.description || '');
    setEditingField(null);
  }

  const handleKeyDown = (e: React.KeyboardEvent, field: 'title' | 'description') => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSaveInlineEdit(field);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      handleCancelInlineEdit();
    }
  }
  
  // Accordion state
  const [openSections, setOpenSections] = useState<string[]>([])
  
  // Track which sections have been saved
  const [savedSections, setSavedSections] = useState<Set<string>>(new Set())
  
  // Track which sections have been modified
  const [modifiedSections, setModifiedSections] = useState<Set<string>>(new Set())
  
  // Track which connections are in read-only mode
  const [savedConnections, setSavedConnections] = useState<Set<number>>(new Set(savedConnectionIndices))
  
  // Track which inputs are in read-only mode
  const [savedInputs, setSavedInputs] = useState<Set<number>>(new Set())
  
  // Track which executors are in read-only mode
  const [savedExecutors, setSavedExecutors] = useState<Set<number>>(new Set())
  
  
  // Filter state for connections
  const [connectionFilters, setConnectionFilters] = useState<Record<number, Array<{id: string, type: string, expression: string, operator?: string}>>>({})
  
  // State to track which connection filters are expanded
  const [expandedFilters, setExpandedFilters] = useState<Set<number>>(new Set())
  
  // State to track inline editing for title and description
  const [editingField, setEditingField] = useState<'title' | 'description' | null>(null)
  const [tempTitle, setTempTitle] = useState(data.title)
  const [tempDescription, setTempDescription] = useState(data.description || '')
  
  // State to track which fields have been modified through inline editing
  const [modifiedFields, setModifiedFields] = useState<Set<'title' | 'description'>>(new Set())
  
  // Input mappings state for each input
  const [inputMappings, setInputMappings] = useState<Record<string, Array<{id: string, connection: string, value: string}>>>({})
  
  // GitHub integration state
  const [showGitHubModal, setShowGitHubModal] = useState(false)
  const [isGitHubConnected, setIsGitHubConnected] = useState(false)
  const [githubProjects, setGithubProjects] = useState<Array<{id: string, name: string, url: string}>>([])
  const [selectedGitHubProject, setSelectedGitHubProject] = useState<string>('')
  const showIcons = new URLSearchParams(window.location.search).get('showIcons') === 'true';
  const executorInHeader = new URLSearchParams(window.location.search).get('executorInHeader') === 'true';
  const consistentStatuses = new URLSearchParams(window.location.search).get('consistentStatuses') === 'true';
  
  // Generate YAML preview
  const generateYamlPreview = () => {
    const yamlData = {
      apiVersion: yamlConfig.apiVersion,
      kind: yamlConfig.kind,
      metadata: yamlConfig.metadata,
      spec: {
        ...(yamlConfig.spec.secrets?.length ? { secrets: yamlConfig.spec.secrets } : {}),
        ...(yamlConfig.spec.connections?.length ? { connections: yamlConfig.spec.connections } : {}),
        ...(yamlConfig.spec.inputs?.length ? { inputs: yamlConfig.spec.inputs } : {}),
        ...(Object.keys(yamlConfig.spec.inputMappings || {}).length ? { inputMappings: yamlConfig.spec.inputMappings } : {}),
        ...(yamlConfig.spec.outputs?.length ? { outputs: yamlConfig.spec.outputs } : {}),
        ...(yamlConfig.spec.executor ? { executor: yamlConfig.spec.executor } : {})
      }
    }
    
    return JSON.stringify(yamlData, null, 2)
      .replace(/"/g, '')
      .replace(/,$/gm, '')
      .replace(/\{/g, '')
      .replace(/\}/g, '')
      .replace(/\[/g, '')
      .replace(/\]/g, '')
      .split('\n')
      .filter(line => line.trim())
      .map(line => line.replace(/^\s+/, match => '  '.repeat(match.length / 2)))
      .join('\n')
  }

  const handleSave = () => {
    onUpdate?.({
      title: editedTitle,
      description: editedDescription,
      type: editedType,
      yamlConfig: yamlConfig
    })
    onSave?.()
  }

  // Individual section save handlers
  const handleConnectionsSave = () => {
    onUpdate?.({
      yamlConfig: {
        ...yamlConfig,
        spec: {
          ...yamlConfig.spec,
          connections: yamlConfig.spec.connections || []
        }
      }
    })
  }

  const handleInputsSave = () => {
    // Convert inputMappings to the format expected by yamlConfig
    const formattedMappings: Record<string, string> = {};
    Object.entries(inputMappings).forEach(([inputId, mappings]) => {
      formattedMappings[inputId] = mappings.map(m => `${m.connection}:${m.value}`).join(',');
    });
    
    onUpdate?.({
      yamlConfig: {
        ...yamlConfig,
        spec: {
          ...yamlConfig.spec,
          inputs: yamlConfig.spec.inputs || [],
          inputMappings: { ...yamlConfig.spec.inputMappings, ...formattedMappings }
        }
      }
    })
    
    // Clear modification status when saving
    clearSectionModified('inputs');
    console.log('All inputs saved:', yamlConfig.spec.inputs);
  }

  const handleExecutorsSave = () => {
    onUpdate?.({
      yamlConfig: {
        ...yamlConfig,
        spec: {
          ...yamlConfig.spec,
          executor: yamlConfig.spec.executor || { type: 'default', config: {} }
        }
      }
    })
    
    // Clear modification status when saving
    clearSectionModified('executor');
    console.log('Executor saved:', yamlConfig.spec.executor);
  }

  const handleOutputsSave = () => {
    onUpdate?.({
      yamlConfig: {
        ...yamlConfig,
        spec: {
          ...yamlConfig.spec,
          outputs: yamlConfig.spec.outputs || []
        }
      }
    })
  }

  const handleSecretsSave = () => {
    onUpdate?.({
      yamlConfig: {
        ...yamlConfig,
        spec: {
          ...yamlConfig.spec,
          secrets: yamlConfig.spec.secrets || []
        }
      }
    })
  }

  const handleExecutorSave = () => {
    onUpdate?.({
      yamlConfig: {
        ...yamlConfig,
        spec: {
          ...yamlConfig.spec,
          executor: yamlConfig.spec.executor || { type: 'default', config: {} }
        }
      }
    })
  }

  // GitHub integration handlers
  const handleExecutorTypeChange = (type: string) => {
    setYamlConfig(prev => ({
      ...prev,
      spec: {
        ...prev.spec,
        executor: {
          type,
          config: {}
        }
      }
    }))
    
    // Reset GitHub state when switching away from GitHub
    if (type !== 'github') {
      setIsGitHubConnected(false)
      setGithubProjects([])
      setSelectedGitHubProject('')
    }
  }

  const handleConnectGitHub = () => {
    setShowGitHubModal(true)
  }

  const handleGitHubLogin = () => {
    // Simulate GitHub authentication and fetch projects
    setTimeout(() => {
      setIsGitHubConnected(true)
      setGithubProjects([
        { id: '1', name: 'my-awesome-project', url: 'https://github.com/user/my-awesome-project' },
        { id: '2', name: 'react-components', url: 'https://github.com/user/react-components' },
        { id: '3', name: 'api-service', url: 'https://github.com/user/api-service' },
        { id: '4', name: 'frontend-app', url: 'https://github.com/user/frontend-app' },
        { id: '5', name: 'backend-service', url: 'https://github.com/user/backend-service' }
      ])
      setShowGitHubModal(false)
    }, 1000)
  }

  const handleGitHubProjectSelect = (projectId: string) => {
    setSelectedGitHubProject(projectId)
    const selectedProject = githubProjects.find(p => p.id === projectId)
    if (selectedProject) {
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
      }))
    }
  }

  const handleCancel = () => {
    setEditedTitle(data.title)
    setEditedDescription(data.description || '')
    setEditedType(data.type)
    setYamlConfig(data.yamlConfig || {
      apiVersion: 'v1',
      kind: 'Stage',
      metadata: {
        name: data.title.toLowerCase().replace(/\s+/g, '-'),
        canvasId: ''
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
    })
    onCancel?.()
  }

  const handleInputFocus = () => {
    onSelect?.()
  }

  const handleAddConnection = () => {
    setYamlConfig(prev => {
      const currentConnections = prev.spec.connections || [];
      const newConnection = { 
        name: '', 
        type: 'stage', 
        config: {} 
      };
      
      // Only mark as modified if this is adding the first connection
      if (currentConnections.length === 0) {
        markSectionModified('connections');
      }
      
      return {
        ...prev,
        spec: {
          ...prev.spec,
          connections: [...currentConnections, newConnection]
        }
      };
    });
  }

  const handleAddInput = () => {
    setYamlConfig(prev => {
      const currentInputs = prev.spec.inputs || [];
      const newInput = { 
        name: '', 
        description: '', 
        type: 'string', 
        required: false 
      };
      
      // Only mark as modified if this is adding the first input
      if (currentInputs.length === 0) {
        markSectionModified('inputs');
      }
      
      return {
        ...prev,
        spec: {
          ...prev.spec,
          inputs: [...currentInputs, newInput]
        }
      };
    });
  }

  const handleAddExecutor = () => {
    setYamlConfig(prev => {
      const newExecutor = { 
        type: 'semaphore', 
        config: {} 
      };
      
      // Mark as modified when adding/changing executor
      markSectionModified('executor');
      
      return {
        ...prev,
        spec: {
          ...prev.spec,
          executor: newExecutor
        }
      };
    });
    
    // Ensure the executor starts in editable mode
    setSavedExecutors(prev => {
      const newSaved = new Set(prev);
      newSaved.delete(0); // Remove from saved to make it editable
      return newSaved;
    });
  }

  const handleAddFilter = (connectionIndex: number) => {
    const existingFilters = connectionFilters[connectionIndex] || []
    
    // If there are existing filters, use their operator, otherwise default to 'AND'
    let currentOperator = 'AND'
    if (existingFilters.length > 0) {
      // Find the current operator from existing filters (they should all be the same)
      const filterWithOperator = existingFilters.find(filter => filter.operator)
      currentOperator = filterWithOperator?.operator || 'AND'
    }
    
    const newFilter = {
      id: `filter_${Date.now()}`,
      type: 'Data',
      expression: '',
      operator: existingFilters.length > 0 ? currentOperator : undefined
    }
    
    setConnectionFilters(prev => ({
      ...prev,
      [connectionIndex]: [...(prev[connectionIndex] || []), newFilter]
    }))
    
    // Only mark as modified if there are existing connections
    if (yamlConfig.spec.connections && yamlConfig.spec.connections.length > 0) {
      markSectionModified('connections');
    }
  }

  const handleRemoveFilter = (connectionIndex: number, filterId: string) => {
    setConnectionFilters(prev => ({
      ...prev,
      [connectionIndex]: (prev[connectionIndex] || []).filter(filter => filter.id !== filterId)
    }))
    
    // Only mark as modified if there are existing connections
    if (yamlConfig.spec.connections && yamlConfig.spec.connections.length > 0) {
      markSectionModified('connections');
    }
  }

  const handleUpdateFilter = (connectionIndex: number, filterId: string, field: 'type' | 'expression', value: string) => {
    setConnectionFilters(prev => ({
      ...prev,
      [connectionIndex]: (prev[connectionIndex] || []).map(filter => 
        filter.id === filterId ? { ...filter, [field]: value } : filter
      )
    }))
    
    // Only mark as modified if there are existing connections
    if (yamlConfig.spec.connections && yamlConfig.spec.connections.length > 0) {
      markSectionModified('connections');
    }
  }

  const handleToggleOperator = (connectionIndex: number, filterId: string) => {
    setConnectionFilters(prev => {
      const currentFilters = prev[connectionIndex] || []
      const clickedFilter = currentFilters.find(filter => filter.id === filterId)
      const newOperator = clickedFilter?.operator === 'OR' ? 'AND' : 'OR'
      
      // Update all filters with operators to have the same operator
      return {
        ...prev,
        [connectionIndex]: currentFilters.map(filter => 
          filter.operator 
            ? { ...filter, operator: newOperator }
            : filter
        )
      }
    })
    
    // Only mark as modified if there are existing connections
    if (yamlConfig.spec.connections && yamlConfig.spec.connections.length > 0) {
      markSectionModified('connections');
    }
  }

  const handleAddInputMapping = (inputId: string) => {
    const newMapping = {
      id: `mapping_${Date.now()}`,
      connection: '',
      value: ''
    }
    
    setInputMappings(prev => ({
      ...prev,
      [inputId]: [...(prev[inputId] || []), newMapping]
    }))
  }

  const handleRemoveInputMapping = (inputId: string, mappingId: string) => {
    setInputMappings(prev => ({
      ...prev,
      [inputId]: (prev[inputId] || []).filter(mapping => mapping.id !== mappingId)
    }))
  }

  const handleUpdateInputMapping = (inputId: string, mappingId: string, field: 'connection' | 'value', value: string) => {
    setInputMappings(prev => ({
      ...prev,
      [inputId]: (prev[inputId] || []).map(mapping => 
        mapping.id === mappingId ? { ...mapping, [field]: value } : mapping
      )
    }))
  }

  const getTypeIcon = (type: WorkflowNodeData['type']) => {
    switch (type) {
      case 'stage':
        return 'rocket_launch'
      case 'event':
        return 'bolt'
      default:
        return 'circle'
    }
  }

  const getStatusColor = (status?: WorkflowNodeData['status']) => {
    switch (status) {
      case 'running':
        return 'text-blue-600 dark:text-blue-400'
      case 'success':
        return 'text-green-600 dark:text-green-400'
      case 'error':
        return 'text-red-600 dark:text-red-400'
      case 'disabled':
        return 'text-zinc-400 dark:text-zinc-500'
      default:
        return 'text-zinc-600 dark:text-zinc-400'
    }
  }

  const getTypeColor = (type: WorkflowNodeData['type']) => {
    switch (type) {
      case 'stage':
        return 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
      case 'event':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400'
      default:
        return 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/20 dark:text-zinc-400'
    }
  }


  // Helper function to render the event trigger chain
  const renderEventChainTooltip = (nodeData: WorkflowNodeData) => {
    // Mock event chain data with real workflow node names
    const eventChain = [
      {
        nodeId: 'event-source',
        nodeName: 'Event Source',
        eventId: 'evt_webhook_received_abc123',
        timestamp: '2024-01-15 14:32:15',
        type: 'webhook_received',
        description: 'Incoming event trigger'
      },
      {
        nodeId: 'stage-1',
        nodeName: 'Sync Cluster',
        eventId: 'run_completed_sync456',
        timestamp: '2024-01-15 14:33:02',
        type: 'run_completed',
        description: 'Cluster sync completed successfully',
        triggeredBy: 'event-source'
      },
      {
        nodeId: 'stage-2',
        nodeName: 'AI Agent triage', 
        eventId: 'run_completed_triage789',
        timestamp: '2024-01-15 14:35:18',
        type: 'run_completed',
        description: 'AI triage process completed',
        triggeredBy: 'stage-1'
      },
      {
        nodeId: nodeData.id,
        nodeName: nodeData.title,
        eventId: nodeData.eventId || 'current_event',
        timestamp: '2024-01-15 14:37:45',
        type: 'run_started',
        description: 'Current workflow execution',
        triggeredBy: nodeData.id === 'stage-2' ? 'stage-1' : nodeData.id === 'stage-3' ? 'stage-2' : nodeData.id === 'stage-4' ? 'stage-3' : 'stage-2'
      }
    ];

    return (
      <div className="bg-white dark:bg-zinc-800 p-4 rounded-lg border border-gray-200 dark:border-zinc-700 max-w-sm">
        <div className="text-sm font-semibold text-gray-900 dark:text-white mb-3">
          Event Trigger Chain
        </div>
        <div className="space-y-3">
          {eventChain.map((event, index) => {
            const isCurrentNode = event.nodeId === nodeData.id;
            const isLastEvent = index === eventChain.length - 1;
            
            return (
              <div key={event.eventId} className="relative">
                {/* Event item */}
                <div className={`flex items-start gap-3 ${isCurrentNode ? 'bg-blue-50 dark:bg-blue-900/20 p-2 rounded border border-blue-200 dark:border-blue-800' : ''}`}>
                  {/* Timeline dot */}
                  <div className="relative flex-shrink-0 mt-1">
                    <div className={`w-2 h-2 rounded-full ${isCurrentNode ? 'bg-blue-500' : 'bg-gray-400 dark:bg-zinc-500'}`}></div>
                    {!isLastEvent && (
                      <div className="absolute top-2 left-1 w-px h-6 bg-gray-300 dark:bg-zinc-600"></div>
                    )}
                  </div>
                  
                  {/* Event details */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <span className={`text-sm font-medium truncate ${isCurrentNode ? 'text-blue-900 dark:text-blue-100' : 'text-gray-900 dark:text-white'}`}>
                        {event.nodeName}
                      </span>
                      <span className="text-xs text-gray-500 dark:text-zinc-400 flex-shrink-0 ml-2">
                        {new Date(event.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}
                      </span>
                    </div>
                    <div className="text-xs text-gray-600 dark:text-zinc-400 mt-1">
                      {event.description}
                    </div>
                    <div className="text-xs font-mono text-gray-500 dark:text-zinc-500 mt-1">
                      {event.eventId}
                    </div>
                  </div>
                </div>
                
                {/* Arrow connecting events */}
                {!isLastEvent && event.triggeredBy && (
                  <div className="flex items-center ml-4 mt-1 mb-1 text-xs text-gray-500 dark:text-zinc-400">
                    <MaterialSymbol name="arrow_downward" size="sm" className="mr-1" />
                    <span>triggered</span>
                  </div>
                )}
              </div>
            );
          })}
        </div>
        
        {/* Summary */}
        <div className="mt-4 pt-3 border-t border-gray-200 dark:border-zinc-700">
          <div className="text-xs text-gray-600 dark:text-zinc-400">
            <span className="font-medium">Current trigger:</span> {nodeData.triggeredBy || 'Unknown'}
          </div>
          <div className="text-xs text-gray-600 dark:text-zinc-400 mt-1">
            <span className="font-medium">Event ID:</span> <span className="font-mono">{nodeData.eventId || 'N/A'}</span>
          </div>
        </div>
      </div>
    );
  };

  // Helper function to render inputs in tooltip format
  const renderInputsTooltip = (outputs = false, inputs: Array<{name: string, type: string, required?: boolean, defaultValue?: any}>) => {
    if (!inputs || inputs.length === 0) return null;
    
    const inputsRecord: Record<string, string> = {};
    inputs.forEach(input => {
      inputsRecord[input.name] = input.defaultValue || `${input.type}${input.required ? ' (required)' : ''}`;
    });

    return (
      <div className="min-w-[250px] max-w-xs">
        <div className="bg-white dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 rounded-lg p-3">
          <div className="flex items-start gap-3">
        
            <div className="flex-1">
              <span className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">
                { !outputs ? "Inputs" : "Outputs" }
              </span>
              <div className="space-y-1">
                {Object.entries(inputsRecord).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between">
                    <span className="text-xs text-gray-600 dark:text-zinc-300 font-medium">{key}</span>
                    <div className="flex items-center gap-2">
                      <Badge className='font-mono !text-xs'>
                        12313123
                      </Badge>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  };

  // Default accordion sections
  const defaultSections: AccordionItem[] = [
    {
      id: 'general',
      title: (
        <div className="flex items-center">
          <span>General</span>
          <ModificationIndicator sectionId="general" />
        </div>
      ),
      defaultOpen: true,
      content: (
        <div className="space-y-4">
          {/* Name Field */}
          <Field>
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              placeholder="Enter stage name"
              value={editedTitle}
              onChange={(e) => {
                const newTitle = e.target.value;
                setEditedTitle(newTitle);
                markSectionModified('general');
                // Update the node title in real-time
                onUpdate?.({
                  title: newTitle
                });
              }}
              onFocus={handleInputFocus}
              className="w-full"
            />
          </Field>

          {/* Description Field */}
          <Field>
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              placeholder="Enter stage description"
              value={editedDescription}
              onChange={(e) => {
                setEditedDescription(e.target.value);
                markSectionModified('general');
              }}
              onFocus={handleInputFocus}
              rows={3}
              className="w-full"
            />
          </Field>

          {partialSave && (
            <>
              <Divider/>
              <Field className='flex justify-end'>
                <Button
                  color='blue'
                  className='flex items-center !text-xs'
                  onClick={() => {
                    onUpdate?.({
                      title: editedTitle,
                      description: editedDescription
                    });
                    console.log('General info saved:', { title: editedTitle, description: editedDescription });
                    
                    // Mark general section as saved
                    setSavedSections(prev => new Set([...prev, 'general']));
                    
                    // Collapse the general accordion section
                    setOpenSections(prev => prev.filter(id => id !== 'general'));
                  }}
                >
                  <MaterialSymbol name="save" size="sm" />
                  Save
                </Button>
              </Field>
            </>
          )}
        </div>
      )
    },
    {
      id: 'connections',
      title: (
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center">
            <span>Connections</span>
            <ModificationIndicator sectionId="connections" />
          </div>
          {yamlConfig.spec.connections && yamlConfig.spec.connections.length > 0 && (
            <span className="text-xs text-gray-600 dark:text-gray-400 text-code !font-normal pr-2">
              {yamlConfig.spec.connections.length} connection{yamlConfig.spec.connections.length !== 1 ? 's' : ''}
            </span>
          )}
        </div>
      ),
      defaultOpen: true,
      content: (
        <div className="space-y-4">
          

          {/* Connections List */}
          <div className="space-y-2">
            {yamlConfig.spec.connections?.map((connection, index) => (
              <div key={index} className="flex connection">
                {savedConnections.has(index) ? (
                  // Read-only mode - entire connection box is read-only
                  <div className="flex-auto space-y-1 border border-zinc-50 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/20 p-2 rounded-sm">
                    {/* Connection name with edit button */}
                    <div className="flex items-center justify-between">
                      <h4 className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                      
                        {connection.type === 'stage' ? (
                          <span className="flex items-center gap-1">
                            <MaterialSymbol name="rocket_launch" size="sm" />
                            Deploy to staging
                          </span>
                        ) : (
                          <span className="flex items-center gap-1">
                            <MaterialSymbol name="bolt" size="sm" />
                            Github webhook
                          </span>
                        )}
                      </h4>
                      <div className="flex items-center">
                      <Button
                        plain
                        className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
                        onClick={() => {
                          // Remove from saved connections to make it editable again
                          setSavedConnections(prev => {
                            const newSaved = new Set(prev);
                            newSaved.delete(index);
                            return newSaved;
                          });
                          console.log('Connection made editable:', connection);
                        }}
                      >
                        <MaterialSymbol name="edit" size="sm" />
                        
                      </Button>
                      <Button
                        plain
                        className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300"
                        onClick={() => {
                          // Remove connection from the list
                          const newConnections = yamlConfig.spec.connections?.filter((_, i) => i !== index) || []
                          setYamlConfig(prev => ({ 
                            ...prev, 
                            spec: { 
                              ...prev.spec, 
                              connections: newConnections 
                            } 
                          }))
                          
                          markSectionModified('connections');
                          
                          // Also remove from saved connections and expanded filters
                          setSavedConnections(prev => {
                            const newSaved = new Set(prev);
                            newSaved.delete(index);
                            // Update indices for remaining connections
                            const updatedSaved = new Set<number>();
                            newSaved.forEach(savedIndex => {
                              if (savedIndex < index) {
                                updatedSaved.add(savedIndex);
                              } else if (savedIndex > index) {
                                updatedSaved.add(savedIndex - 1);
                              }
                            });
                            return updatedSaved;
                          });
                          
                          // Remove from expanded filters
                          setExpandedFilters(prev => {
                            const newExpanded = new Set(prev);
                            newExpanded.delete(index);
                            // Update indices for remaining connections
                            const updatedExpanded = new Set<number>();
                            newExpanded.forEach(expandedIndex => {
                              if (expandedIndex < index) {
                                updatedExpanded.add(expandedIndex);
                              } else if (expandedIndex > index) {
                                updatedExpanded.add(expandedIndex - 1);
                              }
                            });
                            return updatedExpanded;
                          });
                          
                          console.log('Connection deleted:', connection);
                        }}
                      >
                        <MaterialSymbol name="delete" size="sm" />
                        
                      </Button>
                      </div>
                    </div>
                    
                    {/* Collapsible Filters Group */}
                    {connectionFilters[index] && connectionFilters[index].length > 0 && (
                      <div className="mt-2">
                        <Link 
                          href="#"
                          className="flex items-center gap-2 text-xs text-zinc-600 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
                          onClick={() => {
                            // Toggle filter visibility for this specific connection
                            setExpandedFilters(prev => {
                              const newSet = new Set(prev);
                              if (newSet.has(index)) {
                                newSet.delete(index);
                              } else {
                                newSet.add(index);
                              }
                              return newSet;
                            });
                          }}
                        >
                          <MaterialSymbol 
                            name={expandedFilters.has(index) ? "keyboard_arrow_down" : "keyboard_arrow_right"} 
                            size="sm" 
                          />
                          {connectionFilters[index].length} filter{connectionFilters[index].length !== 1 ? 's' : ''}
                        </Link>
                        
                        {/* Collapsible filters content */}
                        {expandedFilters.has(index) && (
                          <div className="mt-2">
                          {connectionFilters[index].map((filter, filterIndex) => (
                            <div key={filter.id} className='relative w-full'>
                              {/* Show AND/OR indicator (read-only) */}
                              {filter.operator && filterIndex > 0 && (
                                <div className="relative justify-center flex items-center">
                                  <span className="!text-xs font-medium !px-2 !py-0 text-zinc-700 dark:text-zinc-300">
                                    {filter.operator}
                                  </span>
                                </div>
                              )}
                              
                              <div className="flex items-center">
                                <div className="flex flex-auto bg-white dark:bg-zinc-900/40 items-center gap-2 rounded-lg text-xs border border-zinc-300 dark:border-zinc-800">
                                  <span className="rounded-md rounded-r-none px-2 bg-zinc-100 dark:bg-zinc-900/40  text-zinc-700 py-1 dark:text-zinc-300 border-r border-zinc-400 dark:border-zinc-800">
                                    {filter.type}
                                  </span>
                                  <span className="text-zinc-600 py-1 dark:text-zinc-400 font-mono">
                                    {filter.expression}
                                  </span>
                                </div>
                              </div>
                            </div>
                          ))}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                ) : (
                  // Editable mode - show full editable connection box
                  <div className="flex-auto space-y-1 border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 p-1 rounded-sm">
                    <div className="flex flex-col">
                      <Field className='flex justify-between'>
                        <Dropdown>
                          <DropdownButton color='white' className="!justify-between flex items-center w-full">
                            Select connection
                            <MaterialSymbol name="expand_more" size="md" />
                          </DropdownButton>
                          <DropdownMenu anchor="bottom start">
                            <DropdownItem className='flex items-center gap-2' onClick={() => {
                              const newConnections = [...(yamlConfig.spec.connections || [])]
                              newConnections[index] = { ...connection, type: 'stage' }
                              setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                              // Mark as modified since we're editing an existing connection
                              markSectionModified('connections');
                            }}>
                              
                                <MaterialSymbol name="rocket_launch" size="md" />
                                <DropdownLabel> Deploy to staging</DropdownLabel>
                            </DropdownItem>
                            <DropdownItem className='flex items-center gap-2' onClick={() => {
                              const newConnections = [...(yamlConfig.spec.connections || [])]
                              newConnections[index] = { ...connection, type: 'event source' }
                              setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                              // Mark as modified since we're editing an existing connection
                              markSectionModified('connections');
                            }}>
                              
                              <MaterialSymbol name="bolt" size="sm" />
                              <DropdownLabel>Github webhook</DropdownLabel>
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </Field>
                    </div>
                  {/* Filters List */}
                  {connectionFilters[index] && connectionFilters[index].length > 0 && (
                    <Field className="">
                      <Label className="!text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-1">
                        Filters
                      </Label>
                      {connectionFilters[index].map((filter, filterIndex) => (
                        <div key={filter.id} className='relative w-full'>
                          {/* Show AND/OR button before filter (except for the first filter) */}
                          {filter.operator && filterIndex > 0 && (
                            <div className={filter.operator === 'AND' ? "relative justify-center flex items-center" : "relative justify-center flex items-center"}>
                              <Link
                                href="#"
                                onClick={() => handleToggleOperator(index, filter.id)}
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
                                  <DropdownMenu anchor="bottom start">
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
                                  onFocus={handleInputFocus}
                                  className="flex-auto text-xs"
                                />
                              </div>
                              <div className='flex items-center'>
                                <Link
                                  href='#'
                                  onClick={() => handleRemoveFilter(index, filter.id)}
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
                    {/* Add Filter Button */}
                    <Link
                      href="#"
                      onClick={() => handleAddFilter(index)}
                      className="flex items-center !text-xs"
                    >
                      <MaterialSymbol name="add" size="sm" />
                      Add Filter
                    </Link>

                    {/* Save Button - only show if saveGranular is true and connection is not saved */}
                    {saveGranular && (
                      <div className='flex items-center justify-end w-full border-t border-zinc-200 dark:border-zinc-700 pt-2'>
                          <Button
                            plain
                            className='flex items-center !text-xs'
                            onClick={() => {
                              setSavedConnections(prev => new Set([...prev, index]));
                              console.log('Connection saved:', connection);
                            }}
                          >
                            Cancel
                          </Button>
                          <Button
                            color='blue'
                            className='flex items-center !text-xs'
                            onClick={() => {
                              setSavedConnections(prev => new Set([...prev, index]));
                              // Clear modification status when saving
                              clearSectionModified('connections');
                              console.log('Connection saved:', connection);
                            }}
                          >
                            <MaterialSymbol name="save" size="sm" />
                            Save
                          </Button>
                        </div>
                    )}
                  </div>
                )}
                
                
              </div>
            ))}
             {totalNodesCount > 1 && (
            <div>
              <Link
                href='#'
                onClick={() => {
                  // Use inline behavior to add connection
                  handleAddConnection();
                }}
                className="text-gray-600 hover:text-gray-500 dark:hover:text-gray-400 flex items-center !text-xs"
              >
                <MaterialSymbol name="add" size="sm" />
                Add Connection
              </Link>
            </div>
          )}
            {yamlConfig.spec.connections?.length === 0 && totalNodesCount <= 1 ? (
              <div className="flex justify-center items-center h-full">
                <div className="flex flex-col items-center justify-center h-full space-y-4">
                  <Text className="text-zinc-500 dark:text-zinc-400">
                    No connections added
                  </Text>
                  <Button
                    onClick={handleAddConnection}
                    className="text-blue-600 hover:text-blue-700 flex items-center !text-xs"
                    plain
                  >
                    <MaterialSymbol name="add" size="sm" />
                    Add Connection
                  </Button>
                </div>
              </div>
            ) : (
              partialSave && (yamlConfig.spec.connections?.length !== undefined && yamlConfig.spec.connections?.length > 0) && (
                <>
                  <Divider/>
                  <Field className='flex justify-end'>
                    <Button
                      color='blue'
                      className='flex items-center !text-xs'
                      onClick={handleConnectionsSave}
                    >
                      <MaterialSymbol name="save" size="sm" />
                      Save
                    </Button>
                  </Field>
                </>
              )
            )}
            
          </div>
         
        </div>
      )
    },
    {
      id: 'inputs',
      title: (
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center">
            <span>Inputs</span>
            <ModificationIndicator sectionId="inputs" />
          </div>
          {yamlConfig.spec.inputs && yamlConfig.spec.inputs.length > 0 && (
            <span className="text-xs text-gray-600 dark:text-gray-400 text-code !font-normal pr-2">
              {yamlConfig.spec.inputs.length} input{yamlConfig.spec.inputs.length !== 1 ? 's' : ''}
            </span>
          )}
        </div>
      ),
      content: (
        <div className="space-y-4">
          
          {/* Inputs List */}
          <div className="space-y-2">
            {yamlConfig.spec.inputs?.map((input, index) => {
              const inputId = `input_${index}`;
              return (
                <div key={index} className="flex connection">
                  {savedInputs.has(index) ? (
                    // Read-only mode - entire input box is read-only
                    <div className="flex-auto space-y-1 border border-zinc-50 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/20 p-2 rounded-sm">
                      {/* Input name and description with edit button */}
                      <div className="flex items-center justify-between">
                        <div className="flex-1">
                          <h4 className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                            {input.name || 'Unnamed Input'}
                          </h4>
                          {input.description && (
                            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                              {input.description}
                            </p>
                          )}
                        </div>
                        <div className="flex items-center">
                          <Button
                            plain
                            className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
                            onClick={() => {
                              // Remove from saved inputs to make it editable again
                              setSavedInputs(prev => {
                                const newSaved = new Set(prev);
                                newSaved.delete(index);
                                return newSaved;
                              });
                              console.log('Input made editable:', input);
                            }}
                          >
                            <MaterialSymbol name="edit" size="sm" />
                          </Button>
                          <Button
                            plain
                            className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300"
                            onClick={() => {
                              // Remove input from the list
                              const newInputs = yamlConfig.spec.inputs?.filter((_, i) => i !== index) || []
                              setYamlConfig(prev => ({ 
                                ...prev, 
                                spec: { 
                                  ...prev.spec, 
                                  inputs: newInputs 
                                } 
                              }))
                              
                              markSectionModified('inputs');
                              
                              // Also remove from saved inputs
                              setSavedInputs(prev => {
                                const newSaved = new Set(prev);
                                newSaved.delete(index);
                                // Update indices for remaining inputs
                                const updatedSaved = new Set<number>();
                                newSaved.forEach(savedIndex => {
                                  if (savedIndex < index) {
                                    updatedSaved.add(savedIndex);
                                  } else if (savedIndex > index) {
                                    updatedSaved.add(savedIndex - 1);
                                  }
                                });
                                return updatedSaved;
                              });
                              
                              // Remove mappings for this input
                              setInputMappings(prev => {
                                const newMappings = { ...prev };
                                delete newMappings[inputId];
                                return newMappings;
                              });
                              
                              console.log('Input deleted:', input);
                            }}
                          >
                            <MaterialSymbol name="delete" size="sm" />
                          </Button>
                        </div>
                      </div>
                      
                      {/* Collapsible Mappings Display */}
                      {inputMappings[inputId] && inputMappings[inputId].length > 0 && (
                        <div className="mt-2 space-y-1">
                          {inputMappings[inputId].map((mapping) => (
                            <div key={mapping.id} className="flex items-center gap-2 text-xs text-zinc-600 dark:text-zinc-400">
                              <span className="px-2 py-1 bg-zinc-100 dark:bg-zinc-800 rounded">
                                {mapping.connection || 'No connection'}
                              </span>
                              <span>→</span>
                              <span className="px-2 py-1 bg-zinc-100 dark:bg-zinc-800 rounded">
                                {mapping.value || 'No value'}
                              </span>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ) : (
                    // Editable mode - input is editable inline
                    <div className="flex-auto space-y-3 border bg-zinc-50 dark:bg-zinc-900 border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 p-3 rounded-sm">
                      <div className="flex items-center justify-between">
                        <div className="flex-1 space-y-3">
                          {/* Input Name */}
                          <Field>
                            <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                              Name
                            </Label>
                            <Input
                              placeholder="Input name"
                              value={input.name}
                              onChange={(e) => {
                                const newInputs = [...(yamlConfig.spec.inputs || [])]
                                newInputs[index] = { ...input, name: e.target.value }
                                setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                                markSectionModified('inputs');
                              }}
                              onFocus={handleInputFocus}
                              className="w-full"
                            />
                          </Field>

                          {/* Input Description */}
                          <Field>
                            <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                              Description
                            </Label>
                            <Textarea
                              placeholder="Input description"
                              value={input.description || ''}
                              onChange={(e) => {
                                const newInputs = [...(yamlConfig.spec.inputs || [])]
                                newInputs[index] = { ...input, description: e.target.value }
                                setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                                markSectionModified('inputs');
                              }}
                              onFocus={handleInputFocus}
                              rows={2}
                              className="w-full"
                            />
                          </Field>

                          {/* Mappings Section */}
                          
                            
                            {/* Existing Mappings */}
                            {inputMappings[inputId] && inputMappings[inputId].length > 0 && (
                             <Field>
                              <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                                Mappings
                              </Label>
                                {inputMappings[inputId].map((mapping) => (
                                  <div key={mapping.id} className="flex items-center gap-2">
                                  <div className="flex w-full justify-between p-1 bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded gap-1">
                                  <div className='flex flex-auto items-center'>
                                    <Dropdown>
                                      <DropdownButton outline className="min-w-[100px] !justify-between flex-auto flex items-center justify-between text-xs">
                                        {mapping.connection || 'Connection'}
                                        <MaterialSymbol name="expand_more" size="sm" />
                                      </DropdownButton>
                                      <DropdownMenu anchor="bottom start">
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
                                    
                                    <Dropdown>
                                      <DropdownButton outline className="min-w-[100px] !justify-between flex-auto flex items-center justify-between text-xs">
                                        {mapping.value || 'Value'}
                                        <MaterialSymbol name="expand_more" size="sm" />
                                      </DropdownButton>
                                      <DropdownMenu anchor="bottom start">
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
                                    </div>
                                    <Button
                                      plain
                                      onClick={() => handleRemoveInputMapping(inputId, mapping.id)}
                                      className="text-red-600 hover:text-red-700"
                                    >
                                      <MaterialSymbol name="close" size="sm" />
                                    </Button>
                                  </div>
                                  </div>
                                ))}
                              </Field>
                             
                            )}

                            {/* Add Mapping Button */}
                            
                            <Link
                              href="#"
                              onClick={() => handleAddInputMapping(inputId)}

                              className="flex items-center !text-xs"
                            >
                              <MaterialSymbol name="add" size="sm" />
                              Add Mapping
                            </Link>
                            {/* Save Button - only show if saveGranular is true and input is not saved */}
                            {saveGranular && (
                              <div className='flex items-center justify-end w-full border-t border-zinc-200 dark:border-zinc-700 pt-2'>
                                  <Button
                                    plain
                                    className='flex items-center !text-xs'
                                    onClick={() => {
                                      setSavedInputs(prev => new Set([...prev, index]));
                                      console.log('Input cancelled:', input);
                                    }}
                                  >
                                    Cancel
                                  </Button>
                                  <Button
                                    color='blue'
                                    className='flex items-center !text-xs'
                                    onClick={() => {
                                      // Save this specific input and its mappings
                                      const formattedMappings: Record<string, string> = {};
                                      if (inputMappings[inputId]) {
                                        formattedMappings[inputId] = inputMappings[inputId].map(m => `${m.connection}:${m.value}`).join(',');
                                      }
                                      
                                      onUpdate?.({
                                        yamlConfig: {
                                          ...yamlConfig,
                                          spec: {
                                            ...yamlConfig.spec,
                                            inputs: yamlConfig.spec.inputs || [],
                                            inputMappings: { ...yamlConfig.spec.inputMappings, ...formattedMappings }
                                          }
                                        }
                                      });
                                      
                                      setSavedInputs(prev => new Set([...prev, index]));
                                      // Clear modification status when saving
                                      clearSectionModified('inputs');
                                      console.log('Input saved:', input);
                                    }}
                                  >
                                    <MaterialSymbol name="save" size="sm" />
                                    Save
                                  </Button>
                                </div>
                            )}
                          
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
          
          {yamlConfig.spec.inputs && yamlConfig.spec.inputs.length > 0 && partialSave && (
            <>
              <Divider/>
              <Field className='flex justify-end'>
                <Button
                  color='blue'
                  className='flex items-center !text-xs'
                  onClick={handleInputsSave}
                >
                  <MaterialSymbol name="save" size="sm" />
                  Save
                </Button>
              </Field>
            </>
          )}
          {/* Add Input Button */}
          <Link
            href='#'
            onClick={handleAddInput}
            className="text-gray-600 hover:text-gray-500 dark:hover:text-gray-400 flex items-center !text-xs"
          >
            <MaterialSymbol name="add" size="sm" />
            Add Input
          </Link>
        </div>
      )
    },
    {
      id: 'outputs',
      title: 'Outputs',
      content: (
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
          
          {yamlConfig.spec.outputs && yamlConfig.spec.outputs.length > 0 && partialSave && (
            <>
              <Divider/>
              <Field className='flex justify-end'>
                <Button
                  color='blue'
                  className='flex items-center !text-xs'
                  onClick={handleOutputsSave}
                >
                  <MaterialSymbol name="save" size="sm" />
                  Save
                </Button>
              </Field>
            </>
          )}
        </div>
      )
    },
    {
      id: 'secrets',
      title: 'Secrets Management',
      content: (
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <Field>
              <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                Secrets
              </Label>
            </Field>
            <Button
              onClick={() => setYamlConfig(prev => ({
                ...prev,
                spec: {
                  ...prev.spec,
                  secrets: [...(prev.spec.secrets || []), { name: '', key: '', value: '' }]
                }
              }))}
              className="text-blue-600 hover:text-blue-700"
              plain
            >
              <MaterialSymbol name="add" size="sm" />
              Add Secret
            </Button>
          </div>
          <div className="space-y-3">
            {yamlConfig.spec.secrets?.map((secret, index) => (
              <div key={index} className="grid grid-cols-3 gap-2 p-3 bg-zinc-50 dark:bg-zinc-900 rounded-lg">
                <Input
                  placeholder="Secret name"
                  value={secret.name}
                  onChange={(e) => {
                    const newSecrets = [...(yamlConfig.spec.secrets || [])]
                    newSecrets[index] = { ...secret, name: e.target.value }
                    setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, secrets: newSecrets } }))
                  }}
                />
                <Input
                  placeholder="Key"
                  value={secret.key}
                  onChange={(e) => {
                    const newSecrets = [...(yamlConfig.spec.secrets || [])]
                    newSecrets[index] = { ...secret, key: e.target.value }
                    setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, secrets: newSecrets } }))
                  }}
                />
                <div className="flex gap-1">
                  <Input
                    placeholder="Value"
                    value={secret.value}
                    onChange={(e) => {
                      const newSecrets = [...(yamlConfig.spec.secrets || [])]
                      newSecrets[index] = { ...secret, value: e.target.value }
                      setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, secrets: newSecrets } }))
                    }}
                  />
                  <Button
                    plain
                    onClick={() => {
                      const newSecrets = yamlConfig.spec.secrets?.filter((_, i) => i !== index) || []
                      setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, secrets: newSecrets } }))
                    }}
                    className="text-red-600 hover:text-red-700"
                  >
                    <MaterialSymbol name="delete" size="sm" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
          
          {yamlConfig.spec.secrets && yamlConfig.spec.secrets.length > 0 && partialSave && (
            <>
              <Divider/>
              <Field className='flex justify-end'>
                <Button
                  color='blue'
                  className='flex items-center !text-xs'
                  onClick={handleSecretsSave}
                >
                  <MaterialSymbol name="save" size="sm" />
                  Save
                </Button>
              </Field>
            </>
          )}
        </div>
      )
    },
    {
      id: 'executor',
      title: (
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center">
            <span>Executor Configuration</span>
            <ModificationIndicator sectionId="executor" />
          </div>
          {yamlConfig.spec.executor && yamlConfig.spec.executor.type !== 'default' && (
            <span className="text-xs text-gray-600 dark:text-gray-400 text-code !font-normal pr-2">
              {yamlConfig.spec.executor.type}
            </span>
          )}
        </div>
      ),
      content: (
        <div className="space-y-4">
          {/* Add Executor Button */}
          {yamlConfig.spec.executor && yamlConfig.spec.executor.type == 'default' && (
          <Link 
            href="#"
            onClick={handleAddExecutor}
            className="flex items-center text-xs"
          >
            <MaterialSymbol name="add" size="sm" />
            Add Executor
          </Link>
          )}
          {/* Executor Display */}
          {yamlConfig.spec.executor && yamlConfig.spec.executor.type !== 'default' && (
            <div className="space-y-2">
              <div className="flex">
                {savedExecutors.has(0) ? (
                  // Read-only mode - entire executor box is read-only
                  <div className="flex-auto space-y-1 border border-zinc-50 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/20 p-2 rounded-sm">
                    {/* Executor type and name with edit button */}
                    <div className="flex items-center justify-between">
                      <div className="flex-1">
                        <h4 className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                          {yamlConfig.spec.executor.type === 'github' ? 'GitHub' : yamlConfig.spec.executor.type === 'semaphore' ? 'Semaphore' : yamlConfig.spec.executor.type}
                        </h4>
                      </div>
                      <div className="flex items-center">
                        <Button
                          plain
                          className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
                          onClick={() => {
                            // Remove from saved executors to make it editable again
                            setSavedExecutors(prev => {
                              const newSaved = new Set(prev);
                              newSaved.delete(0);
                              return newSaved;
                            });
                            console.log('Executor made editable:', yamlConfig.spec.executor);
                          }}
                        >
                          <MaterialSymbol name="edit" size="sm" />
                        </Button>
                        <Button
                          plain
                          className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300"
                          onClick={() => {
                            // Remove executor
                            setYamlConfig(prev => ({ 
                              ...prev, 
                              spec: { 
                                ...prev.spec, 
                                executor: { type: 'default', config: {} }
                              } 
                            }))
                            
                            markSectionModified('executor');
                            
                            // Also remove from saved executors
                            setSavedExecutors(prev => {
                              const newSaved = new Set(prev);
                              newSaved.delete(0);
                              return newSaved;
                            });
                            
                            console.log('Executor deleted');
                          }}
                        >
                          <MaterialSymbol name="delete" size="sm" />
                        </Button>
                      </div>
                    </div>
                  </div>
                ) : (
                  // Editable mode - executor is editable inline
                  <div className="flex-auto space-y-3 bg-zinc-50 dark:bg-zinc-900/20 border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 p-3 rounded-sm">
                    <div className="flex items-center justify-between">
                      <div className="flex-1 space-y-3">
                        {/* Semaphore Integration */}
                        <Field>
                          <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                            Semaphore integration
                          </Label>
                          <Dropdown>
                            <DropdownButton color='white' className="w-full flex items-center !justify-between">
                              <span>
                                {yamlConfig.spec.executor?.config?.integration || 'Select Semaphore integration'}
                              </span>
                              <MaterialSymbol name="expand_more" size="sm" />
                            </DropdownButton>
                            <DropdownMenu anchor="bottom start">
                              <DropdownItem onClick={() => {
                                setYamlConfig(prev => ({ 
                                  ...prev, 
                                  spec: { 
                                    ...prev.spec, 
                                    executor: { 
                                      type: 'semaphore', 
                                      config: { 
                                        integration: "zawkey's semaphore org"
                                      } 
                                    }
                                  }
                                }))
                                markSectionModified('executor');
                              }}>
                                <DropdownLabel>zawkey's semaphore org</DropdownLabel>
                              </DropdownItem>
                              <DropdownItem onClick={() => {
                                setYamlConfig(prev => ({ 
                                  ...prev, 
                                  spec: { 
                                    ...prev.spec, 
                                    executor: { 
                                      type: 'semaphore', 
                                      config: { 
                                        integration: 'semaphore test org'
                                      } 
                                    }
                                  }
                                }))
                                markSectionModified('executor');
                              }}>
                                <DropdownLabel>semaphore test org</DropdownLabel>
                              </DropdownItem>
                              <DropdownItem onClick={() => {
                                setYamlConfig(prev => ({ 
                                  ...prev, 
                                  spec: { 
                                    ...prev.spec, 
                                    executor: { 
                                      type: 'semaphore', 
                                      config: { 
                                        integration: 'my organization'
                                      } 
                                    }
                                  }
                                }))
                                markSectionModified('executor');
                              }}>
                                <DropdownLabel>my organization</DropdownLabel>
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </Field>


                       

                        {/* Semaphore specific fields */}
                        
                        <Field>
                          <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                            Project Name
                          </Label>
                          <Input value={yamlConfig.spec.executor?.config?.projectName} onChange={(e) => {
                            setYamlConfig(prev => ({ 
                              ...prev, 
                              spec: { 
                                ...prev.spec, 
                                executor: { 
                                  type: 'semaphore', 
                                  config: { 
                                    integration: 'my organization',
                                    projectName: e.target.value 
                                  } 
                                }
                              }
                            }))
                            markSectionModified('executor');
                          }} />
                        </Field>
                        <Field>
                          <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                            Execution type
                          </Label>
                        
                          <ControlledTabs 
                            className='mt-3'
                            tabs={[
                              { id: 'workflow', label: 'Workflow' },
                              { id: 'task', label: 'Task' },
                            ]}
                            activeTab='workflow'
                            variant='pills'
                            onTabChange={(tabId) => {
                              console.log('Tab changed to:', tabId);
                            }}  
                          />  
                          
                        </Field>
                        <Field>
                          <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                            Branch
                          </Label>
                          <Input value={yamlConfig.spec.executor?.config?.branch} onChange={(e) => {
                            setYamlConfig(prev => ({ 
                              ...prev, 
                              spec: { 
                                ...prev.spec, 
                                executor: { 
                                  type: 'semaphore', 
                                  config: { 
                                    integration: 'my organization',
                                    branch: e.target.value 
                                  } 
                                }
                              }
                            }))
                            markSectionModified('executor');
                          }} />
                          
                        </Field>
                        <Field>
                          <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                            Pipeline
                          </Label>
                          <Input value={yamlConfig.spec.executor?.config?.pipeline} onChange={(e) => {
                            setYamlConfig(prev => ({ 
                              ...prev, 
                              spec: { 
                                ...prev.spec, 
                                executor: { 
                                  type: 'semaphore', 
                                  config: { 
                                    integration: 'my organization',
                                    branch: e.target.value 
                                  } 
                                }
                              }
                            }))
                            markSectionModified('executor');
                          }} />
                          
                        </Field>

                        {/* Parameters Section */}
                        <div className="space-y-3">
                          <div className="flex items-center justify-between">
                            <div className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                              Parameters
                            </div>
                            
                          </div>
                          
                          {/* Parameters List */}
                          {yamlConfig.spec.executor?.config?.parameters && yamlConfig.spec.executor.config.parameters.length > 0 && yamlConfig.spec.executor.config.parameters.map((param: any, index: number) => (
                               <div className="flex w-full justify-between p-1 bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded gap-1 space-y-2">
                                  <div key={index} className="flex items-center">
                                    <Input
                                      value={param.key}
                                      onChange={(e) => {
                                        const newParams = [...(yamlConfig.spec.executor?.config?.parameters || [])];
                                        newParams[index] = { ...param, key: e.target.value };
                                        setYamlConfig(prev => ({ 
                                          ...prev, 
                                          spec: { 
                                            ...prev.spec, 
                                            executor: { 
                                              ...prev.spec.executor,
                                              type: 'semaphore', 
                                              config: { 
                                                ...prev.spec.executor?.config,
                                                parameters: newParams
                                              } 
                                            }
                                          }
                                        }))
                                        markSectionModified('executor');
                                      }}
                                      placeholder="VERSION"
                                      className="flex-1 text-xs"
                                    />
                                      <Input
                                        value={param.value}
                                        onChange={(e) => {
                                          const newParams = [...(yamlConfig.spec.executor?.config?.parameters || [])];
                                          newParams[index] = { ...param, value: e.target.value };
                                          setYamlConfig(prev => ({ 
                                            ...prev, 
                                            spec: { 
                                              ...prev.spec, 
                                              executor: { 
                                                ...prev.spec.executor,
                                                type: 'semaphore', 
                                                config: { 
                                                  ...prev.spec.executor?.config,
                                                  parameters: newParams
                                                } 
                                              }
                                            }
                                          }))
                                          markSectionModified('executor');
                                        }}
                                        placeholder="VERSION"
                                        className="border-none bg-transparent p-0 text-xs focus:ring-0 focus:border-none flex-1 ml-2"
                                        style={{ boxShadow: 'none' }}
                                      />
                                        
                                    
                                    <Button
                                      plain
                                      onClick={() => {
                                        const newParams = yamlConfig.spec.executor?.config?.parameters?.filter((_: any, i: number) => i !== index) || [];
                                        setYamlConfig(prev => ({ 
                                          ...prev, 
                                          spec: { 
                                            ...prev.spec, 
                                            executor: { 
                                              ...prev.spec.executor,
                                              type: 'semaphore', 
                                              config: { 
                                                ...prev.spec.executor?.config,
                                                parameters: newParams
                                              } 
                                            }
                                          }
                                        }))
                                        markSectionModified('executor');
                                      }}
                                      className="p-1 text-gray-400 hover:text-gray-900 dark:hover:text-gray-400"
                                    >
                                      <MaterialSymbol name="close" size="sm" />
                                    </Button>
                                  </div>
                                 </div>
                          ))}
                           
                          
                          <Link
                              href="#"
                              onClick={() => {
                                const newParams = yamlConfig.spec.executor?.config?.parameters || [];
                                setYamlConfig(prev => ({ 
                                  ...prev, 
                                  spec: { 
                                    ...prev.spec, 
                                    executor: { 
                                      ...prev.spec.executor,
                                      type: 'semaphore', 
                                      config: { 
                                        ...prev.spec.executor?.config,
                                        parameters: [...newParams, { key: '', value: '' }]
                                      } 
                                    }
                                  }
                                }))
                                markSectionModified('executor');
                              }}
                              className="flex items-center text-xs"
                            >
                              <MaterialSymbol name="add" size="sm" className="mr-1" />
                              Add parameter
                            </Link>
                        </div>

                        {/* Save Button - only show if saveGranular is true */}
                        {saveGranular && (
                          <div className='flex items-center justify-end w-full border-t border-zinc-200 dark:border-zinc-700 pt-2'>
                            <Button
                              plain
                              className='flex items-center !text-xs'
                              onClick={() => {
                                setSavedExecutors(prev => new Set([...prev, 0]));
                                console.log('Executor cancelled');
                              }}
                            >
                              Cancel
                            </Button>
                            <Button
                              color='blue'
                              className='flex items-center !text-xs'
                              onClick={() => {
                                // Save executor
                                onUpdate?.({
                                  yamlConfig: {
                                    ...yamlConfig,
                                    spec: {
                                      ...yamlConfig.spec,
                                      executor: yamlConfig.spec.executor || { type: 'default', config: {} }
                                    }
                                  }
                                });
                                
                                setSavedExecutors(prev => new Set([...prev, 0]));
                                // Clear modification status when saving
                                clearSectionModified('executor');
                                console.log('Executor saved:', yamlConfig.spec.executor);
                              }}
                            >
                              <MaterialSymbol name="save" size="sm" />
                              Save
                            </Button>
                          </div>
                        )}
                      </div>
                      
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
          
          {partialSave && (
            <>
              <Divider/>
              <Field className='flex justify-end'>
                <Button
                  color='blue'
                  className='flex items-center !text-xs'
                  onClick={handleExecutorsSave}
                >
                  <MaterialSymbol name="save" size="sm" />
                  Save
                </Button>
              </Field>
            </>
          )}
        </div>
      )
    },
    {
      id: 'advanced',
      title: 'Advanced',
      content: (
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
      )
    }
  ]

  const handleAccordionToggle = (sectionId: string) => {
    if (multiple) {
      setOpenSections(prev => 
        prev.includes(sectionId) 
          ? prev.filter(id => id !== sectionId)
          : [...prev, sectionId]
      )
    } else {
      setOpenSections(prev => 
        prev.includes(sectionId) ? [] : [sectionId]
      )
    }
  }

  if (variant === 'edit') {
    const sections = customSections || defaultSections

    return (
    
      <div className={clsx(
        'bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-sm min-w-sm max-w-sm relative',
        className
      )}>
      {selected && (
        <div 
          className="action-buttons absolute -top-14 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white dark:bg-zinc-800 shadow-xs rounded-lg p-1 border border-gray-200 dark:border-zinc-600 z-50"
          onClick={(e) => e.stopPropagation()}
        >
         

            <Button
              type="button"
              plain
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="code" size="md"/>
              Code
            </Button>
            
          
            
            
            <Dropdown>
              <DropdownButton plain className='flex items-center gap-2 !pr-1'>
                <MaterialSymbol name="save" size="md"/> 
                Save
                <MaterialSymbol name="expand_more" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2'><DropdownLabel>Save & Commit</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><DropdownLabel>Save as Draft</DropdownLabel></DropdownItem>
               
              </DropdownMenu>
            </Dropdown>
           
            <Button
              type="button"
              plain
              onClick={onCancel}
              className="flex items-center gap-2"
            >
              
              Cancel
            </Button>
            <Tippy content="More options" placement="top">
            <Dropdown>
              <DropdownButton plain>
                <MaterialSymbol name="more_vert" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="play_arrow" size="md"/><DropdownLabel>Run</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="tune" size="md"/><DropdownLabel>Advanced configuration</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="menu_book" size="md"/><DropdownLabel>Documentation</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2 text-red-600 dark:text-red-200' color='red'><MaterialSymbol name="delete" size="md"/><DropdownLabel>Delete</DropdownLabel></DropdownItem>

              </DropdownMenu>
            </Dropdown>
          </Tippy>
         
          
          
        </div>
      )}
        <div className="node-header p-4 flex justify-between border-b border-gray-200 dark:border-zinc-700 align-start items-start">
          <div className="flex flex-col w-full">
            <div className="flex items-start gap-2">
              <div className='w-10 h-10 bg-zinc-100 dark:bg-zinc-700 rounded-lg flex items-center justify-center'>
                <img width={24} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt="" />
              </div>
              <div className="flex flex-col w-full">
                {/* Inline editable title */}
                {editingField === 'title' ? (
                  <div className="flex-1">
                    <Input
                      value={tempTitle}
                      onChange={(e) => setTempTitle(e.target.value)}
                      onKeyDown={(e) => handleKeyDown(e, 'title')}
                      onBlur={() => handleSaveInlineEdit('title')}
                      className="font-semibold text-gray-900 dark:text-white"
                      autoFocus
                    />
                  </div>
                ) : (
                  <div className="group relative">
                    <div className="flex items-center">
                      <h3 
                        className="font-semibold text-gray-900 dark:text-white cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700 px-2 py-1 rounded transition-colors mb-0"
                        onClick={() => handleStartEdit('title')}
                        title="Click to edit title"
                      >
                        {data.title}
                      </h3>
                      <FieldModificationIndicator field="title" />
                    </div>
                  </div>
                )}
            
                {/* Inline editable description */}
                {editingField === 'description' ? (
                  <div className="mt-2">
                    <Textarea
                      value={tempDescription}
                      onChange={(e) => setTempDescription(e.target.value)}
                      onKeyDown={(e) => handleKeyDown(e, 'description')}
                      onBlur={() => handleSaveInlineEdit('description')}
                      className="text-sm text-gray-600 dark:text-gray-400"
                      rows={2}
                      autoFocus
                    />
                  </div>
                ) : (
                  <div className="group relative w-full">
                    <div className="flex items-center">
                      <Subheading 
                        className='!font-normal cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700 px-2 py-1 rounded transition-colors'
                        onClick={() => handleStartEdit('description')}
                        title="Click to edit description"
                      >
                        {data.description || 'Click to add description'}
                      </Subheading>
                      <FieldModificationIndicator field="description" />
                    </div>
                    <div className="hidden absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
                      <MaterialSymbol 
                        name="edit" 
                        size="sm" 
                        className="absolute top-0 right-0 text-zinc-500 dark:text-zinc-400 bg-white dark:bg-zinc-800 rounded p-1 shadow-sm"
                      />
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
          <Badge color="zinc">Draft</Badge>
      </div>
        {/* Header */}
        <div className="hidden p-4 flex justify-between items-center border-b border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center gap-3">
            <img width={24} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt="" />

            <div>
              <h3 className="font-semibold text-zinc-900 dark:text-white">YAML Configuration</h3>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">Edit workflow stage configuration</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button 
              onClick={handleSave}
              className="bg-green-600 hover:bg-green-700 text-white"
            >
              <MaterialSymbol name="check" size="sm" className="mr-1" />
              Save
            </Button>
            <Button 
              plain 
              onClick={handleCancel}
              className="text-zinc-600 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300"
            >
              Cancel
            </Button>
          </div>
        </div>

        {/* Accordion Content */}
        <div className="">
          <ControlledAccordion
            items={sections}
            openItems={openSections}
            onToggle={handleAccordionToggle}
            multiple={multiple}
          />
        </div>
        
        {/* GitHub Authentication Modal */}
        <Dialog open={showGitHubModal} onClose={() => setShowGitHubModal(false)}>
          
          <DialogTitle className='flex items-center justify-between'>
            Connect to GitHub
            <Button plain onClick={() => setShowGitHubModal(false)} className="flex items-center gap-2">
              <MaterialSymbol name="close" size="sm" />
            </Button>
          </DialogTitle>
          <DialogBody>
            <div className="flex flex-col items-center space-y-4 py-6">
              
              <div className="text-center">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                  GitHub Authentication
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
                Connect your GitHub account to access your repositories and enable GitHub-based execution.
                </p>
                <Button 
                  type="button"
                  outline
                  onClick={handleGitHubLogin}
                  className="flex items-center w-full text-lg px-6 py-3"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="currentColor" viewBox="0 0 16 16">
                    <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8"/>
                  </svg>
                  Continue with GitHub
                </Button>
              </div>
            </div>
          </DialogBody>
         
        </Dialog>
        <Handle
          type="target"
          position={Position.Left}
          className="!w-1 !h-12 !bg-blue-500 dark:!bg-zinc-300 !border-none !border-white dark:!border-zinc-50 z-50 !rounded-md"
          aria-label="Input connection point"
        />
        <Handle
          type="source"
          position={Position.Right}
          className="!w-1 !h-12 !bg-blue-500 dark:!bg-zinc-300 !border-none !border-white dark:!border-zinc-50 z-50 !rounded-md"
          aria-label="Output connection point"
        />
      </div>  
    )
  }

  // Read variant - StageCard style
  const statusConfig = getStatusConfig(data.status || 'pending')
  
  return (
    <div 
      className={clsx(
        'bg-white dark:bg-zinc-800 rounded-lg border-2 relative transition-all duration-200 hover:shadow-lg min-w-[320px]',
        selected ? 'border-blue-600 dark:border-zinc-200 ring-2 ring-blue-200 dark:ring-white' : 'border-gray-200 dark:border-zinc-700',
        className
      )}
      style={{ width: 340, boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }}
      role="article"
      aria-label={`Workflow stage: ${data.title}`}
      onClick={onSelect}
    >
      {/* Action buttons when selected */}
      {selected && (
        <div 
          className="action-buttons absolute -top-14 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white dark:bg-zinc-800 shadow-xs rounded-lg p-1 border border-gray-200 dark:border-zinc-600 z-50"
          onClick={(e) => e.stopPropagation()}
        >
         

            <Button
              type="button"
              plain
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="play_arrow" size="md"/>
              Run
            </Button>
            <Button
              type="button"
              plain
              className="flex items-center gap-2"
              onClick={onEdit}
            >
              <MaterialSymbol name="edit" size="md"/>
              Edit
            </Button>
            <Button
              type="button"
              plain
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="pause" size="md"/>
              Freeze
            </Button>
            
            
            <Tippy content="More options" placement="top">
            <Dropdown>
              <DropdownButton plain>
                <MaterialSymbol name="more_vert" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="content_copy" size="md"/><DropdownLabel>Duplicate</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="menu_book" size="md"/><DropdownLabel>Documentation</DropdownLabel></DropdownItem>

                <DropdownItem className='flex items-center gap-2 text-red-600 dark:text-red-200' color='red'><MaterialSymbol name="delete" size="md"/><DropdownLabel>Delete</DropdownLabel></DropdownItem>

              </DropdownMenu>
            </Dropdown>
          </Tippy>
         
          
          
        </div>
      )}
      {/* Header section */}
      {!executorInHeader && (
        <div className="p-4 flex flex-col border-b border-gray-200 dark:border-zinc-700">
          <div className="flex items-center gap-3 mb-4">
           <div className='rounded-lg bg-zinc-100 dark:bg-white !p-2'>
            {data.icon == 'github' && <img width={24} src="/images/github-logo.svg" alt="" />}
            {data.icon == 'semaphore' && <img width={24} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt="" />}
            {data.icon == 'openAI' && <img width={24} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAAAAABXZoBIAAABEElEQVR4AbTJIWyDQACG0d+rWsxENQ6Jwi0YFLIWRTKFxecc6pIzJzF4QTKDT23lSby5BPUt6ZJe2i1ze/aJP/x/+ly5/z2PU7ne1rIYm4/tR9YDsNeaFqP3l9wFx6AJgPLylLN6rrpEcBaiQko6dR/YDq4659po55SL8D1ujI0MLGpbl/K84nr8SWbSCBj5lNrvWewQCy1wU0gZsDV+ABi603nHtI9sJ0KW9d/paCxBjwy6wawyQiwdg2VPiZeBY5S10j3XjJRNoxWMqn20DB4tKeeWTUWhDfqJsX9rSRl0gLUQe20McpCSSwWAUxdBgaek0rQ6lQEwFS/JZ1cNebWFrVaElInLlNmv0TNpYgIAMy6KDbFgKo8AAAAASUVORK5CYII=" alt="" />}
          </div>
            <div className='flex flex-col'> 
              <h3 className="font-semibold text-gray-900 dark:text-white">{data.title}</h3>
              <Link className='text-xs text-blue-500 dark:text-blue-400 hidden' href="#">semaphore-project/semaphore.yml</Link>
              
              
            </div>
          </div>

          {data.description && (
            <h4 className="text-xs text-gray-600 dark:text-zinc-300 mb-4">{data.description}</h4>
          )}
          { data.icon == 'semaphore' && (
            <div className='flex items-center gap-3 text-blue-600 dark:text-blue-300'>
              <BadgeButton color='zinc' href='#' className='!text-xs'><MaterialSymbol name="assignment" size="md"/> semaphore-project</BadgeButton>
              <BadgeButton color='zinc' href='#' className='!text-xs'><MaterialSymbol name="code" size="md"/> semaphore.yml</BadgeButton>
            </div>
          )}
          { data.icon == 'github' && (
          <div className='flex items-center gap-3 text-blue-600 dark:text-blue-300'>
            <BadgeButton color='zinc' href='#' className='!text-xs'><MaterialSymbol name="book" size="md"/> superplane</BadgeButton>
            <BadgeButton color='zinc' href='#' className='!text-xs'><MaterialSymbol name="code" size="md"/> terraform.yml</BadgeButton>
            <BadgeButton color='zinc' href='#' className='!text-xs'><MaterialSymbol name="graph_1" size="md"/> main</BadgeButton>
          </div>
          )}
        </div>
      )}
      {executorInHeader && (
        <div>
          <div className="p-4 flex flex-col border-b border-gray-200 dark:border-zinc-700">
            <div className="flex items-center gap-3">
              <div className='w-8 h-8 bg-zinc-100 dark:bg-zinc-700 rounded-lg flex items-center justify-center'>
              {data.icon == 'semaphore' && <img width={20} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt="" />}
              {data.icon == 'openAI' && <img width={20} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAAAAABXZoBIAAABEElEQVR4AbTJIWyDQACG0d+rWsxENQ6Jwi0YFLIWRTKFxecc6pIzJzF4QTKDT23lSby5BPUt6ZJe2i1ze/aJP/x/+ly5/z2PU7ne1rIYm4/tR9YDsNeaFqP3l9wFx6AJgPLylLN6rrpEcBaiQko6dR/YDq4659po55SL8D1ujI0MLGpbl/K84nr8SWbSCBj5lNrvWewQCy1wU0gZsDV+ABi603nHtI9sJ0KW9d/paCxBjwy6wawyQiwdg2VPiZeBY5S10j3XjJRNoxWMqn20DB4tKeeWTUWhDfqJsX9rSRl0gLUQe20McpCSSwWAUxdBgaek0rQ6lQEwFS/JZ1cNebWFrVaElInLlNmv0TNpYgIAMy6KDbFgKo8AAAAASUVORK5CYII=" alt="" />}

              </div>
              <div className='flex flex-col'> 
                <h3 className="font-semibold text-gray-900 dark:text-white">{data.title}</h3>
                
              </div>
            </div>
            <h4 className="text-xs text-gray-600 dark:text-zinc-300 mt-4">{data.description}</h4>
            
          </div>
          
        </div>
      )}
      

      {/* Status section */}
      <div className={clsx('p-4 border-b border-gray-200 dark:border-zinc-700', statusConfig.borderColor, statusConfig.bgColor)}>
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-bold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
            Last Run
          </span>
          <span className="text-xs text-gray-500 dark:text-zinc-300 flex items-center">
            <MaterialSymbol name="timer" size="sm" className="mr-1"/> 14s | 2h ago
          </span>
        </div>

        <div className="flex items-center mb-3">
          <div className={`flex items-center gap-2 ${consistentStatuses ? 'visible' : 'hidden'}`}>
            {data.status == 'success' && (
            <Badge color='green' className='!flex !items-center mr-2'>
                <MaterialSymbol name='check_circle' size='md'/>
              <span className='uppercase'>Passed</span>
            </Badge>
            )}
            {data.status == 'failed' && (
            <Badge color='red' className='!flex !items-center mr-2'>
                <MaterialSymbol name='cancel' size='md'/>
              <span className='uppercase'>Failed</span>
            </Badge>
            )}
            {data.status == 'running' && (
            <Badge color='blue' className='!flex !items-center mr-2'>
              <MaterialSymbol name='sync' size='md' className='animate-spin'/>
              <span className='uppercase'>Running</span>
            </Badge>
            )}
            
            
         
          </div>
          <div className={`flex items-center gap-2 ${consistentStatuses ? 'hidden' : 'visible'}`}>
          <Badge color={data.status == 'success' ? 'green' : data.status == 'failed' ? 'red' : 'blue'} className='!flex !items-center mr-2'>
            <MaterialSymbol 
              name={statusConfig.icon} 
              size='xl'
              className={statusConfig.iconColor}
            />
            </Badge>
          </div>
          <div className="flex-1 min-w-0">
            <div className="font-medium text-sm text-gray-900 dark:text-white truncate">
              {data.runName || '2348932urejhwejkhr2304958ru2ioefwjh20389ruie'}
            </div>
            
          </div>
        </div>
        {/* Compact trigger information */}
        <div className="flex items-center gap-3 mb-3 hidden">
              <div className="flex items-center gap-1 text-xs text-gray-600 dark:text-zinc-400">
                Triggered by 
                <span className="font-medium text-gray-700 dark:text-zinc-300 truncate max-w-24" title={data.triggeredBy}>
                  {data.triggeredBy || 'No trigger data'}
                </span>
              </div>
              <div className="flex items-center gap-1 text-xs text-gray-600 dark:text-zinc-400 hidden">
                Event ID
                <span className="font-mono text-gray-700 dark:text-zinc-300" title={data.eventId}>
                  {data.eventId?.slice(-8) || 'abc123def456'.slice(-8)}
                </span>
              </div>
            </div>
        {/* Executor and connection info */}
        <div className="flex gap-2">
          
          {yamlConfig.spec.inputs && yamlConfig.spec.inputs.length > 0 && (
            <Tippy content={renderInputsTooltip(false, yamlConfig.spec.inputs)} placement='top' interactive={true}>
              <BadgeButton className="text-xs flex-grow-1 whitespace-nowrap" href="#">
                <MaterialSymbol name="input" size="md"/>
                {yamlConfig.spec.inputs?.length} input{yamlConfig.spec.inputs?.length !== 1 ? 's' : ''}
              </BadgeButton>
            </Tippy>
          )}
          <Tippy content={renderEventChainTooltip(data)} placement='top' interactive={true}>
              <BadgeButton className="text-xs event-trigger min-w-0 flex-grow-0 truncate flex items-center" href="#">
                <MaterialSymbol name="bolt" size="md"/> 
                <span className="truncate flex-grow-1">Event {data.eventId || '423...'}</span>
              </BadgeButton>
            </Tippy>
          {yamlConfig.spec.connections && yamlConfig.spec.connections.length > 0 && (
            <Tippy content={renderInputsTooltip(true, yamlConfig.spec.outputs || [])} placement='top' interactive={true}>
              <BadgeButton className="text-xs flex-grow-1 whitespace-nowrap" href="#">
              <MaterialSymbol name="output" size="md"/>
                {yamlConfig.spec.outputs?.length} output {yamlConfig.spec.outputs?.length !== 1 ? 's' : ''}
               
              </BadgeButton>
            </Tippy>
          )}
        </div>
      </div>

      {/* Summary section */}
      <div className="p-4 dark:bg-zinc-800 rounded-b-lg">
        <h4 className="text-xs text-gray-700 font-bold dark:text-zinc-400 uppercase tracking-wide mb-2 flex items-center justify-between">
          NEXT IN QUEUE 
          <span className="text-xs text-gray-500 dark:text-zinc-300 font-normal normal-case">+{data.nodeNumber} more</span>
        </h4>
        <div className="space-y-2">
          {yamlConfig.spec.inputs && yamlConfig.spec.inputs.length > 0 && (
            <div className={data.queueIcon == 'how_to_reg' ? "flex items-center p-2 border bg-orange-50 dark:bg-orange-900/20 border-orange-400 dark:border-orange-800 rounded-sm justify-between" : "flex items-center p-2 border bg-zinc-50 dark:bg-zinc-700 border-gray-200 dark:border-gray-700 rounded-md gap-2 justify-between"}>
              <div className="flex items-center gap-2 truncate">
                { showIcons && (
                  <MaterialSymbol name="how_to_reg" size="lg" className='text-orange-600 dark:text-orange-400' />
                )}
                <div className={`flex items-center ${consistentStatuses ? 'hidden' : 'visible'}`}>
                  {data.queueIcon == 'how_to_reg' && (
                   <Badge color='amber' className='!text-xs'>
                    <MaterialSymbol name="how_to_reg" size="xl" className='text-orange-600 dark:text-orange-400 animate-pulse' />
                    </Badge>
                  
                  )}
                  {data.queueIcon == 'pause' && (
                  <Badge color='zinc' className='!text-xs'>
                    <MaterialSymbol name="pending" size="xl" className='text-gray-600 dark:text-gray-400 animate-pulse' />
                    
                  </Badge>
                  )}
                  {data.queueIcon == 'timer' && (
                  <Badge color='zinc' className='!text-xs'>
                    <MaterialSymbol name="schedule" size="xl" className='text-gray-600 dark:text-gray-400 animate-pulse' />
                    
                  </Badge>
                  )}
                </div>
                <div className={`flex items-center ${consistentStatuses ? 'visible' : 'hidden'}`}>
                  {data.queueIcon == 'how_to_reg' && (
                   <Badge color='amber' className='!text-xs'>
                    <MaterialSymbol name="how_to_reg" size="md" className='text-orange-600 dark:text-orange-400 animate-pulse' />
                    APPROVAL
                    </Badge>
                  
                  )}
                  {data.queueIcon != 'how_to_reg' && (
                  <Badge color='zinc' className='!text-xs'>
                    <MaterialSymbol name="pending" size="md" className='text-gray-600 dark:text-gray-400 animate-pulse' />
                    PENDING
                  </Badge>
                  )}
                  
                </div>
                <div className={`flex items-center gap-2 hidden`}>
                  <div className={`w-2 h-2 rounded-full flex-shrink-0 bg-orange-600 dark:bg-orange-500 animate-pulse`}></div>
                  <span className={`text-xs font-medium text-orange-700 dark:text-orange-500`}>
                    {data.queueIcon == 'how_to_reg' ? 'Action required' : 'Pending'}
                  </span>
                </div>
                <span className="text-sm text-gray-700 dark:text-gray-200 truncate font-medium">
                  {data.queueTitle}
                </span>
              </div>
              { !showIcons && data.queueIcon == 'how_to_reg' && (
               
                <div className='flex items-center'>
                <Button plain>
                <MaterialSymbol name="close" size="sm" className='text-black-700 dark:text-black-400' />
                </Button>
                <Button color='white'>
                <MaterialSymbol name="check" size="sm" className='text-black-700 dark:text-black-400' />
                </Button>
                </div>
              )}
              { !showIcons && data.queueIcon != 'how_to_reg' && (
                <div className='flex items-center hidden'>
                
                <MaterialSymbol name={data.queueIcon || 'how_to_reg'} size="lg" className='text-orange-700 dark:text-orange-600 px-2' />
                
                </div>
              )}
            </div>
          )}
          
         
        </div>
      </div>
      {(executorInHeader && (
          <div className='flex flex-col px-4 py-4 border-t border-gray-200 dark:border-zinc-700'>
             <h4 className="text-xs text-gray-700 font-bold dark:text-zinc-400 uppercase tracking-wide mb-2 flex items-center justify-between">
                EXECUTOR
              </h4>
          <div className='flex items-center gap-2 '>
          <BadgeButton color='zinc' href='#' className='!text-xs'><MaterialSymbol name="assignment" size="md"/> semaphore-project</BadgeButton>
          <BadgeButton color='zinc' href='#' className='!text-xs'><MaterialSymbol name="code" size="md"/> semaphore.yml</BadgeButton>
          </div>
          </div>
      ))}
     <Handle
        type="target"
        position={Position.Left}
        className="!w-1 !h-12 !bg-blue-500 dark:!bg-zinc-300 !border-none !border-white dark:!border-zinc-50 z-50 !rounded-md"
        aria-label="Input connection point"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="!w-1 !h-12 !bg-blue-500 dark:!bg-zinc-300 !border-none !border-white dark:!border-zinc-50 z-50 !rounded-md"
        aria-label="Output connection point"
      />
    </div>
  )
}
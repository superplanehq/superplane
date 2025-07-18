import { useState, useEffect, useCallback, useRef } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
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
  DialogDescription, 
  DialogBody, 
  DialogActions 
} from '../Dialog/dialog'
import { ControlledAccordion, type AccordionItem } from '../Accordion/accordion'
import { type WorkflowNodeData, type WorkflowNodeProps } from './workflow-node'
import clsx from 'clsx'
import { Subheading } from '../Heading/heading'
import { Divider } from '../Divider/divider'
import { Text } from '../Text/text'
import { Link } from '../Link/link'
import Tippy from '@tippyjs/react'
import { Badge } from '../Badge/badge'

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
                <Text className="text-zinc-500 dark:text-zinc-400">
                  Add event source or stage to connect to
                </Text>
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
                        {/* Executor Type */}
                        <Field>
                          <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                            Executor Type
                          </Label>
                          <Dropdown>
                            <DropdownButton color='white' className="w-full flex items-center !justify-between">
                              <span>{yamlConfig.spec.executor?.type || 'Select executor type'}</span>
                              <MaterialSymbol name="expand_more" size="sm" />
                            </DropdownButton>
                            <DropdownMenu anchor="bottom start">
                              <DropdownItem onClick={() => {
                                setYamlConfig(prev => ({ 
                                  ...prev, 
                                  spec: { 
                                    ...prev.spec, 
                                    executor: { type: 'semaphore', config: {} }
                                  }
                                }))
                                markSectionModified('executor');
                              }}>
                                <DropdownLabel>Semaphore</DropdownLabel>
                              </DropdownItem>
                              <DropdownItem onClick={() => {
                                setYamlConfig(prev => ({ 
                                  ...prev, 
                                  spec: { 
                                    ...prev.spec, 
                                    executor: { type: 'github', config: {} }
                                  }
                                }))
                                markSectionModified('executor');
                              }} className='flex items-center gap-2'>
                             
                                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="currentColor" viewBox="0 0 16 16">
                                    <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8"/>
                                  </svg>
                                  <DropdownLabel>GitHub</DropdownLabel>
                                
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </Field>


                        {/* GitHub specific fields */}
                        {yamlConfig.spec.executor?.type === 'github' && (
                          <Field>
                            <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                              GitHub Project
                            </Label>
                            {!isGitHubConnected ? (
                              <div className="space-y-2">
                                <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                                  Connect with GitHub to proceed
                                </Text>
                                <Link
                                  href='#'
                                  onClick={handleConnectGitHub}
                                  className="w-full flex items-center text-sm gap-2 text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300"
                                  
                                >
                                  <MaterialSymbol name="link" size="sm" />
                                  Connect with GitHub
                                </Link>
                              </div>
                            ) : (
                              <Dropdown>
                                <DropdownButton outline className="w-full flex items-center !justify-between">
                                  <span>
                                    {selectedGitHubProject ? 
                                      githubProjects.find(p => p.id === selectedGitHubProject)?.name : 
                                      'Select a project'
                                    }
                                  </span>
                                  <MaterialSymbol name="expand_more" size="sm" />
                                </DropdownButton>
                                <DropdownMenu anchor="bottom start">
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
                            )}
                          </Field>
                        )}

                        {/* Semaphore specific fields */}
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
                                        type: prev.spec.executor?.type || 'semaphore',
                                        config 
                                      }
                                    }
                                  }))
                                  markSectionModified('executor');
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
          className="action-buttons absolute -top-13 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white shadow-lg rounded-lg px-2 py-1 border border-gray-200 z-50"
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
            <Button
              type="button"
              plain
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="play_arrow" size="md"/>
              Run
            </Button>
          
            
            
            <Dropdown>
              <DropdownButton plain className='flex items-center gap-2'>
                <MaterialSymbol name="save" size="md"/>
                Save
                <MaterialSymbol name="expand_more" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2'><DropdownLabel>Save & Commit</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><DropdownLabel>Save as Draft</DropdownLabel></DropdownItem>
              </DropdownMenu>
            </Dropdown>
          <Tippy content="" placement="top">
            <Dropdown>
              <DropdownButton plain>
                <MaterialSymbol name="more_vert" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="tune" size="md"/><DropdownLabel>Advanced configuration</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="delete" size="md"/><DropdownLabel>Delete</DropdownLabel></DropdownItem>
              </DropdownMenu>
            </Dropdown>
          </Tippy>
          
          
        </div>
      )}
        <div className="node-header p-4 flex justify-between border-b border-gray-200 align-start items-start">
          <div className="flex flex-col w-full">
            <div className="flex items-center">
              <span className="material-symbols-outlined mr-2 text-gray-600 p-2 bg-zinc-100 dark:bg-zinc-700 rounded-xl">
                {getTypeIcon(data.type)}
              </span>
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
                      className="font-semibold text-gray-900 dark:text-white cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700 px-2 py-1 rounded transition-colors"
                      onClick={() => handleStartEdit('title')}
                      title="Click to edit title"
                    >
                      {data.title}
                    </h3>
                    <FieldModificationIndicator field="title" />
                  </div>
                </div>
              )}
            </div>
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
              <div className="group relative mt-2 w-full">
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
          <Badge color="zinc">Draft</Badge>
      </div>
        {/* Header */}
        <div className="hidden p-4 flex justify-between items-center border-b border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center gap-3">
            <MaterialSymbol 
              name={getTypeIcon(data.type)} 
              className={clsx('text-lg', getStatusColor(data.status))}
            />
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
      </div>
    )
  }

  // Read variant (same as original)
  return (
    <div className={clsx(
      'bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg p-4 shadow-sm min-w-[250px] cursor-pointer transition-all hover:shadow-md hover:border-zinc-300 dark:hover:border-zinc-600',
      className
    )}>
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <MaterialSymbol 
            name={getTypeIcon(data.type)} 
            className={clsx('text-lg', getStatusColor(data.status))}
          />
          <span className={clsx(
            'px-2 py-1 text-xs font-medium rounded-full',
            getTypeColor(data.type)
          )}>
            {data.type}
          </span>
        </div>
        <div className="flex items-center gap-1">
          {onEdit && (
            <Button 
              plain 
              onClick={onEdit}
              className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300"
            >
              <MaterialSymbol name="edit" size="sm" />
            </Button>
          )}
          {onDelete && (
            <Button 
              plain 
              onClick={onDelete}
              className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-500 hover:text-red-600 dark:text-zinc-400 dark:hover:text-red-400"
            >
              <MaterialSymbol name="delete" size="sm" />
            </Button>
          )}
        </div>
      </div>

      {/* Content */}
      <div>
        {/* Inline editable title */}
        {editingField === 'title' ? (
          <div className="mb-1">
            <Input
              value={tempTitle}
              onChange={(e) => setTempTitle(e.target.value)}
              onKeyDown={(e) => handleKeyDown(e, 'title')}
              onBlur={() => handleSaveInlineEdit('title')}
              className="font-semibold text-zinc-900 dark:text-white"
              autoFocus
            />
          </div>
        ) : (
          <div className="flex items-center mb-1">
            <h3 
              className="font-semibold text-zinc-900 dark:text-white cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700 px-2 py-1 rounded transition-colors"
              onClick={() => handleStartEdit('title')}
              title="Click to edit title"
            >
              {data.title}
            </h3>
            <FieldModificationIndicator field="title" />
          </div>
        )}
        
        {/* Inline editable description */}
        {editingField === 'description' ? (
          <div className="mb-1">
            <Textarea
              value={tempDescription}
              onChange={(e) => setTempDescription(e.target.value)}
              onKeyDown={(e) => handleKeyDown(e, 'description')}
              onBlur={() => handleSaveInlineEdit('description')}
              className="text-sm text-zinc-600 dark:text-zinc-400"
              rows={2}
              autoFocus
            />
          </div>
        ) : (
          <div className="flex items-center">
            <p 
              className="text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700 px-2 py-1 rounded transition-colors"
              onClick={() => handleStartEdit('description')}
              title="Click to edit description"
            >
              {data.description || 'Click to add description'}
            </p>
            <FieldModificationIndicator field="description" />
          </div>
        )}
      </div>

      {/* Status indicator */}
      {data.status && data.status !== 'pending' && (
        <div className="mt-3 flex items-center gap-2">
          <div className={clsx(
            'w-2 h-2 rounded-full',
            data.status === 'running' && 'bg-blue-500 animate-pulse',
            data.status === 'success' && 'bg-green-500',
            data.status === 'error' && 'bg-red-500',
            data.status === 'disabled' && 'bg-zinc-400'
          )} />
          <span className={clsx('text-xs font-medium capitalize', getStatusColor(data.status))}>
            {data.status}
          </span>
        </div>
      )}
      
    </div>
  )
}
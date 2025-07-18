import { useState } from 'react'
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
import { ControlledTabs, type Tab } from '../Tabs/tabs'
import clsx from 'clsx'
import { Subheading } from '../Heading/heading'

export interface WorkflowNodeData {
  id: string
  title: string
  description?: string
  type: 'stage' | 'event'
  status?: 'pending' | 'running' | 'success' | 'error' | 'disabled'
  config?: Record<string, any>
  // YAML configuration properties
  yamlConfig?: {
    apiVersion: string
    kind: string
    metadata: {
      name: string
      canvasId: string
    }
    spec: {
      secrets?: Array<{
        name: string
        key: string
        value: string
      }>
      connections?: Array<{
        name: string
        type: string
        config: Record<string, any>
      }>
      inputs?: Array<{
        name: string
        type: string
        required?: boolean
        defaultValue?: any
        description?: string
      }>
      inputMappings?: Record<string, string>
      outputs?: Array<{
        name: string
        type: string
        value?: any
        description?: string
        required?: boolean
      }>
      executor?: {
        type: string
        config?: Record<string, any>
      }
    }
  }
}

export interface WorkflowNodeProps {
  data: WorkflowNodeData
  variant?: 'read' | 'edit'
  selected: boolean
  className?: string
  tabs?: Tab[]
  onUpdate?: (data: Partial<WorkflowNodeData>) => void
  onDelete?: () => void
  onEdit?: () => void
  onSave?: () => void
  onCancel?: () => void
}

export function WorkflowNode({
  data,
  variant = 'read',
  className,
  tabs: customTabs,
  onUpdate,
  onDelete,
  onEdit,
  onSave,
  onCancel
}: WorkflowNodeProps) {
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
  
  // Default tab configuration
  const defaultTabs: Tab[] = [
    { id: 'basic', label: 'Basic' },
    { id: 'secrets', label: 'Secrets' },
    { id: 'connections', label: 'Connections' },
    { id: 'inputs', label: 'Inputs' },
    { id: 'outputs', label: 'Outputs' },
    { id: 'executor', label: 'Executor' },
    { id: 'preview', label: 'Preview' }
  ]
  
  // Use custom tabs if provided, otherwise use default tabs
  const tabs = customTabs || defaultTabs
  
  // Individual form states for easier management
  const [activeTab, setActiveTab] = useState<string>(tabs[0]?.id || 'basic')
  
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
    
    // Simple YAML-like string generation (for preview purposes)
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

  if (variant === 'edit') {
    return (
      <div className={clsx(
        'bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-sm min-w-[600px] max-w-[800px]',
        className
      )}>
        {/* Action buttons when selected */}
       

      

      {/* Header section */}
      <div className="p-4 flex justify-between items-center border-b border-gray-200 justify-between">
        <div className='flex flex-col'>
          <div className="flex items-center">
            <span className="material-symbols-outlined mr-2 text-gray-600">
              {getTypeIcon(data.type)}
            </span>
            <h3 className="font-semibold text-gray-900">{data.title}</h3>
          </div>
          <Subheading>{data.description}</Subheading>
        </div>
        <div className="flex items-center gap-2 hidden">
          <Button 
            onClick={handleSave}
            className="bg-green-600 hover:bg-green-700 text-white flex items-center"
          >
          <MaterialSymbol name="save" size="sm" />
          </Button>
       
        </div>
        
      </div>
       

        {/* Tab Navigation */}
        <ControlledTabs
          tabs={tabs}
          activeTab={activeTab}
          onTabChange={setActiveTab}
          variant="underline"
        />

        {/* Tab Content */}
        <div className="p-4 max-h-96 overflow-y-auto">
          {activeTab === 'basic' && (
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
          )}

          {activeTab === 'secrets' && (
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
            </div>
          )}

          {activeTab === 'executor' && (
            <div className="space-y-4">
              <Field>
                <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  Executor Type
                </Label>
                <Input
                  type="text"
                  value={yamlConfig.spec.executor?.type || ''}
                  onChange={(e) => setYamlConfig(prev => ({ 
                    ...prev, 
                    spec: { 
                      ...prev.spec, 
                      executor: { 
                        type: e.target.value,
                        config: prev.spec.executor?.config || {}
                      }
                    }
                  }))}
                  placeholder="kubernetes"
                  className="w-full"
                />
              </Field>
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
                            type: prev.spec.executor?.type || 'default',
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
            </div>
          )}

          {activeTab === 'connections' && (
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <Field>
                  <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                    Connections
                  </Label>
                </Field>
                <Button
                  onClick={() => setYamlConfig(prev => ({
                    ...prev,
                    spec: {
                      ...prev.spec,
                      connections: [...(prev.spec.connections || []), { name: '', type: '', config: {} }]
                    }
                  }))}
                  className="text-blue-600 hover:text-blue-700"
                  plain
                >
                  <MaterialSymbol name="add" size="sm" />
                  Add Connection
                </Button>
              </div>
              <div className="space-y-3">
                {yamlConfig.spec.connections?.map((connection, index) => (
                  <div key={index} className="p-3 bg-zinc-50 dark:bg-zinc-900 rounded-lg space-y-2">
                    <div className="grid grid-cols-2 gap-2">
                      <Input
                        placeholder="Connection name"
                        value={connection.name}
                        onChange={(e) => {
                          const newConnections = [...(yamlConfig.spec.connections || [])]
                          newConnections[index] = { ...connection, name: e.target.value }
                          setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                        }}
                      />
                      <Input
                        placeholder="Type (e.g., database, api)"
                        value={connection.type}
                        onChange={(e) => {
                          const newConnections = [...(yamlConfig.spec.connections || [])]
                          newConnections[index] = { ...connection, type: e.target.value }
                          setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                        }}
                      />
                    </div>
                    <div className="flex gap-2">
                      <Textarea
                        placeholder="Configuration (JSON)"
                        value={JSON.stringify(connection.config, null, 2)}
                        onChange={(e) => {
                          try {
                            const config = JSON.parse(e.target.value)
                            const newConnections = [...(yamlConfig.spec.connections || [])]
                            newConnections[index] = { ...connection, config }
                            setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                          } catch (err) {
                            // Invalid JSON, don't update
                          }
                        }}
                        rows={3}
                        className="flex-1 font-mono text-sm"
                      />
                      <Button
                        plain
                        onClick={() => {
                          const newConnections = yamlConfig.spec.connections?.filter((_, i) => i !== index) || []
                          setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, connections: newConnections } }))
                        }}
                        className="text-red-600 hover:text-red-700 self-start"
                      >
                        <MaterialSymbol name="delete" size="sm" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {activeTab === 'inputs' && (
            <div className="space-y-4">
              <div className="flex justify-between items-center">
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
              <div className="space-y-3">
                {yamlConfig.spec.inputs?.map((input, index) => (
                  <div key={index} className="grid grid-cols-4 gap-2 p-3 bg-zinc-50 dark:bg-zinc-900 rounded-lg">
                    <Input
                      placeholder="Input name"
                      value={input.name}
                      onChange={(e) => {
                        const newInputs = [...(yamlConfig.spec.inputs || [])]
                        newInputs[index] = { ...input, name: e.target.value }
                        setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                      }}
                    />
                    <Dropdown>
                      <DropdownButton outline className="text-sm w-full justify-between">
                        {input.type}
                        <MaterialSymbol name="keyboard_arrow_down" />
                      </DropdownButton>
                      <DropdownMenu>
                        {['string', 'number', 'boolean', 'object', 'array'].map(type => (
                          <DropdownItem key={type} onClick={() => {
                            const newInputs = [...(yamlConfig.spec.inputs || [])]
                            newInputs[index] = { ...input, type }
                            setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                          }}>
                            <DropdownLabel>{type}</DropdownLabel>
                          </DropdownItem>
                        ))}
                      </DropdownMenu>
                    </Dropdown>
                    <Input
                      placeholder="Default value"
                      value={input.defaultValue || ''}
                      onChange={(e) => {
                        const newInputs = [...(yamlConfig.spec.inputs || [])]
                        newInputs[index] = { ...input, defaultValue: e.target.value }
                        setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                      }}
                    />
                    <div className="flex items-center gap-1">
                      <label className="flex items-center gap-1 text-sm">
                        <input
                          type="checkbox"
                          checked={input.required || false}
                          onChange={(e) => {
                            const newInputs = [...(yamlConfig.spec.inputs || [])]
                            newInputs[index] = { ...input, required: e.target.checked }
                            setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                          }}
                          className="rounded"
                        />
                        Required
                      </label>
                      <Button
                        plain
                        onClick={() => {
                          const newInputs = yamlConfig.spec.inputs?.filter((_, i) => i !== index) || []
                          setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputs: newInputs } }))
                        }}
                        className="text-red-600 hover:text-red-700 ml-2"
                      >
                        <MaterialSymbol name="delete" size="sm" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
              
              {/* Input Mappings */}
              <div className="mt-6">
                <Field>
                  <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2 block">
                    Input Mappings
                  </Label>
                  <Textarea
                    value={JSON.stringify(yamlConfig.spec.inputMappings || {}, null, 2)}
                    onChange={(e) => {
                      try {
                        const inputMappings = JSON.parse(e.target.value)
                        setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, inputMappings } }))
                      } catch (err) {
                        // Invalid JSON, don't update
                      }
                    }}
                    placeholder='{\n  "inputName": "mappedValue"\n}'
                    rows={4}
                    className="w-full font-mono text-sm"
                  />
                </Field>
              </div>
            </div>
          )}

          {activeTab === 'outputs' && (
            <div className="space-y-4">
              <div className="flex justify-between items-center">
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
                      outputs: [...(prev.spec.outputs || []), { name: '', type: 'string' }]
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
                  <div key={index} className="grid grid-cols-3 gap-2 p-3 bg-zinc-50 dark:bg-zinc-900 rounded-lg">
                    <Input
                      placeholder="Output name"
                      value={output.name}
                      onChange={(e) => {
                        const newOutputs = [...(yamlConfig.spec.outputs || [])]
                        newOutputs[index] = { ...output, name: e.target.value }
                        setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, outputs: newOutputs } }))
                      }}
                    />
                    <Dropdown>
                      <DropdownButton outline className="text-sm w-full justify-between">
                        {output.type}
                        <MaterialSymbol name="keyboard_arrow_down" />
                      </DropdownButton>
                      <DropdownMenu>
                        {['string', 'number', 'boolean', 'object', 'array'].map(type => (
                          <DropdownItem key={type} onClick={() => {
                            const newOutputs = [...(yamlConfig.spec.outputs || [])]
                            newOutputs[index] = { ...output, type }
                            setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, outputs: newOutputs } }))
                          }}>
                            <DropdownLabel>{type}</DropdownLabel>
                          </DropdownItem>
                        ))}
                      </DropdownMenu>
                    </Dropdown>
                    <div className="flex gap-1">
                      <Input
                        placeholder="Value/Expression"
                        value={output.value || ''}
                        onChange={(e) => {
                          const newOutputs = [...(yamlConfig.spec.outputs || [])]
                          newOutputs[index] = { ...output, value: e.target.value }
                          setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, outputs: newOutputs } }))
                        }}
                      />
                      <Button
                        plain
                        onClick={() => {
                          const newOutputs = yamlConfig.spec.outputs?.filter((_, i) => i !== index) || []
                          setYamlConfig(prev => ({ ...prev, spec: { ...prev.spec, outputs: newOutputs } }))
                        }}
                        className="text-red-600 hover:text-red-700"
                      >
                        <MaterialSymbol name="delete" size="sm" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {activeTab === 'preview' && (
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <Field>
                  <Label className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                    YAML Preview
                  </Label>
                </Field>
                <Button
                  onClick={() => {
                    navigator.clipboard.writeText(generateYamlPreview())
                  }}
                  className="text-blue-600 hover:text-blue-700"
                  plain
                >
                  <MaterialSymbol name="content_copy" size="sm" />
                  Copy YAML
                </Button>
              </div>
              <div className="bg-zinc-50 dark:bg-zinc-900 rounded-lg p-4">
                <pre className="text-sm font-mono text-zinc-800 dark:text-zinc-200 whitespace-pre-wrap">
                  {generateYamlPreview()}
                </pre>
              </div>
              <div className="text-xs text-zinc-500 dark:text-zinc-400">
                This is a simplified YAML preview. The actual YAML output may have different formatting.
              </div>
            </div>
          )}
        </div>
      </div>
    )
  }

  // Read variant
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
        <h3 className="font-semibold text-zinc-900 dark:text-white mb-1">
          {data.title}
        </h3>
        {data.description && (
          <p className="text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2">
            {data.description}
          </p>
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
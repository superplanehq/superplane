import { useState, useCallback, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Node,
  Edge,
  addEdge,
  Connection,
  applyNodeChanges,
  applyEdgeChanges,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useBlueprint, useUpdateBlueprint } from '../../hooks/useBlueprintData'
import { useComponents } from '../../hooks/useBlueprintData'
import { Button } from '../../components/ui/button'
import { AlertCircle, Plus, Trash2 } from 'lucide-react'
import { BuildingBlock } from '../../ui/BuildingBlocksSidebar'
import { BlueprintBuilderPage } from '../../ui/BlueprintBuilderPage'
import type { BreadcrumbItem } from '../../ui/BlueprintBuilderPage'
import { BlockData } from '../../ui/CanvasPage/Block'
import { Heading } from '../../components/Heading/heading'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../components/ui/select'
import { ComponentsComponent } from '../../api-client'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from '../../components/ui/dialog'
import { VisuallyHidden } from '../../components/ui/visually-hidden'
import { showSuccessToast, showErrorToast } from '../../utils/toast'
import { filterVisibleConfiguration } from '../../utils/components'
import { ConfigurationFieldRenderer } from '@/ui/configurationFieldRenderer'
import { ScrollArea } from '../../components/ui/scroll-area'
import ELK from 'elkjs/lib/elk.bundled.js'

const elk = new ELK()

const getLayoutedElements = async (nodes: Node[], edges: Edge[]) => {
  const graph = {
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': 'RIGHT',
      'elk.spacing.nodeNode': '80',
      'elk.layered.spacing.nodeNodeBetweenLayers': '100',
    },
    children: nodes.map((node) => ({
      id: node.id,
      width: 180,
      height: 100,
    })),
    edges: edges.map((edge) => ({
      id: edge.id,
      sources: [edge.source],
      targets: [edge.target],
    })),
  }

  const layoutedGraph = await elk.layout(graph)

  const layoutedNodes = nodes.map((node) => {
    const layoutedNode = layoutedGraph.children?.find((n) => n.id === node.id)
    return {
      ...node,
      position: {
        x: layoutedNode?.x ?? 0,
        y: layoutedNode?.y ?? 0,
      },
    }
  })

  return { nodes: layoutedNodes, edges }
}

export const Blueprint = () => {
  const { organizationId, blueprintId } = useParams<{ organizationId: string; blueprintId: string }>()
  const navigate = useNavigate()
  const [isAddNodeModalOpen, setIsAddNodeModalOpen] = useState(false)
  const [selectedComponent, setSelectedComponent] = useState<any>(null)
  const [nodeName, setNodeName] = useState('')
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, any>>({})
  const [editingNodeId, setEditingNodeId] = useState<string | null>(null)
  const [blueprintConfiguration, setBlueprintConfiguration] = useState<any[]>([])
  const [isEditConfigFieldModalOpen, setIsEditConfigFieldModalOpen] = useState(false)
  const [editingConfigFieldIndex, setEditingConfigFieldIndex] = useState<number | null>(null)
  const [configFieldForm, setConfigFieldForm] = useState<any>({})
  const [blueprintOutputChannels, setBlueprintOutputChannels] = useState<any[]>([])
  const [isEditOutputChannelModalOpen, setIsEditOutputChannelModalOpen] = useState(false)
  const [editingOutputChannelIndex, setEditingOutputChannelIndex] = useState<number | null>(null)
  const [outputChannelForm, setOutputChannelForm] = useState<any>({})
  const [blueprintName, setBlueprintName] = useState('')
  const [blueprintDescription, setBlueprintDescription] = useState('')
  const [blueprintIcon, setBlueprintIcon] = useState('')
  const [blueprintColor, setBlueprintColor] = useState('')

  // Handler for metadata changes
  const handleMetadataChange = useCallback((metadata: any) => {
    setBlueprintName(metadata.name)
    setBlueprintDescription(metadata.description)
    setBlueprintIcon(metadata.icon)
    setBlueprintColor(metadata.color)
  }, [])

  // Fetch blueprint and components
  const { data: blueprint, isLoading: blueprintLoading } = useBlueprint(organizationId!, blueprintId!)
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!)
  const updateBlueprintMutation = useUpdateBlueprint(organizationId!, blueprintId!)

  const [nodes, setNodes] = useState<Node[]>([])
  const [edges, setEdges] = useState<Edge[]>([])

  // Update blueprint configuration and output channels when blueprint loads
  useEffect(() => {
    if (blueprint) {
      if (blueprint.configuration) {
        setBlueprintConfiguration(blueprint.configuration)
      }
      if (blueprint.outputChannels) {
        setBlueprintOutputChannels(blueprint.outputChannels)
      }
      setBlueprintName(blueprint.name || '')
      setBlueprintDescription(blueprint.description || '')
      setBlueprintIcon(blueprint.icon || '')
      setBlueprintColor(blueprint.color || '')
    }
  }, [blueprint])

  // Helper function to map component type to block type
  const getBlockType = (componentName: string): BlockData['type'] => {
    const typeMap: Record<string, BlockData['type']> = {
      'if': 'if',
      'filter': 'filter',
      'approval': 'approval',
      'noop': 'noop',
    }
    return typeMap[componentName] || 'noop' // Default to noop for unknown components
  }

  // Helper function to create minimal BlockData for a component
  const createBlockData = (node: any, component: ComponentsComponent | undefined): BlockData => {
    const componentName = node.component?.name || ''
    const blockType = getBlockType(componentName)
    const channels = component?.outputChannels?.map((channel: any) => channel.name) || ['default']

    const baseData: BlockData = {
      label: node.name,
      state: 'pending',
      type: blockType,
      outputChannels: channels,
    }

    // Add type-specific props based on component type
    switch (blockType) {
      case 'if':
        baseData.if = {
          title: node.name,
          conditions: [],
          collapsed: false,
        }
        break
      case 'filter':
        baseData.filter = {
          title: node.name,
          filters: [],
          collapsed: false,
        }
        break
      case 'approval':
        baseData.approval = {
          title: node.name,
          description: component?.description,
          iconSlug: component?.icon,
          iconColor: 'text-orange-500',
          headerColor: 'bg-orange-100',
          collapsedBackground: 'bg-orange-100',
          approvals: [],
          collapsed: false,
        }
        break
      case 'noop':
        baseData.noop = {
          title: node.name,
          collapsed: false,
        }
        break
    }

    return baseData
  }

  // Update nodes and edges when blueprint or components data changes
  useEffect(() => {
    if (!blueprint || components.length === 0) return

    const allNodes: Node[] = (blueprint.nodes || []).map((node: any) => {
      // Handle output channel nodes - skip for now as Block component doesn't support them
      if (node.type === 'TYPE_OUTPUT_CHANNEL') {
        return null
      }

      // Handle component nodes
      const component = components.find((p: any) => p.name === node.component?.name)
      const blockData = createBlockData(node, component)

      return {
        id: node.id,
        type: 'default', // BlueprintBuilderPage uses 'default' type for all nodes
        data: {
          ...blockData,
          // Store original data for serialization
          _originalComponent: node.component?.name,
          _originalConfiguration: node.configuration || {},
        },
        position: node.position || { x: 0, y: 0 },
      }
    }).filter(Boolean) as Node[]

    const loadedEdges: Edge[] = (blueprint.edges || []).map((edge: any, index: number) => ({
      id: `e${index}`,
      source: edge.sourceId,
      sourceHandle: edge.channel || 'default',
      target: edge.targetId,
      style: { strokeWidth: 3, stroke: '#C9D5E1' },
    }))

    // Check if we have saved positions
    const hasPositions = allNodes.some(node => node.position && (node.position.x !== 0 || node.position.y !== 0))

    if (hasPositions) {
      // Use saved positions
      setNodes(allNodes)
      setEdges(loadedEdges)
    } else {
      // Apply elk layout for blueprints without saved positions
      getLayoutedElements(allNodes, loadedEdges).then(({ nodes: layoutedNodes, edges: layoutedEdges }) => {
        setNodes(layoutedNodes)
        setEdges(layoutedEdges)
      })
    }
  }, [blueprint, components])

  // Node and edge change handlers
  const onNodesChange = useCallback((changes: any) => {
    setNodes((nds) => applyNodeChanges(changes, nds))
  }, [])

  const onEdgesChange = useCallback((changes: any) => {
    setEdges((eds) => applyEdgeChanges(changes, eds))
  }, [])

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, style: { strokeWidth: 3, stroke: '#C9D5E1' } }, eds))
    },
    []
  )

  const handleBlockClick = (block: BuildingBlock) => {
    // Find the full component data from the components array
    const component = components.find((c: ComponentsComponent) => c.name === block.name)
    if (!component) return

    setSelectedComponent(component)
    setNodeName(block.label || block.name)
    setNodeConfiguration({})
    setIsAddNodeModalOpen(true)
  }

  const generateNodeId = (componentName: string, nodeName: string) => {
    const randomChars = Math.random().toString(36).substring(2, 8)
    const sanitizedComponent = componentName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    return `${sanitizedComponent}-${sanitizedName}-${randomChars}`
  }

  const handleNodeEdit = useCallback((nodeId: string) => {
    const node = nodes.find((n) => n.id === nodeId)
    if (!node) return

    const component = components.find((p: any) => p.name === (node.data as any)._originalComponent)
    if (!component) return

    setEditingNodeId(node.id)
    setSelectedComponent(component)
    setNodeName((node.data as any).label as string)
    setNodeConfiguration((node.data as any)._originalConfiguration || {})
    setIsAddNodeModalOpen(true)
  }, [nodes, components])

  const handleNodeDelete = useCallback((nodeId: string) => {
    setNodes((nds) => nds.filter((n) => n.id !== nodeId))
    setEdges((eds) => eds.filter((e) => e.source !== nodeId && e.target !== nodeId))
  }, [])

  const handleAddNode = () => {
    if (!selectedComponent || !nodeName.trim()) return

    // Filter configuration to only include visible fields
    const filteredConfiguration = filterVisibleConfiguration(
      nodeConfiguration,
      selectedComponent.configuration || []
    )

    if (editingNodeId) {
      // Update existing node
      setNodes((nds) =>
        nds.map((node) => {
          if (node.id !== editingNodeId) return node

          const nodeData = node.data as any
          const updatedData = {
            ...nodeData,
            label: nodeName.trim(),
            _originalConfiguration: filteredConfiguration,
          }

          // Update the title in the type-specific props
          if (nodeData.if) {
            updatedData.if = { ...nodeData.if, title: nodeName.trim() }
          }
          if (nodeData.filter) {
            updatedData.filter = { ...nodeData.filter, title: nodeName.trim() }
          }
          if (nodeData.approval) {
            updatedData.approval = { ...nodeData.approval, title: nodeName.trim() }
          }
          if (nodeData.noop) {
            updatedData.noop = { ...nodeData.noop, title: nodeName.trim() }
          }

          return {
            ...node,
            data: updatedData,
          }
        })
      )
    } else {
      // Add new node
      const newNodeId = generateNodeId(selectedComponent.name, nodeName.trim())
      const mockNode = { component: { name: selectedComponent.name }, name: nodeName.trim() }
      const blockData = createBlockData(mockNode, selectedComponent)

      const newNode: Node = {
        id: newNodeId,
        type: 'default',
        position: { x: nodes.length * 250, y: 100 },
        data: {
          ...blockData,
          _originalComponent: selectedComponent.name,
          _originalConfiguration: filteredConfiguration,
        },
      }
      setNodes((nds) => [...nds, newNode])
    }

    setIsAddNodeModalOpen(false)
    setSelectedComponent(null)
    setNodeName('')
    setNodeConfiguration({})
    setEditingNodeId(null)
  }

  const handleCloseModal = () => {
    setIsAddNodeModalOpen(false)
    setSelectedComponent(null)
    setNodeName('')
    setNodeConfiguration({})
    setEditingNodeId(null)
  }

  const handleOpenConfigFieldModal = (index?: number) => {
    if (index !== undefined) {
      setEditingConfigFieldIndex(index)
      setConfigFieldForm(blueprintConfiguration[index])
    } else {
      setEditingConfigFieldIndex(null)
      setConfigFieldForm({
        name: '',
        label: '',
        type: 'string',
        description: '',
        required: false,
        typeOptions: {},
      })
    }
    setIsEditConfigFieldModalOpen(true)
  }

  const handleCloseConfigFieldModal = () => {
    setIsEditConfigFieldModalOpen(false)
    setEditingConfigFieldIndex(null)
    setConfigFieldForm({})
  }

  const handleSaveConfigField = () => {
    if (!configFieldForm.name.trim()) {
      showErrorToast('Field name is required')
      return
    }

    // Validate options for select/multi_select types
    if (configFieldForm.type === 'select') {
      const options = configFieldForm.typeOptions?.select?.options || []
      if (options.length === 0) {
        showErrorToast('At least one option is required for select fields')
        return
      }

      // Validate that all options have both label and value
      const hasInvalidOption = options.some((opt: any) => !opt.label?.trim() || !opt.value?.trim())
      if (hasInvalidOption) {
        showErrorToast('All options must have both label and value')
        return
      }
    } else if (configFieldForm.type === 'multi_select') {
      const options = configFieldForm.typeOptions?.multiSelect?.options || []
      if (options.length === 0) {
        showErrorToast('At least one option is required for multi-select fields')
        return
      }

      // Validate that all options have both label and value
      const hasInvalidOption = options.some((opt: any) => !opt.label?.trim() || !opt.value?.trim())
      if (hasInvalidOption) {
        showErrorToast('All options must have both label and value')
        return
      }
    }

    if (editingConfigFieldIndex !== null) {
      // Update existing field
      const newConfig = [...blueprintConfiguration]
      newConfig[editingConfigFieldIndex] = configFieldForm
      setBlueprintConfiguration(newConfig)
    } else {
      // Add new field
      setBlueprintConfiguration([...blueprintConfiguration, configFieldForm])
    }

    handleCloseConfigFieldModal()
  }

  const handleOpenOutputChannelModal = (index?: number) => {
    if (index !== undefined) {
      setEditingOutputChannelIndex(index)
      setOutputChannelForm(blueprintOutputChannels[index])
    } else {
      setEditingOutputChannelIndex(null)
      setOutputChannelForm({
        name: '',
        nodeId: '',
        nodeOutputChannel: 'default',
      })
    }
    setIsEditOutputChannelModalOpen(true)
  }

  const handleCloseOutputChannelModal = () => {
    setIsEditOutputChannelModalOpen(false)
    setEditingOutputChannelIndex(null)
    setOutputChannelForm({})
  }

  const handleSaveOutputChannel = () => {
    if (!outputChannelForm.name.trim()) {
      showErrorToast('Output channel name is required')
      return
    }

    if (!outputChannelForm.nodeId) {
      showErrorToast('Node selection is required')
      return
    }

    if (editingOutputChannelIndex !== null) {
      // Update existing output channel
      const newOutputChannels = [...blueprintOutputChannels]
      newOutputChannels[editingOutputChannelIndex] = outputChannelForm
      setBlueprintOutputChannels(newOutputChannels)
    } else {
      // Add new output channel
      setBlueprintOutputChannels([...blueprintOutputChannels, outputChannelForm])
    }

    handleCloseOutputChannelModal()
  }

  const handleSave = async () => {
    try {
      // Serialize all nodes
      const blueprintNodes = nodes.map((node) => {
        const nodeData = node.data as any
        return {
          id: node.id,
          name: nodeData.label as string,
          type: 'TYPE_COMPONENT',
          component: {
            name: nodeData._originalComponent as string,
          },
          configuration: nodeData._originalConfiguration || {},
          position: {
            x: Math.round(node.position.x),
            y: Math.round(node.position.y),
          },
        }
      })

      const blueprintEdges = edges.map((edge) => ({
        sourceId: edge.source!,
        targetId: edge.target!,
        channel: edge.sourceHandle || 'default',
      }))

      await updateBlueprintMutation.mutateAsync({
        name: blueprintName,
        description: blueprintDescription,
        nodes: blueprintNodes,
        edges: blueprintEdges,
        configuration: blueprintConfiguration,
        outputChannels: blueprintOutputChannels,
        icon: blueprintIcon,
        color: blueprintColor,
      })

      showSuccessToast('Blueprint saved successfully')
    } catch (error) {
      console.error('Error saving blueprint:', error)
      showErrorToast('Failed to save blueprint')
    }
  }

  if (blueprintLoading || componentsLoading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <p className="ml-3 text-gray-500">Loading blueprint...</p>
      </div>
    )
  }

  if (!blueprint) {
    return (
      <div className="flex flex-col items-center justify-center h-screen">
        <AlertCircle className="text-red-500 mb-4" size={32} />
        <Heading level={2}>Blueprint not found</Heading>
        <Button variant="outline" onClick={() => navigate(`/${organizationId}`)} className="mt-4">
          Go back to home
        </Button>
      </div>
    )
  }

  // Create breadcrumbs
  const breadcrumbs: BreadcrumbItem[] = [
    { label: 'Blueprints', onClick: () => navigate(`/${organizationId}`) },
    { label: blueprintName, iconSlug: blueprintIcon, iconColor: `text-${blueprintColor}-600` },
  ]

  return (
    <>
      <BlueprintBuilderPage
        blueprintName={blueprintName}
        breadcrumbs={breadcrumbs}
        metadata={{
          name: blueprintName,
          description: blueprintDescription,
          icon: blueprintIcon,
          color: blueprintColor,
        }}
        onMetadataChange={handleMetadataChange}
        configurationFields={blueprintConfiguration}
        onConfigurationFieldsChange={setBlueprintConfiguration}
        onAddConfigField={() => handleOpenConfigFieldModal()}
        onEditConfigField={handleOpenConfigFieldModal}
        outputChannels={blueprintOutputChannels}
        onOutputChannelsChange={setBlueprintOutputChannels}
        onAddOutputChannel={() => handleOpenOutputChannelModal()}
        onEditOutputChannel={handleOpenOutputChannelModal}
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={(nodeId) => console.log('Node clicked:', nodeId)}
        onNodeEdit={handleNodeEdit}
        onNodeDelete={handleNodeDelete}
        components={components}
        onComponentClick={handleBlockClick}
        onSave={handleSave}
        isSaving={updateBlueprintMutation.isPending}
      />

      {/* Add/Edit Node Modal */}
      <Dialog open={isAddNodeModalOpen} onOpenChange={(open) => !open && handleCloseModal()}>
        <DialogContent className="max-w-2xl p-0" showCloseButton={false}>
          <VisuallyHidden>
            <DialogTitle>{editingNodeId ? 'Edit Node' : 'Add Node'}</DialogTitle>
            <DialogDescription>Configure the node settings and parameters</DialogDescription>
          </VisuallyHidden>
          <ScrollArea className="max-h-[80vh]">
            <div className="p-6">
              <div className="space-y-6">
                {/* Node identification section */}
                <div className="flex items-center gap-3">
                  <Label className="min-w-[100px] text-left">Node Name</Label>
                  <Input
                    type="text"
                    value={nodeName}
                    onChange={(e) => setNodeName(e.target.value)}
                    placeholder="Enter a name for this node"
                    autoFocus
                    className="flex-1"
                  />
                </div>

                {/* Configuration section */}
                {selectedComponent?.configuration && selectedComponent.configuration.length > 0 && (
                  <div className="border-t border-gray-200 dark:border-zinc-700 pt-6 space-y-4">
                    {selectedComponent.configuration.map((field: any) => (
                      <ConfigurationFieldRenderer
                        key={field.name}
                        field={field}
                        value={nodeConfiguration[field.name]}
                        onChange={(value) => setNodeConfiguration({ ...nodeConfiguration, [field.name]: value })}
                        allValues={nodeConfiguration}
                        domainId={organizationId}
                        domainType="DOMAIN_TYPE_ORGANIZATION"
                      />
                    ))}
                  </div>
                )}
              </div>

              <DialogFooter className="mt-6">
                <Button variant="outline" onClick={handleCloseModal}>
                  Cancel
                </Button>
                <Button
                  variant="default"
                  onClick={handleAddNode}
                  disabled={!nodeName.trim()}
                >
                  {editingNodeId ? 'Save' : 'Add Node'}
                </Button>
              </DialogFooter>
            </div>
          </ScrollArea>
        </DialogContent>
      </Dialog>

      {/* Configuration Field Editor Modal */}
      <Dialog open={isEditConfigFieldModalOpen} onOpenChange={(open) => !open && handleCloseConfigFieldModal()}>
        <DialogContent className="max-w-2xl" showCloseButton={false}>
          <VisuallyHidden>
            <DialogTitle>
              {editingConfigFieldIndex !== null ? 'Edit Configuration Field' : 'Add Configuration Field'}
            </DialogTitle>
            <DialogDescription>Configure the blueprint configuration field</DialogDescription>
          </VisuallyHidden>
          <div className="p-6">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-zinc-100 mb-6">
              {editingConfigFieldIndex !== null ? 'Edit Configuration Field' : 'Add Configuration Field'}
            </h3>

            <div className="space-y-4">
              {/* Field Name */}
              <div>
                <Label className="block text-sm font-medium mb-2">Field Name *</Label>
                <Input
                  type="text"
                  value={configFieldForm.name || ''}
                  onChange={(e) => setConfigFieldForm({ ...configFieldForm, name: e.target.value })}
                  placeholder="e.g., threshold_expression"
                  autoFocus
                />
                <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                  This is the internal name used in templates (e.g., $config.threshold_expression)
                </p>
              </div>

              {/* Field Label */}
              <div>
                <Label className="block text-sm font-medium mb-2">Label *</Label>
                <Input
                  type="text"
                  value={configFieldForm.label || ''}
                  onChange={(e) => setConfigFieldForm({ ...configFieldForm, label: e.target.value })}
                  placeholder="e.g., Threshold Expression"
                />
                <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                  Display name shown in the UI
                </p>
              </div>

              {/* Field Type */}
              <div>
                <Label className="block text-sm font-medium mb-2">Type *</Label>
                <Select
                  value={configFieldForm.type || 'string'}
                  onValueChange={(val) => setConfigFieldForm({ ...configFieldForm, type: val })}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="string">String</SelectItem>
                    <SelectItem value="number">Number</SelectItem>
                    <SelectItem value="boolean">Boolean</SelectItem>
                    <SelectItem value="select">Select</SelectItem>
                    <SelectItem value="multi_select">Multi-Select</SelectItem>
                    <SelectItem value="date">Date</SelectItem>
                    <SelectItem value="url">URL</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Options Section (for select and multi_select types) */}
              {(configFieldForm.type === 'select' || configFieldForm.type === 'multi_select') && (() => {
                const isSelect = configFieldForm.type === 'select'
                const currentOptions = isSelect
                  ? (configFieldForm.typeOptions?.select?.options || [])
                  : (configFieldForm.typeOptions?.multiSelect?.options || [])

                const updateOptions = (newOptions: any[]) => {
                  if (isSelect) {
                    setConfigFieldForm({
                      ...configFieldForm,
                      typeOptions: {
                        ...configFieldForm.typeOptions,
                        select: { options: newOptions }
                      }
                    })
                  } else {
                    setConfigFieldForm({
                      ...configFieldForm,
                      typeOptions: {
                        ...configFieldForm.typeOptions,
                        multiSelect: { options: newOptions }
                      }
                    })
                  }
                }

                return (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4 space-y-3">
                    <div className="flex items-center justify-between">
                      <Label className="block text-sm font-medium">Options *</Label>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          updateOptions([...currentOptions, { label: '', value: '' }])
                        }}
                      >
                        <Plus />
                        Add Option
                      </Button>
                    </div>

                    {currentOptions.length > 0 ? (
                      <div className="space-y-2">
                        {currentOptions.map((option: any, index: number) => (
                          <div key={index} className="flex gap-2 items-start">
                            <div className="flex-1 grid grid-cols-2 gap-2">
                              <Input
                                type="text"
                                value={option.label || ''}
                                onChange={(e) => {
                                  const newOptions = [...currentOptions]
                                  newOptions[index] = { ...option, label: e.target.value }
                                  updateOptions(newOptions)
                                }}
                                placeholder="Label (e.g., Low)"
                              />
                              <Input
                                type="text"
                                value={option.value || ''}
                                onChange={(e) => {
                                  const newOptions = [...currentOptions]
                                  newOptions[index] = { ...option, value: e.target.value }
                                  updateOptions(newOptions)
                                }}
                                placeholder="Value (e.g., low)"
                              />
                            </div>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={() => {
                                const newOptions = currentOptions.filter((_: any, i: number) => i !== index)
                                updateOptions(newOptions)
                              }}
                            >
                              <Trash2 className="text-red-500" />
                            </Button>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-xs text-gray-500 dark:text-zinc-400">
                        No options added yet. Click "Add Option" to add options.
                      </p>
                    )}
                  </div>
                )
              })()}

              {/* Field Description */}
              <div>
                <Label className="block text-sm font-medium mb-2">Description</Label>
                <Input
                  type="text"
                  value={configFieldForm.description || ''}
                  onChange={(e) => setConfigFieldForm({ ...configFieldForm, description: e.target.value })}
                  placeholder="Describe the purpose of this field"
                />
              </div>

              {/* Required Checkbox */}
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={configFieldForm.required || false}
                  onChange={(e) => setConfigFieldForm({ ...configFieldForm, required: e.target.checked })}
                  className="h-4 w-4 rounded border-gray-300 dark:border-zinc-700"
                  id="required-checkbox"
                />
                <Label htmlFor="required-checkbox" className="cursor-pointer">
                  Required field
                </Label>
              </div>
            </div>

            <DialogFooter className="mt-6">
              <Button variant="outline" onClick={handleCloseConfigFieldModal}>
                Cancel
              </Button>
              <Button
                variant="default"
                onClick={handleSaveConfigField}
                disabled={!configFieldForm.name?.trim() || !configFieldForm.label?.trim()}
              >
                {editingConfigFieldIndex !== null ? 'Save Changes' : 'Add Field'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* Output Channel Editor Modal */}
      <Dialog open={isEditOutputChannelModalOpen} onOpenChange={(open) => !open && handleCloseOutputChannelModal()}>
        <DialogContent className="max-w-2xl" showCloseButton={false}>
          <VisuallyHidden>
            <DialogTitle>
              {editingOutputChannelIndex !== null ? 'Edit Output Channel' : 'Add Output Channel'}
            </DialogTitle>
            <DialogDescription>Configure the blueprint output channel</DialogDescription>
          </VisuallyHidden>
          <div className="p-6">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-zinc-100 mb-6">
              {editingOutputChannelIndex !== null ? 'Edit Output Channel' : 'Add Output Channel'}
            </h3>

            <div className="space-y-4">
              {/* Output Channel Name */}
              <div>
                <Label className="block text-sm font-medium mb-2">Output Channel Name *</Label>
                <Input
                  type="text"
                  value={outputChannelForm.name || ''}
                  onChange={(e) => setOutputChannelForm({ ...outputChannelForm, name: e.target.value })}
                  placeholder="e.g., success, error, default"
                  autoFocus
                />
                <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                  The name of this output channel
                </p>
              </div>

              {/* Node Selection */}
              <div>
                <Label className="block text-sm font-medium mb-2">Node *</Label>
                <Select
                  value={outputChannelForm.nodeId || ''}
                  onValueChange={(val) => {
                    // When node changes, reset the channel to default
                    setOutputChannelForm({ ...outputChannelForm, nodeId: val, nodeOutputChannel: 'default' })
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select a node" />
                  </SelectTrigger>
                  <SelectContent>
                    {nodes
                      .filter((node) => node.type !== 'outputChannel')
                      .map((node) => (
                        <SelectItem key={node.id} value={node.id}>
                          {(node.data as any).label} ({node.id})
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                  Select which node's output to use for this channel
                </p>
              </div>

              {/* Node Output Channel Selection */}
              {outputChannelForm.nodeId && (() => {
                const selectedNode = nodes.find((n) => n.id === outputChannelForm.nodeId)
                const nodeChannels = (selectedNode?.data as any)?.outputChannels || ['default']

                return (
                  <div>
                    <Label className="block text-sm font-medium mb-2">Node Output Channel *</Label>
                    <Select
                      value={outputChannelForm.nodeOutputChannel || 'default'}
                      onValueChange={(val) => setOutputChannelForm({ ...outputChannelForm, nodeOutputChannel: val })}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {nodeChannels.map((channel: string) => (
                          <SelectItem key={channel} value={channel}>
                            {channel}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                      Select which output channel from the node to expose
                    </p>
                  </div>
                )
              })()}
            </div>

            <DialogFooter className="mt-6">
              <Button variant="outline" onClick={handleCloseOutputChannelModal}>
                Cancel
              </Button>
              <Button
                variant="default"
                onClick={handleSaveOutputChannel}
                disabled={!outputChannelForm.name?.trim() || !outputChannelForm.nodeId}
              >
                {editingOutputChannelIndex !== null ? 'Save Changes' : 'Add Output Channel'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}

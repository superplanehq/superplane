import { useState, useCallback, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  ReactFlow,
  Node,
  Edge,
  addEdge,
  Background,
  BackgroundVariant,
  Controls,
  Connection,
  useNodesState,
  useEdgesState,
  NodeTypes,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useBlueprint, useUpdateBlueprint } from '../../hooks/useBlueprintData'
import { useComponents } from '../../hooks/useBlueprintData'
import { Button } from '../../components/ui/button'
import { MaterialSymbol } from '../../components/MaterialSymbol/material-symbol'
import { Heading } from '../../components/Heading/heading'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../components/ui/select'
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
import { IfNode } from './components/nodes/IfNode'
import { HttpNode } from './components/nodes/HttpNode'
import { FilterNode } from './components/nodes/FilterNode'
import { DefaultNode } from './components/nodes/DefaultNode'
import { ApprovalNode } from './components/nodes/ApprovalNode'
import { GithubTriggerNode } from './components/nodes/GithubTriggerNode'
import { SemaphoreTriggerNode } from './components/nodes/SemaphoreTriggerNode'
import { ConfigurationFieldRenderer } from '../../components/ConfigurationFieldRenderer'
import { ScrollArea } from '../../components/ui/scroll-area'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../../components/ui/tabs'
import { ItemGroup, Item, ItemMedia, ItemContent, ItemTitle, ItemDescription } from '../../components/ui/item'
import ELK from 'elkjs/lib/elk.bundled.js'
import { getColorClass } from '../../utils/colors'

const nodeTypes: NodeTypes = {
  if: IfNode,
  http: HttpNode,
  filter: FilterNode,
  approval: ApprovalNode,
  default: DefaultNode,
  github: GithubTriggerNode,
  semaphore: SemaphoreTriggerNode,
}

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
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)
  const [isAddNodeModalOpen, setIsAddNodeModalOpen] = useState(false)
  const [selectedComponent, setSelectedComponent] = useState<any>(null)
  const [nodeName, setNodeName] = useState('')
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, any>>({})
  const [editingNodeId, setEditingNodeId] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'components' | 'configuration' | 'outputChannels'>('components')
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

  // Fetch blueprint and components
  const { data: blueprint, isLoading: blueprintLoading } = useBlueprint(organizationId!, blueprintId!)
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!)
  const updateBlueprintMutation = useUpdateBlueprint(organizationId!, blueprintId!)

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])

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

  // Update nodes and edges when blueprint or components data changes
  useEffect(() => {
    if (!blueprint || components.length === 0) return

    const allNodes: Node[] = (blueprint.nodes || []).map((node: any) => {
      // Handle output channel nodes
      if (node.type === 'TYPE_OUTPUT_CHANNEL') {
        return {
          id: node.id,
          type: 'outputChannel',
          data: {
            label: node.name,
          },
          position: { x: 0, y: 0 }, // Will be set by elk
        }
      }

      // Handle component nodes
      const component = components.find((p: any) => p.name === node.component?.name)
      const channels = component?.outputChannels?.map((channel: any) => channel.name) || ['default']
      const componentName = node.component?.name

      // Use the component name as node type if it exists in nodeTypes, otherwise use 'default'
      const nodeType = componentName && nodeTypes[componentName as keyof typeof nodeTypes] ? componentName : 'default'

      return {
        id: node.id,
        type: nodeType,
        data: {
          label: node.name,
          component: componentName,
          channels,
          configuration: node.configuration || {},
          icon: component?.icon,
          color: component?.color,
        },
        position: node.position || { x: 0, y: 0 },
      }
    })

    const loadedEdges: Edge[] = (blueprint.edges || []).map((edge: any, index: number) => ({
      id: `e${index}`,
      source: edge.sourceId,
      sourceHandle: edge.channel || 'default',
      target: edge.targetId,
      label: edge.channel,
      style: { strokeWidth: 2, stroke: '#64748b' },
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
  }, [blueprint, components, setNodes, setEdges])

  const onConnect = useCallback(
    (params: Connection) => {
      // Prevent connections from output channels (output channels can only have incoming connections)
      const sourceNode = nodes.find(n => n.id === params.source)
      if (sourceNode?.type === 'outputChannel') {
        showErrorToast('Output channels cannot have outgoing connections')
        return
      }

      setEdges((eds) => addEdge({ ...params, style: { strokeWidth: 2, stroke: '#64748b' } }, eds))
    },
    [setEdges, nodes]
  )

  const handleComponentClick = (component: any) => {
    setSelectedComponent(component)
    setNodeName(component.name)
    setNodeConfiguration({})
    setIsAddNodeModalOpen(true)
  }

  const generateNodeId = (componentName: string, nodeName: string) => {
    const randomChars = Math.random().toString(36).substring(2, 8)
    const sanitizedComponent = componentName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    return `${sanitizedComponent}-${sanitizedName}-${randomChars}`
  }

  const handleNodeDoubleClick = useCallback((_: any, node: Node) => {
    const component = components.find((p: any) => p.name === (node.data as any).component)
    if (!component) return

    setEditingNodeId(node.id)
    setSelectedComponent(component)
    setNodeName((node.data as any).label as string)
    setNodeConfiguration((node.data as any).configuration || {})
    setIsAddNodeModalOpen(true)
  }, [components])

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
        nds.map((node) =>
          node.id === editingNodeId
            ? {
                ...node,
                data: {
                  ...node.data,
                  label: nodeName.trim(),
                  configuration: filteredConfiguration,
                },
              }
            : node
        )
      )
    } else {
      // Add new node with left-to-right positioning
      const newNodeId = generateNodeId(selectedComponent.name, nodeName.trim())
      const channels = selectedComponent?.outputChannels?.map((channel: any) => channel.name) || ['default']

      // Use component name as node type if it exists in nodeTypes, otherwise use 'default'
      const nodeType = selectedComponent.name && nodeTypes[selectedComponent.name as keyof typeof nodeTypes]
        ? selectedComponent.name
        : 'default'

      const newNode: Node = {
        id: newNodeId,
        type: nodeType,
        position: { x: nodes.length * 250, y: 100 },
        data: {
          label: nodeName.trim(),
          component: selectedComponent.name,
          channels,
          configuration: filteredConfiguration,
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
        options: [],
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
    if (configFieldForm.type === 'select' || configFieldForm.type === 'multi_select') {
      if (!configFieldForm.options || configFieldForm.options.length === 0) {
        showErrorToast('At least one option is required for select/multi-select fields')
        return
      }

      // Validate that all options have both label and value
      const hasInvalidOption = configFieldForm.options.some((opt: any) => !opt.label.trim() || !opt.value.trim())
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
      // Serialize all nodes including output channels
      const blueprintNodes = nodes.map((node) => {
        if (node.type === 'outputChannel') {
          // Output channel nodes
          return {
            id: node.id,
            name: (node.data as any).label as string,
            type: 'TYPE_OUTPUT_CHANNEL',
            outputChannel: {
              name: (node.data as any).label as string,
            },
            configuration: {},
            position: {
              x: Math.round(node.position.x),
              y: Math.round(node.position.y),
            },
          }
        } else {
          // Component nodes
          return {
            id: node.id,
            name: (node.data as any).label as string,
            type: 'TYPE_COMPONENT',
            component: {
              name: (node.data as any).component as string,
            },
            configuration: (node.data as any).configuration || {},
            position: {
              x: Math.round(node.position.x),
              y: Math.round(node.position.y),
            },
          }
        }
      })

      const blueprintEdges = edges.map((edge) => ({
        sourceId: edge.source!,
        targetId: edge.target!,
        channel: edge.sourceHandle || edge.label as string || 'default',
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
        <MaterialSymbol name="error" className="text-red-500 mb-4" size="xl" />
        <Heading level={2}>Blueprint not found</Heading>
        <Button variant="outline" onClick={() => navigate(`/${organizationId}`)} className="mt-4">
          Go back to home
        </Button>
      </div>
    )
  }

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="bg-white dark:bg-zinc-900 border-b border-zinc-200 dark:border-zinc-800 p-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(`/${organizationId}`)}>
            <MaterialSymbol name="arrow_back" />
          </Button>
          <div>
            <Heading level={2} className="!text-xl !mb-0">{blueprint.name}</Heading>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            onClick={handleSave}
            disabled={updateBlueprintMutation.isPending}
          >
            <MaterialSymbol name="save" />
            {updateBlueprintMutation.isPending ? 'Saving...' : 'Save'}
          </Button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex relative">
        {/* Sidebar */}
        {isSidebarOpen && (
          <div className="w-96 bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col z-50">
            {/* Sidebar Header */}
            <div className="flex items-center justify-between px-4 pt-4 pb-0">
              <div className="flex items-center gap-3">
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => setIsSidebarOpen(false)}
                  aria-label="Close sidebar"
                >
                  <MaterialSymbol name="menu_open" size="lg" />
                </Button>
                <h2 className="text-md font-semibold text-gray-900 dark:text-zinc-100">
                  Blueprint Builder
                </h2>
              </div>
            </div>

            {/* Blueprint Settings */}
            <div className="px-4 py-4 border-b border-zinc-200 dark:border-zinc-800">
              <div className="space-y-3">
                <div>
                  <Label htmlFor="blueprint-name" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                    Name
                  </Label>
                  <Input
                    id="blueprint-name"
                    value={blueprintName}
                    onChange={(e) => setBlueprintName(e.target.value)}
                    className="mt-1"
                    placeholder="Blueprint name"
                  />
                </div>
                <div>
                  <Label htmlFor="blueprint-description" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                    Description
                  </Label>
                  <Input
                    id="blueprint-description"
                    value={blueprintDescription}
                    onChange={(e) => setBlueprintDescription(e.target.value)}
                    className="mt-1"
                    placeholder="Blueprint description"
                  />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label htmlFor="blueprint-icon" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                      Icon
                    </Label>
                    <Input
                      id="blueprint-icon"
                      value={blueprintIcon}
                      onChange={(e) => setBlueprintIcon(e.target.value)}
                      className="mt-1"
                      placeholder="Icon name"
                    />
                  </div>
                  <div>
                    <Label htmlFor="blueprint-color" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                      Color
                    </Label>
                    <Input
                      id="blueprint-color"
                      value={blueprintColor}
                      onChange={(e) => setBlueprintColor(e.target.value)}
                      className="mt-1"
                      placeholder="Color"
                    />
                  </div>
                </div>
              </div>
            </div>

            {/* Tabs */}
            <Tabs value={activeTab} onValueChange={(value: any) => setActiveTab(value)} className="flex-1 flex flex-col">
              <TabsList className="mx-4 mt-4 grid w-auto grid-cols-3">
                <TabsTrigger value="components">Components</TabsTrigger>
                <TabsTrigger value="configuration">Configuration</TabsTrigger>
                <TabsTrigger value="outputChannels">Output Channels</TabsTrigger>
              </TabsList>

              {/* Components Tab */}
              <TabsContent value="components" className="flex-1 overflow-y-auto mt-0">
                <div className="text-left p-4">
                  <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                    Click on a component to add it to your blueprint
                  </div>
                  <ItemGroup>
                    {components.map((component: any) => {
                      const icon = component.icon || 'widgets'
                      const colorClass = getColorClass(component.color)

                      return (
                        <Item
                          key={component.name}
                          onClick={() => handleComponentClick(component)}
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                          size="sm"
                        >
                          <ItemMedia>
                            <MaterialSymbol name={icon} size="lg" className={colorClass} />
                          </ItemMedia>
                          <ItemContent>
                            <ItemTitle>{component.label || component.name}</ItemTitle>
                            {component.description && (
                              <ItemDescription>{component.description}</ItemDescription>
                            )}
                          </ItemContent>
                        </Item>
                      )
                    })}
                  </ItemGroup>
                </div>
              </TabsContent>

              {/* Configuration Tab */}
              <TabsContent value="configuration" className="flex-1 overflow-y-auto mt-0">
                <div className="text-left p-4 space-y-6">
                  <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                    Add configuration fields that can be used in your blueprint nodes
                  </div>

                  {/* Configuration Fields List */}
                  {blueprintConfiguration.length > 0 && (
                    <div className="space-y-4">
                      {blueprintConfiguration.map((field: any, index: number) => (
                        <div
                          key={index}
                          className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4 space-y-3 cursor-pointer hover:border-blue-400 dark:hover:border-blue-600 transition-colors"
                          onClick={() => handleOpenConfigFieldModal(index)}
                        >
                          <div className="flex items-start justify-between">
                            <div className="flex-1">
                              <p className="font-medium text-sm text-gray-900 dark:text-zinc-100">
                                {field.label || field.name}
                              </p>
                              <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                                Type: {field.type} {field.required && '(required)'}
                              </p>
                              {field.description && (
                                <p className="text-xs text-gray-600 dark:text-zinc-400 mt-1">
                                  {field.description}
                                </p>
                              )}
                              {(field.type === 'select' || field.type === 'multi_select') && field.options && field.options.length > 0 && (
                                <p className="text-xs text-gray-600 dark:text-zinc-400 mt-1">
                                  Options: {field.options.map((opt: any) => opt.label).join(', ')}
                                </p>
                              )}
                            </div>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={(e) => {
                                e.stopPropagation()
                                const newConfig = blueprintConfiguration.filter((_, i) => i !== index)
                                setBlueprintConfiguration(newConfig)
                              }}
                            >
                              <MaterialSymbol name="delete" className="text-red-500" />
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Add New Configuration Field Button */}
                  <Button
                    variant="outline"
                    onClick={() => handleOpenConfigFieldModal()}
                    className="w-full"
                  >
                    <MaterialSymbol name="add" />
                    Add Configuration Field
                  </Button>
                </div>
              </TabsContent>

              {/* Output Channels Tab */}
              <TabsContent value="outputChannels" className="flex-1 overflow-y-auto mt-0">
                <div className="text-left p-4 space-y-6">
                  <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                    Define output channels for this blueprint by selecting which node and channel should be exposed
                  </div>

                  {/* Output Channels List */}
                  {blueprintOutputChannels.length > 0 && (
                    <div className="space-y-4">
                      {blueprintOutputChannels.map((outputChannel: any, index: number) => (
                        <div
                          key={index}
                          className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4 space-y-3 cursor-pointer hover:border-green-400 dark:hover:border-green-600 transition-colors"
                          onClick={() => handleOpenOutputChannelModal(index)}
                        >
                          <div className="flex items-start justify-between">
                            <div className="flex-1">
                              <div className="flex items-center gap-2">
                                <MaterialSymbol name="output" className="text-green-600 dark:text-green-400" />
                                <p className="font-medium text-sm text-gray-900 dark:text-zinc-100">
                                  {outputChannel.name}
                                </p>
                              </div>
                              <p className="text-xs text-gray-500 dark:text-zinc-400 mt-2">
                                Node: {outputChannel.nodeId}
                              </p>
                              <p className="text-xs text-gray-500 dark:text-zinc-400">
                                Channel: {outputChannel.nodeOutputChannel || 'default'}
                              </p>
                            </div>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={(e) => {
                                e.stopPropagation()
                                const newOutputChannels = blueprintOutputChannels.filter((_, i) => i !== index)
                                setBlueprintOutputChannels(newOutputChannels)
                              }}
                            >
                              <MaterialSymbol name="delete" className="text-red-500" />
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Add New Output Channel Button */}
                  <Button
                    variant="outline"
                    onClick={() => handleOpenOutputChannelModal()}
                    className="w-full"
                  >
                    <MaterialSymbol name="add" />
                    Add Output Channel
                  </Button>
                </div>
              </TabsContent>
            </Tabs>
          </div>
        )}

        {/* React Flow Canvas */}
        <div className="flex-1 relative">
          {!isSidebarOpen && (
            <Button
              variant="outline"
              size="icon"
              onClick={() => setIsSidebarOpen(true)}
              aria-label="Open sidebar"
              className="absolute top-4 left-4 z-10 shadow-md"
            >
              <MaterialSymbol name="menu" size="lg" />
            </Button>
          )}
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={nodeTypes}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeDoubleClick={handleNodeDoubleClick}
            fitView
            colorMode="system"
          >
            <Background
              variant={BackgroundVariant.Dots}
              gap={24}
              size={1}
            />
            <Controls />
          </ReactFlow>
        </div>
      </div>

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
              {(configFieldForm.type === 'select' || configFieldForm.type === 'multi_select') && (
                <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <Label className="block text-sm font-medium">Options *</Label>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        const currentOptions = configFieldForm.options || []
                        setConfigFieldForm({
                          ...configFieldForm,
                          options: [...currentOptions, { label: '', value: '' }]
                        })
                      }}
                    >
                      <MaterialSymbol name="add" />
                      Add Option
                    </Button>
                  </div>

                  {configFieldForm.options && configFieldForm.options.length > 0 ? (
                    <div className="space-y-2">
                      {configFieldForm.options.map((option: any, index: number) => (
                        <div key={index} className="flex gap-2 items-start">
                          <div className="flex-1 grid grid-cols-2 gap-2">
                            <Input
                              type="text"
                              value={option.label || ''}
                              onChange={(e) => {
                                const newOptions = [...configFieldForm.options]
                                newOptions[index] = { ...option, label: e.target.value }
                                setConfigFieldForm({ ...configFieldForm, options: newOptions })
                              }}
                              placeholder="Label (e.g., Low)"
                            />
                            <Input
                              type="text"
                              value={option.value || ''}
                              onChange={(e) => {
                                const newOptions = [...configFieldForm.options]
                                newOptions[index] = { ...option, value: e.target.value }
                                setConfigFieldForm({ ...configFieldForm, options: newOptions })
                              }}
                              placeholder="Value (e.g., low)"
                            />
                          </div>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => {
                              const newOptions = configFieldForm.options.filter((_: any, i: number) => i !== index)
                              setConfigFieldForm({ ...configFieldForm, options: newOptions })
                            }}
                          >
                            <MaterialSymbol name="delete" className="text-red-500" />
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
              )}

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
                const nodeChannels = (selectedNode?.data as any)?.channels || ['default']

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
    </div>
  )
}

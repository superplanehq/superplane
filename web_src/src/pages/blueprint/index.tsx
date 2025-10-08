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
  MarkerType,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useBlueprint, useUpdateBlueprint } from '../../hooks/useBlueprintData'
import { useComponents } from '../../hooks/useBlueprintData'
import { Button } from '../../components/ui/button'
import { MaterialSymbol } from '../../components/MaterialSymbol/material-symbol'
import { Heading } from '../../components/Heading/heading'
import { Text } from '../../components/Text/text'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogFooter,
} from '../../components/ui/dialog'
import { showSuccessToast, showErrorToast } from '../../utils/toast'
import { IfNode } from './components/nodes/IfNode'
import { HttpNode } from './components/nodes/HttpNode'
import { FilterNode } from './components/nodes/FilterNode'
import { SwitchNode } from './components/nodes/SwitchNode'
import { DefaultNode } from './components/nodes/DefaultNode'
import { ConfigurationFieldRenderer } from '../../components/ConfigurationFieldRenderer'
import { ScrollArea } from '../../components/ui/scroll-area'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../../components/ui/tabs'
import { ItemGroup, Item, ItemMedia, ItemContent, ItemTitle, ItemDescription } from '../../components/ui/item'
import ELK from 'elkjs/lib/elk.bundled.js'

const nodeTypes = {
  if: IfNode,
  http: HttpNode,
  filter: FilterNode,
  switch: SwitchNode,
  default: DefaultNode,
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
  const [connectingFrom, setConnectingFrom] = useState<{ nodeId: string; branch: string } | null>(null)

  // Fetch blueprint and components
  const { data: blueprint, isLoading: blueprintLoading } = useBlueprint(organizationId!, blueprintId!)
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!)
  const updateBlueprintMutation = useUpdateBlueprint(organizationId!, blueprintId!)

  const handleAddNodeFromBranch = useCallback((sourceNodeId: string, branch: string) => {
    setConnectingFrom({ nodeId: sourceNodeId, branch })
  }, [])

  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  // Update nodes and edges when blueprint or components data changes
  useEffect(() => {
    if (!blueprint || components.length === 0) return

    const loadedNodes: Node[] = (blueprint.nodes || []).map((node: any) => {
      const component = components.find((p: any) => p.name === node.component?.name)
      const branches = component?.branches?.map((branch: any) => branch.name) || ['default']
      const componentName = node.component?.name

      // Use the component name as node type if it exists in nodeTypes, otherwise use 'default'
      const nodeType = componentName && nodeTypes[componentName as keyof typeof nodeTypes] ? componentName : 'default'

      return {
        id: node.id,
        type: nodeType,
        data: {
          label: node.name,
          component: componentName,
          branches,
          configuration: node.configuration || {},
          onAddNode: handleAddNodeFromBranch,
        },
        position: { x: 0, y: 0 }, // Will be set by elk
      }
    })

    const loadedEdges: Edge[] = (blueprint.edges || []).map((edge: any, index: number) => ({
      id: `e${index}`,
      source: edge.sourceId,
      sourceHandle: edge.branch || 'default',
      target: edge.targetId,
      label: edge.branch,
      style: { strokeWidth: 2, stroke: '#64748b' },
    }))

    // Apply elk layout
    getLayoutedElements(loadedNodes, loadedEdges).then(({ nodes: layoutedNodes, edges: layoutedEdges }) => {
      setNodes(layoutedNodes)
      setEdges(layoutedEdges)
    })
  }, [blueprint, components, setNodes, setEdges, handleAddNodeFromBranch])

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, style: { strokeWidth: 2, stroke: '#64748b' } }, eds))
    },
    [setEdges]
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
    const component = components.find((p: any) => p.name === node.data.component)
    if (!component) return

    setEditingNodeId(node.id)
    setSelectedComponent(component)
    setNodeName(node.data.label)
    setNodeConfiguration(node.data.configuration || {})
    setIsAddNodeModalOpen(true)
  }, [components])

  const handleAddNode = () => {
    if (!selectedComponent || !nodeName.trim()) return

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
                  configuration: nodeConfiguration,
                },
              }
            : node
        )
      )
    } else {
      // Add new node with left-to-right positioning
      const branches = selectedComponent?.branches?.map((branch: any) => branch.name) || ['default']
      const newNodeId = generateNodeId(selectedComponent.name, nodeName.trim())

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
          branches,
          configuration: nodeConfiguration,
          onAddNode: handleAddNodeFromBranch,
        },
      }
      setNodes((nds) => [...nds, newNode])

      // If connecting from a branch button, create the edge
      if (connectingFrom) {
        const newEdge: Edge = {
          id: `e-${connectingFrom.nodeId}-${newNodeId}-${Date.now()}`,
          source: connectingFrom.nodeId,
          sourceHandle: connectingFrom.branch,
          target: newNodeId,
          label: connectingFrom.branch,
          style: { strokeWidth: 2, stroke: '#64748b' },
        }
        setEdges((eds) => [...eds, newEdge])
        setConnectingFrom(null)
      }
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

  const handleSave = async () => {
    try {
      const blueprintNodes = nodes.map((node) => ({
        id: node.id,
        name: node.data.label,
        refType: 'REF_TYPE_COMPONENT',
        component: {
          name: node.data.component,
        },
        configuration: node.data.configuration || {},
      }))

      const blueprintEdges = edges.map((edge) => ({
        sourceId: edge.source!,
        targetId: edge.target!,
        branch: edge.sourceHandle || edge.label as string || 'default',
      }))

      await updateBlueprintMutation.mutateAsync({
        name: blueprint?.name || '',
        description: blueprint?.description,
        nodes: blueprintNodes,
        edges: blueprintEdges,
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
            {blueprint.description && (
              <Text className="text-sm text-zinc-600 dark:text-zinc-400">{blueprint.description}</Text>
            )}
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
                  Components
                </h2>
              </div>
            </div>

            {/* Sidebar Content */}
            <div className="flex-1 overflow-y-auto text-left p-4">
              <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                Click on a component to add it to your blueprint
              </div>
              <ItemGroup>
                {components.map((component: any) => {
                  const iconMap: Record<string, string> = {
                    if: 'alt_route',
                    http: 'http',
                    filter: 'filter_alt',
                    switch: 'settings_input_component',
                  }
                  const icon = iconMap[component.name] || 'widgets'

                  return (
                    <Item
                      key={component.name}
                      onClick={() => handleComponentClick(component)}
                      className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                      size="sm"
                    >
                      <ItemMedia>
                        <MaterialSymbol name={icon} size="lg" className="text-blue-600 dark:text-blue-400" />
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
    </div>
  )
}

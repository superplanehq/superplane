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
import { useWorkflow, useUpdateWorkflow } from '../../hooks/useWorkflowData'
import { useComponents, useBlueprints } from '../../hooks/useBlueprintData'
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
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../../components/ui/tabs'
import { ItemGroup, Item, ItemMedia, ItemContent, ItemTitle, ItemDescription } from '../../components/ui/item'
import { showSuccessToast, showErrorToast } from '../../utils/toast'
import { IfNode } from '../blueprint/components/nodes/IfNode'
import { HttpNode } from '../blueprint/components/nodes/HttpNode'
import { FilterNode } from '../blueprint/components/nodes/FilterNode'
import { SwitchNode } from '../blueprint/components/nodes/SwitchNode'
import { ApprovalNode } from '../blueprint/components/nodes/ApprovalNode'
import { DefaultNode } from '../blueprint/components/nodes/DefaultNode'
import { WorkflowNodeSidebar } from '../../components/WorkflowNodeSidebar'
import { ConfigurationFieldRenderer } from '../../components/ConfigurationFieldRenderer'
import { ScrollArea } from '../../components/ui/scroll-area'
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from '../../components/ui/resizable'
import ELK from 'elkjs/lib/elk.bundled.js'

const nodeTypes: NodeTypes = {
  if: IfNode,
  http: HttpNode,
  filter: FilterNode,
  switch: SwitchNode,
  approval: ApprovalNode,
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

type BuildingBlock = {
  name: string
  label?: string
  description?: string
  type: 'component' | 'blueprint'
  channels?: { name: string }[]
  configuration?: any[]
}

export const Workflow = () => {
  const { organizationId, workflowId } = useParams<{ organizationId: string; workflowId: string }>()
  const navigate = useNavigate()
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)
  const [isAddNodeModalOpen, setIsAddNodeModalOpen] = useState(false)
  const [selectedBlock, setSelectedBlock] = useState<BuildingBlock | null>(null)
  const [nodeName, setNodeName] = useState('')
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, any>>({})
  const [editingNodeId, setEditingNodeId] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'components' | 'blueprints'>('components')
  const [selectedNode, setSelectedNode] = useState<{ id: string; name: string; isBlueprintNode: boolean; nodeType: string; componentLabel?: string } | null>(null)

  // Fetch workflow, components, and blueprints
  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!)
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!)
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId!)
  const updateWorkflowMutation = useUpdateWorkflow(organizationId!, workflowId!)

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])

  // Combine components and blueprints into building blocks
  const buildingBlocks: BuildingBlock[] = [
    ...components.map((p: any) => ({ ...p, type: 'component' as const })),
    ...blueprints.map((b: any) => ({ ...b, type: 'blueprint' as const }))
  ]

  // Update nodes and edges when workflow data changes
  useEffect(() => {
    if (!workflow || buildingBlocks.length === 0) return

    const loadedNodes: Node[] = (workflow.nodes || []).map((node: any) => {
      const isComponent = node.type === 'TYPE_COMPONENT'
      const blockName = isComponent ? node.component?.name : node.blueprint?.name
      const blockId = isComponent ? node.component?.name : node.blueprint?.id
      const block = buildingBlocks.find((b: BuildingBlock) => {
        if (isComponent) {
          return b.name === blockName && b.type === 'component'
        } else {
          // For blueprints, match by ID since blueprint nodes reference by ID, not name
          return (b as any).id === blockId && b.type === 'blueprint'
        }
      })

      const channels = block?.channels?.map((channel: any) => channel.name) || ['default']

      // Use component name as node type if it exists in nodeTypes, otherwise use 'default'
      const nodeType = isComponent && blockName && nodeTypes[blockName as keyof typeof nodeTypes]
        ? blockName
        : 'default'

      return {
        id: node.id,
        type: nodeType,
        data: {
          label: node.name,
          blockName,
          blockId,
          blockType: isComponent ? 'component' : 'blueprint',
          channels,
          configuration: node.configuration || {},
        },
        position: { x: 0, y: 0 }, // Will be set by elk
      }
    })

    const loadedEdges: Edge[] = (workflow.edges || []).map((edge: any, index: number) => ({
      id: `e${index}`,
      source: edge.sourceId,
      sourceHandle: edge.channel || 'default',
      target: edge.targetId,
      label: edge.channel,
      style: { strokeWidth: 2, stroke: '#64748b' },
    }))

    // Apply elk layout
    getLayoutedElements(loadedNodes, loadedEdges).then(({ nodes: layoutedNodes, edges: layoutedEdges }) => {
      setNodes(layoutedNodes)
      setEdges(layoutedEdges)
    })
  }, [workflow, buildingBlocks.length, setNodes, setEdges])

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, style: { strokeWidth: 2, stroke: '#64748b' } }, eds))
    },
    [setEdges]
  )

  const handleBlockClick = (block: BuildingBlock) => {
    setSelectedBlock(block)
    setNodeName(block.name || '')
    setNodeConfiguration({})
    setIsAddNodeModalOpen(true)
  }

  const generateNodeId = (blockName: string, nodeName: string) => {
    const randomChars = Math.random().toString(36).substring(2, 8)
    const sanitizedBlock = blockName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    return `${sanitizedBlock}-${sanitizedName}-${randomChars}`
  }

  const handleNodeClick = useCallback((_: any, node: Node) => {
    const block = buildingBlocks.find((b: BuildingBlock) => {
      if (node.data.blockType === 'component') {
        return b.name === node.data.blockName && b.type === 'component'
      } else {
        // For blueprints, match by ID
        return (b as any).id === node.data.blockId && b.type === 'blueprint'
      }
    })
    setSelectedNode({
      id: node.id,
      name: node.data.label as string,
      isBlueprintNode: node.data.blockType === 'blueprint',
      nodeType: node.data.blockName as string,
      componentLabel: block?.label
    })
  }, [buildingBlocks])

  const handleNodeDoubleClick = useCallback((_: any, node: Node) => {
    const block = buildingBlocks.find((b: BuildingBlock) => {
      if (node.data.blockType === 'component') {
        return b.name === node.data.blockName && b.type === 'component'
      } else {
        // For blueprints, match by ID
        return (b as any).id === node.data.blockId && b.type === 'blueprint'
      }
    })
    if (!block) return

    setEditingNodeId(node.id)
    setSelectedBlock(block)
    setNodeName(node.data.label as string)
    setNodeConfiguration((node.data.configuration as Record<string, any>) || {})
    setIsAddNodeModalOpen(true)
  }, [buildingBlocks])

  const handleAddNode = () => {
    if (!selectedBlock || !nodeName.trim()) return

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
      // Add new node
      const channels = selectedBlock?.channels?.map((channel: any) => channel.name) || ['default']
      const newNodeId = generateNodeId(selectedBlock.name || 'node', nodeName.trim())

      // Use block name as node type if it exists in nodeTypes and is a component
      const nodeType = selectedBlock.type === 'component' &&
                      selectedBlock.name &&
                      nodeTypes[selectedBlock.name as keyof typeof nodeTypes]
        ? selectedBlock.name
        : 'default'

      const newNode: Node = {
        id: newNodeId,
        type: nodeType,
        position: { x: nodes.length * 250, y: 100 },
        data: {
          label: nodeName.trim(),
          blockName: selectedBlock.name || '',
          blockId: selectedBlock.type === 'blueprint' ? (selectedBlock as any).id : selectedBlock.name,
          blockType: selectedBlock.type,
          channels,
          configuration: nodeConfiguration,
        } as Record<string, unknown>,
      }
      setNodes((nds) => [...nds, newNode])
    }

    setIsAddNodeModalOpen(false)
    setSelectedBlock(null)
    setNodeName('')
    setNodeConfiguration({})
    setEditingNodeId(null)
  }

  const handleCloseModal = () => {
    setIsAddNodeModalOpen(false)
    setSelectedBlock(null)
    setNodeName('')
    setNodeConfiguration({})
    setEditingNodeId(null)
  }

  const handleSave = async () => {
    try {
      const workflowNodes = nodes.map((node) => {
        const baseNode: any = {
          id: node.id,
          name: node.data.label,
          type: node.data.blockType === 'component' ? 'TYPE_COMPONENT' : 'TYPE_BLUEPRINT',
          configuration: node.data.configuration || {},
        }

        // Add either component or blueprint reference directly on the node
        if (node.data.blockType === 'component') {
          baseNode.component = { name: node.data.blockName }
        } else {
          baseNode.blueprint = { id: node.data.blockId }
        }

        return baseNode
      })

      const workflowEdges = edges.map((edge) => ({
        sourceId: edge.source!,
        targetId: edge.target!,
        channel: edge.sourceHandle || edge.label as string || 'default',
      }))

      await updateWorkflowMutation.mutateAsync({
        name: workflow?.name || '',
        description: workflow?.description,
        nodes: workflowNodes,
        edges: workflowEdges,
      })

      showSuccessToast('Workflow saved successfully')
    } catch (error) {
      console.error('Error saving workflow:', error)
      showErrorToast('Failed to save workflow')
    }
  }

  if (workflowLoading || componentsLoading || blueprintsLoading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <p className="ml-3 text-gray-500">Loading workflow...</p>
      </div>
    )
  }

  if (!workflow) {
    return (
      <div className="flex flex-col items-center justify-center h-screen">
        <MaterialSymbol name="error" className="text-red-500 mb-4" size="xl" />
        <Heading level={2}>Workflow not found</Heading>
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
            <Heading level={2} className="!text-xl !mb-0">{workflow.name}</Heading>
            {workflow.description && (
              <Text className="text-sm text-zinc-600 dark:text-zinc-400">{workflow.description}</Text>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => navigate(`/${organizationId}/workflows/${workflowId}/events`)}
          >
            <MaterialSymbol name="timeline" />
            Event Execution Chains
          </Button>
          <Button
            onClick={handleSave}
            disabled={updateWorkflowMutation.isPending}
          >
            <MaterialSymbol name="save" />
            {updateWorkflowMutation.isPending ? 'Saving...' : 'Save'}
          </Button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex relative">
        {/* Sidebar */}
        {isSidebarOpen && (
          <div className="w-96 bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col z-50">
            {/* Sidebar Header with Tabs */}
            <div className="flex items-center gap-3 px-4 pt-4 pb-0">
              <Button
                variant="outline"
                size="icon"
                onClick={() => setIsSidebarOpen(false)}
                aria-label="Close sidebar"
              >
                <MaterialSymbol name="menu_open" size="lg" />
              </Button>
              <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as 'components' | 'blueprints')} className="flex-1">
                <TabsList className="w-full">
                  <TabsTrigger value="components" className="flex-1">
                    Components
                  </TabsTrigger>
                  <TabsTrigger value="blueprints" className="flex-1">
                    Blueprints
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </div>

            {/* Tab Content */}
            <div className="flex-1 overflow-hidden px-4">
              <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as 'components' | 'blueprints')} className="flex-1 flex flex-col h-full">
                <TabsContent value="components" className="flex-1 overflow-y-auto text-left mt-4 data-[state=inactive]:hidden">
                  <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                    Click on a component to add it to your workflow
                  </div>
                  <ItemGroup>
                    {buildingBlocks.filter(b => b.type === 'component').map((block: BuildingBlock) => {
                      // Map block name to icon
                      const iconMap: Record<string, string> = {
                        if: 'alt_route',
                        http: 'http',
                        filter: 'filter_alt',
                        switch: 'settings_input_component',
                      }
                      const icon = block.type === 'component'
                        ? (iconMap[block.name] || 'widgets')
                        : 'account_tree'

                      return (
                        <Item
                          key={`${block.type}-${block.name}`}
                          onClick={() => handleBlockClick(block)}
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                          size="sm"
                        >
                          <ItemMedia>
                            <MaterialSymbol name={icon} size="lg" className="text-blue-600 dark:text-blue-400" />
                          </ItemMedia>
                          <ItemContent>
                            <ItemTitle>{block.label || block.name}</ItemTitle>
                            {block.description && (
                              <ItemDescription>{block.description}</ItemDescription>
                            )}
                          </ItemContent>
                        </Item>
                      )
                    })}
                  </ItemGroup>
                </TabsContent>

                <TabsContent value="blueprints" className="flex-1 overflow-y-auto text-left mt-4 data-[state=inactive]:hidden">
                  <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                    Click on a blueprint to add it to your workflow
                  </div>
                  <ItemGroup>
                    {buildingBlocks.filter(b => b.type === 'blueprint').map((block: BuildingBlock) => {
                      const icon = 'account_tree'

                      return (
                        <Item
                          key={`${block.type}-${block.name}`}
                          onClick={() => handleBlockClick(block)}
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                          size="sm"
                        >
                          <ItemMedia>
                            <MaterialSymbol name={icon} size="lg" className="text-blue-600 dark:text-blue-400" />
                          </ItemMedia>
                          <ItemContent>
                            <ItemTitle>{block.label || block.name}</ItemTitle>
                            {block.description && (
                              <ItemDescription>{block.description}</ItemDescription>
                            )}
                          </ItemContent>
                        </Item>
                      )
                    })}
                  </ItemGroup>
                </TabsContent>
              </Tabs>
            </div>
          </div>
        )}

        {/* React Flow Canvas and Right Sidebar */}
        <ResizablePanelGroup direction="horizontal" className="flex-1">
          <ResizablePanel defaultSize={selectedNode ? 65 : 100} minSize={30}>
            <div className="relative h-full">
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
                onNodeClick={handleNodeClick}
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
          </ResizablePanel>

          {/* Node Details Sidebar */}
          {selectedNode && (
            <>
              <ResizableHandle />
              <ResizablePanel defaultSize={35} minSize={20} maxSize={60}>
                <WorkflowNodeSidebar
                  workflowId={workflowId!}
                  nodeId={selectedNode.id}
                  nodeName={selectedNode.name}
                  onClose={() => setSelectedNode(null)}
                  isBlueprintNode={selectedNode.isBlueprintNode}
                  nodeType={selectedNode.nodeType}
                  componentLabel={selectedNode.componentLabel}
                />
              </ResizablePanel>
            </>
          )}
        </ResizablePanelGroup>
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
                {selectedBlock?.configuration && selectedBlock.configuration.length > 0 && (
                  <div className="border-t border-gray-200 dark:border-zinc-700 pt-6 space-y-4">
                    {selectedBlock.configuration.map((field: any) => (
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

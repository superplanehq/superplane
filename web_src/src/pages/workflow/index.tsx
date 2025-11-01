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
import { useWorkflow, useUpdateWorkflow, useTriggers } from '../../hooks/useWorkflowData'
import { useComponents, useBlueprints } from '../../hooks/useBlueprintData'
import { Button } from '../../components/ui/button'
import { AlertCircle, ArrowLeft, Activity, Save, PanelLeftClose, Menu } from 'lucide-react'
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
import { filterVisibleConfiguration } from '../../utils/components'
import { WorkflowNodeSidebar } from '../../components/WorkflowNodeSidebar'
import { ConfigurationFieldRenderer } from '@/ui/configurationFieldRenderer'
import { ScrollArea } from '../../components/ui/scroll-area'
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from '../../components/ui/resizable'
import { EmitEventModal } from '@/ui/EmitEventModal'
import { workflowsEmitNodeEvent } from '../../api-client/sdk.gen'
import { withOrganizationHeader } from '../../utils/withOrganizationHeader'
import {
  WorkflowIfNode,
  WorkflowHttpNode,
  WorkflowFilterNode,
  WorkflowApprovalNode,
  WorkflowDefaultNode,
  WorkflowStartTriggerNode,
  WorkflowScheduledTriggerNode,
  WorkflowWebhookTriggerNode,
  WorkflowGithubTriggerNode,
  WorkflowSemaphoreTriggerNode
} from './components/nodes'
import ELK from 'elkjs/lib/elk.bundled.js'
import { getColorClass } from '../../utils/colors'
import { resolveIcon } from '../../lib/utils'

const nodeTypes: NodeTypes = {
  if: WorkflowIfNode,
  http: WorkflowHttpNode,
  filter: WorkflowFilterNode,
  approval: WorkflowApprovalNode,
  start: WorkflowStartTriggerNode,
  schedule: WorkflowScheduledTriggerNode,
  webhook: WorkflowWebhookTriggerNode,
  github: WorkflowGithubTriggerNode,
  semaphore: WorkflowSemaphoreTriggerNode,
  default: WorkflowDefaultNode,
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
  type: 'trigger' | 'component' | 'blueprint'
  outputChannels?: { name: string }[]
  configuration?: any[]
  icon?: string
  color?: string
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
  const [activeTab, setActiveTab] = useState<'triggers' | 'components' | 'blueprints'>('triggers')
  const [selectedNode, setSelectedNode] = useState<{ id: string; name: string; isBlueprintNode: boolean; nodeType: 'trigger' | 'component' | 'blueprint'; componentName?: string; componentLabel?: string; blueprintId?: string } | null>(null)
  const [emitModalNode, setEmitModalNode] = useState<{ id: string; name: string; channels: string[] } | null>(null)

  // Fetch workflow, components, blueprints, and triggers
  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!)
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!)
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId!)
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers()
  const updateWorkflowMutation = useUpdateWorkflow(organizationId!, workflowId!)

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])

  // Combine components, blueprints, and triggers into building blocks
  const buildingBlocks: BuildingBlock[] = [
    ...triggers.map((t: any) => ({ ...t, type: 'trigger' as const })),
    ...components.map((p: any) => ({ ...p, type: 'component' as const })),
    ...blueprints.map((b: any) => ({ ...b, type: 'blueprint' as const }))
  ]

  // Define handlers before they're used in useEffect
  // Use setNodes to access current nodes without adding it as a dependency
  const handleNodeEdit = useCallback((nodeId: string) => {
    setNodes((currentNodes) => {
      const node = currentNodes.find(n => n.id === nodeId)
      if (!node) return currentNodes

      const block = buildingBlocks.find((b: BuildingBlock) => {
        if (node.data.blockType === 'component') {
          return b.name === node.data.blockName && b.type === 'component'
        } else if (node.data.blockType === 'trigger') {
          return b.name === node.data.blockName && b.type === 'trigger'
        } else {
          return (b as any).id === node.data.blockId && b.type === 'blueprint'
        }
      })
      if (!block) return currentNodes

      setEditingNodeId(node.id)
      setSelectedBlock(block)
      setNodeName(node.data.label as string)
      setNodeConfiguration((node.data.configuration as Record<string, any>) || {})
      setIsAddNodeModalOpen(true)

      return currentNodes
    })
  }, [buildingBlocks])

  const handleNodeEmit = useCallback((nodeId: string) => {
    setNodes((currentNodes) => {
      const node = currentNodes.find(n => n.id === nodeId)
      if (!node) return currentNodes

      setEmitModalNode({
        id: node.id,
        name: node.data.label as string,
        channels: (node.data.channels as string[]) || ['default']
      })

      return currentNodes
    })
  }, [])

  // Update nodes and edges when workflow data changes
  useEffect(() => {
    if (!workflow || buildingBlocks.length === 0) return

    const loadedNodes: Node[] = (workflow.nodes || []).map((node: any) => {
      const isComponent = node.type === 'TYPE_COMPONENT'
      const isTrigger = node.type === 'TYPE_TRIGGER'
      const isBlueprint = node.type === 'TYPE_BLUEPRINT'

      const blockName = isComponent ? node.component?.name
                      : isTrigger ? node.trigger?.name
                      : node.blueprint?.name
      const blockId = isComponent ? node.component?.name
                    : isTrigger ? node.trigger?.name
                    : node.blueprint?.id

      const block = buildingBlocks.find((b: BuildingBlock) => {
        if (isComponent) {
          return b.name === blockName && b.type === 'component'
        } else if (isTrigger) {
          return b.name === blockName && b.type === 'trigger'
        } else if (isBlueprint) {
          // For blueprints, match by ID since blueprint nodes reference by ID, not name
          return (b as any).id === blockId && b.type === 'blueprint'
        }
        return false
      })

      const channels = block?.outputChannels?.map((channel: any) => channel.name) || ['default']

      // Use component or trigger name as node type if it exists in nodeTypes, otherwise use 'default'
      // For blueprints, always use 'default' since they don't have specific node types
      const nodeType = isBlueprint
        ? 'default'
        : blockName && nodeTypes[blockName as keyof typeof nodeTypes]
        ? blockName
        : 'default'

      // For blueprint nodes, use the blueprint's icon/color; for components/triggers, use from block
      const icon = isBlueprint ? block?.icon : block?.icon
      const color = isBlueprint ? block?.color : block?.color

      return {
        id: node.id,
        type: nodeType,
        data: {
          label: node.name,
          blockName,
          blockId,
          blockType: isComponent ? 'component' : isTrigger ? 'trigger' : 'blueprint',
          channels,
          configuration: node.configuration || {},
          metadata: node.metadata || {},
          onEdit: () => handleNodeEdit(node.id),
          onEmit: () => handleNodeEmit(node.id),
          icon: icon || (isBlueprint ? 'account_tree' : 'widgets'),
          color: color || 'blue',
        },
        position: node.position || { x: 0, y: 0 },
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

    // Check if we have saved positions
    const hasPositions = loadedNodes.some(node => node.position && (node.position.x !== 0 || node.position.y !== 0))

    if (hasPositions) {
      // Use saved positions
      setNodes(loadedNodes)
      setEdges(loadedEdges)
    } else {
      // Apply elk layout for workflows without saved positions
      getLayoutedElements(loadedNodes, loadedEdges).then(({ nodes: layoutedNodes, edges: layoutedEdges }) => {
        setNodes(layoutedNodes)
        setEdges(layoutedEdges)
      })
    }
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
      } else if (node.data.blockType === 'trigger') {
        return b.name === node.data.blockName && b.type === 'trigger'
      } else {
        // For blueprints, match by ID
        return (b as any).id === node.data.blockId && b.type === 'blueprint'
      }
    })
    setSelectedNode({
      id: node.id,
      name: node.data.label as string,
      isBlueprintNode: node.data.blockType === 'blueprint',
      nodeType: node.data.blockType as 'trigger' | 'component' | 'blueprint',
      componentName: (node.data.blockType === 'component' || node.data.blockType === 'trigger')
        ? node.data.blockName as string
        : undefined,
      componentLabel: block?.label,
      blueprintId: node.data.blockType === 'blueprint' ? node.data.blockId as string : undefined
    })
  }, [buildingBlocks])

  const handleEmit = async (channel: string, data: any) => {
    if (!emitModalNode) return

    await workflowsEmitNodeEvent(
      withOrganizationHeader({
        path: {
          workflowId: workflowId!,
          nodeId: emitModalNode.id
        },
        body: {
          channel,
          data
        }
      })
    )
  }

  const handleAddNode = () => {
    if (!selectedBlock || !nodeName.trim()) return

    // Filter configuration to only include visible fields
    const filteredConfiguration = filterVisibleConfiguration(
      nodeConfiguration,
      selectedBlock.configuration || []
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
      // Add new node
      const channels = selectedBlock?.outputChannels?.map((channel: any) => channel.name) || ['default']
      const newNodeId = generateNodeId(selectedBlock.name || 'node', nodeName.trim())

      // Use block name as node type if it exists in nodeTypes (works for both components and triggers)
      const nodeType = selectedBlock.name && nodeTypes[selectedBlock.name as keyof typeof nodeTypes]
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
          configuration: filteredConfiguration,
          onEdit: () => handleNodeEdit(newNodeId),
          onEmit: () => handleNodeEmit(newNodeId),
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
          type: node.data.blockType === 'component' ? 'TYPE_COMPONENT'
              : node.data.blockType === 'trigger' ? 'TYPE_TRIGGER'
              : 'TYPE_BLUEPRINT',
          configuration: node.data.configuration || {},
          position: {
            x: Math.round(node.position.x),
            y: Math.round(node.position.y),
          },
        }

        // Add either component, blueprint, or trigger reference directly on the node
        if (node.data.blockType === 'component') {
          baseNode.component = { name: node.data.blockName }
        } else if (node.data.blockType === 'trigger') {
          baseNode.trigger = { name: node.data.blockName }
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

  if (workflowLoading || componentsLoading || blueprintsLoading || triggersLoading) {
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
        <AlertCircle className="text-red-500 mb-4" size={32} />
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
            <ArrowLeft />
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
            <Activity />
            Event Execution Chains
          </Button>
          <Button
            onClick={handleSave}
            disabled={updateWorkflowMutation.isPending}
          >
            <Save />
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
                <PanelLeftClose size={24} />
              </Button>
              <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as 'triggers' | 'components' | 'blueprints')} className="flex-1">
                <TabsList className="w-full">
                  <TabsTrigger value="triggers" className="flex-1">
                    Triggers
                  </TabsTrigger>
                  <TabsTrigger value="components" className="flex-1">
                    Components
                  </TabsTrigger>
                  <TabsTrigger value="blueprints" className="flex-1">
                    Components
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </div>

            {/* Tab Content */}
            <div className="flex-1 overflow-hidden px-4">
              <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as 'triggers' | 'components' | 'blueprints')} className="flex-1 flex flex-col h-full">
                <TabsContent value="triggers" className="flex-1 overflow-y-auto text-left mt-4 data-[state=inactive]:hidden">
                  <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                    Click on a trigger to add it to your workflow
                  </div>
                  <ItemGroup>
                    {buildingBlocks.filter(b => b.type === 'trigger').map((block: BuildingBlock) => {
                      const IconComponent = resolveIcon(block.icon || 'zap')
                      const colorClass = getColorClass(block.color)

                      return (
                        <Item
                          key={`${block.type}-${block.name}`}
                          onClick={() => handleBlockClick(block)}
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                          size="sm"
                        >
                          <ItemMedia>
                            <IconComponent size={24} className={colorClass} />
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

                <TabsContent value="components" className="flex-1 overflow-y-auto text-left mt-4 data-[state=inactive]:hidden">
                  <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                    Click on a component to add it to your workflow
                  </div>
                  <ItemGroup>
                    {buildingBlocks.filter(b => b.type === 'component').map((block: BuildingBlock) => {
                      const IconComponent = resolveIcon(block.icon || 'boxes')
                      const colorClass = getColorClass(block.color)

                      return (
                        <Item
                          key={`${block.type}-${block.name}`}
                          onClick={() => handleBlockClick(block)}
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                          size="sm"
                        >
                          <ItemMedia>
                            <IconComponent size={24} className={colorClass} />
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
                    Click on a component to add it to your workflow
                  </div>
                  <ItemGroup>
                    {buildingBlocks.filter(b => b.type === 'blueprint').map((block: BuildingBlock) => {
                      const IconComponent = resolveIcon(block.icon || 'git-branch')
                      const colorClass = getColorClass(block.color)

                      return (
                        <Item
                          key={`${block.type}-${block.name}`}
                          onClick={() => handleBlockClick(block)}
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                          size="sm"
                        >
                          <ItemMedia>
                            <IconComponent size={24} className={colorClass} />
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
                  <Menu size={24} />
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
                  componentName={selectedNode.componentName}
                  componentLabel={selectedNode.componentLabel}
                  organizationId={organizationId!}
                  blueprintId={selectedNode.blueprintId}
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

      {/* Emit Event Modal */}
      {emitModalNode && (
        <EmitEventModal
          isOpen={true}
          onClose={() => setEmitModalNode(null)}
          nodeId={emitModalNode.id}
          nodeName={emitModalNode.name}
          workflowId={workflowId!}
          organizationId={organizationId!}
          channels={emitModalNode.channels}
          onEmit={handleEmit}
        />
      )}
    </div>
  )
}

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
import { useWorkflow, useUpdateWorkflow } from '../../hooks/useWorkflowData'
import { usePrimitives, useBlueprints } from '../../hooks/useBlueprintData'
import { Button } from '../../components/Button/button'
import { MaterialSymbol } from '../../components/MaterialSymbol/material-symbol'
import { Heading } from '../../components/Heading/heading'
import { Text } from '../../components/Text/text'
import { Input } from '../../components/Input/input'
import { Field, Label } from '../../components/Fieldset/fieldset'
import {
  Dialog,
  DialogTitle,
  DialogDescription,
  DialogBody,
  DialogActions
} from '../../components/Dialog/dialog'
import { showSuccessToast, showErrorToast } from '../../utils/toast'
import { IfNode } from '../blueprint/components/nodes/IfNode'
import { HttpNode } from '../blueprint/components/nodes/HttpNode'
import { FilterNode } from '../blueprint/components/nodes/FilterNode'
import { SwitchNode } from '../blueprint/components/nodes/SwitchNode'
import { ApprovalNode } from '../blueprint/components/nodes/ApprovalNode'
import { DefaultNode } from '../blueprint/components/nodes/DefaultNode'
import { WorkflowNodeSidebar } from '../../components/WorkflowNodeSidebar'

const nodeTypes = {
  if: IfNode,
  http: HttpNode,
  filter: FilterNode,
  switch: SwitchNode,
  approval: ApprovalNode,
  default: DefaultNode,
}

type BuildingBlock = {
  name: string
  description?: string
  type: 'primitive' | 'blueprint'
  branches?: { name: string }[]
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
  const [connectingFrom, setConnectingFrom] = useState<{ nodeId: string; branch: string } | null>(null)
  const [activeTab, setActiveTab] = useState<'primitives' | 'blueprints'>('primitives')
  const [selectedNode, setSelectedNode] = useState<{ id: string; name: string; isBlueprintNode: boolean; nodeType: string } | null>(null)

  // Fetch workflow, primitives, and blueprints
  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!)
  const { data: primitives = [], isLoading: primitivesLoading } = usePrimitives(organizationId!)
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId!)
  const updateWorkflowMutation = useUpdateWorkflow(organizationId!, workflowId!)

  const handleAddNodeFromBranch = useCallback((sourceNodeId: string, branch: string) => {
    setConnectingFrom({ nodeId: sourceNodeId, branch })
  }, [])

  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  // Combine primitives and blueprints into building blocks
  const buildingBlocks: BuildingBlock[] = [
    ...primitives.map((p: any) => ({ ...p, type: 'primitive' as const })),
    ...blueprints.map((b: any) => ({ ...b, type: 'blueprint' as const }))
  ]

  // Update nodes and edges when workflow data changes
  useEffect(() => {
    if (!workflow || buildingBlocks.length === 0) return

    const loadedNodes: Node[] = (workflow.nodes || []).map((node: any, index: number) => {
      const isPrimitive = node.refType === 'REF_TYPE_PRIMITIVE'
      const blockName = isPrimitive ? node.primitive?.name : node.blueprint?.name
      const block = buildingBlocks.find((b: BuildingBlock) =>
        b.name === blockName && b.type === (isPrimitive ? 'primitive' : 'blueprint')
      )

      const branches = block?.branches?.map((branch: any) => branch.name) || ['default']

      // Use primitive name as node type if it exists in nodeTypes, otherwise use 'default'
      const nodeType = isPrimitive && blockName && nodeTypes[blockName as keyof typeof nodeTypes]
        ? blockName
        : 'default'

      return {
        id: node.id,
        type: nodeType,
        data: {
          label: node.name,
          blockName,
          blockType: isPrimitive ? 'primitive' : 'blueprint',
          branches,
          configuration: node.configuration || {},
          onAddNode: handleAddNodeFromBranch,
        },
        position: { x: index * 250, y: 100 },
      }
    })

    const loadedEdges: Edge[] = (workflow.edges || []).map((edge: any, index: number) => ({
      id: `e${index}`,
      source: edge.sourceId,
      sourceHandle: edge.branch || 'default',
      target: edge.targetId,
      label: edge.branch,
      markerEnd: { type: MarkerType.ArrowClosed },
    }))

    setNodes(loadedNodes)
    setEdges(loadedEdges)
  }, [workflow, buildingBlocks.length, setNodes, setEdges, handleAddNodeFromBranch])

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, markerEnd: { type: MarkerType.ArrowClosed } }, eds))
    },
    [setEdges]
  )

  const handleBlockClick = (block: BuildingBlock) => {
    setSelectedBlock(block)
    setNodeName(block.name)
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
    setSelectedNode({
      id: node.id,
      name: node.data.label,
      isBlueprintNode: node.data.blockType === 'blueprint',
      nodeType: node.data.blockName
    })
  }, [])

  const handleNodeDoubleClick = useCallback((_: any, node: Node) => {
    const block = buildingBlocks.find((b: BuildingBlock) =>
      b.name === node.data.blockName && b.type === node.data.blockType
    )
    if (!block) return

    setEditingNodeId(node.id)
    setSelectedBlock(block)
    setNodeName(node.data.label)
    setNodeConfiguration(node.data.configuration || {})
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
      const branches = selectedBlock?.branches?.map((branch: any) => branch.name) || ['default']
      const newNodeId = generateNodeId(selectedBlock.name, nodeName.trim())

      // Use block name as node type if it exists in nodeTypes and is a primitive
      const nodeType = selectedBlock.type === 'primitive' &&
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
          blockName: selectedBlock.name,
          blockType: selectedBlock.type,
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
          markerEnd: { type: MarkerType.ArrowClosed },
        }
        setEdges((eds) => [...eds, newEdge])
        setConnectingFrom(null)
      }
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
          refType: node.data.blockType === 'primitive' ? 'REF_TYPE_PRIMITIVE' : 'REF_TYPE_BLUEPRINT',
          configuration: node.data.configuration || {},
        }

        // Add either primitive or blueprint reference directly on the node
        if (node.data.blockType === 'primitive') {
          baseNode.primitive = { name: node.data.blockName }
        } else {
          baseNode.blueprint = { name: node.data.blockName }
        }

        return baseNode
      })

      const workflowEdges = edges.map((edge) => ({
        sourceId: edge.source!,
        targetId: edge.target!,
        branch: edge.sourceHandle || edge.label as string || 'default',
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

  if (workflowLoading || primitivesLoading || blueprintsLoading) {
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
        <Button onClick={() => navigate(`/${organizationId}`)} className="mt-4">
          Go back to home
        </Button>
      </div>
    )
  }

  const displayedBlocks = activeTab === 'primitives'
    ? buildingBlocks.filter(b => b.type === 'primitive')
    : buildingBlocks.filter(b => b.type === 'blueprint')

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="bg-white dark:bg-zinc-900 border-b border-zinc-200 dark:border-zinc-800 p-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button plain onClick={() => navigate(`/${organizationId}`)}>
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
            color="blue"
            onClick={handleSave}
            disabled={updateWorkflowMutation.isPending}
            className="flex items-center gap-2"
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
            {/* Sidebar Header */}
            <div className="flex items-center justify-between px-4 pt-4 pb-0">
              <div className="flex items-center gap-3">
                <button
                  onClick={() => setIsSidebarOpen(false)}
                  aria-label="Close sidebar"
                  className="px-2 py-1 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded-md shadow-md hover:bg-gray-50 dark:hover:bg-zinc-800 transition-all duration-300 flex items-center gap-2"
                >
                  <MaterialSymbol name="menu_open" size="lg" className="text-gray-600 dark:text-zinc-300" />
                </button>
                <h2 className="text-md font-semibold text-gray-900 dark:text-zinc-100">
                  Building Blocks
                </h2>
              </div>
            </div>

            {/* Tab selector */}
            <div className="px-4 pt-4">
              <div className="flex gap-2 border-b border-zinc-200 dark:border-zinc-700">
                <button
                  onClick={() => setActiveTab('primitives')}
                  className={`px-4 py-2 text-sm font-medium transition-colors ${
                    activeTab === 'primitives'
                      ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400'
                      : 'text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:hover:text-zinc-300'
                  }`}
                >
                  Primitives
                </button>
                <button
                  onClick={() => setActiveTab('blueprints')}
                  className={`px-4 py-2 text-sm font-medium transition-colors ${
                    activeTab === 'blueprints'
                      ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400'
                      : 'text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:hover:text-zinc-300'
                  }`}
                >
                  Blueprints
                </button>
              </div>
            </div>

            {/* Sidebar Content */}
            <div className="flex-1 overflow-y-auto text-left p-4">
              <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                Click on a {activeTab === 'primitives' ? 'primitive' : 'blueprint'} to add it to your workflow
              </div>
              <div className="space-y-1">
                {displayedBlocks.map((block: BuildingBlock) => {
                  // Map block name to icon
                  const iconMap: Record<string, string> = {
                    if: 'alt_route',
                    http: 'http',
                    filter: 'filter_alt',
                    switch: 'settings_input_component',
                  }
                  const icon = block.type === 'primitive'
                    ? (iconMap[block.name] || 'widgets')
                    : 'account_tree'

                  return (
                    <div
                      key={`${block.type}-${block.name}`}
                      onClick={() => handleBlockClick(block)}
                      className="p-3 rounded-lg cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50 border border-gray-200 dark:border-zinc-700 transition-colors"
                    >
                      <div className="flex items-start gap-3">
                        <MaterialSymbol name={icon} size="lg" className="text-zinc-600 dark:text-zinc-400 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-sm text-gray-900 dark:text-zinc-100 mb-1">{block.name}</div>
                          {block.description && (
                            <div className="text-xs text-gray-500 dark:text-zinc-400">{block.description}</div>
                          )}
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        )}

        {/* React Flow Canvas */}
        <div className="flex-1 relative">
          {!isSidebarOpen && (
            <button
              onClick={() => setIsSidebarOpen(true)}
              className="absolute top-4 left-4 z-10 px-2 py-1 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded-md shadow-md hover:bg-gray-50 dark:hover:bg-zinc-800 transition-all duration-300 flex items-center gap-2"
              aria-label="Open sidebar"
            >
              <MaterialSymbol name="menu" size="lg" className="text-gray-600 dark:text-zinc-300" />
            </button>
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

        {/* Node Details Sidebar */}
        {selectedNode && (
          <WorkflowNodeSidebar
            workflowId={workflowId!}
            nodeId={selectedNode.id}
            nodeName={selectedNode.name}
            onClose={() => setSelectedNode(null)}
            isBlueprintNode={selectedNode.isBlueprintNode}
            nodeType={selectedNode.nodeType}
          />
        )}
      </div>

      {/* Add/Edit Node Modal */}
      <Dialog open={isAddNodeModalOpen} onClose={handleCloseModal} size="lg">
        <DialogTitle>{editingNodeId ? 'Edit' : 'Add'} {selectedBlock?.name}</DialogTitle>
        <DialogDescription>
          Configure the node for this {selectedBlock?.type}
        </DialogDescription>
        <DialogBody>
          <div className="space-y-4">
            <Field>
              <Label>Node Name *</Label>
              <Input
                type="text"
                value={nodeName}
                onChange={(e) => setNodeName(e.target.value)}
                placeholder="Enter a name for this node"
                autoFocus
              />
            </Field>

            {/* Dynamic configuration fields */}
            {selectedBlock?.configuration?.map((field: any) => (
              <Field key={field.name}>
                <Label>
                  {field.name} {field.required && '*'}
                </Label>
                {field.description && (
                  <p className="text-xs text-gray-500 dark:text-zinc-400 mb-1">{field.description}</p>
                )}
                {field.type === 'map' ? (
                  <textarea
                    className="w-full px-3 py-2 border border-gray-300 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-800 text-gray-900 dark:text-zinc-100"
                    value={nodeConfiguration[field.name] ? JSON.stringify(nodeConfiguration[field.name], null, 2) : ''}
                    onChange={(e) => {
                      try {
                        const parsed = e.target.value ? JSON.parse(e.target.value) : {}
                        setNodeConfiguration({ ...nodeConfiguration, [field.name]: parsed })
                      } catch {
                        // Invalid JSON, keep as string for now
                      }
                    }}
                    placeholder={`Enter ${field.name} as JSON`}
                    rows={3}
                  />
                ) : field.type === 'array' ? (
                  <textarea
                    className="w-full px-3 py-2 border border-gray-300 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-800 text-gray-900 dark:text-zinc-100"
                    value={nodeConfiguration[field.name] ? JSON.stringify(nodeConfiguration[field.name], null, 2) : ''}
                    onChange={(e) => {
                      try {
                        const parsed = e.target.value ? JSON.parse(e.target.value) : []
                        setNodeConfiguration({ ...nodeConfiguration, [field.name]: parsed })
                      } catch {
                        // Invalid JSON, keep as string for now
                      }
                    }}
                    placeholder={`Enter ${field.name} as JSON array`}
                    rows={4}
                  />
                ) : field.type === 'number' ? (
                  <Input
                    type="number"
                    value={nodeConfiguration[field.name] ?? ''}
                    onChange={(e) => {
                      const value = e.target.value === '' ? undefined : Number(e.target.value)
                      setNodeConfiguration({ ...nodeConfiguration, [field.name]: value })
                    }}
                    placeholder={`Enter ${field.name}`}
                  />
                ) : (
                  <Input
                    type="text"
                    value={nodeConfiguration[field.name] || ''}
                    onChange={(e) => setNodeConfiguration({ ...nodeConfiguration, [field.name]: e.target.value })}
                    placeholder={`Enter ${field.name}`}
                  />
                )}
              </Field>
            ))}
          </div>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={handleCloseModal}>
            Cancel
          </Button>
          <Button
            color="blue"
            onClick={handleAddNode}
            disabled={!nodeName.trim()}
          >
            {editingNodeId ? 'Save' : 'Add Node'}
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  )
}

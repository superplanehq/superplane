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
  Position,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useBlueprint, useUpdateBlueprint } from '../../hooks/useBlueprintData'
import { usePrimitives } from '../../hooks/useBlueprintData'
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
import { IfNode } from './components/nodes/IfNode'
import { HttpNode } from './components/nodes/HttpNode'
import { FilterNode } from './components/nodes/FilterNode'
import { SwitchNode } from './components/nodes/SwitchNode'
import { DefaultNode } from './components/nodes/DefaultNode'

const nodeTypes = {
  if: IfNode,
  http: HttpNode,
  filter: FilterNode,
  switch: SwitchNode,
  default: DefaultNode,
}

export const Blueprint = () => {
  const { organizationId, blueprintId } = useParams<{ organizationId: string; blueprintId: string }>()
  const navigate = useNavigate()
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)
  const [isAddNodeModalOpen, setIsAddNodeModalOpen] = useState(false)
  const [selectedPrimitive, setSelectedPrimitive] = useState<any>(null)
  const [nodeName, setNodeName] = useState('')
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, any>>({})
  const [editingNodeId, setEditingNodeId] = useState<string | null>(null)
  const [connectingFrom, setConnectingFrom] = useState<{ nodeId: string; branch: string } | null>(null)

  // Fetch blueprint and primitives
  const { data: blueprint, isLoading: blueprintLoading } = useBlueprint(organizationId!, blueprintId!)
  const { data: primitives = [], isLoading: primitivesLoading } = usePrimitives(organizationId!)
  const updateBlueprintMutation = useUpdateBlueprint(organizationId!, blueprintId!)

  const handleAddNodeFromBranch = useCallback((sourceNodeId: string, branch: string) => {
    setConnectingFrom({ nodeId: sourceNodeId, branch })
  }, [])

  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  // Update nodes and edges when blueprint or primitives data changes
  useEffect(() => {
    if (!blueprint || primitives.length === 0) return

    const loadedNodes: Node[] = (blueprint.nodes || []).map((node: any, index: number) => {
      const primitive = primitives.find((p: any) => p.name === node.primitive?.name)
      const branches = primitive?.branches?.map((branch: any) => branch.name) || ['default']
      const primitiveName = node.primitive?.name

      // Use primitive name as node type if it exists in nodeTypes, otherwise use 'default'
      const nodeType = primitiveName && nodeTypes[primitiveName as keyof typeof nodeTypes] ? primitiveName : 'default'

      return {
        id: node.id,
        type: nodeType,
        data: {
          label: node.name,
          primitive: primitiveName,
          branches,
          configuration: node.configuration || {},
          onAddNode: handleAddNodeFromBranch,
        },
        position: { x: index * 250, y: 100 }, // Left-to-right layout
      }
    })

    const loadedEdges: Edge[] = (blueprint.edges || []).map((edge: any, index: number) => ({
      id: `e${index}`,
      source: edge.sourceId,
      sourceHandle: edge.branch || 'default',
      target: edge.targetId,
      label: edge.branch,
      markerEnd: { type: MarkerType.ArrowClosed },
    }))

    setNodes(loadedNodes)
    setEdges(loadedEdges)
  }, [blueprint, primitives, setNodes, setEdges, handleAddNodeFromBranch])

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, markerEnd: { type: MarkerType.ArrowClosed } }, eds))
    },
    [setEdges]
  )

  const handlePrimitiveClick = (primitive: any) => {
    setSelectedPrimitive(primitive)
    setNodeName(primitive.name)
    setNodeConfiguration({})
    setIsAddNodeModalOpen(true)
  }

  const generateNodeId = (primitiveName: string, nodeName: string) => {
    const randomChars = Math.random().toString(36).substring(2, 8)
    const sanitizedPrimitive = primitiveName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    return `${sanitizedPrimitive}-${sanitizedName}-${randomChars}`
  }

  const handleNodeDoubleClick = useCallback((_: any, node: Node) => {
    const primitive = primitives.find((p: any) => p.name === node.data.primitive)
    if (!primitive) return

    setEditingNodeId(node.id)
    setSelectedPrimitive(primitive)
    setNodeName(node.data.label)
    setNodeConfiguration(node.data.configuration || {})
    setIsAddNodeModalOpen(true)
  }, [primitives])

  const handleAddNode = () => {
    if (!selectedPrimitive || !nodeName.trim()) return

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
      const branches = selectedPrimitive?.branches?.map((branch: any) => branch.name) || ['default']
      const newNodeId = generateNodeId(selectedPrimitive.name, nodeName.trim())

      // Use primitive name as node type if it exists in nodeTypes, otherwise use 'default'
      const nodeType = selectedPrimitive.name && nodeTypes[selectedPrimitive.name as keyof typeof nodeTypes]
        ? selectedPrimitive.name
        : 'default'

      const newNode: Node = {
        id: newNodeId,
        type: nodeType,
        position: { x: nodes.length * 250, y: 100 },
        data: {
          label: nodeName.trim(),
          primitive: selectedPrimitive.name,
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
    setSelectedPrimitive(null)
    setNodeName('')
    setNodeConfiguration({})
    setEditingNodeId(null)
  }

  const handleCloseModal = () => {
    setIsAddNodeModalOpen(false)
    setSelectedPrimitive(null)
    setNodeName('')
    setNodeConfiguration({})
    setEditingNodeId(null)
  }

  const handleSave = async () => {
    try {
      const blueprintNodes = nodes.map((node) => ({
        id: node.id,
        name: node.data.label,
        refType: 'REF_TYPE_PRIMITIVE',
        primitive: {
          name: node.data.primitive,
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

  if (blueprintLoading || primitivesLoading) {
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
        <Button onClick={() => navigate(`/${organizationId}`)} className="mt-4">
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
          <Button plain onClick={() => navigate(`/${organizationId}`)}>
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
            color="blue"
            onClick={handleSave}
            disabled={updateBlueprintMutation.isPending}
            className="flex items-center gap-2"
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
                <button
                  onClick={() => setIsSidebarOpen(false)}
                  aria-label="Close sidebar"
                  className="px-2 py-1 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded-md shadow-md hover:bg-gray-50 dark:hover:bg-zinc-800 transition-all duration-300 flex items-center gap-2"
                >
                  <MaterialSymbol name="menu_open" size="lg" className="text-gray-600 dark:text-zinc-300" />
                </button>
                <h2 className="text-md font-semibold text-gray-900 dark:text-zinc-100">
                  Primitives
                </h2>
              </div>
            </div>

            {/* Sidebar Content */}
            <div className="flex-1 overflow-y-auto text-left p-4">
              <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                Click on a primitive to add it to your blueprint
              </div>
              <div className="space-y-1">
                {primitives.map((primitive: any) => {
                  // Map primitive name to icon
                  const iconMap: Record<string, string> = {
                    if: 'alt_route',
                    http: 'http',
                    filter: 'filter_alt',
                    switch: 'settings_input_component',
                  }
                  const icon = iconMap[primitive.name] || 'widgets'

                  return (
                    <div
                      key={primitive.name}
                      onClick={() => handlePrimitiveClick(primitive)}
                      className="p-3 rounded-lg cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50 border border-gray-200 dark:border-zinc-700 transition-colors"
                    >
                      <div className="flex items-start gap-3">
                        <MaterialSymbol name={icon} size="lg" className="text-zinc-600 dark:text-zinc-400 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-sm text-gray-900 dark:text-zinc-100 mb-1">{primitive.name}</div>
                          {primitive.description && (
                            <div className="text-xs text-gray-500 dark:text-zinc-400">{primitive.description}</div>
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
      <Dialog open={isAddNodeModalOpen} onClose={handleCloseModal} size="lg">
        <DialogTitle>{editingNodeId ? 'Edit' : 'Add'} {selectedPrimitive?.name}</DialogTitle>
        <DialogDescription>
          Configure the node for this primitive
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
            {selectedPrimitive?.configuration?.map((field: any) => (
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

import { useState, useCallback, useEffect, useRef } from 'react'
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
import { AlertCircle } from 'lucide-react'
import { CustomComponentBuilderPage } from '../../ui/CustomComponentBuilderPage'
import type { BreadcrumbItem, NewNodeData } from '../../ui/CustomComponentBuilderPage'
import { BlockData } from '../../ui/CanvasPage/Block'
import { Heading } from '../../components/Heading/heading'
import { ComponentsComponent, ComponentsNode } from '../../api-client'
import { showSuccessToast, showErrorToast } from '../../utils/toast'
import { filterVisibleConfiguration } from '../../utils/components'
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
  const expression = node.configuration?.expression
  // Add type-specific props based on component type
  switch (blockType) {
    case 'if':
      baseData.if = {
        title: node.name,
        expression,
        trueEvent: {
          eventTitle: "No events received yet",
          eventState: "neutral" as const
        },
        falseEvent: {
          eventTitle: "No events received yet",
          eventState: "neutral" as const
        },
        trueSectionLabel: "TRUE",
        falseSectionLabel: "FALSE",
        collapsed: false,
      }
      break
    case 'filter':
      baseData.filter = {
        title: node.name,
        expression,
        lastEvent: {
          eventTitle: "No events received yet",
          eventState: "neutral" as const
        },
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
        lastEvent: {
          eventTitle: "No events received yet",
          eventState: "neutral" as const
        },
        collapsed: false,
      }
      break
  }

  return baseData
}

export const CustomComponent = () => {
  const { organizationId, blueprintId } = useParams<{ organizationId: string; blueprintId: string }>()
  const navigate = useNavigate()
  const [blueprintConfiguration, setBlueprintConfiguration] = useState<any[]>([])
  const [blueprintOutputChannels, setBlueprintOutputChannels] = useState<any[]>([])
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

  // Update nodes and edges when blueprint or components data changes
  useEffect(() => {
    if (!blueprint || components.length === 0) return

    const allNodes: Node[] = (blueprint.nodes || []).map((node: ComponentsNode) => {
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

  // Use refs to access latest values without causing re-renders
  const nodesRef = useRef(nodes)
  const componentsRef = useRef(components)

  // Keep refs updated
  useEffect(() => {
    nodesRef.current = nodes
  }, [nodes])

  useEffect(() => {
    componentsRef.current = components
  }, [components])

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

  const generateNodeId = (componentName: string, nodeName: string) => {
    const randomChars = Math.random().toString(36).substring(2, 8)
    const sanitizedComponent = componentName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, '-')
    return `${sanitizedComponent}-${sanitizedName}-${randomChars}`
  }

  const getNodeEditData = useCallback((nodeId: string) => {
    const node = nodesRef.current.find((n) => n.id === nodeId)
    if (!node) return null

    const component = componentsRef.current.find((p: any) => p.name === (node.data as any)._originalComponent)
    if (!component) return null

    return {
      nodeId: node.id,
      nodeName: (node.data as any).label as string,
      configuration: (node.data as any)._originalConfiguration || {},
      configurationFields: component.configuration || [],
    }
  }, [])

  const handleNodeConfigurationSave = useCallback((
    nodeId: string,
    configuration: Record<string, any>,
    nodeName: string
  ) => {
    const node = nodesRef.current.find((n) => n.id === nodeId)
    if (!node) return

    const component = componentsRef.current.find((p: any) => p.name === (node.data as any)._originalComponent)
    if (!component) return

    // Filter configuration to only include visible fields
    const filteredConfiguration = filterVisibleConfiguration(
      configuration,
      component.configuration || []
    )

    // Update existing node
    setNodes((nds) =>
      nds.map((n) => {
        if (n.id !== nodeId) return n

        const nodeData = n.data as any
        const updatedData = {
          ...nodeData,
          label: nodeName.trim(),
          _originalConfiguration: filteredConfiguration,
        }

        // Update the title and expression in the type-specific props
        if (nodeData.if) {
          updatedData.if = {
            ...nodeData.if,
            title: nodeName.trim(),
            expression: filteredConfiguration.expression
          }
        }
        if (nodeData.filter) {
          updatedData.filter = {
            ...nodeData.filter,
            title: nodeName.trim(),
            expression: filteredConfiguration.expression
          }
        }
        if (nodeData.approval) {
          updatedData.approval = { ...nodeData.approval, title: nodeName.trim() }
        }
        if (nodeData.noop) {
          updatedData.noop = { ...nodeData.noop, title: nodeName.trim() }
        }

        return {
          ...n,
          data: updatedData,
        }
      })
    )
  }, [])

  const handleNodeAdd = useCallback((newNodeData: NewNodeData) => {
    const component = componentsRef.current.find((c: ComponentsComponent) => c.name === newNodeData.buildingBlock.name)
    if (!component) return

    // Filter configuration to only include visible fields
    const filteredConfiguration = filterVisibleConfiguration(
      newNodeData.configuration,
      component.configuration || []
    )

    // Add new node
    const newNodeId = generateNodeId(component.name!, newNodeData.nodeName.trim())
    const mockNode = {
      component: { name: component.name },
      name: newNodeData.nodeName.trim(),
      configuration: filteredConfiguration
    }
    const blockData = createBlockData(mockNode, component)

    const newNode: Node = {
      id: newNodeId,
      type: 'default',
      position: newNodeData.position || { x: nodesRef.current.length * 250, y: 100 },
      data: {
        ...blockData,
        _originalComponent: component.name,
        _originalConfiguration: filteredConfiguration,
      },
    }
    setNodes((nds) => [...nds, newNode])
  }, [])

  const handleNodeDelete = useCallback((nodeId: string) => {
    setNodes((nds) => nds.filter((n) => n.id !== nodeId))
    setEdges((eds) => eds.filter((e) => e.source !== nodeId && e.target !== nodeId))
  }, [])


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

      showSuccessToast('Component saved successfully')
    } catch (error: any) {
      console.error('Error saving component:', error)
      const errorMessage = error?.response?.data?.message || error?.message || 'Failed to save component'
      showErrorToast(errorMessage)
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

  const breadcrumbs: BreadcrumbItem[] = [
    { label: 'Components', onClick: () => navigate(`/${organizationId}`) },
    { label: blueprintName, iconSlug: blueprintIcon, iconColor: `text-${blueprintColor}-600` },
  ]

  return (
    <>
      <CustomComponentBuilderPage
        customComponentName={blueprintName}
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
        outputChannels={blueprintOutputChannels}
        onOutputChannelsChange={setBlueprintOutputChannels}
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeDelete={handleNodeDelete}
        getNodeEditData={getNodeEditData}
        onNodeConfigurationSave={handleNodeConfigurationSave}
        onNodeAdd={handleNodeAdd}
        organizationId={organizationId}
        components={components}
        onSave={handleSave}
        isSaving={updateBlueprintMutation.isPending}
      />
    </>
  )
}

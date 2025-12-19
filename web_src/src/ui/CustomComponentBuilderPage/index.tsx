import {
  Background,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Connection,
  type Edge,
  type Node,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ZoomSlider } from "@/components/zoom-slider";
import { NodeSearch } from "@/components/node-search";
import "./blueprint-canvas-reset.css";

import {
  ComponentsComponent,
  ConfigurationField,
  SuperplaneBlueprintsOutputChannel,
  AuthorizationDomainType,
  ApplicationsApplicationDefinition,
  OrganizationsAppInstallation,
  ComponentsAppInstallationRef,
} from "@/api-client";
import { BuildingBlock, BuildingBlockCategory, BuildingBlocksSidebar } from "../BuildingBlocksSidebar";
import { Block, BlockData } from "../CanvasPage/Block";
import { CustomEdge } from "../CanvasPage/CustomEdge";
import { BreadcrumbItem, Header } from "../CanvasPage/Header";
import {
  BlueprintMetadata,
  CustomComponentConfigurationSidebar,
  OutputChannel,
} from "../CustomComponentConfigurationSidebar";
import { ComponentSidebar } from "../componentSidebar";
import { getBackgroundColorClass } from "@/utils/colors";
import { ComponentBaseProps } from "../componentBase";
import { buildBuildingBlockCategories } from "../buildingBlocks";
import { ConfigurationFieldModal } from "./ConfigurationFieldModal";
import { OutputChannelConfigurationModal } from "./OutputChannelConfigurationModal";

export interface NodeEditData {
  nodeId: string;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, any>;
  configurationFields: ConfigurationField[];
  appName?: string;
  appInstallationRef?: any;
}

export interface NewNodeData {
  icon?: string;
  buildingBlock: BuildingBlock;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, any>;
  position?: { x: number; y: number };
  sourceConnection?: { nodeId: string; handleId: string | null };
  appName?: string;
  appInstallationRef?: ComponentsAppInstallationRef;
}

export interface CustomComponentBuilderPageProps {
  // Custom Component data
  customComponentName: string;
  breadcrumbs?: BreadcrumbItem[];
  metadata: BlueprintMetadata;
  onMetadataChange: (metadata: BlueprintMetadata) => void;

  // Configuration
  configurationFields: ConfigurationField[];
  onConfigurationFieldsChange: (fields: ConfigurationField[]) => void;

  // Output channels
  outputChannels: OutputChannel[];
  onOutputChannelsChange: (channels: OutputChannel[]) => void;

  // Canvas
  nodes: Node[];
  edges: Edge[];
  onNodesChange: (changes: any) => void;
  onEdgesChange: (changes: any) => void;
  onConnect: (connection: Connection) => void;
  onNodeDoubleClick?: (event: any, node: Node) => void;
  onNodeClick?: (nodeId: string) => void;
  onNodeDelete?: (nodeId: string) => void;
  onNodeDuplicate?: (nodeId: string) => void;

  // Node configuration
  getNodeEditData?: (nodeId: string) => NodeEditData | null;
  onNodeConfigurationSave?: (
    nodeId: string,
    configuration: Record<string, any>,
    nodeName: string,
    appInstallationRef?: ComponentsAppInstallationRef,
  ) => void;
  onNodeAdd?: (newNodeData: NewNodeData) => void;
  organizationId?: string;

  // Building blocks
  components: ComponentsComponent[];
  availableApplications?: ApplicationsApplicationDefinition[];
  installedApplications?: OrganizationsAppInstallation[];

  // Template node helpers
  onAddTemplateNode?: (node: Node) => void;
  onRemoveTemplateNode?: (nodeId: string) => void;

  // Drag-edge-to-empty-space functionality
  templateNodeId?: string | null;
  newNodeData?: NewNodeData | null;
  isBuildingBlocksSidebarOpen?: boolean;
  onBuildingBlocksSidebarToggle?: (open: boolean) => void;
  onConnectionDropInEmptySpace?: (
    position: { x: number; y: number },
    sourceConnection: { nodeId: string; handleId: string | null },
  ) => void;
  onPendingConnectionNodeClick?: (nodeId: string) => void;
  onTemplateNodeClick?: (nodeId: string) => void;
  onBuildingBlockClick?: (block: BuildingBlock) => void;
  onCancelTemplate?: () => void;

  // Actions
  onSave: () => void;
  isSaving?: boolean;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  // Undo functionality
  onUndo?: () => void;
  canUndo?: boolean;
}

/*
 * nodeTypes must be defined outside of the component to prevent
 * react-flow from remounting the node types on every render.
 */
const nodeTypes = {
  default: (nodeProps: { data: BlockData & { _callbacksRef?: any }; id: string; selected?: boolean }) => {
    const { _callbacksRef, ...blockData } = nodeProps.data;
    const callbacks = _callbacksRef?.current;

    if (!callbacks) {
      return <Block data={blockData} nodeId={nodeProps.id} selected={nodeProps.selected} />;
    }

    return (
      <Block
        data={blockData}
        nodeId={nodeProps.id}
        selected={nodeProps.selected}
        onEdit={() => callbacks.onEdit?.current?.(nodeProps.id)}
        onDelete={() => callbacks.onDelete?.current?.(nodeProps.id)}
        onDuplicate={() => callbacks.onDuplicate?.current?.(nodeProps.id)}
      />
    );
  },
};

// Canvas content component with ReactFlow hooks - defined outside to prevent re-creation
function CanvasContent({
  nodes,
  edges,
  edgeTypes,
  onNodesChange,
  onEdgesChange,
  onConnect,
  onNodeDoubleClick,
  onNodeClick,
  onBuildingBlockDrop,
  onZoomChange,
  onEdgeMouseEnter,
  onEdgeMouseLeave,
  onConnectStart,
  onConnectEnd,
  onConnectionDropInEmptySpace,
  connectionCompletedRef,
  connectingFromRef,
  templateNodeId,
}: {
  nodes: Node[];
  edges: Edge[];
  edgeTypes: any;
  onNodesChange: (changes: any) => void;
  onEdgesChange: (changes: any) => void;
  onConnect: (connection: Connection) => void;
  onNodeDoubleClick?: (event: any, node: Node) => void;
  onNodeClick?: (nodeId: string) => void;
  onBuildingBlockDrop?: (block: BuildingBlock, position?: { x: number; y: number }) => void;
  onZoomChange?: (zoom: number) => void;
  onEdgeMouseEnter?: (event: React.MouseEvent, edge: any) => void;
  onEdgeMouseLeave?: () => void;
  onConnectStart?: (
    event: any,
    params: { nodeId: string | null; handleId: string | null; handleType: "source" | "target" | null },
  ) => void;
  onConnectEnd?: () => void;
  onConnectionDropInEmptySpace?: (
    position: { x: number; y: number },
    sourceConnection: { nodeId: string; handleId: string | null },
  ) => void;
  connectionCompletedRef?: React.MutableRefObject<boolean>;
  connectingFromRef?: React.MutableRefObject<{
    nodeId: string;
    handleId: string | null;
    handleType: "source" | "target" | null;
  } | null>;
  templateNodeId?: string | null;
}) {
  const { fitView, screenToFlowPosition, getViewport } = useReactFlow();

  const handleDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }, []);

  const handleDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const blockData = event.dataTransfer.getData("application/reactflow");
      if (!blockData || !onBuildingBlockDrop) {
        return;
      }

      try {
        const block: BuildingBlock = JSON.parse(blockData);
        const cursorPosition = screenToFlowPosition({
          x: event.clientX,
          y: event.clientY,
        });

        // Adjust position to place node exactly where preview was shown
        const nodeWidth = 420;
        const cursorOffsetY = 30;
        const position = {
          x: cursorPosition.x - nodeWidth / 2,
          y: cursorPosition.y - cursorOffsetY,
        };

        onBuildingBlockDrop(block, position);
      } catch (error) {
        console.error("Failed to parse building block data:", error);
      }
    },
    [onBuildingBlockDrop, screenToFlowPosition],
  );

  const handleMove = useCallback(
    (_event: any, viewport: { x: number; y: number; zoom: number }) => {
      if (onZoomChange) {
        onZoomChange(viewport.zoom);
      }
    },
    [onZoomChange],
  );

  const handleNodeClick = useCallback(
    (_event: any, node: Node) => {
      // Allow clicking on the same template node, other pending connection nodes, or other template nodes
      // But block clicking on regular nodes when a configured template is being created
      const clickedIsPending = (node.data as any)?.isPendingConnection;
      const clickedIsTemplate = (node.data as any)?.isTemplate && !clickedIsPending;

      if (templateNodeId && node.id !== templateNodeId) {
        // Check if current template is configured (not just pending)
        const currentTemplate = nodes.find((n) => n.id === templateNodeId);
        const currentIsConfigured =
          currentTemplate &&
          (currentTemplate.data as any)?.isTemplate &&
          !(currentTemplate.data as any)?.isPendingConnection;

        // Block if: there's a configured template AND we're not clicking on a pending node or template node
        if (currentIsConfigured && !clickedIsPending && !clickedIsTemplate) {
          return;
        }
      }

      onNodeClick?.(node.id);
    },
    [onNodeClick, templateNodeId, nodes],
  );

  const handlePaneClick = useCallback(() => {
    // do not close sidebar while we are creating a new component
    if (templateNodeId) return;
    // Could add pane click handling here if needed
  }, [templateNodeId]);

  const handleConnect = useCallback(
    (connection: Connection) => {
      // Mark that a connection was successfully completed
      if (connectionCompletedRef) {
        connectionCompletedRef.current = true;
      }
      onConnect(connection);
    },
    [onConnect, connectionCompletedRef],
  );

  const handleConnectEndInternal = useCallback(
    (event: MouseEvent | TouchEvent) => {
      const currentConnectingFrom = connectingFromRef?.current;

      if (currentConnectingFrom && connectionCompletedRef && !connectionCompletedRef.current) {
        const mouseEvent = event as MouseEvent;
        const canvasPosition = screenToFlowPosition({
          x: mouseEvent.clientX,
          y: mouseEvent.clientY,
        });

        if (onConnectionDropInEmptySpace) {
          onConnectionDropInEmptySpace(canvasPosition, currentConnectingFrom);
        }
      }

      // Call the parent's onConnectEnd
      if (onConnectEnd) {
        onConnectEnd();
      }
    },
    [screenToFlowPosition, onConnectionDropInEmptySpace, onConnectEnd, connectionCompletedRef, connectingFromRef],
  );

  // Initialize: fit to view on mount with zoom constraints
  useEffect(() => {
    fitView({ maxZoom: 1.0, padding: 0.5 });

    if (onZoomChange) {
      const viewport = getViewport();
      onZoomChange(viewport.zoom);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      edgeTypes={edgeTypes}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onConnect={handleConnect}
      onConnectStart={onConnectStart}
      onConnectEnd={handleConnectEndInternal}
      onNodeClick={handleNodeClick}
      onNodeDoubleClick={onNodeDoubleClick}
      onPaneClick={handlePaneClick}
      onEdgeMouseEnter={onEdgeMouseEnter}
      onEdgeMouseLeave={onEdgeMouseLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      onMove={handleMove}
      minZoom={0.4}
      maxZoom={1.5}
      zoomOnScroll={true}
      zoomOnPinch={true}
      zoomOnDoubleClick={false}
      panOnScroll={true}
      panOnDrag={true}
      selectionOnDrag={false}
      panOnScrollSpeed={0.8}
      nodesDraggable={true}
      nodesConnectable={true}
      elementsSelectable={true}
    >
      <Background gap={8} size={2} bgColor="#F1F5F9" color="#d9d9d9ff" />
      <ZoomSlider position="bottom-left" orientation="horizontal">
        <NodeSearch />
      </ZoomSlider>
    </ReactFlow>
  );
}

export function CustomComponentBuilderPage(props: CustomComponentBuilderPageProps) {
  const [isLeftSidebarOpen, setIsLeftSidebarOpen] = useState(true);
  const [isRightSidebarOpen, setIsRightSidebarOpen] = useState(false);
  const [isNodeSidebarOpen, setIsNodeSidebarOpen] = useState(false);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [canvasZoom, setCanvasZoom] = useState(1);

  // Use parent's state if provided (delegated mode), otherwise use local state
  const [localTemplateNodeId, setLocalTemplateNodeId] = useState<string | null>(null);
  const [localNewNodeData, setLocalNewNodeData] = useState<NewNodeData | null>(null);
  const templateNodeId = props.templateNodeId !== undefined ? props.templateNodeId : localTemplateNodeId;
  const newNodeData = props.newNodeData !== undefined ? props.newNodeData : localNewNodeData;
  const setTemplateNodeId = props.templateNodeId !== undefined ? () => {} : setLocalTemplateNodeId;
  const setNewNodeData = props.newNodeData !== undefined ? () => {} : setLocalNewNodeData;

  // In delegated mode, open the ComponentSidebar when newNodeData is set, close when cleared
  useEffect(() => {
    if (props.templateNodeId !== undefined && props.newNodeData && templateNodeId) {
      setIsNodeSidebarOpen(true);
      setSelectedNodeId(templateNodeId);
    } else if (props.newNodeData === null) {
      setIsNodeSidebarOpen(false);
      setSelectedNodeId(null);
    }
  }, [props.templateNodeId, props.newNodeData, templateNodeId]);

  // Modal state management
  const [isConfigFieldModalOpen, setIsConfigFieldModalOpen] = useState(false);
  const [editingConfigFieldIndex, setEditingConfigFieldIndex] = useState<number | null>(null);
  const [isOutputChannelModalOpen, setIsOutputChannelModalOpen] = useState(false);
  const [editingOutputChannelIndex, setEditingOutputChannelIndex] = useState<number | null>(null);

  // Modal handlers
  const handleAddConfigField = useCallback(() => {
    setEditingConfigFieldIndex(null);
    setIsConfigFieldModalOpen(true);
  }, []);

  const handleEditConfigField = useCallback((index: number) => {
    setEditingConfigFieldIndex(index);
    setIsConfigFieldModalOpen(true);
  }, []);

  const handleSaveConfigField = useCallback(
    (field: ConfigurationField) => {
      if (editingConfigFieldIndex !== null) {
        // Update existing field
        const newFields = [...props.configurationFields];
        newFields[editingConfigFieldIndex] = field as ConfigurationField;
        props.onConfigurationFieldsChange(newFields);
      } else {
        // Add new field
        props.onConfigurationFieldsChange([...props.configurationFields, field as ConfigurationField]);
      }
      setIsConfigFieldModalOpen(false);
      setEditingConfigFieldIndex(null);
    },
    [editingConfigFieldIndex, props.configurationFields, props.onConfigurationFieldsChange],
  );

  const handleAddOutputChannel = useCallback(() => {
    setEditingOutputChannelIndex(null);
    setIsOutputChannelModalOpen(true);
  }, []);

  const handleEditOutputChannel = useCallback((index: number) => {
    setEditingOutputChannelIndex(index);
    setIsOutputChannelModalOpen(true);
  }, []);

  const handleSaveOutputChannel = useCallback(
    (outputChannel: SuperplaneBlueprintsOutputChannel) => {
      if (editingOutputChannelIndex !== null) {
        // Update existing output channel
        const newChannels = [...props.outputChannels];
        newChannels[editingOutputChannelIndex] = outputChannel as OutputChannel;
        props.onOutputChannelsChange(newChannels);
      } else {
        // Add new output channel
        props.onOutputChannelsChange([...props.outputChannels, outputChannel as OutputChannel]);
      }
      setIsOutputChannelModalOpen(false);
      setEditingOutputChannelIndex(null);
    },
    [editingOutputChannelIndex, props.outputChannels, props.onOutputChannelsChange],
  );

  // Node configuration handlers
  const handleNodeEdit = useCallback(
    (nodeId: string) => {
      if (!isNodeSidebarOpen || selectedNodeId !== nodeId) {
        setIsNodeSidebarOpen(true);
        setSelectedNodeId(nodeId);
      }
      // Always shows settings tab - no tab switching needed
    },
    [isNodeSidebarOpen, selectedNodeId],
  );

  // Get editing data for the currently selected node
  const editingNodeData = useMemo(() => {
    if (selectedNodeId && isNodeSidebarOpen && props.getNodeEditData) {
      return props.getNodeEditData(selectedNodeId);
    }
    return null;
  }, [selectedNodeId, isNodeSidebarOpen, props.getNodeEditData]);

  const handleBuildingBlockDrop = useCallback(
    (block: BuildingBlock, position?: { x: number; y: number }) => {
      if (templateNodeId) {
        return;
      }

      // Generate unique template node ID
      const newTemplateId = `template_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

      // Deselect all existing nodes first
      props.onNodesChange(
        props.nodes.map((node) => ({
          type: "select",
          id: node.id,
          selected: false,
        })),
      );

      // Create template node data
      const templateNode: Node = {
        id: newTemplateId,
        type: "default",
        position: position || { x: props.nodes.length * 250, y: 100 },
        selected: true,
        data: {
          type: "component",
          label: block.label || block.name || "New Component",
          state: "pending" as const,
          outputChannels: ["default"],
          component: {
            title: block.label || block.name || "New Component",
            headerColor: "#e5e7eb",
            iconSlug: block.icon,
            iconColor: "text-gray-800",
            collapsedBackground: getBackgroundColorClass("white"),
            hideActionsButton: true,
            includeEmptyState: true,
            emptyStateTitle: block.type === "trigger" ? "Waiting for the first event" : undefined,
          } as ComponentBaseProps,
          isTemplate: true,
          buildingBlock: block,
          tempConfiguration: {},
          tempNodeName: block.name || "",
          _originalComponent: block.name,
          _originalConfiguration: {},
        } as any,
      };

      // Add the template node using the helper callback
      if (props.onAddTemplateNode) {
        props.onAddTemplateNode(templateNode);
      }

      // In delegated mode, we need to notify parent to set template state
      // In non-delegated mode, set local state
      if (props.templateNodeId !== undefined) {
        // Delegated mode - parent manages state via onAddTemplateNode callback
        // The parent's onAddTemplateNode will set both templateNodeId and newNodeData
        // Then the useEffect will open the ComponentSidebar automatically
      } else {
        // Non-delegated mode - manage state locally
        setTemplateNodeId(newTemplateId);
        setNewNodeData({
          icon: block.icon || "circle-off",
          buildingBlock: block,
          nodeName: block.name || "",
          displayLabel: block.label || block.name || "",
          configuration: {},
          position,
          appName: block.appName,
        });

        // Open sidebar in non-delegated mode
        setIsNodeSidebarOpen(true);
        setSelectedNodeId(newTemplateId);
      }

      // Close building blocks sidebar after dropping a block
      if (props.onBuildingBlocksSidebarToggle) {
        props.onBuildingBlocksSidebarToggle(false);
      } else {
        setIsLeftSidebarOpen(false);
      }
    },
    [templateNodeId, props],
  );

  const handleSaveConfiguration = useCallback(
    (configuration: Record<string, any>, nodeName: string, appInstallationRef?: any) => {
      if (templateNodeId && newNodeData) {
        // This is a template node being saved
        handleSaveNewNode(configuration, nodeName, appInstallationRef);
      } else if (editingNodeData && props.onNodeConfigurationSave) {
        props.onNodeConfigurationSave(editingNodeData.nodeId, configuration, nodeName, appInstallationRef);
      }
    },
    [templateNodeId, newNodeData, editingNodeData, props],
  );

  const handleSaveNewNode = useCallback(
    (configuration: Record<string, any>, nodeName: string, appInstallationRef?: any) => {
      if (newNodeData && props.onNodeAdd && templateNodeId) {
        // Remove the template node first
        if (props.onRemoveTemplateNode) {
          props.onRemoveTemplateNode(templateNodeId);
        }

        // Create the real node through the normal flow
        props.onNodeAdd({
          buildingBlock: newNodeData.buildingBlock,
          nodeName,
          configuration,
          appInstallationRef,
          position: newNodeData.position,
          sourceConnection: newNodeData.sourceConnection,
        });

        // Clear template state
        setTemplateNodeId(null);
        setNewNodeData(null);
        setIsNodeSidebarOpen(false);
        setSelectedNodeId(null);
      }
    },
    [newNodeData, props, templateNodeId],
  );

  const handleCancelTemplate = useCallback(() => {
    // If parent provides onCancelTemplate, delegate to it (for pending connection nodes)
    if (props.onCancelTemplate) {
      props.onCancelTemplate();
      return;
    }

    // Otherwise, handle locally (for regular template nodes)
    if (templateNodeId) {
      if (props.onRemoveTemplateNode) {
        props.onRemoveTemplateNode(templateNodeId);
      }
      setLocalTemplateNodeId(null);
      setLocalNewNodeData(null);
      setIsNodeSidebarOpen(false);
      setSelectedNodeId(null);
    }
  }, [templateNodeId, props, setLocalTemplateNodeId, setLocalNewNodeData]);

  const handleNodeSidebarClose = useCallback(() => {
    setIsNodeSidebarOpen(false);
    setSelectedNodeId(null);

    if (templateNodeId) {
      setNewNodeData(null);
      setTemplateNodeId(null);
      if (props.onRemoveTemplateNode) {
        props.onRemoveTemplateNode(templateNodeId);
      }
    }
  }, [templateNodeId, props]);

  // Use shared builder (merge mocks + live components)
  // Filter out triggers from applications since triggers can't be used in custom components
  const availableApplicationsWithoutTriggers = useMemo(() => {
    return (props.availableApplications || []).map((app) => ({
      ...app,
      triggers: undefined, // Remove triggers from applications
    }));
  }, [props.availableApplications]);

  const buildingBlockCategories = useMemo<BuildingBlockCategory[]>(
    () => buildBuildingBlockCategories([], props.components, [], availableApplicationsWithoutTriggers),
    [props.components, availableApplicationsWithoutTriggers],
  );

  const handleNodeClick = useCallback(
    (nodeId: string) => {
      // Check if this is a pending connection node
      const clickedNode = props.nodes.find((n) => n.id === nodeId);
      const isPendingConnection = (clickedNode?.data as any)?.isPendingConnection;
      const isTemplateNode = (clickedNode?.data as any)?.isTemplate && !isPendingConnection;

      // Check if the current template is a configured template (not just pending connection)
      const currentTemplateNode = templateNodeId ? props.nodes.find((n) => n.id === templateNodeId) : null;
      const isCurrentTemplateConfigured =
        (currentTemplateNode?.data as any)?.isTemplate && !(currentTemplateNode?.data as any)?.isPendingConnection;

      // Allow switching to pending connection nodes or other template nodes even if there's a configured template
      // But block switching to other regular/real nodes
      if (isCurrentTemplateConfigured && nodeId !== templateNodeId && !isPendingConnection && !isTemplateNode) {
        return;
      }

      if (isPendingConnection && props.onPendingConnectionNodeClick) {
        // Notify parent that a pending connection node was clicked
        props.onPendingConnectionNodeClick(nodeId);
      } else if (isTemplateNode && props.onTemplateNodeClick) {
        // Notify parent to restore template state
        props.onTemplateNodeClick(nodeId);
      } else {
        // Regular node click - only in non-delegated mode
        if (props.templateNodeId === undefined) {
          setIsNodeSidebarOpen(true);
          setSelectedNodeId(nodeId);
        }
      }

      // Update selection
      props.onNodesChange(
        props.nodes.map((node) => ({
          type: "select",
          id: node.id,
          selected: node.id === nodeId,
        })),
      );
    },
    [templateNodeId, props],
  );

  const handleConnect = useCallback(
    (connection: Connection) => {
      props.onConnect(connection);
    },
    [props.onConnect],
  );

  const handleNodeDelete = useCallback(
    (nodeId: string) => {
      props.onNodeDelete?.(nodeId);
    },
    [props.onNodeDelete],
  );

  const handleNodeDuplicate = useCallback(
    (nodeId: string) => {
      props.onNodeDuplicate?.(nodeId);
    },
    [props.onNodeDuplicate],
  );

  // Use refs for callbacks to avoid recreating nodeTypes
  const handleNodeEditRef = useRef(handleNodeEdit);
  handleNodeEditRef.current = handleNodeEdit;

  const handleNodeDeleteRef = useRef(handleNodeDelete);
  handleNodeDeleteRef.current = handleNodeDelete;

  const handleNodeDuplicateRef = useRef(handleNodeDuplicate);
  handleNodeDuplicateRef.current = handleNodeDuplicate;

  const callbacksRef = useRef({
    onEdit: handleNodeEditRef,
    onDelete: handleNodeDeleteRef,
    onDuplicate: handleNodeDuplicateRef,
  });

  const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null);
  const [connectingFrom, setConnectingFrom] = useState<{
    nodeId: string;
    handleId: string | null;
    handleType: "source" | "target" | null;
  } | null>(null);

  // Track connection completion for empty space drop detection
  const connectionCompletedRef = useRef(false);
  const connectingFromRef = useRef<{
    nodeId: string;
    handleId: string | null;
    handleType: "source" | "target" | null;
  } | null>(null);

  const handleEdgeMouseEnter = useCallback((_event: React.MouseEvent, edge: any) => {
    setHoveredEdgeId(edge.id);
  }, []);

  const handleEdgeMouseLeave = useCallback(() => {
    setHoveredEdgeId(null);
  }, []);

  const handleConnectStart = useCallback(
    (
      _event: any,
      params: { nodeId: string | null; handleId: string | null; handleType: "source" | "target" | null },
    ) => {
      if (params.nodeId) {
        const connectionInfo = { nodeId: params.nodeId, handleId: params.handleId, handleType: params.handleType };
        setConnectingFrom(connectionInfo);
        connectingFromRef.current = connectionInfo;
      }
    },
    [],
  );

  const handleConnectEnd = useCallback(() => {
    setConnectingFrom(null);
    connectingFromRef.current = null;
    connectionCompletedRef.current = false;
  }, []);

  // Find the hovered edge to get its source and target
  const hoveredEdge = useMemo(() => {
    if (!hoveredEdgeId) return null;
    return props.edges?.find((e) => e.id === hoveredEdgeId);
  }, [hoveredEdgeId, props.edges]);

  const edgeTypes = useMemo(
    () => ({
      custom: CustomEdge,
    }),
    [],
  );

  // Style edges with custom type and hover state
  const styledEdges = useMemo(
    () =>
      props.edges.map((edge) => ({
        ...edge,
        type: "custom",
        data: { ...edge.data, isHovered: edge.id === hoveredEdgeId },
        zIndex: edge.id === hoveredEdgeId ? 1000 : 0,
      })),
    [props.edges, hoveredEdgeId],
  );

  // Add hovered edge and connecting state to nodes
  const nodesWithHoveredEdge = useMemo(
    () =>
      props.nodes.map((node) => ({
        ...node,
        data: {
          ...node.data,
          _hoveredEdge: hoveredEdge,
          _connectingFrom: connectingFrom,
          _allEdges: props.edges,
          _callbacksRef: callbacksRef,
        },
      })),
    [props.nodes, hoveredEdge, connectingFrom, props.edges],
  );

  const handleLogoClick = useCallback(() => {
    if (props.organizationId) {
      window.location.href = `/${props.organizationId}`;
    }
  }, [props.organizationId]);

  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden relative flex flex-col sp-blueprint-canvas">
      {/* Header */}
      <div className="relative z-20">
        <Header
          breadcrumbs={props.breadcrumbs || [{ label: props.customComponentName }]}
          onSave={props.isSaving ? undefined : props.onSave}
          onUndo={props.onUndo}
          canUndo={props.canUndo}
          onLogoClick={props.organizationId ? handleLogoClick : undefined}
          organizationId={props.organizationId}
          unsavedMessage={props.unsavedMessage}
          saveIsPrimary={props.saveIsPrimary}
          saveButtonHidden={props.saveButtonHidden}
        />
      </div>

      {/* Main content */}
      <div className="flex-1 flex relative overflow-hidden">
        {/* Left Sidebar - Building Blocks */}
        <BuildingBlocksSidebar
          isOpen={
            props.isBuildingBlocksSidebarOpen !== undefined ? props.isBuildingBlocksSidebarOpen : isLeftSidebarOpen
          }
          onToggle={props.onBuildingBlocksSidebarToggle || setIsLeftSidebarOpen}
          blocks={buildingBlockCategories}
          canvasZoom={canvasZoom}
          disabled={
            !!templateNodeId && !props.nodes.find((n) => n.id === templateNodeId && (n.data as any).isPendingConnection)
          }
          onBlockClick={props.onBuildingBlockClick}
        />

        {/* React Flow Canvas */}
        <div className="flex-1 relative h-full w-full">
          <ReactFlowProvider>
            <CanvasContent
              nodes={nodesWithHoveredEdge}
              edges={styledEdges}
              edgeTypes={edgeTypes}
              onNodesChange={props.onNodesChange}
              onEdgesChange={props.onEdgesChange}
              onConnect={handleConnect}
              onNodeDoubleClick={props.onNodeDoubleClick}
              onNodeClick={handleNodeClick}
              onBuildingBlockDrop={handleBuildingBlockDrop}
              onZoomChange={setCanvasZoom}
              onEdgeMouseEnter={handleEdgeMouseEnter}
              onEdgeMouseLeave={handleEdgeMouseLeave}
              onConnectStart={handleConnectStart}
              onConnectEnd={handleConnectEnd}
              onConnectionDropInEmptySpace={props.onConnectionDropInEmptySpace}
              connectionCompletedRef={connectionCompletedRef}
              connectingFromRef={connectingFromRef}
              templateNodeId={templateNodeId}
            />
          </ReactFlowProvider>
        </div>

        {/* Right Sidebar - Configuration & Settings */}
        <CustomComponentConfigurationSidebar
          isOpen={isRightSidebarOpen}
          onToggle={setIsRightSidebarOpen}
          metadata={props.metadata}
          onMetadataChange={props.onMetadataChange}
          configurationFields={props.configurationFields}
          onConfigurationFieldsChange={props.onConfigurationFieldsChange}
          onAddConfigField={handleAddConfigField}
          onEditConfigField={handleEditConfigField}
          outputChannels={props.outputChannels}
          onOutputChannelsChange={props.onOutputChannelsChange}
          onAddOutputChannel={handleAddOutputChannel}
          onEditOutputChannel={handleEditOutputChannel}
        />

        {/* Node Configuration Sidebar */}
        {isNodeSidebarOpen && selectedNodeId && (
          <ComponentSidebar
            isOpen={isNodeSidebarOpen}
            onClose={handleNodeSidebarClose}
            nodeId={selectedNodeId}
            iconSlug={newNodeData?.icon || "gear"}
            iconColor="text-black"
            latestEvents={[]}
            nextInQueueEvents={[]}
            iconBackground=""
            totalInQueueCount={0}
            totalInHistoryCount={0}
            hideQueueEvents={true}
            showSettingsTab={true}
            currentTab="settings"
            onTabChange={() => {}} // No tab switching in custom component builder
            templateNodeId={templateNodeId}
            newNodeData={newNodeData}
            onCancelTemplate={handleCancelTemplate}
            nodeConfigMode={templateNodeId ? "create" : "edit"}
            nodeName={editingNodeData?.nodeName || ""}
            nodeLabel={editingNodeData?.displayLabel}
            nodeConfiguration={editingNodeData?.configuration || {}}
            nodeConfigurationFields={editingNodeData?.configurationFields || []}
            onNodeConfigSave={handleSaveConfiguration}
            onNodeConfigCancel={undefined}
            domainId={props.organizationId}
            domainType={"DOMAIN_TYPE_ORGANIZATION" as AuthorizationDomainType}
            customField={undefined}
            appName={editingNodeData?.appName}
            appInstallationRef={editingNodeData?.appInstallationRef}
            installedApplications={props.installedApplications}
          />
        )}
      </div>

      {/* Configuration Field Modal */}
      <ConfigurationFieldModal
        isOpen={isConfigFieldModalOpen}
        onClose={() => {
          setIsConfigFieldModalOpen(false);
          setEditingConfigFieldIndex(null);
        }}
        field={
          editingConfigFieldIndex !== null
            ? (props.configurationFields[editingConfigFieldIndex] as ConfigurationField)
            : undefined
        }
        onSave={handleSaveConfigField}
      />

      {/* Output Channel Modal */}
      <OutputChannelConfigurationModal
        isOpen={isOutputChannelModalOpen}
        onClose={() => {
          setIsOutputChannelModalOpen(false);
          setEditingOutputChannelIndex(null);
        }}
        outputChannel={
          editingOutputChannelIndex !== null
            ? (props.outputChannels[editingOutputChannelIndex] as SuperplaneBlueprintsOutputChannel)
            : undefined
        }
        nodes={props.nodes}
        onSave={handleSaveOutputChannel}
      />

      {/* Node configuration is now handled by the ComponentSidebar */}
    </div>
  );
}

export type { BreadcrumbItem };

import {
  Background,
  Controls,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Connection,
  type Edge,
  type Node,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "./blueprint-canvas-reset.css";
import { useCallback, useMemo, useState, useEffect } from "react";

import { ComponentsComponent } from "@/api-client";
import { Block, BlockData } from "../CanvasPage/Block";
import { CustomEdge } from "../CanvasPage/CustomEdge";
import { Header, BreadcrumbItem } from "../CanvasPage/Header";
import {
  BuildingBlock,
  BuildingBlockCategory,
  BuildingBlocksSidebar,
} from "../BuildingBlocksSidebar";
import {
  CustomComponentConfigurationSidebar,
  BlueprintMetadata,
  OutputChannel,
} from "../CustomComponentConfigurationSidebar";
import { ConfigurationFieldModal } from "./ConfigurationFieldModal";
import { OutputChannelConfigurationModal } from "./OutputChannelConfigurationModal";
import {
  ConfigurationField,
  SuperplaneBlueprintsOutputChannel,
} from "@/api-client";
import { NodeConfigurationModal } from "../CanvasPage/NodeConfigurationModal";
import { buildBuildingBlockCategories } from "../buildingBlocks";

export interface NodeEditData {
  nodeId: string;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, any>;
  configurationFields: ConfigurationField[];
}

export interface NewNodeData {
  buildingBlock: BuildingBlock;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, any>;
  position?: { x: number; y: number };
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

  // Node configuration
  getNodeEditData?: (nodeId: string) => NodeEditData | null;
  onNodeConfigurationSave?: (
    nodeId: string,
    configuration: Record<string, any>,
    nodeName: string
  ) => void;
  onNodeAdd?: (newNodeData: NewNodeData) => void;
  organizationId?: string;

  // Building blocks
  components: ComponentsComponent[];

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

// Canvas content component with ReactFlow hooks - defined outside to prevent re-creation
function CanvasContent({
  nodes,
  edges,
  nodeTypes,
  edgeTypes,
  onNodesChange,
  onEdgesChange,
  onConnect,
  onNodeDoubleClick,
  onBuildingBlockDrop,
  onZoomChange,
}: {
  nodes: Node[];
  edges: Edge[];
  nodeTypes: any;
  edgeTypes: any;
  onNodesChange: (changes: any) => void;
  onEdgesChange: (changes: any) => void;
  onConnect: (connection: Connection) => void;
  onNodeDoubleClick?: (event: any, node: Node) => void;
  onBuildingBlockDrop?: (block: BuildingBlock, position?: { x: number; y: number }) => void;
  onZoomChange?: (zoom: number) => void;
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
    [onBuildingBlockDrop, screenToFlowPosition]
  );

  const handleMove = useCallback(
    (_event: any, viewport: { x: number; y: number; zoom: number }) => {
      if (onZoomChange) {
        onZoomChange(viewport.zoom);
      }
    },
    [onZoomChange]
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
      onConnect={onConnect}
      onNodeDoubleClick={onNodeDoubleClick}
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
      <Background bgColor="#F1F5F9" color="#F1F5F9" />
      <Controls />
    </ReactFlow>
  );
}

export function CustomComponentBuilderPage(props: CustomComponentBuilderPageProps) {
  const [isLeftSidebarOpen, setIsLeftSidebarOpen] = useState(true);
  const [isRightSidebarOpen, setIsRightSidebarOpen] = useState(false);
  const [canvasZoom, setCanvasZoom] = useState(1);

  // Modal state management
  const [isConfigFieldModalOpen, setIsConfigFieldModalOpen] = useState(false);
  const [editingConfigFieldIndex, setEditingConfigFieldIndex] = useState<
    number | null
  >(null);
  const [isOutputChannelModalOpen, setIsOutputChannelModalOpen] =
    useState(false);
  const [editingOutputChannelIndex, setEditingOutputChannelIndex] = useState<
    number | null
  >(null);

  // Node configuration modal state
  const [editingNodeData, setEditingNodeData] = useState<NodeEditData | null>(
    null
  );
  const [newNodeData, setNewNodeData] = useState<NewNodeData | null>(null);

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
        props.onConfigurationFieldsChange([
          ...props.configurationFields,
          field as ConfigurationField,
        ]);
      }
      setIsConfigFieldModalOpen(false);
      setEditingConfigFieldIndex(null);
    },
    [
      editingConfigFieldIndex,
      props.configurationFields,
      props.onConfigurationFieldsChange,
    ]
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
        props.onOutputChannelsChange([
          ...props.outputChannels,
          outputChannel as OutputChannel,
        ]);
      }
      setIsOutputChannelModalOpen(false);
      setEditingOutputChannelIndex(null);
    },
    [
      editingOutputChannelIndex,
      props.outputChannels,
      props.onOutputChannelsChange,
    ]
  );

  // Node configuration handlers
  const handleNodeEdit = useCallback(
    (nodeId: string) => {
      // Try the modal-based edit first (for node configuration)
      if (props.getNodeEditData) {
        const editData = props.getNodeEditData(nodeId);
        if (editData) {
          setEditingNodeData(editData);
        }
      }
    },
    [props.getNodeEditData]
  );

  const handleBuildingBlockDrop = useCallback((block: BuildingBlock, position?: { x: number; y: number }) => {
    setNewNodeData({
      buildingBlock: block,
      nodeName: block.name || "",
      displayLabel: block.label || block.name || "",
      configuration: {},
      position,
    });
  }, []);

  const handleSaveConfiguration = useCallback(
    (configuration: Record<string, any>, nodeName: string) => {
      if (editingNodeData && props.onNodeConfigurationSave) {
        props.onNodeConfigurationSave(
          editingNodeData.nodeId,
          configuration,
          nodeName
        );
      }
      setEditingNodeData(null);
    },
    [editingNodeData, props]
  );

  const handleSaveNewNode = useCallback(
    (configuration: Record<string, any>, nodeName: string) => {
      if (newNodeData && props.onNodeAdd) {
        props.onNodeAdd({
          buildingBlock: newNodeData.buildingBlock,
          nodeName,
          configuration,
          position: newNodeData.position,
        });
      }
      setNewNodeData(null);
    },
    [newNodeData, props]
  );

  // Use shared builder (merge mocks + live components)
  const buildingBlockCategories = useMemo<BuildingBlockCategory[]>(
    () => buildBuildingBlockCategories([], props.components, []),
    [props.components]
  );

  const handleConnect = useCallback(
    (connection: Connection) => {
      props.onConnect(connection);
    },
    [props.onConnect]
  );

  const handleNodeDelete = useCallback(
    (nodeId: string) => {
      props.onNodeDelete?.(nodeId);
    },
    [props.onNodeDelete]
  );

  const nodeTypes = useMemo(
    () => ({
      default: (nodeProps: {
        data: unknown;
        id: string;
        selected?: boolean;
      }) => (
        <Block
          data={nodeProps.data as BlockData}
          nodeId={nodeProps.id}
          onEdit={() => handleNodeEdit(nodeProps.id)}
          onDelete={() => handleNodeDelete(nodeProps.id)}
          selected={nodeProps.selected}
        />
      ),
    }),
    [handleNodeEdit, handleNodeDelete]
  );

  const edgeTypes = useMemo(() => ({
    custom: CustomEdge,
  }), []);

  // Style edges with custom type
  const styledEdges = useMemo(() =>
    props.edges.map((edge) => ({
      ...edge,
      type: 'custom',
    })),
    [props.edges]
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
          isOpen={isLeftSidebarOpen}
          onToggle={setIsLeftSidebarOpen}
          blocks={buildingBlockCategories}
          canvasZoom={canvasZoom}
        />

        {/* React Flow Canvas */}
        <div className="flex-1 relative h-full w-full">
          <ReactFlowProvider>
            <CanvasContent
              nodes={props.nodes}
              edges={styledEdges}
              nodeTypes={nodeTypes}
              edgeTypes={edgeTypes}
              onNodesChange={props.onNodesChange}
              onEdgesChange={props.onEdgesChange}
              onConnect={handleConnect}
              onNodeDoubleClick={props.onNodeDoubleClick}
              onBuildingBlockDrop={handleBuildingBlockDrop}
              onZoomChange={setCanvasZoom}
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
            ? (props.outputChannels[
              editingOutputChannelIndex
            ] as SuperplaneBlueprintsOutputChannel)
            : undefined
        }
        nodes={props.nodes}
        onSave={handleSaveOutputChannel}
      />

      {/* Edit existing node modal */}
      {editingNodeData && (
        <NodeConfigurationModal
          isOpen={true}
          onClose={() => setEditingNodeData(null)}
          nodeName={editingNodeData.nodeName}
          nodeLabel={editingNodeData.displayLabel}
          configuration={editingNodeData.configuration}
          configurationFields={editingNodeData.configurationFields}
          onSave={handleSaveConfiguration}
          domainId={props.organizationId}
          domainType="DOMAIN_TYPE_ORGANIZATION"
        />
      )}

      {/* Add new node modal */}
      {newNodeData && (
        <NodeConfigurationModal
          isOpen={true}
          onClose={() => setNewNodeData(null)}
          nodeName={newNodeData.nodeName}
          nodeLabel={newNodeData.displayLabel}
          configuration={newNodeData.configuration}
          configurationFields={newNodeData.buildingBlock.configuration || []}
          onSave={handleSaveNewNode}
          domainId={props.organizationId}
          domainType="DOMAIN_TYPE_ORGANIZATION"
        />
      )}
    </div>
  );
}

export type { BreadcrumbItem };

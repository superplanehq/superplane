import {
  Background,
  Controls,
  ReactFlow,
  ReactFlowProvider,
  type Connection,
  type Edge,
  type Node,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "./blueprint-canvas-reset.css";
import { useCallback, useMemo, useState } from "react";

import { ComponentsComponent } from "@/api-client";
import { Block, BlockData } from "../CanvasPage/Block";
import { Header, BreadcrumbItem } from "../CanvasPage/Header";
import {
  BuildingBlock,
  BuildingBlockCategory,
  BuildingBlocksSidebar,
} from "../BuildingBlocksSidebar";
import {
  BlueprintConfigurationSidebar,
  BlueprintMetadata,
  ConfigurationField,
  OutputChannel,
} from "../BlueprintConfigurationSidebar";
import { ConfigurationFieldModal } from "./ConfigurationFieldModal";
import { OutputChannelConfigurationModal } from "./OutputChannelConfigurationModal";
import {
  ComponentsConfigurationField,
  SuperplaneBlueprintsOutputChannel,
} from "@/api-client";

export interface BlueprintBuilderPageProps {
  // Blueprint data
  blueprintName: string;
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
  onNodeEdit?: (nodeId: string) => void;
  onNodeDelete?: (nodeId: string) => void;

  // Building blocks
  components: ComponentsComponent[];
  onComponentClick: (block: BuildingBlock) => void;

  // Actions
  onSave: () => void;
  isSaving?: boolean;
}

export function BlueprintBuilderPage(props: BlueprintBuilderPageProps) {
  const [isLeftSidebarOpen, setIsLeftSidebarOpen] = useState(true);
  const [isRightSidebarOpen, setIsRightSidebarOpen] = useState(false);

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
    (field: ComponentsConfigurationField) => {
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

  // Transform components into building block categories
  const buildingBlockCategories = useMemo(() => {
    const categoryMap = new Map<string, BuildingBlock[]>();

    props.components.forEach((component: ComponentsComponent) => {
      const categoryName = "Components";

      const block: BuildingBlock = {
        name: component.name || "",
        label: component.label || component.name || "",
        description: component.description,
        type: "component",
        outputChannels: component.outputChannels || [],
        configuration: component.configuration || [],
        icon: component.icon,
        color: component.color,
      };

      if (!categoryMap.has(categoryName)) {
        categoryMap.set(categoryName, []);
      }
      categoryMap.get(categoryName)!.push(block);
    });

    const categories: BuildingBlockCategory[] = Array.from(
      categoryMap.entries()
    ).map(([name, blocks]) => ({
      name,
      blocks,
    }));

    return categories;
  }, [props.components]);

  const handleConnect = useCallback(
    (connection: Connection) => {
      props.onConnect(connection);
    },
    [props]
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
          onClick={() => props.onNodeClick?.(nodeProps.id)}
          onEdit={() => props.onNodeEdit?.(nodeProps.id)}
          onDelete={() => props.onNodeDelete?.(nodeProps.id)}
          selected={nodeProps.selected}
        />
      ),
    }),
    [props.onNodeClick, props.onNodeEdit, props.onNodeDelete]
  );

  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden relative flex flex-col sp-blueprint-canvas">
      {/* Header */}
      <div className="relative z-20">
        <Header
          breadcrumbs={props.breadcrumbs || [{ label: props.blueprintName }]}
          onSave={props.isSaving ? undefined : props.onSave}
        />
      </div>

      {/* Main content */}
      <div className="flex-1 flex relative overflow-hidden">
        {/* Left Sidebar - Building Blocks */}
        <BuildingBlocksSidebar
          isOpen={isLeftSidebarOpen}
          onToggle={setIsLeftSidebarOpen}
          onBlockClick={props.onComponentClick}
          blocks={buildingBlockCategories}
        />

        {/* React Flow Canvas */}
        <div className="flex-1 relative h-full w-full">
          <ReactFlowProvider>
            <ReactFlow
              nodes={props.nodes}
              edges={props.edges}
              nodeTypes={nodeTypes}
              onNodesChange={props.onNodesChange}
              onEdgesChange={props.onEdgesChange}
              onConnect={handleConnect}
              onNodeDoubleClick={props.onNodeDoubleClick}
              fitView
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
          </ReactFlowProvider>
        </div>

        {/* Right Sidebar - Configuration & Settings */}
        <BlueprintConfigurationSidebar
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
            ? (props.configurationFields[editingConfigFieldIndex] as ComponentsConfigurationField)
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
    </div>
  );
}

export type { BreadcrumbItem };

import type { Meta, StoryObj } from "@storybook/react";
import "@xyflow/react/dist/style.css";
import "./blueprint-canvas-reset.css";
import { useState } from "react";
import { CustomComponentBuilderPage } from "./index";
import { mockComponents } from "./storybooks/mockComponents";
import { mockNodes, mockEdges } from "./storybooks/mockBlueprint";
import type { Node, Edge } from "@xyflow/react";
import { applyNodeChanges, applyEdgeChanges } from "@xyflow/react";
import type { ConfigurationField, OutputChannel } from "../CustomComponentConfigurationSidebar";

const meta = {
  title: "Pages/CustomComponentBuilderPage",
  component: CustomComponentBuilderPage,
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {},
} satisfies Meta<typeof CustomComponentBuilderPage>;

export default meta;

type Story = StoryObj<typeof CustomComponentBuilderPage>;

export const Default: Story = {
  render: () => {
    const [nodes, setNodes] = useState<Node[]>(mockNodes);
    const [edges, setEdges] = useState<Edge[]>(mockEdges);
    const [blueprintName, setBlueprintName] = useState("Deploy to Production");
    const [description, setDescription] = useState("Automated deployment workflow with approval gates");
    const [icon, setIcon] = useState("rocket");
    const [color, setColor] = useState("blue");
    const [configurationFields, setConfigurationFields] = useState<ConfigurationField[]>([
      {
        name: "environment",
        label: "Environment",
        type: "select",
        description: "Target deployment environment",
        required: true,
        typeOptions: {
          select: {
            options: [
              { label: "Development", value: "dev" },
              { label: "Staging", value: "staging" },
              { label: "Production", value: "prod" },
            ],
          },
        },
      },
      {
        name: "notification_email",
        label: "Notification Email",
        type: "string",
        description: "Email to receive deployment notifications",
        required: false,
      },
    ]);
    const [outputChannels, setOutputChannels] = useState<OutputChannel[]>([
      {
        name: "success",
        nodeId: "deploy-node-1",
        nodeOutputChannel: "default",
      },
      {
        name: "failure",
        nodeId: "deploy-node-1",
        nodeOutputChannel: "error",
      },
    ]);

    return (
      <CustomComponentBuilderPage
        customComponentName={blueprintName}
        breadcrumbs={[
          { label: "Components" },
          { label: blueprintName, iconSlug: "rocket", iconColor: "text-blue-600" },
        ]}
        metadata={{
          name: blueprintName,
          description,
          icon,
          color,
        }}
        onMetadataChange={(metadata) => {
          setBlueprintName(metadata.name);
          setDescription(metadata.description);
          setIcon(metadata.icon);
          setColor(metadata.color);
        }}
        configurationFields={configurationFields}
        onConfigurationFieldsChange={setConfigurationFields}
        outputChannels={outputChannels}
        onOutputChannelsChange={setOutputChannels}
        nodes={nodes}
        edges={edges}
        onNodesChange={(changes) => {
          setNodes((nds) => applyNodeChanges(changes, nds));
        }}
        onEdgesChange={(changes) => {
          setEdges((eds) => applyEdgeChanges(changes, eds));
        }}
        onConnect={(connection) => {
          console.log("Connection created:", connection);
        }}
        onNodeDoubleClick={(event, node) => {
          console.log("Node double clicked:", node.id);
        }}
        onNodeClick={(nodeId) => {
          console.log("Node clicked:", nodeId);
        }}
        onNodeEdit={(nodeId) => {
          console.log("Node edit:", nodeId);
        }}
        onNodeDelete={(nodeId) => {
          console.log("Node delete:", nodeId);
        }}
        components={mockComponents}
        onComponentClick={(block) => {
          console.log("Component clicked:", block.name);
        }}
        onSave={() => {
          console.log("Save component");
          console.log("Nodes:", nodes);
          console.log("Edges:", edges);
          console.log("Configuration:", configurationFields);
          console.log("Output channels:", outputChannels);
        }}
        isSaving={false}
      />
    );
  },
};

export const EmptyBlueprint: Story = {
  render: () => {
    const [nodes, setNodes] = useState<Node[]>([]);
    const [edges, setEdges] = useState<Edge[]>([]);
    const [blueprintName, setBlueprintName] = useState("New Component");
    const [description, setDescription] = useState("");
    const [icon, setIcon] = useState("");
    const [color, setColor] = useState("");
    const [configurationFields, setConfigurationFields] = useState<ConfigurationField[]>([]);
    const [outputChannels, setOutputChannels] = useState<OutputChannel[]>([]);

    return (
      <CustomComponentBuilderPage
        customComponentName={blueprintName}
        breadcrumbs={[
          { label: "Components" },
          { label: blueprintName, iconSlug: "rocket", iconColor: "text-blue-600" },
        ]}
        metadata={{
          name: blueprintName,
          description,
          icon,
          color,
        }}
        onMetadataChange={(metadata) => {
          setBlueprintName(metadata.name);
          setDescription(metadata.description);
          setIcon(metadata.icon);
          setColor(metadata.color);
        }}
        configurationFields={configurationFields}
        onConfigurationFieldsChange={setConfigurationFields}
        outputChannels={outputChannels}
        onOutputChannelsChange={setOutputChannels}
        nodes={nodes}
        edges={edges}
        onNodesChange={(changes) => {
          setNodes((nds) => applyNodeChanges(changes, nds));
        }}
        onEdgesChange={(changes) => {
          setEdges((eds) => applyEdgeChanges(changes, eds));
        }}
        onConnect={(connection) => {
          console.log("Connection created:", connection);
        }}
        onNodeClick={(nodeId) => {
          console.log("Node clicked:", nodeId);
        }}
        onNodeEdit={(nodeId) => {
          console.log("Node edit:", nodeId);
        }}
        onNodeDelete={(nodeId) => {
          console.log("Node delete:", nodeId);
        }}
        components={mockComponents}
        onComponentClick={(block) => {
          console.log("Component clicked:", block.name);
        }}
        onSave={() => {
          console.log("Save component");
        }}
        isSaving={false}
      />
    );
  },
};

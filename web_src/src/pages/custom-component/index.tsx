import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Connection, Edge, Node, addEdge, applyEdgeChanges, applyNodeChanges } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import ELK from "elkjs/lib/elk.bundled.js";
import { AlertCircle } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { ComponentsComponent, ComponentsNode } from "../../api-client";
import { Heading } from "../../components/Heading/heading";
import { Button } from "../../components/ui/button";
import { useBlueprint, useComponents, useUpdateBlueprint } from "../../hooks/useBlueprintData";
import { BlockData } from "../../ui/CanvasPage/Block";
import type { BreadcrumbItem, NewNodeData } from "../../ui/CustomComponentBuilderPage";
import { CustomComponentBuilderPage } from "../../ui/CustomComponentBuilderPage";
import { filterVisibleConfiguration } from "../../utils/components";
import { showErrorToast, showSuccessToast } from "../../utils/toast";
import { getComponentBaseMapper } from "../workflowv2/mappers";

const elk = new ELK();

const getLayoutedElements = async (nodes: Node[], edges: Edge[]) => {
  const graph = {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.spacing.nodeNode": "80",
      "elk.layered.spacing.nodeNodeBetweenLayers": "100",
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
  };

  const layoutedGraph = await elk.layout(graph);

  const layoutedNodes = nodes.map((node) => {
    const layoutedNode = layoutedGraph.children?.find((n) => n.id === node.id);
    return {
      ...node,
      position: {
        x: layoutedNode?.x ?? 0,
        y: layoutedNode?.y ?? 0,
      },
    };
  });

  return { nodes: layoutedNodes, edges };
};

// Helper function to map component type to block type
const getBlockType = (componentName: string): BlockData["type"] => {
  const typeMap: Record<string, BlockData["type"]> = {
    if: "if",
    filter: "filter",
    approval: "approval",
    noop: "component",
    http: "component",
    semaphore: "semaphore",
    wait: "wait",
    time_gate: "time_gate",
  };
  return typeMap[componentName] || "noop"; // Default to noop for unknown components
};

// Helper function to create minimal BlockData for a component
const createBlockData = (node: any, component: ComponentsComponent | undefined): BlockData => {
  const componentName = node.component?.name || "";
  const blockType = getBlockType(componentName);
  const channels = component?.outputChannels?.map((channel: any) => channel.name) || ["default"];

  const baseData: BlockData = {
    label: node.name,
    state: "pending",
    type: blockType,
    outputChannels: channels,
  };
  const expression = node.configuration?.expression;
  // Add type-specific props based on component type
  switch (blockType) {
    case "if":
      baseData.if = {
        title: node.name,
        expression,
        trueEvent: {
          eventTitle: "No events received yet",
          eventState: "neutral" as const,
        },
        falseEvent: {
          eventTitle: "No events received yet",
          eventState: "neutral" as const,
        },
        trueSectionLabel: "TRUE",
        falseSectionLabel: "FALSE",
        collapsed: false,
      };
      break;
    case "filter":
      baseData.filter = {
        title: node.name,
        expression,
        lastEvent: {
          eventTitle: "No events received yet",
          eventState: "neutral" as const,
        },
        collapsed: false,
      };
      break;
    case "approval":
      baseData.approval = {
        title: node.name,
        description: component?.description,
        iconSlug: component?.icon,
        iconColor: "text-orange-500",
        headerColor: "bg-orange-100",
        collapsedBackground: "bg-orange-100",
        approvals: [],
        collapsed: false,
      };
      break;
    case "semaphore":
      // Build metadata array
      const metadataItems = [];
      if (node.configuration?.project) {
        metadataItems.push({ icon: "folder", label: node.configuration.project });
      }
      if (node.configuration?.ref) {
        metadataItems.push({ icon: "git-branch", label: node.configuration.ref });
      }
      if (node.configuration?.pipelineFile) {
        metadataItems.push({ icon: "file-code", label: node.configuration.pipelineFile });
      }

      baseData.semaphore = {
        title: node.name,
        iconSrc: SemaphoreLogo,
        iconSlug: component?.icon || "workflow",
        iconColor: "text-gray-700",
        iconBackground: "bg-gray-100",
        headerColor: "bg-gray-50",
        collapsedBackground: "bg-gray-100",
        collapsed: false,
        hideLastRun: true,
        metadata: metadataItems,
        parameters: node.configuration?.parameters,
      };
      break;
    case "wait":
      baseData.wait = {
        title: node.name,
        duration: node.configuration?.duration,
        iconColor: "text-yellow-600",
        iconBackground: "bg-yellow-100",
        headerColor: "bg-yellow-50",
        collapsedBackground: "bg-yellow-50",
        collapsed: false,
        hideLastRun: true,
      };
      break;
    case "component":
      baseData.component = getComponentBaseMapper(component?.name!).props([], node, component!, null);
      break;
    case "time_gate":
      const mode = node.configuration?.mode || "include_range";
      const days = node.configuration?.days || [];
      const daysDisplay = days.length > 0 ? days.join(", ") : "";

      // Get timezone information
      const timezone = node.configuration?.timezone || "0";
      const getTimezoneDisplay = (timezoneOffset: string) => {
        const offset = parseFloat(timezoneOffset);
        if (offset === 0) return "GMT+0 (UTC)";
        if (offset > 0) return `GMT+${offset}`;
        return `GMT${offset}`; // Already has the minus sign
      };
      const timezoneDisplay = getTimezoneDisplay(timezone);

      let startTime = "00:00";
      let endTime = "23:59";

      if (mode === "include_specific" || mode === "exclude_specific") {
        startTime = `${node.configuration.startDayInYear} ${node.configuration.startTime}`;
        endTime = `${node.configuration.endDayInYear} ${node.configuration.endTime}`;
      } else {
        startTime = `${node.configuration.startTime}`;
        endTime = `${node.configuration.endTime}`;
      }

      const timeWindow = `${startTime} - ${endTime}`;

      baseData.time_gate = {
        title: node.name,
        mode,
        timeWindow,
        days: daysDisplay,
        timezone: timezoneDisplay,
        lastExecution: undefined,
        nextInQueue: undefined,
        iconColor: "text-blue-600",
        iconBackground: "bg-blue-100",
        headerColor: "bg-blue-50",
        collapsedBackground: "bg-white",
        collapsed: false,
      };
      break;
  }

  return baseData;
};

export const CustomComponent = () => {
  const { organizationId, blueprintId } = useParams<{ organizationId: string; blueprintId: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [blueprintConfiguration, setBlueprintConfiguration] = useState<any[]>([]);
  const [blueprintOutputChannels, setBlueprintOutputChannels] = useState<any[]>([]);
  const [blueprintName, setBlueprintName] = useState("");
  const [blueprintDescription, setBlueprintDescription] = useState("");
  const [blueprintIcon, setBlueprintIcon] = useState("");
  const [blueprintColor, setBlueprintColor] = useState("");
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

  // Fetch blueprint and components
  const { data: blueprint, isLoading: blueprintLoading } = useBlueprint(organizationId || "", blueprintId || "");
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!);
  const updateBlueprintMutation = useUpdateBlueprint(organizationId!, blueprintId!);

  usePageTitle([blueprint?.name || "Custom Component"]);

  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);

  // Revert functionality - track initial blueprint snapshot
  const [initialBlueprintSnapshot, setInitialBlueprintSnapshot] = useState<any>(null);

  // Save initial blueprint snapshot for revert functionality
  const saveSnapshot = useCallback(() => {
    // Only save if we don't already have a snapshot
    if (!initialBlueprintSnapshot) {
      setInitialBlueprintSnapshot({
        nodes,
        edges,
        blueprintName,
        blueprintDescription,
        blueprintIcon,
        blueprintColor,
        timestamp: Date.now(),
      });
    }
  }, [initialBlueprintSnapshot, nodes, edges, blueprintName, blueprintDescription, blueprintIcon, blueprintColor]);

  // Revert to initial state
  const handleRevert = useCallback(() => {
    if (initialBlueprintSnapshot) {
      // Restore the initial state
      setNodes(initialBlueprintSnapshot.nodes);
      setEdges(initialBlueprintSnapshot.edges);
      setBlueprintName(initialBlueprintSnapshot.blueprintName);
      setBlueprintDescription(initialBlueprintSnapshot.blueprintDescription);
      setBlueprintIcon(initialBlueprintSnapshot.blueprintIcon);
      setBlueprintColor(initialBlueprintSnapshot.blueprintColor);

      // Clear the snapshot since we're back to the initial state
      setInitialBlueprintSnapshot(null);

      // Mark as no unsaved changes since we're back to the saved state
      setHasUnsavedChanges(false);
    }
  }, [initialBlueprintSnapshot]);

  // Handler for metadata changes
  const handleMetadataChange = useCallback(
    (metadata: any) => {
      saveSnapshot();
      setBlueprintName(metadata.name);
      setBlueprintDescription(metadata.description);
      setBlueprintIcon(metadata.icon);
      setBlueprintColor(metadata.color);
      setHasUnsavedChanges(true);
    },
    [saveSnapshot],
  );

  // Update blueprint configuration and output channels when blueprint loads
  useEffect(() => {
    if (blueprint) {
      if (blueprint.configuration) {
        setBlueprintConfiguration(blueprint.configuration);
      }
      if (blueprint.outputChannels) {
        setBlueprintOutputChannels(blueprint.outputChannels);
      }
      setBlueprintName(blueprint.name || "");
      setBlueprintDescription(blueprint.description || "");
      setBlueprintIcon(blueprint.icon || "");
      setBlueprintColor(blueprint.color || "");
    }
  }, [blueprint]);

  // Update nodes and edges when blueprint or components data changes
  useEffect(() => {
    if (!blueprint || components.length === 0) return;

    const allNodes: Node[] = (blueprint.nodes || [])
      .map((node: ComponentsNode) => {
        // Handle component nodes
        const component = components.find((p: any) => p.name === node.component?.name);
        const blockData = createBlockData(node, component);

        return {
          id: node.id,
          type: "default", // BlueprintBuilderPage uses 'default' type for all nodes
          data: {
            ...blockData,
            // Store original data for serialization
            _originalComponent: node.component?.name,
            _originalConfiguration: node.configuration || {},
          },
          position: node.position || { x: 0, y: 0 },
        };
      })
      .filter(Boolean) as Node[];

    const loadedEdges: Edge[] = (blueprint.edges || []).map((edge: any, index: number) => ({
      id: `e${index}`,
      source: edge.sourceId,
      sourceHandle: edge.channel || "default",
      target: edge.targetId,
      style: { strokeWidth: 3, stroke: "#C9D5E1" },
    }));

    // Check if we have saved positions
    const hasPositions = allNodes.some((node) => node.position && (node.position.x !== 0 || node.position.y !== 0));

    if (hasPositions) {
      // Use saved positions
      setNodes(allNodes);
      setEdges(loadedEdges);
    } else {
      // Apply elk layout for blueprints without saved positions
      getLayoutedElements(allNodes, loadedEdges).then(({ nodes: layoutedNodes, edges: layoutedEdges }) => {
        setNodes(layoutedNodes);
        setEdges(layoutedEdges);
      });
    }
  }, [blueprint, components]);

  // Use refs to access latest values without causing re-renders
  const nodesRef = useRef(nodes);
  const componentsRef = useRef(components);

  // Keep refs updated
  useEffect(() => {
    nodesRef.current = nodes;
  }, [nodes]);

  useEffect(() => {
    componentsRef.current = components;
  }, [components]);

  // Warn user before leaving page with unsaved changes
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (hasUnsavedChanges) {
        e.preventDefault();
        e.returnValue = "Your work isn't saved, unsaved changes will be lost. Are you sure you want to leave?";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [hasUnsavedChanges]);

  // Node and edge change handlers
  const onNodesChange = useCallback(
    (changes: any) => {
      setNodes((nds) => applyNodeChanges(changes, nds));
      // Only mark as unsaved if it's a position change (user dragging nodes)
      // Don't mark for 'select' or 'dimensions' changes which happen automatically
      const hasPositionChange = changes.some((change: any) => change.type === "position" && change.dragging);
      if (hasPositionChange) {
        saveSnapshot();
        setHasUnsavedChanges(true);
      }
    },
    [saveSnapshot],
  );

  const onEdgesChange = useCallback(
    (changes: any) => {
      setEdges((eds) => applyEdgeChanges(changes, eds));
      // Only mark as unsaved for edge removal, not for selection changes
      const hasRemoval = changes.some((change: any) => change.type === "remove");
      if (hasRemoval) {
        saveSnapshot();
        setHasUnsavedChanges(true);
      }
    },
    [saveSnapshot],
  );

  const onConnect = useCallback(
    (params: Connection) => {
      saveSnapshot();
      setEdges((eds) => addEdge({ ...params, style: { strokeWidth: 3, stroke: "#C9D5E1" } }, eds));
      setHasUnsavedChanges(true);
    },
    [saveSnapshot],
  );

  const generateNodeId = (componentName: string, nodeName: string) => {
    const randomChars = Math.random().toString(36).substring(2, 8);
    const sanitizedComponent = componentName.toLowerCase().replace(/[^a-z0-9]/g, "-");
    const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, "-");
    return `${sanitizedComponent}-${sanitizedName}-${randomChars}`;
  };

  const getNodeEditData = useCallback((nodeId: string) => {
    const node = nodesRef.current.find((n) => n.id === nodeId);
    if (!node) return null;

    const component = componentsRef.current.find((p: any) => p.name === (node.data as any)._originalComponent);
    if (!component) return null;

    return {
      nodeId: node.id,
      nodeName: (node.data as any).label as string,
      displayLabel: component.label || ((node.data as any).label as string),
      configuration: (node.data as any)._originalConfiguration || {},
      configurationFields: component.configuration || [],
    };
  }, []);

  const handleNodeConfigurationSave = useCallback(
    (nodeId: string, configuration: Record<string, any>, nodeName: string) => {
      saveSnapshot();

      const node = nodesRef.current.find((n) => n.id === nodeId);
      if (!node) return;

      const component = componentsRef.current.find((p: any) => p.name === (node.data as any)._originalComponent);
      if (!component) return;

      // Filter configuration to only include visible fields
      const filteredConfiguration = filterVisibleConfiguration(configuration, component.configuration || []);

      // Update existing node
      setNodes((nds) =>
        nds.map((n) => {
          if (n.id !== nodeId) return n;

          const nodeData = n.data as any;
          const updatedData = {
            ...nodeData,
            label: nodeName.trim(),
            _originalConfiguration: filteredConfiguration,
          };

          // Update the title and expression in the type-specific props
          if (nodeData.if) {
            updatedData.if = {
              ...nodeData.if,
              title: nodeName.trim(),
              expression: filteredConfiguration.expression,
            };
          }
          if (nodeData.filter) {
            updatedData.filter = {
              ...nodeData.filter,
              title: nodeName.trim(),
              expression: filteredConfiguration.expression,
            };
          }
          if (nodeData.approval) {
            updatedData.approval = { ...nodeData.approval, title: nodeName.trim() };
          }
          if (nodeData.semaphore) {
            // Rebuild metadata array from configuration
            const metadataItems = [];
            if (filteredConfiguration.project) {
              metadataItems.push({ icon: "folder", label: filteredConfiguration.project });
            }
            if (filteredConfiguration.ref) {
              metadataItems.push({ icon: "git-branch", label: filteredConfiguration.ref });
            }
            if (filteredConfiguration.pipelineFile) {
              metadataItems.push({ icon: "file-code", label: filteredConfiguration.pipelineFile });
            }

            updatedData.semaphore = {
              ...nodeData.semaphore,
              title: nodeName.trim(),
              metadata: metadataItems,
              parameters: filteredConfiguration.parameters,
            };
          }
          if (nodeData.wait) {
            updatedData.wait = {
              ...nodeData.wait,
              title: nodeName.trim(),
              duration: filteredConfiguration.duration,
            };
          }
          if (nodeData.component) {
            const updatedNode: ComponentsNode = {
              id: node.id,
              name: nodeName.trim(),
              type: "TYPE_COMPONENT",
              configuration: filteredConfiguration,
              component: {
                name: component.name,
              },
            };
            updatedData.component = getComponentBaseMapper(component.name!).props([], updatedNode, component, null);
          }
          if (nodeData.time_gate) {
            const mode = filteredConfiguration.mode || "include_range";
            const days = (filteredConfiguration.days as string[]) || [];
            const daysDisplay = days.length > 0 ? days.join(", ") : "";

            // Handle different time window formats based on mode
            let startTime = "00:00";
            let endTime = "23:59";

            if (mode === "include_specific" || mode === "exclude_specific") {
              startTime = `${filteredConfiguration.startDayInYear} ${filteredConfiguration.startTime}`;
              endTime = `${filteredConfiguration.endDayInYear} ${filteredConfiguration.endTime}`;
            } else {
              startTime = `${filteredConfiguration.startTime}`;
              endTime = `${filteredConfiguration.endTime}`;
            }

            const timeWindow = `${startTime} - ${endTime}`;

            updatedData.time_gate = {
              ...nodeData.time_gate,
              title: nodeName.trim(),
              mode,
              timeWindow,
              days: daysDisplay,
            };
          }

          return {
            ...n,
            data: updatedData,
          };
        }),
      );
      setHasUnsavedChanges(true);
    },
    [saveSnapshot],
  );

  const handleNodeAdd = useCallback(
    (newNodeData: NewNodeData) => {
      // Save snapshot before making changes
      saveSnapshot();

      const component = componentsRef.current.find(
        (c: ComponentsComponent) => c.name === newNodeData.buildingBlock.name,
      );
      if (!component) return;

      // Filter configuration to only include visible fields
      const filteredConfiguration = filterVisibleConfiguration(
        newNodeData.configuration,
        component.configuration || [],
      );

      // Add new node
      const newNodeId = generateNodeId(component.name!, newNodeData.nodeName.trim());
      const mockNode = {
        component: { name: component.name },
        name: newNodeData.nodeName.trim(),
        configuration: filteredConfiguration,
      };
      const blockData = createBlockData(mockNode, component);

      const newNode: Node = {
        id: newNodeId,
        type: "default",
        position: newNodeData.position || { x: nodesRef.current.length * 250, y: 100 },
        data: {
          ...blockData,
          _originalComponent: component.name,
          _originalConfiguration: filteredConfiguration,
        },
      };
      setNodes((nds) => [...nds, newNode]);
      setHasUnsavedChanges(true);
    },
    [saveSnapshot],
  );

  const handleNodeDelete = useCallback(
    (nodeId: string) => {
      saveSnapshot();
      setNodes((nds) => nds.filter((n) => n.id !== nodeId));
      setEdges((eds) => eds.filter((e) => e.source !== nodeId && e.target !== nodeId));
      setHasUnsavedChanges(true);
    },
    [saveSnapshot],
  );

  const handleNodeDuplicate = useCallback(
    (nodeId: string) => {
      const nodeToDuplicate = nodesRef.current.find((node) => node.id === nodeId);
      if (!nodeToDuplicate) return;

      // Save snapshot before making changes
      saveSnapshot();

      // Generate a new unique node ID
      const nodeData = nodeToDuplicate.data as any;
      const originalName = nodeData.label || "node";
      const duplicateName = `${originalName} copy`;

      // Get component name for ID generation
      const componentName = nodeData._originalComponent || "component";
      const newNodeId = generateNodeId(componentName, duplicateName);

      // Create the duplicate node with offset position
      const offsetX = 50; // Offset to the right
      const offsetY = 50; // Offset down

      const duplicateNode: Node = {
        ...nodeToDuplicate,
        id: newNodeId,
        position: {
          x: nodeToDuplicate.position.x + offsetX,
          y: nodeToDuplicate.position.y + offsetY,
        },
        data: {
          ...nodeData,
          label: duplicateName,
          // Update type-specific props with new title
          ...(nodeData.if && {
            if: {
              ...nodeData.if,
              title: duplicateName,
            },
          }),
          ...(nodeData.filter && {
            filter: {
              ...nodeData.filter,
              title: duplicateName,
            },
          }),
          ...(nodeData.approval && {
            approval: {
              ...nodeData.approval,
              title: duplicateName,
            },
          }),
          ...(nodeData.semaphore && {
            semaphore: {
              ...nodeData.semaphore,
              title: duplicateName,
            },
          }),
          ...(nodeData.wait && {
            wait: {
              ...nodeData.wait,
              title: duplicateName,
            },
          }),
          ...(nodeData.component && {
            component: {
              ...nodeData.component,
              title: duplicateName,
            },
          }),
          ...(nodeData.time_gate && {
            time_gate: {
              ...nodeData.time_gate,
              title: duplicateName,
            },
          }),
        },
      };

      // Add the duplicate node to the nodes array
      setNodes((nds) => [...nds, duplicateNode]);
      setHasUnsavedChanges(true);
    },
    [saveSnapshot],
  );

  const handleConfigurationFieldsChange = useCallback(
    (fields: any[]) => {
      saveSnapshot();
      setBlueprintConfiguration(fields);
      setHasUnsavedChanges(true);
    },
    [saveSnapshot],
  );

  const handleOutputChannelsChange = useCallback(
    (channels: any[]) => {
      saveSnapshot();
      setBlueprintOutputChannels(channels);
      setHasUnsavedChanges(true);
    },
    [saveSnapshot],
  );

  const handleSave = async () => {
    try {
      // Serialize all nodes
      const blueprintNodes = nodes.map((node) => {
        const nodeData = node.data as any;
        return {
          id: node.id,
          name: nodeData.label as string,
          type: "TYPE_COMPONENT",
          component: {
            name: nodeData._originalComponent as string,
          },
          configuration: nodeData._originalConfiguration || {},
          position: {
            x: Math.round(node.position.x),
            y: Math.round(node.position.y),
          },
        };
      });

      const blueprintEdges = edges.map((edge) => ({
        sourceId: edge.source!,
        targetId: edge.target!,
        channel: edge.sourceHandle || "default",
      }));

      await updateBlueprintMutation.mutateAsync({
        name: blueprintName,
        description: blueprintDescription,
        nodes: blueprintNodes,
        edges: blueprintEdges,
        configuration: blueprintConfiguration,
        outputChannels: blueprintOutputChannels,
        icon: blueprintIcon,
        color: blueprintColor,
      });

      showSuccessToast("Component saved successfully");
      setHasUnsavedChanges(false);

      // Clear the snapshot since changes are now saved
      setInitialBlueprintSnapshot(null);
    } catch (error: any) {
      console.error("Error saving component:", error);
      const errorMessage = error?.response?.data?.message || error?.message || "Failed to save component";
      showErrorToast(errorMessage);
    }
  };

  if (blueprintLoading || componentsLoading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <p className="ml-3 text-gray-500">Loading blueprint...</p>
      </div>
    );
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
    );
  }

  // Get workflow info from URL parameters
  const fromWorkflowId = searchParams.get("fromWorkflow");
  const workflowName = searchParams.get("workflowName");

  const breadcrumbs: BreadcrumbItem[] =
    fromWorkflowId && workflowName
      ? [
          { label: workflowName, href: `/${organizationId}/workflows/${fromWorkflowId}` },
          { label: blueprintName, iconSlug: blueprintIcon, iconColor: `text-${blueprintColor}-600` },
        ]
      : [
          { label: "Components", href: `/${organizationId}` },
          { label: blueprintName, iconSlug: blueprintIcon, iconColor: `text-${blueprintColor}-600` },
        ];

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
        onConfigurationFieldsChange={handleConfigurationFieldsChange}
        outputChannels={blueprintOutputChannels}
        onOutputChannelsChange={handleOutputChannelsChange}
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeDuplicate={handleNodeDuplicate}
        onNodeDelete={handleNodeDelete}
        getNodeEditData={getNodeEditData}
        onNodeConfigurationSave={handleNodeConfigurationSave}
        onNodeAdd={handleNodeAdd}
        organizationId={organizationId}
        components={components}
        onSave={handleSave}
        isSaving={updateBlueprintMutation.isPending}
        unsavedMessage={hasUnsavedChanges ? "You have unsaved changes" : undefined}
        saveButtonHidden={!hasUnsavedChanges}
        saveIsPrimary={hasUnsavedChanges}
        onUndo={handleRevert}
        canUndo={initialBlueprintSnapshot !== null}
      />
    </>
  );
};

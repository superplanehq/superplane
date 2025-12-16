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
    if: "component",
    filter: "component",
    approval: "component",
    noop: "component",
    http: "component",
    semaphore: "component",
    wait: "component",
    time_gate: "component",
    merge: "merge",
  };
  return typeMap[componentName] || "component"; // Default to noop for unknown components
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
    component: getComponentBaseMapper(component?.name!).props([], node, component!, [], undefined),
  };

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

  // Template node state for drag-edge-to-empty-space functionality
  const [templateNodeId, setTemplateNodeId] = useState<string | null>(null);
  const [newNodeData, setNewNodeData] = useState<NewNodeData | null>(null);
  const [isBuildingBlocksSidebarOpen, setIsBuildingBlocksSidebarOpen] = useState(false);

  // Track template node IDs that have been converted to real nodes and should not be preserved
  const convertedTemplateIdsRef = useRef<Set<string>>(new Set());

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

    setNodes((currentNodes) => {
      // Preserve pending connection nodes and template nodes
      // But exclude template nodes that have been converted to real nodes
      const localOnlyNodes = currentNodes.filter((node) => {
        const isPending = (node.data as any).isPendingConnection;
        const isTemplate = (node.data as any).isTemplate;

        // Always preserve pending connection nodes
        if (isPending) return true;

        // Preserve template nodes UNLESS they've been converted to real nodes
        if (isTemplate) {
          const shouldPreserve = !convertedTemplateIdsRef.current.has(node.id);
          if (shouldPreserve) return true;
        }

        return false;
      });

      const allNodes: Node[] = (blueprint.nodes || [])
        .map((node: ComponentsNode) => {
          // Handle component nodes
          const component = components.find((p: any) => p.name === node.component?.name);
          const blockData = createBlockData(node, component);

          // Find if this node exists in current state to preserve selection
          const existingNode = currentNodes.find((n) => n.id === node.id);

          return {
            id: node.id,
            type: "default",
            data: {
              ...blockData,
              _originalComponent: node.component?.name,
              _originalConfiguration: node.configuration || {},
            },
            position: node.position || { x: 0, y: 0 },
            selected: existingNode?.selected ?? false, // Preserve selection
          };
        })
        .filter(Boolean) as Node[];

      // Check if we have saved positions
      const hasPositions = allNodes.some((node) => node.position && (node.position.x !== 0 || node.position.y !== 0));

      if (hasPositions) {
        // Use saved positions and append local-only nodes
        return [...allNodes, ...localOnlyNodes];
      } else {
        // Apply elk layout for blueprints without saved positions
        // We'll handle this async, so just return current for now
        getLayoutedElements(allNodes, []).then(({ nodes: layoutedNodes }) => {
          setNodes((current) => {
            const localOnly = current.filter(
              (node) => (node.data as any).isTemplate || (node.data as any).isPendingConnection,
            );
            return [...layoutedNodes, ...localOnly];
          });
        });
        return [...allNodes, ...localOnlyNodes];
      }
    });

    setEdges((currentEdges) => {
      // Preserve edges connected to pending connection nodes or template nodes
      // But exclude edges to template nodes that have been converted
      const localOnlyEdges = currentEdges.filter((edge) => {
        const sourceNode = nodes.find((n) => n.id === edge.source);
        const targetNode = nodes.find((n) => n.id === edge.target);

        const sourceIsLocal =
          sourceNode &&
          ((sourceNode.data as any).isPendingConnection ||
            ((sourceNode.data as any).isTemplate && !convertedTemplateIdsRef.current.has(sourceNode.id)));

        const targetIsLocal =
          targetNode &&
          ((targetNode.data as any).isPendingConnection ||
            ((targetNode.data as any).isTemplate && !convertedTemplateIdsRef.current.has(targetNode.id)));

        return sourceIsLocal || targetIsLocal;
      });

      const loadedEdges: Edge[] = (blueprint.edges || []).map((edge: any, index: number) => ({
        id: `e${index}`,
        source: edge.sourceId,
        sourceHandle: edge.channel || "default",
        target: edge.targetId,
        style: { strokeWidth: 3, stroke: "#C9D5E1" },
      }));

      return [...loadedEdges, ...localOnlyEdges];
    });
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

  const handleSaveBlueprint = useCallback(
    async (customNodes?: Node[], customEdges?: Edge[]) => {
      try {
        // Use provided nodes or current nodes from ref
        const rawNodes = customNodes || nodesRef.current || [];
        const currentNodes = Array.isArray(rawNodes) ? rawNodes : [];
        // Filter out template and pending connection nodes and serialize remaining nodes
        const blueprintNodes = currentNodes
          .filter((node) => !node.id.startsWith("template_") && !node.id.startsWith("pending_connection_")) // Exclude template and pending nodes
          .map((node) => {
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

        // Use provided edges or current edges
        const currentEdges = customEdges || edges;
        const blueprintEdges = currentEdges
          .filter(
            (edge) =>
              !edge.source.startsWith("template_") &&
              !edge.target.startsWith("template_") &&
              !edge.source.startsWith("pending_connection_") &&
              !edge.target.startsWith("pending_connection_"),
          )
          .map((edge) => ({
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
    },
    [
      nodes,
      edges,
      blueprintName,
      blueprintDescription,
      blueprintConfiguration,
      blueprintOutputChannels,
      blueprintIcon,
      blueprintColor,
      updateBlueprintMutation,
    ],
  );

  const getNodeEditData = useCallback((nodeId: string) => {
    const node = (nodesRef.current || []).find((n) => n.id === nodeId);
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
    async (nodeId: string, configuration: Record<string, any>, nodeName: string) => {
      saveSnapshot();

      // Check if this is a template node with sourceConnection (needs to be converted to real node)
      if (templateNodeId && newNodeData && (newNodeData as any).sourceConnection) {
        // This is a template node created from dragging edge - convert to real node
        const component = componentsRef.current.find(
          (c: ComponentsComponent) => c.name === newNodeData.buildingBlock.name,
        );
        if (!component) return;

        // Filter configuration to only include visible fields
        const filteredConfiguration = filterVisibleConfiguration(configuration, component.configuration || []);

        // Create new node
        const newNodeId = generateNodeId(component.name!, nodeName.trim());
        const mockNode = {
          component: { name: component.name },
          name: nodeName.trim(),
          configuration: filteredConfiguration,
        };
        const blockData = createBlockData(mockNode, component);

        const newNode: Node = {
          id: newNodeId,
          type: "default",
          position: newNodeData.position
            ? {
                x: Math.round(newNodeData.position.x),
                y: Math.round(newNodeData.position.y),
              }
            : {
                x: (nodesRef.current || []).length * 250,
                y: 100,
              },
          data: {
            ...blockData,
            _originalComponent: component.name,
            _originalConfiguration: filteredConfiguration,
          },
          selected: true,
        };

        // Remove the template node and add the real node - deselect all other nodes
        const nodesWithoutTemplate = (nodesRef.current || [])
          .filter((n) => n.id !== templateNodeId)
          .map((n) => ({ ...n, selected: false }));
        const updatedNodes = [...nodesWithoutTemplate, newNode];

        // Calculate updated edges synchronously
        const sourceConnection = (newNodeData as any).sourceConnection;
        let updatedEdges = edges;

        if (sourceConnection) {
          const newEdge = {
            id: `${sourceConnection.nodeId}--${newNodeId}--${sourceConnection.handleId || "default"}`,
            source: sourceConnection.nodeId,
            target: newNodeId,
            sourceHandle: sourceConnection.handleId || "default",
            style: { strokeWidth: 3, stroke: "#C9D5E1" },
          };
          // Remove any edges connected to the template node and add new edge
          const edgesWithoutTemplate = edges.filter((e) => e.source !== templateNodeId && e.target !== templateNodeId);
          updatedEdges = [...edgesWithoutTemplate, newEdge];
        }

        // Mark this template as converted so it won't be preserved in useEffect
        console.log("[SAVE] Marking template as converted:", templateNodeId);
        console.log("[SAVE] convertedTemplateIdsRef now contains:", Array.from(convertedTemplateIdsRef.current));
        convertedTemplateIdsRef.current.add(templateNodeId);

        // Update state
        console.log("[SAVE] Setting nodes - removing template, adding real node");
        console.log(
          "[SAVE] updatedNodes:",
          updatedNodes.map((n) => ({
            id: n.id,
            isTemplate: (n.data as any).isTemplate,
            isPending: (n.data as any).isPendingConnection,
          })),
        );
        setNodes(updatedNodes);
        setEdges(updatedEdges);

        // Clear template state
        setTemplateNodeId(null);
        setNewNodeData(null);
        setHasUnsavedChanges(true);

        // Save to server with both updated nodes and edges
        await handleSaveBlueprint(updatedNodes, updatedEdges);

        // Clean up the converted template ID after save completes
        // (give it time for the useEffect to run first)
        setTimeout(() => {
          convertedTemplateIdsRef.current.delete(templateNodeId);
        }, 1000);

        return;
      }

      // Check if this is a template node being updated in place (no sourceConnection)
      if (templateNodeId && newNodeData) {
        // Update template node configuration in place
        const updatedNodes = (nodesRef.current || []).map((n) =>
          n.id === templateNodeId
            ? {
                ...n,
                data: {
                  ...(n.data as any),
                  configuration,
                  nodeName,
                },
              }
            : n,
        );
        setNodes(updatedNodes);

        // Clear template state
        setTemplateNodeId(null);
        setNewNodeData(null);
        setHasUnsavedChanges(true);
        return;
      }

      // Regular node configuration save
      const node = (nodesRef.current || []).find((n) => n.id === nodeId);
      if (!node) return;

      const component = componentsRef.current.find((p: any) => p.name === (node.data as any)._originalComponent);
      if (!component) return;

      // Filter configuration to only include visible fields
      const filteredConfiguration = filterVisibleConfiguration(configuration, component.configuration || []);

      // Update existing node
      const updatedNodes = (nodesRef.current || []).map((n) => {
        if (n.id !== nodeId) return n;

        const nodeData = n.data as any;
        const updatedData = {
          ...nodeData,
          label: nodeName.trim(),
          _originalConfiguration: filteredConfiguration,
        };

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
          updatedData.component = getComponentBaseMapper(component.name!).props(
            [],
            updatedNode,
            component,
            [],
            undefined,
          );
        }

        return {
          ...n,
          data: updatedData,
        };
      });

      setNodes(updatedNodes);

      // Save to server immediately with the updated nodes
      await handleSaveBlueprint(updatedNodes);
    },
    [saveSnapshot, handleSaveBlueprint, templateNodeId, newNodeData],
  );

  const handleNodeAdd = useCallback(
    async (newNodeData: NewNodeData) => {
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
        position: newNodeData.position
          ? {
              x: Math.round(newNodeData.position.x),
              y: Math.round(newNodeData.position.y),
            }
          : {
              x: (nodesRef.current || []).length * 250,
              y: 100,
            },
        data: {
          ...blockData,
          _originalComponent: component.name,
          _originalConfiguration: filteredConfiguration,
        },
        selected: true,
      };

      // Mark the current template as converted before filtering it out
      const currentNodes = nodesRef.current || [];
      // Save the template ID before clearing it
      const savedTemplateId = templateNodeId;

      // Only mark the template being saved, not all templates
      if (savedTemplateId) {
        convertedTemplateIdsRef.current.add(savedTemplateId);
      }

      // Update nodes state - remove only the saved template, keep other templates, deselect all, add new node
      const updatedNodes = (() => {
        const filteredNodes = currentNodes
          .filter((n) => {
            // Keep all nodes except:
            // 1. The template being saved (savedTemplateId)
            // 2. Old-style template_ nodes
            if (n.id.startsWith("template_")) return false;
            if (n.id === savedTemplateId) return false;
            return true;
          })
          .map((n) => ({ ...n, selected: false }));
        return [...filteredNodes, newNode];
      })();

      // Handle edge creation if there's a sourceConnection
      let updatedEdges = edges;
      if (newNodeData.sourceConnection) {
        const newEdge = {
          id: `${newNodeData.sourceConnection.nodeId}--${newNodeId}--${newNodeData.sourceConnection.handleId || "default"}`,
          source: newNodeData.sourceConnection.nodeId,
          target: newNodeId,
          sourceHandle: newNodeData.sourceConnection.handleId || "default",
          style: { strokeWidth: 3, stroke: "#C9D5E1" },
        };
        updatedEdges = [...edges, newEdge];
      }

      setNodes(updatedNodes);
      setEdges(updatedEdges);

      // Clear template state
      setTemplateNodeId(null);
      setNewNodeData(null);

      // Save to server immediately with the updated nodes and edges
      await handleSaveBlueprint(updatedNodes, updatedEdges);

      // Clean up the converted template ID after save completes
      if (savedTemplateId) {
        setTimeout(() => {
          convertedTemplateIdsRef.current.delete(savedTemplateId);
        }, 1000);
      }
    },
    [saveSnapshot, handleSaveBlueprint, edges],
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
      const nodeToDuplicate = (nodesRef.current || []).find((node) => node.id === nodeId);
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
          ...(nodeData.component && {
            component: {
              ...nodeData.component,
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

  // Handle dropping edge in empty space to create pending connection node
  const handleConnectionDropInEmptySpace = useCallback(
    (position: { x: number; y: number }, sourceConnection: { nodeId: string; handleId: string | null }) => {
      saveSnapshot();
      const pendingNodeId = `pending_connection_${Date.now()}`;

      // Create a placeholder "New Component" node
      const placeholderNode: Node = {
        id: pendingNodeId,
        type: "default",
        position,
        data: {
          type: "component",
          label: "New Component",
          state: "neutral",
          component: {
            title: "New Component",
            headerColor: "#e5e7eb",
            iconSlug: "Puzzle",
            iconColor: "text-gray-500",
            collapsedBackground: "bg-white",
            hideActionsButton: true,
            includeEmptyState: true,
          },
          isPendingConnection: true,
          sourceConnection,
          emptyState: {
            icon: "Puzzle",
            title: "Select the component from sidebar",
          },
        },
      };

      setNodes((nodes) => [...nodes, placeholderNode]);

      // Check if current template is a configured template (not just pending connection)
      const currentTemplateNode = templateNodeId ? nodes.find((n) => n.id === templateNodeId) : null;
      const isCurrentTemplateConfigured =
        currentTemplateNode?.data &&
        (currentTemplateNode.data as any).isTemplate &&
        !(currentTemplateNode.data as any).isPendingConnection;

      // Only select and set as template if there isn't a configured template being created
      // Allow switching between pending nodes, but prevent overwriting configured templates
      if (!isCurrentTemplateConfigured) {
        // Then update all nodes to set selection (deselect others, select the new one)
        // This needs to happen in a separate setNodes call to ensure ReactFlow processes the selection
        setTimeout(() => {
          setNodes((nodes) =>
            nodes.map((node) => ({
              ...node,
              selected: node.id === pendingNodeId,
            })),
          );
        }, 0);
        setTemplateNodeId(pendingNodeId);
      }

      // Create edge
      const edgeId = `${sourceConnection.nodeId}--${pendingNodeId}--${sourceConnection.handleId || "default"}`;
      setEdges((edges) => [
        ...edges,
        {
          id: edgeId,
          source: sourceConnection.nodeId,
          sourceHandle: sourceConnection.handleId || "default",
          target: pendingNodeId,
          style: { strokeWidth: 3, stroke: "#C9D5E1" },
        },
      ]);

      // Open building blocks sidebar
      setIsBuildingBlocksSidebarOpen(true);
      setHasUnsavedChanges(true);
    },
    [saveSnapshot, nodes, templateNodeId],
  );

  // Handle clicking on a pending connection node
  const handlePendingConnectionNodeClick = useCallback((nodeId: string) => {
    setTemplateNodeId(nodeId);
    setIsBuildingBlocksSidebarOpen(true);
    // Clear any existing template configuration (close ComponentSidebar)
    setNewNodeData(null);
  }, []);

  // Handle clicking on a template node (already configured, re-opening for editing)
  const handleTemplateNodeClick = useCallback(
    (nodeId: string) => {
      const templateNode = nodes.find((n) => n.id === nodeId);
      if (!templateNode) return;

      const buildingBlock = (templateNode.data as any).buildingBlock;

      setTemplateNodeId(nodeId);
      setNewNodeData({
        buildingBlock: buildingBlock,
        nodeName: (templateNode.data as any).nodeName || buildingBlock?.name || "New Component",
        icon: (templateNode.data as any).icon || buildingBlock?.icon || "Box",
        configuration: (templateNode.data as any).configuration || {},
        position: templateNode.position,
        sourceConnection: (templateNode.data as any).sourceConnection as
          | { nodeId: string; handleId: string | null }
          | undefined,
      });
    },
    [nodes],
  );

  // Handle selecting a building block for a pending connection node
  const handleBuildingBlockClick = useCallback(
    (block: any) => {
      if (!templateNodeId) return;

      saveSnapshot();

      // Find the template node - could be either a pending connection or an existing template (from drag-and-drop)
      const templateNode = nodes.find((n) => n.id === templateNodeId);
      if (!templateNode) return;

      const isPendingConnection = (templateNode.data as any).isPendingConnection;
      const isExistingTemplate = (templateNode.data as any).isTemplate && !isPendingConnection;

      // If it's already a configured template (from drag-and-drop), just set the newNodeData
      if (isExistingTemplate) {
        setNewNodeData({
          buildingBlock: block,
          nodeName: block.name || "",
          displayLabel: block.label || block.name || "",
          configuration: {},
          position: templateNode.position,
          sourceConnection: (templateNode.data as any).sourceConnection,
        });
        setIsBuildingBlocksSidebarOpen(false);
        return;
      }

      // Otherwise, it's a pending connection node - convert it to a template
      if (!isPendingConnection) return;

      const pendingNodePosition = templateNode.position;
      const pendingNodeSourceConnection = (templateNode.data as any).sourceConnection;

      // Update the node to template state
      setNodes((currentNodes) =>
        currentNodes.map((n) =>
          n.id === templateNodeId
            ? {
                ...n,
                data: {
                  type: "component",
                  label: block.label || block.name || "New Component",
                  state: "neutral",
                  component: {
                    title: block.label || block.name || "New Component",
                    headerColor: "#e5e7eb",
                    iconSlug: block.icon,
                    iconColor: "text-indigo-700",
                    collapsedBackground: "bg-white",
                    hideActionsButton: true,
                    includeEmptyState: true,
                  },
                  isTemplate: true,
                  isPendingConnection: false, // Remove pending connection flag
                  buildingBlock: block,
                  tempConfiguration: {},
                  tempNodeName: block.name || "",
                  sourceConnection: pendingNodeSourceConnection,
                  configuration: {},
                  nodeName: block.name || "",
                  icon: block.icon,
                },
              }
            : n,
        ),
      );

      // Set newNodeData with the pending node's data
      setNewNodeData({
        buildingBlock: block,
        nodeName: block.name || "",
        displayLabel: block.label || block.name || "",
        configuration: {},
        position: pendingNodePosition,
        sourceConnection: pendingNodeSourceConnection,
      });

      setIsBuildingBlocksSidebarOpen(false);
      setHasUnsavedChanges(true);
    },
    [templateNodeId, saveSnapshot],
  );

  // Handle canceling template creation
  const handleCancelTemplate = useCallback(() => {
    if (templateNodeId) {
      setTemplateNodeId(null);
      setNewNodeData(null);
    }
  }, [templateNodeId]);

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

  const handleSave = useCallback(() => {
    return handleSaveBlueprint();
  }, [handleSaveBlueprint]);

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
        onAddTemplateNode={(templateNode) => {
          setNodes((nds) => [...nds, templateNode]);
          setTemplateNodeId(templateNode.id);

          // Set newNodeData to populate the ComponentSidebar
          const buildingBlock = (templateNode.data as any).buildingBlock;
          if (buildingBlock) {
            setNewNodeData({
              buildingBlock: buildingBlock,
              nodeName: buildingBlock.name || "",
              displayLabel: buildingBlock.label || buildingBlock.name || "",
              icon: buildingBlock.icon || "circle-off",
              configuration: {},
              position: templateNode.position,
              sourceConnection: (templateNode.data as any).sourceConnection,
            });
          }
        }}
        onRemoveTemplateNode={(nodeId) => setNodes((nds) => nds.filter((n) => n.id !== nodeId))}
        organizationId={organizationId}
        components={components}
        onSave={handleSave}
        isSaving={updateBlueprintMutation.isPending}
        unsavedMessage={hasUnsavedChanges ? "You have unsaved changes" : undefined}
        saveButtonHidden={!hasUnsavedChanges}
        saveIsPrimary={hasUnsavedChanges}
        onUndo={handleRevert}
        canUndo={initialBlueprintSnapshot !== null}
        templateNodeId={templateNodeId}
        newNodeData={newNodeData}
        isBuildingBlocksSidebarOpen={isBuildingBlocksSidebarOpen}
        onBuildingBlocksSidebarToggle={setIsBuildingBlocksSidebarOpen}
        onConnectionDropInEmptySpace={handleConnectionDropInEmptySpace}
        onPendingConnectionNodeClick={handlePendingConnectionNodeClick}
        onTemplateNodeClick={handleTemplateNodeClick}
        onBuildingBlockClick={handleBuildingBlockClick}
        onCancelTemplate={handleCancelTemplate}
      />
    </>
  );
};

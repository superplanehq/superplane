import { useMemo, useCallback } from "react";
import { useParams } from "react-router-dom";
import { useQueries, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { showSuccessToast, showErrorToast } from "@/utils/toast";

import {
  BlueprintsBlueprint,
  ComponentsComponent,
  ComponentsEdge,
  ComponentsNode,
  TriggersTrigger,
  WorkflowsWorkflow,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
  workflowsInvokeNodeExecutionAction,
} from "@/api-client";

import { useWorkflow, useTriggers, useUpdateWorkflow, nodeEventsQueryOptions, nodeExecutionsQueryOptions, nodeQueueItemsQueryOptions, workflowKeys } from "@/hooks/useWorkflowData";
import { useBlueprints, useComponents } from "@/hooks/useBlueprintData";
import { CanvasEdge, CanvasNode, CanvasPage, SidebarData, NodeEditData } from "@/ui/CanvasPage";
import { CompositeProps, LastRunState } from "@/ui/composite";
import { getTriggerRenderer } from "./renderers";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { formatTimeAgo } from "@/utils/date";

export function WorkflowPageV2() {
  const { organizationId, workflowId } = useParams<{
    organizationId: string;
    workflowId: string;
  }>();

  const queryClient = useQueryClient();
  const updateWorkflowMutation = useUpdateWorkflow(organizationId!, workflowId!);
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId!);
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!);
  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!);

  //
  // Get last event for triggers
  // Memoize to prevent unnecessary re-renders and query recreations
  //
  const triggerNodes = useMemo(
    () => workflow?.nodes?.filter((node) => node.type === 'TYPE_TRIGGER') || [],
    [workflow?.nodes]
  );

  const compositeNodes = useMemo(
    () => workflow?.nodes?.filter((node) => node.type === 'TYPE_BLUEPRINT') || [],
    [workflow?.nodes]
  );

  const componentNodes = useMemo(
    () => workflow?.nodes?.filter((node) => node.type === 'TYPE_COMPONENT') || [],
    [workflow?.nodes]
  );

  // Fetch executions for both composite and component nodes
  const nodesWithExecutions = useMemo(
    () => [...compositeNodes, ...componentNodes],
    [compositeNodes, componentNodes]
  );

  const { eventsMap: nodeEventsMap, isLoading: nodeEventsLoading } = useTriggerNodeEvents(workflowId!, triggerNodes);
  const { nodeExecutionsMap, nodeQueueItemsMap, isLoading: nodeDataLoading } = useCompositeNodeData(workflowId!, nodesWithExecutions);

  const { nodes, edges } = useMemo(
    () => {
      if (!workflow) return { nodes: [], edges: [] };
      return prepareData(workflow, triggers, blueprints, components, nodeEventsMap, nodeExecutionsMap, nodeQueueItemsMap, workflowId!, queryClient);
    },
    [workflow, triggers, blueprints, nodeEventsMap, nodeExecutionsMap, nodeQueueItemsMap, workflowId, queryClient]
  );

  const getSidebarData = useCallback((nodeId: string): SidebarData | null => {
    const node = workflow?.nodes?.find((n) => n.id === nodeId);
    if (!node) return null;

    return prepareSidebarData(
      node,
      blueprints,
      components,
      nodeExecutionsMap,
      nodeQueueItemsMap
    );
  }, [workflow, blueprints, components, nodeExecutionsMap, nodeQueueItemsMap]);

  const getNodeEditData = useCallback((nodeId: string): NodeEditData | null => {
    const node = workflow?.nodes?.find((n) => n.id === nodeId);
    if (!node) return null;

    // Get configuration fields from metadata based on node type
    let configurationFields: ComponentsComponent['configuration'] = [];

    if (node.type === "TYPE_BLUEPRINT") {
      const blueprintMetadata = blueprints.find((b) => b.id === node.blueprint?.id);
      configurationFields = blueprintMetadata?.configuration || [];
    } else if (node.type === "TYPE_COMPONENT") {
      const componentMetadata = components.find((c) => c.name === node.component?.name);
      configurationFields = componentMetadata?.configuration || [];
    } else if (node.type === "TYPE_TRIGGER") {
      const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
      configurationFields = triggerMetadata?.configuration || [];
    }

    return {
      nodeId: node.id!,
      nodeName: node.name!,
      configuration: node.configuration || {},
      configurationFields,
    };
  }, [workflow, blueprints, components, triggers]);

  const handleNodeConfigurationSave = useCallback((nodeId: string, updatedConfiguration: Record<string, any>, updatedNodeName: string) => {
    if (!workflow || !organizationId || !workflowId) return;

    // Update the node's configuration and name in local cache only
    const updatedNodes = workflow.nodes?.map((node) =>
      node.id === nodeId
        ? { ...node, configuration: updatedConfiguration, name: updatedNodeName }
        : node
    );

    const updatedWorkflow = {
      ...workflow,
      nodes: updatedNodes,
    };

    // Update local cache without triggering API call
    queryClient.setQueryData(
      workflowKeys.detail(organizationId, workflowId),
      updatedWorkflow
    );
  }, [workflow, organizationId, workflowId, queryClient]);

  const handleSave = useCallback(async (canvasNodes: CanvasNode[]) => {
    if (!workflow || !organizationId || !workflowId) return;

    // Map canvas nodes back to ComponentsNode format with updated positions
    const updatedNodes = workflow.nodes?.map((node) => {
      const canvasNode = canvasNodes.find((cn) => cn.id === node.id);
      if (canvasNode) {
        return {
          ...node,
          position: {
            x: Math.round(canvasNode.position.x),
            y: Math.round(canvasNode.position.y),
          },
        };
      }
      return node;
    });

    // Save previous state for rollback
    const previousWorkflow = queryClient.getQueryData(
      workflowKeys.detail(organizationId, workflowId)
    );

    // Optimistically update the cache to prevent flicker
    const updatedWorkflow = {
      ...workflow,
      nodes: updatedNodes,
    };

    queryClient.setQueryData(
      workflowKeys.detail(organizationId, workflowId),
      updatedWorkflow
    );

    try {
      await updateWorkflowMutation.mutateAsync({
        name: workflow.name!,
        description: workflow.description,
        nodes: updatedNodes,
        edges: workflow.edges,
      });

      showSuccessToast("Workflow saved successfully");
    } catch (error) {
      console.error("Failed to save workflow:", error);
      showErrorToast("Failed to save workflow");

      // Rollback to previous state on error
      if (previousWorkflow) {
        queryClient.setQueryData(
          workflowKeys.detail(organizationId, workflowId),
          previousWorkflow
        );
      }
    }
  }, [workflow, organizationId, workflowId, updateWorkflowMutation, queryClient]);

  // Show loading indicator while data is being fetched
  if (workflowLoading || triggersLoading || blueprintsLoading || componentsLoading || nodeEventsLoading || nodeDataLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-gray-500" />
          <p className="text-sm text-gray-500">Loading workflow...</p>
        </div>
      </div>
    );
  }

  if (!workflow) {
    return null;
  }

  return (
    <CanvasPage
      title={workflow.name!}
      nodes={nodes}
      edges={edges}
      organizationId={organizationId}
      getSidebarData={getSidebarData}
      getNodeEditData={getNodeEditData}
      onNodeConfigurationSave={handleNodeConfigurationSave}
      onSave={handleSave}
    />
  );
}

function useTriggerNodeEvents(workflowId: string, triggerNodes: ComponentsNode[]) {
  const results = useQueries({
    queries: triggerNodes.map((node) =>
      nodeEventsQueryOptions(workflowId, node.id!, { limit: 1 })
    ),
  });

  // Check if any queries are still loading
  const isLoading = results.some((result) => result.isLoading);

  // Build a map of nodeId -> last event
  // Memoize to prevent unnecessary re-renders downstream
  const eventsMap = useMemo(() => {
    const map: Record<string, WorkflowsWorkflowEvent> = {};
    triggerNodes.forEach((node, index) => {
      const result = results[index];
      if (result.data?.events && result.data.events.length > 0) {
        map[node.id!] = result.data.events[0];
      }
    });
    return map;
  }, [results, triggerNodes]);

  return { eventsMap, isLoading };
}

function useCompositeNodeData(workflowId: string, compositeNodes: ComponentsNode[]) {
  // Fetch last executions for each composite node
  const executionResults = useQueries({
    queries: compositeNodes.map((node) =>
      nodeExecutionsQueryOptions(workflowId, node.id!, { limit: 1 })
    ),
  });

  // Fetch queue items for each composite node
  const queueItemResults = useQueries({
    queries: compositeNodes.map((node) =>
      nodeQueueItemsQueryOptions(workflowId, node.id!)
    ),
  });

  // Check if any queries are still loading
  const isLoading =
    executionResults.some((result) => result.isLoading) ||
    queueItemResults.some((result) => result.isLoading);

  // Build maps of nodeId -> data
  // Memoize to prevent unnecessary re-renders downstream
  const nodeExecutionsMap = useMemo(() => {
    const map: Record<string, any> = {};
    compositeNodes.forEach((node, index) => {
      const result = executionResults[index];
      if (result.data?.executions && result.data.executions.length > 0) {
        map[node.id!] = result.data.executions;
      }
    });
    return map;
  }, [executionResults, compositeNodes]);

  const nodeQueueItemsMap = useMemo(() => {
    const map: Record<string, any> = {};
    compositeNodes.forEach((node, index) => {
      const result = queueItemResults[index];
      if (result.data?.items && result.data.items.length > 0) {
        map[node.id!] = result.data.items;
      }
    });
    return map;
  }, [queueItemResults, compositeNodes]);

  return { nodeExecutionsMap, nodeQueueItemsMap, isLoading };
}

function prepareData(
  data: WorkflowsWorkflow,
  triggers: TriggersTrigger[],
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  nodeEventsMap: Record<string, any>,
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem>,
  workflowId: string,
  queryClient: any
): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
} {
  const edges = data?.edges!.map(prepareEdge);
  const nodes = data?.nodes!.map((node) => {
    return prepareNode(
      node,
      triggers,
      blueprints,
      components,
      nodeEventsMap,
      nodeExecutionsMap,
      nodeQueueItemsMap,
      workflowId,
      queryClient
    )
  });

  return { nodes, edges };
}

function prepareTriggerNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  nodeEventsMap: Record<string, WorkflowsWorkflowEvent>
): CanvasNode {
  const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
  const renderer = getTriggerRenderer(node.trigger?.name || "");
  const lastEvent = nodeEventsMap[node.id!];

  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "trigger",
      label: node.name!,
      state: "pending" as const,
      trigger: renderer.getTriggerProps(node, triggerMetadata!, lastEvent),
    },
  };
}

function prepareCompositeNode(
  node: ComponentsNode,
  blueprints: any[],
  nodeExecutionsMap: Record<string, any>,
  nodeQueueItemsMap: Record<string, any>
): CanvasNode {
  const blueprintMetadata = blueprints.find((b) => b.id === node.blueprint?.id);
  const color = blueprintMetadata?.color || "indigo";
  const executions = nodeExecutionsMap[node.id!] || [];
  const queueItems = nodeQueueItemsMap[node.id!] || [];

  let canvasNode: CanvasNode = {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "composite",
      label: node.name!,
      state: "pending" as const,
      composite: {
        iconSlug: blueprintMetadata?.icon || "boxes",
        iconColor: getColorClass(color),
        iconBackground: getBackgroundColorClass(color),
        headerColor: getBackgroundColorClass(color),
        collapsedBackground: getBackgroundColorClass(color),
        title: node.name!,
        description: blueprintMetadata?.description,
        parameters: Object.keys(node.configuration!).map((key) => {
          return {
            icon: "cog",
            items: [`${node.configuration![key]}`],
          };
        })
      },
    },
  };

  if (executions.length > 0) {
    const execution = executions[0];
    (canvasNode.data.composite as CompositeProps).lastRunItem = {
      title: "",
      subtitle: "",
      receivedAt: new Date(execution.createdAt),
      state: getRunItemState(execution),

      //
      // TODO: we should either load child executions from /children endpoint,
      // or return them as part of the execution payload here.
      //
      // TODO: what is ChildEventsInfo.waitingInfos supposed to be???
      //
      childEventsInfo: {
        count: 3,
        waitingInfos: []
      },

      //
      // TODO: from the storybook pages, it seems like this comes from the root event.
      // We kind of have this in execution.input, but it's the raw data.
      //
      values: {},
    }
  }

  if (queueItems.length > 0) {
    (canvasNode.data.composite as CompositeProps).nextInQueue = {
      title: "",
      subtitle: "",
      receivedAt: new Date(queueItems[0].createdAt),
    }
  }

  return canvasNode;
}

function getRunItemState(execution: WorkflowsWorkflowNodeExecution): LastRunState {
  if (execution.state == "STATE_STARTED") {
    return "running";
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}

function prepareNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  nodeEventsMap: Record<string, any>,
  nodeExecutionsMap: Record<string, any>,
  nodeQueueItemsMap: Record<string, any>,
  workflowId: string,
  queryClient: any
): CanvasNode {
  switch (node.type) {
    case "TYPE_TRIGGER":
      return prepareTriggerNode(node, triggers, nodeEventsMap);
    case "TYPE_BLUEPRINT":
      return prepareCompositeNode(node, blueprints, nodeExecutionsMap, nodeQueueItemsMap);
    default:
      return prepareComponentNode(node, blueprints, components, nodeExecutionsMap, nodeQueueItemsMap, workflowId, queryClient);
  }
}

function prepareComponentNode(
  node: ComponentsNode,
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, any>,
  nodeQueueItemsMap: Record<string, any>,
  workflowId: string,
  queryClient: any
): CanvasNode {
  switch (node.component?.name) {
    case "approval":
      return prepareApprovalNode(node, components, nodeExecutionsMap, workflowId, queryClient);
  }

  //
  // TODO: render other component-type nodes as composites for now
  //
  return prepareCompositeNode(node, blueprints, nodeExecutionsMap, nodeQueueItemsMap);
}

function prepareApprovalNode(
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  workflowId: string,
  queryClient: any
): CanvasNode {
  const metadata = components.find((c) => c.name === "approval");
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;
  const executionMetadata = execution?.metadata as any;

  // Map backend records to approval items
  const approvals = (executionMetadata?.records || []).map((record: any) => {
    const isPending = record.state === 'pending';
    const isExecutionActive = execution?.state === 'STATE_STARTED';

    const approvalComment = record.approval?.comment;
    const hasApprovalArtifacts = record.state === 'approved' && approvalComment;

    return {
      id: `${record.index}`,
      title: record.type === 'user' && record.user ? record.user.name || record.user.email :
             record.type === 'role' && record.role ? record.role :
             record.type === 'group' && record.group ? record.group : 'Unknown',
      approved: record.state === 'approved',
      rejected: record.state === 'rejected',
      approverName: record.user?.name,
      approverAvatar: record.user?.avatarUrl,
      rejectionComment: record.rejection?.reason,
      interactive: isPending && isExecutionActive,
      requireArtifacts: isPending && isExecutionActive ? [
        {
          label: "comment",
          optional: true,
        }
      ] : undefined,
      artifacts: hasApprovalArtifacts ? {
        "Comment": approvalComment,
      } : undefined,
      artifactCount: hasApprovalArtifacts ? 1 : undefined,
      onApprove: async (artifacts?: Record<string, string>) => {
        if (!execution?.id) return;

        try {
          await workflowsInvokeNodeExecutionAction(
            withOrganizationHeader({
              path: {
                workflowId: workflowId,
                executionId: execution.id,
                actionName: 'approve',
              },
              body: {
                parameters: {
                  index: record.index,
                  comment: artifacts?.comment,
                },
              },
            })
          );

          queryClient.invalidateQueries({ queryKey: workflowKeys.nodeExecution(workflowId, node.id!) });
        } catch (error: any) {
          console.error('Failed to approve:', error);
        }
      },
      onReject: async (comment?: string) => {
        if (!execution?.id) return;

        try {
          await workflowsInvokeNodeExecutionAction(
            withOrganizationHeader({
              path: {
                workflowId: workflowId,
                executionId: execution.id,
                actionName: 'reject',
              },
              body: {
                parameters: {
                  index: record.index,
                  reason: comment,
                },
              },
            })
          );

          queryClient.invalidateQueries({ queryKey: workflowKeys.nodeExecution(workflowId, node.id!) });
        } catch (error: any) {
          console.error('Failed to reject:', error);
        }
      },
    };
  });

  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "approval",
      label: node.name!,
      state: "pending" as const,
      approval: {
        iconSlug: metadata?.icon || "hand",
        iconColor: getColorClass(metadata?.color || "orange"),
        iconBackground: getBackgroundColorClass(metadata?.color || "orange"),
        headerColor: getBackgroundColorClass(metadata?.color || "orange"),
        collapsedBackground: getBackgroundColorClass(metadata?.color || "orange"),
        title: node.name!,
        description: metadata?.description,
        receivedAt: execution ? new Date(execution.createdAt!) : undefined,
        approvals,

        //
        // TODO: this also comes from the input event
        //
        awaitingEvent: execution?.state === 'STATE_STARTED' ? {
          title: "",
          subtitle: "",
        } : undefined,
      }
    },
  };
}

function prepareEdge(edge: ComponentsEdge): CanvasEdge {
  const id = `${edge.sourceId!}--${edge.targetId!}--${edge.channel!}`;

  return {
    id: id,
    source: edge.sourceId!,
    target: edge.targetId!,
  };
}

function prepareSidebarData(
  node: ComponentsNode,
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>
): SidebarData {
  const executions = nodeExecutionsMap[node.id!] || [];
  const queueItems = nodeQueueItemsMap[node.id!] || [];

  // Get metadata based on node type
  let metadata: any;
  let nodeTitle = node.name || "Unknown";
  let iconSlug = "boxes";
  let color = "indigo";

  if (node.type === "TYPE_BLUEPRINT") {
    metadata = blueprints.find((b) => b.id === node.blueprint?.id);
    if (metadata) {
      iconSlug = metadata.icon || "boxes";
      color = metadata.color || "indigo";
    }
  } else if (node.type === "TYPE_COMPONENT") {
    metadata = components.find((c) => c.name === node.component?.name);
    if (metadata) {
      iconSlug = metadata.icon || "boxes";
      color = metadata.color || "indigo";
    }
  }

  // Convert executions to sidebar events (latest events)
  const latestEvents = executions.slice(0, 5).map((execution) => {
    const state = execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED"
      ? "processed" as const
      : execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED"
      ? "discarded" as const
      : "waiting" as const;

    const timestamp = execution.createdAt
      ? formatTimeAgo(new Date(execution.createdAt)).replace(' ago', '')
      : '';

    return {
      title: execution.id || "Execution",
      subtitle: timestamp,
      state,
      isOpen: false,
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      values: execution.input as Record<string, string> || {},
      childEventsInfo: {
        count: 0,
        waitingInfos: []
      }
    };
  });

  // Convert queue items to sidebar events (next in queue)
  const nextInQueueEvents = queueItems.slice(0, 5).map((item) => {
    const timestamp = item.createdAt
      ? formatTimeAgo(new Date(item.createdAt)).replace(' ago', '')
      : '';

    return {
      title: item.id || "Queued",
      subtitle: timestamp,
      state: "waiting" as const,
      isOpen: false,
      receivedAt: item.createdAt ? new Date(item.createdAt) : undefined,
      childEventsInfo: {
        count: 0,
        waitingInfos: []
      }
    };
  });

  // Build metadata from node configuration
  const metadataItems = [
    {
      icon: "cog",
      label: `Node ID: ${node.id}`,
    },
  ];

  // Add configuration fields to metadata (only simple types)
  if (node.configuration) {
    Object.entries(node.configuration).forEach(([key, value]) => {
      // Only include simple types (string, number, boolean)
      // Exclude objects, arrays, null, undefined
      const valueType = typeof value;
      const isSimpleType =
        valueType === 'string' ||
        valueType === 'number' ||
        valueType === 'boolean';

      if (isSimpleType) {
        metadataItems.push({
          icon: "settings",
          label: `${key}: ${value}`,
        });
      }
    });
  }

  return {
    latestEvents,
    nextInQueueEvents,
    metadata: metadataItems,
    title: nodeTitle,
    iconSlug,
    iconColor: getColorClass(color),
    iconBackground: getBackgroundColorClass(color),
    moreInQueueCount: Math.max(0, queueItems.length - 5),
  };
}

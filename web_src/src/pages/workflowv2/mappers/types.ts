import {
  ComponentsComponent,
  ComponentsNode,
  TriggersTrigger,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventState, EventStateMap } from "@/ui/componentBase";
import { TriggerProps } from "@/ui/trigger";
import { QueryClient } from "@tanstack/react-query";
import { ReactNode } from "react";

/**
 * A trigger renderer converts backend data into UI props for a specific trigger type.
 * Each trigger type can register its own renderer with custom logic.
 *
 * To add a new trigger type:
 * 1. Create a new file in this renderers folder (e.g., 'mytrigger.ts')
 * 2. Implement the TriggerRenderer interface
 * 3. Export it from index.ts and add it to the registry
 */
export interface TriggerRenderer {
  /**
   * Converts node and trigger metadata from the backend into props for the Trigger UI component.
   *
   * @param node The node from the backend
   * @param trigger The trigger metadata from the backend
   * @returns the props needed to render the Trigger UI component
   */
  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: any) => TriggerProps;

  /**
   * Display values for the root event.
   * @param event The root event from the backend
   * @returns The values to display
   */
  getRootEventValues: (event: WorkflowsWorkflowEvent) => Record<string, string>;

  /**
   * Get the title and subtitle for the trigger.
   * @param node The node from the backend
   * @param trigger The trigger metadata from the backend
   * @returns The title and subtitle to display
   */
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent) => { title: string; subtitle: string };
}

export interface ComponentBaseMapper {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
    additionalData?: unknown,
  ): ComponentBaseProps;

  subtitle?(
    node: ComponentsNode,
    execution: WorkflowsWorkflowNodeExecution,
    additionalData?: unknown,
  ): string | React.ReactNode;
}

/**
 * A component additional data builder creates component-specific data
 * that cannot be derived from the standard parameters alone.
 */
export interface ComponentAdditionalDataBuilder {
  buildAdditionalData(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    workflowId: string,
    queryClient: QueryClient,
    organizationId?: string,
  ): unknown;
}

/**
 * A state function that determines the current state based on execution data
 */
export type StateFunction = (execution: WorkflowsWorkflowNodeExecution) => EventState;

/**
 * Event state registry for components with custom state logic and styling
 */
export interface EventStateRegistry {
  stateMap: EventStateMap;
  getState: StateFunction;
}

/**
 * A custom field renderer renders additional UI elements in the settings tab
 * for specific component/trigger types
 */
export interface CustomFieldRenderer {
  /**
   * Render custom UI for the given node configuration
   * @param node The node from the backend
   * @param configuration Current node configuration
   * @returns React node to render
   */
  render(node: ComponentsNode, configuration: Record<string, unknown>): ReactNode;
}

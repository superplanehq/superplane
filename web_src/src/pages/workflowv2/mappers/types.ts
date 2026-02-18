import { CanvasNodeExecutionResult, CanvasNodeExecutionResultReason, CanvasNodeExecutionState } from "@/api-client";
import { ComponentBaseProps, EventState, EventStateMap } from "@/ui/componentBase";
import { TriggerProps } from "@/ui/trigger";
import { QueryClient } from "@tanstack/react-query";
import { ReactNode } from "react";

/**
 * A trigger renderer converts backend data into UI props for a specific trigger type.
 * Each trigger type can register its own renderer with custom logic.
 *
 * To add a new trigger type:
 * 1. Create a new file in this mappers folder (e.g., 'mytrigger.ts')
 * 2. Implement the TriggerRenderer interface
 * 3. Export it from index.ts and add it to the registry
 */
export interface TriggerRenderer {
  /**
   * Converts node and trigger metadata from the backend into props for the Trigger UI component.
   *
   * @param context The context for the trigger renderer
   * @returns the props needed to render the Trigger UI component
   */
  getTriggerProps: (context: TriggerRendererContext) => TriggerProps;

  /**
   * Display values for the root event.
   * @param context The context for the trigger event
   * @returns The values to display
   */
  getRootEventValues: (context: TriggerEventContext) => Record<string, any>;

  /**
   * Get the title and subtitle for the trigger.
   * @param context The context for the trigger event
   * @returns The title and subtitle to display
   */
  getTitleAndSubtitle: (context: TriggerEventContext) => { title: string; subtitle: string };
}

export type TriggerEventContext = {
  event: EventInfo;
};

export type TriggerRendererContext = {
  node: NodeInfo;
  definition: ComponentDefinition;
  lastEvent: EventInfo;
};

export type EventInfo =
  | {
      id: string;
      createdAt: string;
      customName?: string;
      data: any;
      nodeId: string;
      type: string;
    }
  | undefined;

export type ExecutionInfo = {
  id: string;
  createdAt: string;
  updatedAt: string;
  state: CanvasNodeExecutionState;
  result: CanvasNodeExecutionResult;
  resultReason: CanvasNodeExecutionResultReason;
  resultMessage: string;
  metadata: any;
  configuration: any;
  rootEvent: EventInfo;
  input?: {
    [key: string]: unknown;
  };
  outputs?: {
    [key: string]: unknown;
  };
};

export type QueueItemInfo = {
  id: string;
  createdAt: string;
  rootEvent: EventInfo;
};

export interface NodeInfo {
  id: string;
  name: string;
  componentName: string;
  isCollapsed: boolean;
  configuration?: unknown;
  metadata?: unknown;
}

export interface ComponentDefinition {
  name: string;
  label: string;
  description: string;
  icon: string;
  color: string;
}

export interface ComponentBaseMapper {
  props(context: ComponentBaseContext): ComponentBaseProps;
  subtitle(context: SubtitleContext): string | React.ReactNode;
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any>;
}

export type ComponentBaseContext = {
  nodes: NodeInfo[];
  node: NodeInfo;
  componentDefinition: ComponentDefinition;
  lastExecutions: ExecutionInfo[];
  nodeQueueItems?: QueueItemInfo[];
  additionalData?: unknown;
};

export type SubtitleContext = {
  node: NodeInfo;
  execution: ExecutionInfo;
  additionalData?: unknown;
};

export type ExecutionDetailsContext = {
  nodes: NodeInfo[];
  node: NodeInfo;
  execution: ExecutionInfo;
};

/**
 * A component additional data builder creates component-specific data
 * that cannot be derived from the standard parameters alone.
 */
export interface ComponentAdditionalDataBuilder {
  buildAdditionalData(context: AdditionalDataBuilderContext): unknown;
}

export type AdditionalDataBuilderContext = {
  nodes: NodeInfo[];
  node: NodeInfo;
  componentDefinition: ComponentDefinition;
  lastExecutions: ExecutionInfo[];
  canvasId: string;
  queryClient: QueryClient;
  organizationId?: string;
  currentUser?: User;
};

export type User = {
  id?: string;
  email?: string;
};

/**
 * A state function that determines the current state based on execution data
 */
export type StateFunction = (execution: ExecutionInfo) => EventState;

/**
 * Event state registry for components with custom state logic and styling
 */
export interface EventStateRegistry {
  stateMap: EventStateMap;
  getState: StateFunction;
}

/**
 * A custom field renderer renders additional UI elements on canvas nodes
 * for specific component/trigger types. Can be used for both settings sidebar
 * (via getCustomFieldRenderer) and canvas nodes (via customField prop).
 */
export interface CustomFieldRendererContext {
  onRun?: (initialData?: string) => void;
  /** Full integration object when editing an app trigger/component (e.g. for incident webhook status) */
  integration?: import("@/api-client").OrganizationsIntegration;
}

export interface CustomFieldRenderer {
  /**
   * Render custom UI for the given node configuration
   * @param node The node from the backend
   * @param context Optional context (e.g., onRun, integration for app nodes)
   * @returns React node to render
   */
  render(node: NodeInfo, context?: CustomFieldRendererContext): ReactNode;
}

export interface OutputPayload {
  type: string;
  timestamp: string;
  data: any;
}

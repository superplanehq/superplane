import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent, WorkflowsWorkflowNodeExecution } from "@/api-client";
import { ComponentBaseProps } from "@/ui/componentBase";
import { TriggerProps } from "@/ui/trigger";

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
    lastExecution: WorkflowsWorkflowNodeExecution | null,
  ): ComponentBaseProps;
}

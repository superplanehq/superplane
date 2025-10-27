import { ComponentsNode, TriggersTrigger } from "@/api-client";
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
  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger) => TriggerProps;
}

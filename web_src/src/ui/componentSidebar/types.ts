import { WorkflowsWorkflowEvent, WorkflowsWorkflowNodeExecution } from "@/api-client";
import { ChildEventsState } from "../composite";

export interface SidebarEvent {
  // Unique UI identifier for the item (remains stable across types)
  id: string;
  title: string;
  subtitle?: string | React.ReactNode;
  state: ChildEventsState;
  isOpen: boolean;
  receivedAt?: Date;
  values?: Record<string, string>;
  originalEvent?: WorkflowsWorkflowEvent;
  originalExecution?: WorkflowsWorkflowNodeExecution;

  // Optional specific identifiers to avoid overloading `id`
  // Present for execution items
  executionId?: string;
  nodeId?: string;
  // Present for trigger events
  triggerEventId?: string;
  // Optional explicit kind for clarity
  kind?: "execution" | "trigger" | "queue";
}

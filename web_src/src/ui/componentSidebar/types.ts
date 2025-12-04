import { ChildEventsState } from "../composite";

export interface SidebarEvent {
  // Unique UI identifier for the item (remains stable across types)
  id: string;
  title: string;
  subtitle?: string;
  state: ChildEventsState;
  isOpen: boolean;
  receivedAt?: Date;
  values?: Record<string, string>;

  // Optional specific identifiers to avoid overloading `id`
  // Present for execution items
  executionId?: string;
  nodeId?: string;
  // Present for trigger events
  triggerEventId?: string;
  // Optional explicit kind for clarity
  kind?: "execution" | "trigger" | "queue";
  // Component type for renderer selection
  componentType?: string;
  // Event data for component-specific rendering
  eventData?: Record<string, unknown>;
}

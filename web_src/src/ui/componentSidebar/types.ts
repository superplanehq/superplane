import { ChildEventsInfo } from "../childEvents";
import { ChildEventsState } from "../composite";

export interface SidebarEvent {
  id: string;
  title: string;
  subtitle?: string;
  state: ChildEventsState;
  isOpen: boolean;
  receivedAt?: Date;
  values?: Record<string, string>;
  childEventsInfo?: ChildEventsInfo;
}

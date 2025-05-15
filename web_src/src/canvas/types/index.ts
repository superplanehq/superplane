// Define interfaces for our data types to ensure type safety
import { Connection, Condition, RunTemplate } from './flow';

export interface Stage {
  id: string;
  name: string;
  status?: string;
  labels?: string[];
  timestamp?: string;
  icon?: string;
  queue?: string[];
  connections?: Connection[];
  conditions?: Condition[];
  run_template?: RunTemplate;
  [key: string]: any;
}

export interface EventSource {
  id: string;
  name: string;
  url?: string;
  type?: string;
  filters?: string[];
  filter_operator?: string;
  lastEvent?: {
    type: string;
    release: string;
    timestamp: string;
  };
  [key: string]: any;
}

export interface CanvasData {
  canvas: Record<string, any>;
  stages: Stage[];
  event_sources: EventSource[];
}

// We need this type for the live_react handlers
export interface CanvasInitialData extends CanvasData {
  handleEvent: unknown;
  removeHandleEvent: unknown;
  pushEvent: unknown;
}

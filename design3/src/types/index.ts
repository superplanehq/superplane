import { Node, Edge, NodeProps, EdgeProps } from '@xyflow/react';

// Base component props
export interface BaseComponentProps {
  className?: string;
  id?: string;
}

// Workflow Editor Types
export interface WorkflowData extends Record<string, unknown> {
  id: string;
  label: string;
  type: 'deployment' | 'action' | 'condition';
  status?: 'running' | 'success' | 'failed' | 'pending' | 'stopped';
  hasHealthCheck?: boolean;
  healthCheckStatus?: 'healthy' | 'unhealthy' | 'unknown';
  style?: {
    width?: number;
    height?: number;
  };
}

export interface WorkflowNode extends Node {
  data: WorkflowData;
}

export interface WorkflowEdge extends Edge {
  animated?: boolean;
}

// Component Props
export interface DeploymentCardStageProps extends NodeProps {
  data: WorkflowData;
  selected: boolean;
  onIconAction?: (action: string) => void;
  onDelete?: (id: string) => void;
}

export interface RunItemProps extends BaseComponentProps {
  status: 'running' | 'success' | 'failed' | 'pending' | 'stopped';
  title: string;
  subtitle?: string;
  duration?: string;
  onClick?: () => void;
}

export interface MessageItemProps extends BaseComponentProps {
  type: 'info' | 'warning' | 'error' | 'success';
  title: string;
  message: string;
  timestamp?: string;
  onDismiss?: () => void;
}

export interface NavigationProps extends BaseComponentProps {
  items: NavigationItem[];
  activeItem?: string;
  onItemClick?: (itemId: string) => void;
}

export interface NavigationItem {
  id: string;
  label: string;
  icon?: string;
  href?: string;
  children?: NavigationItem[];
}

export interface ComponentSidebarProps extends BaseComponentProps {
  isOpen: boolean;
  onClose: () => void;
  onNodeAdd: (nodeType: string) => void;
}

// Utility Types
export type Status = 'idle' | 'loading' | 'success' | 'error';

export interface ApiResponse<T> {
  data: T;
  status: number;
  message: string;
}

// Hook Types
export interface UseWorkflowState {
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  selectedNode: string | null;
  isLoading: boolean;
  error: string | null;
}

export interface UseWorkflowActions {
  addNode: (node: Omit<WorkflowNode, 'id'>) => void;
  updateNode: (id: string, data: Partial<WorkflowData>) => void;
  deleteNode: (id: string) => void;
  addEdge: (edge: Omit<WorkflowEdge, 'id'>) => void;
  deleteEdge: (id: string) => void;
  selectNode: (id: string | null) => void;
  exportWorkflow: () => Promise<string>;
}

// Event Types
export interface NodeActionEvent {
  nodeId: string;
  action: 'run' | 'edit' | 'delete' | 'code' | 'more';
  data?: unknown;
}

export interface WorkflowEvent {
  type: 'node_added' | 'node_updated' | 'node_deleted' | 'edge_added' | 'edge_deleted';
  payload: unknown;
  timestamp: Date;
}
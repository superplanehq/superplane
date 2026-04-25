import type { CanvasesCanvasEvent, CanvasesCanvasNodeExecution, SuperplaneComponentsNode } from "@/api-client";
import type React from "react";

export interface ChildExecution {
  name: string;
  state: string;
  nodeId: string;
  executionId: string;
  badgeColor?: string;
  backgroundColor?: string;
  componentIcon?: string;
  componentIconSrc?: string;
}

export interface ChainItemData {
  id: string;
  nodeId: string;
  componentName: string;
  nodeName?: string;
  nodeDisplayName?: string;
  nodeIcon?: string;
  nodeIconSlug?: string;
  nodeIconSrc?: string;
  state?: string;
  executionId?: string;
  originalExecution?: CanvasesCanvasNodeExecution;
  originalEvent?: CanvasesCanvasEvent;
  childExecutions?: ChildExecution[];
  workflowNode?: SuperplaneComponentsNode;
  tabData?: {
    current?: Record<string, string | number | boolean | React.ReactElement | null | undefined>;
    payload?: Record<string, unknown>;
    configuration?: Record<string, unknown>;
  };
}

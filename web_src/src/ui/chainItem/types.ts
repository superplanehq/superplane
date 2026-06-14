import type { CanvasesCanvasEvent, CanvasesCanvasNodeExecution, SuperplaneComponentsNode } from "@/api-client";
import type React from "react";

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
  workflowNode?: SuperplaneComponentsNode;
  tabData?: {
    current?: Record<string, string | number | boolean | React.ReactElement | null | undefined>;
    payload?: Record<string, unknown>;
    configuration?: Record<string, unknown>;
  };
}

import type { QueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type {
  ActionsAction,
  CanvasesCanvasNodeExecution,
  OrganizationsIntegration,
  SuperplaneComponentsNode as ComponentsNode,
  SuperplaneMeUser,
} from "@/api-client";
import { getCustomFieldRenderer } from "./mappers";
import { renderCanvasNodeCustomField } from "./lib/render-canvas-node-custom-field";
import {
  hasSidebarMapperCustomField,
  resolveSidebarMapperCustomField,
} from "./lib/resolve-sidebar-mapper-custom-field";

type UseCanvasSidebarCustomFieldOptions = {
  allComponentsByName: Map<string, ActionsAction>;
  canvasId?: string;
  canvasNodes: ComponentsNode[];
  canvasNodesById: Map<string, ComponentsNode>;
  getNodeData: (nodeId: string) => { executions: CanvasesCanvasNodeExecution[] };
  isEditing: boolean;
  me?: SuperplaneMeUser | null;
  queryClient: QueryClient;
  storeVersion: number;
};

export function useCanvasSidebarCustomField({
  allComponentsByName,
  canvasId,
  canvasNodes,
  canvasNodesById,
  getNodeData,
  isEditing,
  me,
  queryClient,
  storeVersion,
}: UseCanvasSidebarCustomFieldOptions) {
  return useCallback(
    (nodeId: string, integration?: OrganizationsIntegration) => {
      void storeVersion;

      const node = canvasNodesById.get(nodeId);
      if (!node) return null;

      let componentName = "";
      if (node.type === "TYPE_TRIGGER" && node.component) {
        componentName = node.component;
      } else if (node.type === "TYPE_ACTION" && node.component) {
        componentName = node.component;
      }

      const renderer = getCustomFieldRenderer(componentName);
      if (renderer) {
        const context: {
          integration?: OrganizationsIntegration;
        } = {};
        if (integration) {
          context.integration = integration;
        }

        return (configuration?: Record<string, unknown>) => {
          return renderCanvasNodeCustomField({
            renderer,
            node,
            configuration,
            context: Object.keys(context).length > 0 ? context : undefined,
          });
        };
      }

      if (node.type === "TYPE_ACTION" && node.component && canvasId && hasSidebarMapperCustomField(node.component)) {
        const componentDef = allComponentsByName.get(node.component);
        if (!componentDef) {
          return null;
        }

        return () => {
          const nodeData = getNodeData(nodeId);
          return resolveSidebarMapperCustomField({
            node,
            canvasNodes,
            executions: nodeData.executions,
            componentDef,
            canvasMode: isEditing ? "edit" : "live",
            canvasId,
            queryClient,
            me,
          });
        };
      }

      return null;
    },
    [
      allComponentsByName,
      canvasId,
      canvasNodes,
      canvasNodesById,
      getNodeData,
      isEditing,
      me,
      queryClient,
      storeVersion,
    ],
  );
}

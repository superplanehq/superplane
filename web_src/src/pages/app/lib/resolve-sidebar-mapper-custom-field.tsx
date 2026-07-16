import type { QueryClient } from "@tanstack/react-query";
import type {
  ActionsAction,
  CanvasesCanvasNodeExecution,
  SuperplaneComponentsNode as ComponentsNode,
  SuperplaneMeUser,
} from "@/api-client";
import { canvasesInvokeNodeExecutionHook } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { getComponentBaseMapper } from "../mappers";
import { buildComponentDefinition, buildExecutionInfo, buildNodeInfo, buildUserInfo } from "../utils";

const SIDEBAR_MAPPER_CUSTOM_FIELD_COMPONENTS = new Set(["approval"]);

export function hasSidebarMapperCustomField(componentName: string | undefined | null): boolean {
  return Boolean(componentName && SIDEBAR_MAPPER_CUSTOM_FIELD_COMPONENTS.has(componentName));
}

export function resolveSidebarMapperCustomField({
  node,
  canvasNodes,
  executions,
  componentDef,
  canvasMode,
  canvasId,
  queryClient,
  me,
}: {
  node: ComponentsNode;
  canvasNodes: ComponentsNode[];
  executions: CanvasesCanvasNodeExecution[];
  componentDef: ActionsAction;
  canvasMode: "live" | "edit";
  canvasId: string;
  queryClient: QueryClient;
  me?: SuperplaneMeUser | null;
}) {
  const componentName = node.component;
  if (!hasSidebarMapperCustomField(componentName) || !node.id) {
    return null;
  }

  const mapper = getComponentBaseMapper(componentName);
  const props = mapper.props({
    nodes: canvasNodes.map((canvasNode) => buildNodeInfo(canvasNode)),
    node: buildNodeInfo(node),
    componentDefinition: buildComponentDefinition(componentDef),
    lastExecutions: executions.map((execution) => buildExecutionInfo(execution)),
    currentUser: buildUserInfo(me),
    actions: {
      invokeNodeExecutionHook: async (executionId: string, hookName: string, parameters: unknown) => {
        try {
          await canvasesInvokeNodeExecutionHook(
            withOrganizationHeader({
              path: {
                canvasId,
                executionId,
                hookName,
              },
              body: {
                parameters,
              },
            }),
          );
          queryClient.invalidateQueries({
            queryKey: canvasKeys.nodeExecution(canvasId, node.id!),
          });
        } catch (error) {
          showErrorToast(getApiErrorMessage(error, "failed to invoke hook"));
        }
      },
    },
    canvasMode,
  });

  return props.customField ?? null;
}

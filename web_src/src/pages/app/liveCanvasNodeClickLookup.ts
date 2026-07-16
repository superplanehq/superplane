import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { shouldOpenConfigurationSidebarForLiveNodeClick } from "./runInspectionLiveNodeLookup";

type LiveCanvasNodeClickActions = {
  openConfigurationSidebar: (options?: { preferSettingsTab?: boolean }) => void;
};

type RunLiveCanvasNodeClickLookupOptions = {
  fetchRunIdForSidebarEvent: (event: SidebarEvent, options?: { maxPages?: number }) => Promise<string | null>;
  handleSelectRunFromSidebarEvent: (runId: string, options?: { nodeId?: string }) => void;
  isLookupStale: () => boolean;
  nodeId: string;
  openConfigurationSidebar: LiveCanvasNodeClickActions["openConfigurationSidebar"];
  resolveLatestNodeRunLookupEvent: (nodeId: string) => Promise<SidebarEvent | null>;
  workflowNode: ComponentsNode | undefined;
};

export async function runLiveCanvasNodeClickLookup({
  nodeId,
  workflowNode,
  isLookupStale,
  resolveLatestNodeRunLookupEvent,
  openConfigurationSidebar,
  fetchRunIdForSidebarEvent,
  handleSelectRunFromSidebarEvent,
}: RunLiveCanvasNodeClickLookupOptions) {
  try {
    const lookupEvent = await resolveLatestNodeRunLookupEvent(nodeId);
    if (isLookupStale()) return;

    const refreshedNodeActivity = useNodeExecutionStore.getState().getNodeData(nodeId);
    if (shouldOpenConfigurationSidebarForLiveNodeClick(workflowNode, refreshedNodeActivity)) {
      openConfigurationSidebar({ preferSettingsTab: true });
      return;
    }

    if (!lookupEvent) {
      openConfigurationSidebar();
      return;
    }

    const runId = await fetchRunIdForSidebarEvent(lookupEvent, { maxPages: 1 });
    if (isLookupStale()) return;

    if (!runId) {
      openConfigurationSidebar();
      return;
    }

    handleSelectRunFromSidebarEvent(runId, { nodeId });
  } catch (error) {
    console.error("Failed to inspect latest node run", error);
    if (isLookupStale()) return;
    openConfigurationSidebar();
  }
}

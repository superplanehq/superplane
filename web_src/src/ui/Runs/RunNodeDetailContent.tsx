import { Loader2 } from "lucide-react";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { RunNodeDetailHeader } from "./RunNodeDetailHeader";
import { RunNodeDetailTabSection } from "./RunNodeDetailTabSection";
import { useRunNodeDetailEscape, useRunNodeDetailTabs } from "./useRunNodeDetailTabs";
import { useRunNodeDetailPresentation } from "./useRunNodeDetailPresentation";

export interface RunNodeDetailContentProps {
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  executions: CanvasesCanvasNodeExecution[];
  isExecutionsLoading?: boolean;
  onClose: () => void;
  onNavigateNode?: (nodeId: string) => void;
  testId?: string;
}

export function RunNodeDetailContent({
  run,
  nodeId,
  workflowNodes = [],
  componentIconMap = {},
  executions,
  isExecutionsLoading = false,
  onClose,
  onNavigateNode,
  testId = "run-node-detail-content",
}: RunNodeDetailContentProps) {
  const presentation = useRunNodeDetailPresentation({ run, nodeId, workflowNodes, executions });
  const { activeTab, selectTab } = useRunNodeDetailTabs(nodeId, presentation.tabAvailability);

  useRunNodeDetailEscape(onClose);

  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-hidden bg-white"
      data-testid={testId}
      aria-label={`${presentation.nodeName} run details`}
    >
      <RunNodeDetailHeader
        nodeName={presentation.nodeName}
        workflowNode={presentation.workflowNode}
        componentIconMap={componentIconMap}
        previousNodeId={presentation.previousNodeId}
        nextNodeId={presentation.nextNodeId}
        onClose={onClose}
        onNavigateNode={onNavigateNode}
      />

      {isExecutionsLoading && !presentation.isTriggerNode ? (
        <div className="flex items-center justify-center gap-2 px-4 py-8 text-xs text-gray-400">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading run details...
        </div>
      ) : presentation.hasAnyTab ? (
        <RunNodeDetailTabSection
          activeTab={activeTab}
          tabData={presentation.tabData}
          hasDetailsSection={presentation.hasDetailsSection}
          hasPayload={presentation.hasPayload}
          hasConfig={presentation.hasConfig}
          headerEventBadge={presentation.headerEventBadge}
          createdAt={presentation.createdAt}
          onSelectTab={selectTab}
        />
      ) : (
        <div className="px-4 py-6 text-center text-xs text-gray-400">No execution data for this node in this run.</div>
      )}
    </div>
  );
}

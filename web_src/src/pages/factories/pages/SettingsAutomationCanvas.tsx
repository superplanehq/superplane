import type { CanvasesCanvasRun } from "@/api-client";
import { CanvasPage } from "@/ui/CanvasPage";

import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

interface SettingsAutomationCanvasProps {
  graph: IntakeAutomationGraph;
  isRunInspectionMode?: boolean;
  runCanvasLoading?: boolean;
  selectedRun?: CanvasesCanvasRun | null;
  runParticipantNodeIds?: string[];
  fitAllRequest?: number | null;
  fitAllFocusNodeIds?: string[];
  onNodeSelect?: (nodeId: string) => void;
  showStatusControls?: boolean;
  showFindControls?: boolean;
  focusNodeId?: string | null;
  focusNonce?: number;
  layoutFitNonce?: number | null;
}

/**
 * Read-only factory automation preview. Uses the same leaf-right layout as
 * factory run inspection and fits every node into the popup. Edit lives
 * as a pencil on the canvas.
 */
export function SettingsAutomationCanvas({
  graph,
  isRunInspectionMode = false,
  runCanvasLoading = false,
  selectedRun = null,
  runParticipantNodeIds,
  fitAllRequest = null,
  fitAllFocusNodeIds,
  onNodeSelect,
  showStatusControls = true,
  showFindControls = true,
  focusNodeId = null,
  focusNonce = 0,
  layoutFitNonce = null,
}: SettingsAutomationCanvasProps) {
  return (
    <CanvasPage
      nodes={graph.nodes}
      edges={graph.edges}
      factoryId={graph.factoryId}
      factoryEmbed
      factoryDisplayLayout
      isEditing={false}
      readOnly
      hidePageChrome
      hideAddControls
      hideCanvasToolSidebar
      hideRightSideControls
      buildingBlocks={[]}
      activeCanvasVersionId=""
      isRunInspectionMode={isRunInspectionMode}
      runCanvasLoading={runCanvasLoading}
      runNodeDetailRun={selectedRun}
      runParticipantNodeIds={runParticipantNodeIds}
      fitAllRequest={layoutFitNonce ?? fitAllRequest}
      fitAllFocusNodeIds={fitAllFocusNodeIds}
      onNodeClick={onNodeSelect}
      showBottomStatusControls={showStatusControls}
      showCanvasFindControls={showFindControls}
      focusRequest={
        focusNodeId && focusNonce > 0
          ? {
              nodeId: focusNodeId,
              requestId: focusNonce,
              targetMode: "live",
            }
          : null
      }
    />
  );
}

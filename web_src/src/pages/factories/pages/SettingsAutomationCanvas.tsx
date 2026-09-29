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
      fitAllRequest={fitAllRequest}
      fitAllFocusNodeIds={fitAllFocusNodeIds}
    />
  );
}

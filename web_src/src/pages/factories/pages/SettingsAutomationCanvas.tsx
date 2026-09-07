import { CanvasPage } from "@/ui/CanvasPage";

import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

/**
 * Read-only factory automation preview. Uses the same leaf-right layout as
 * factory run inspection and fits every node into the popup. Edit lives
 * behind the Edit automation link.
 */
export function SettingsAutomationCanvas({ graph }: { graph: IntakeAutomationGraph }) {
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
    />
  );
}

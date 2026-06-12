import type { DraftDiffLine } from "./draftNodeDiff";
import { UnifiedDiffView } from "./UnifiedDiffView";

export function DraftNodeDiffView({ nodeID, lines }: { nodeID: string; lines: DraftDiffLine[] }) {
  return <UnifiedDiffView diffId={nodeID} emptyMessage="No diff available for this node." lines={lines} />;
}

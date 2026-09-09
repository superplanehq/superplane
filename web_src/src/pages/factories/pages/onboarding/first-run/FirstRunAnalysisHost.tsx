import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";
import { useFirstRunAnalysis } from "./useFirstRunAnalysis";

export function FirstRunAnalysisHost(args: {
  organizationId: string;
  factoryId: string;
  chrome: FirstRunChrome;
  selectedRepo: string | null;
  onGoToBoard: () => void;
}) {
  const { organizationId, factoryId, chrome, selectedRepo, onGoToBoard } = args;
  const { progress, failed } = useFirstRunAnalysis(organizationId, factoryId);
  const copy = FIRST_RUN_COPY.sphere;
  const sphere: FirstRunSphereProps = {
    level: 1,
    animate: true,
    phasesLit: true,
    caption: selectedRepo ?? "",
    captionHighlight: "Scoring:",
    leftChip:
      progress.total > 0
        ? { label: copy.discover, value: copy.ticketsFound(progress.total), tone: "amber" }
        : { label: copy.discover, value: copy.awaitingCode, tone: "ghost" },
    rightChip: { label: copy.verify, value: copy.reviewReadyPr, tone: "amber" },
  };
  return (
    <FirstRunAnalysisScreen
      progress={progress}
      failed={failed}
      chrome={chrome}
      sphere={sphere}
      onGoToBoard={onGoToBoard}
    />
  );
}

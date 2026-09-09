import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";
import { analysisSphereFor } from "./firstRunSphereFor";
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
  const { progress, sourceName, failed } = useFirstRunAnalysis(organizationId, factoryId);
  return (
    <FirstRunAnalysisScreen
      progress={progress}
      sourceName={sourceName}
      failed={failed}
      chrome={chrome}
      sphere={analysisSphereFor(selectedRepo, progress.total)}
      onGoToBoard={onGoToBoard}
    />
  );
}

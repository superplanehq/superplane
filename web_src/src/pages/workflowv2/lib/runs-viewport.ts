type ViewportState = { x: number; y: number; zoom: number };

type PrepareRunsViewportOnModeEntryParams = {
  currentCanvasViewport: ViewportState | undefined;
  existingRunsViewport: ViewportState | undefined;
  hasFitToView: boolean;
};

type PrepareRunsViewportOnModeEntryResult = {
  runsViewport: ViewportState | undefined;
  hasFitToView: boolean;
  seededFromCanvasViewport: boolean;
};

export function prepareRunsViewportOnModeEntry({
  currentCanvasViewport,
  existingRunsViewport,
  hasFitToView,
}: PrepareRunsViewportOnModeEntryParams): PrepareRunsViewportOnModeEntryResult {
  const runsViewport = currentCanvasViewport ?? existingRunsViewport;
  if (runsViewport) {
    return {
      runsViewport,
      hasFitToView: true,
      seededFromCanvasViewport: !!currentCanvasViewport,
    };
  }

  return {
    runsViewport: undefined,
    hasFitToView,
    seededFromCanvasViewport: false,
  };
}

type RunsFitAllDecisionParams = {
  isRunsMode: boolean;
  runCanvasNodeIdsKey: string | null;
  skipInitialRunsFitAll: boolean;
};

type RunsFitAllDecision = {
  shouldFitAll: boolean;
  skipInitialRunsFitAll: boolean;
};

export function getRunsFitAllDecision({
  isRunsMode,
  runCanvasNodeIdsKey,
  skipInitialRunsFitAll,
}: RunsFitAllDecisionParams): RunsFitAllDecision {
  if (!isRunsMode || !runCanvasNodeIdsKey) {
    return {
      shouldFitAll: false,
      skipInitialRunsFitAll,
    };
  }

  if (skipInitialRunsFitAll) {
    return {
      shouldFitAll: false,
      skipInitialRunsFitAll: false,
    };
  }

  return {
    shouldFitAll: true,
    skipInitialRunsFitAll: false,
  };
}

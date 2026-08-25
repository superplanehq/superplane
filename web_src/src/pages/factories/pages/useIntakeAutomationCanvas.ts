import type { CanvasesCanvas } from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";
import { useMemo } from "react";

import { intakeAutomationCanvasFromApp } from "./intakeAutomationCanvasModel";
import type { SplitRunCanvasModel } from "./work-order-split-run/splitRunCanvases";

interface IntakeAutomationCanvasResult {
  canvas?: SplitRunCanvasModel;
  sourceCanvas?: CanvasesCanvas;
  isLoading: boolean;
  isError: boolean;
  refetch: () => Promise<unknown>;
}

/** Loads the graph of the app that backs an intake source. */
export function useIntakeAutomationCanvas(
  organizationId: string | undefined,
  appId: string | undefined,
  title: string,
): IntakeAutomationCanvasResult {
  const enabled = Boolean(organizationId && appId);
  const query = useCanvas(organizationId ?? "", appId ?? "", { enabled });
  const canvas = useMemo(() => intakeAutomationCanvasFromApp(title, query.data), [query.data, title]);

  return {
    canvas,
    sourceCanvas: query.data,
    isLoading: enabled && query.isPending,
    isError: query.isError,
    refetch: query.refetch,
  };
}

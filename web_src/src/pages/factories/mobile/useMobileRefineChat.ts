import type { FactoriesFactory, FactoriesWorkOrderArtifact } from "@/api-client";
import { useWorkOrderFileUpload } from "@/hooks/useWorkOrderFileUpload";
import type { ReactNode } from "react";

import { analysisFirstResultDelivered } from "../lib/analysisOutcome";
import { factoryPlanningEnabled } from "../pages/planningSettingsModel";
import { draftStripAnalysis } from "../pages/work-order-split-run/draftStripAnalysis";
import type { IntentAnalysisChat } from "../pages/work-order-split-run/intentAnalysisChat";
import type { SplitRunFixture } from "../pages/work-order-split-run/splitRunMocks";
import { useAnalysisPlanningSession } from "../pages/work-order-split-run/useAnalysisPlanningSession";

export type MobileRefineChat = {
  /** Refine chat for a Planning draft that has a session. Undefined otherwise. */
  chat?: IntentAnalysisChat;
  loadFailed: boolean;
  /** Lookup succeeded, but this draft has no planning session. */
  sessionMissing: boolean;
};

/**
 * Planning session for the phone task page. It uses the same session hook
 * and draft chat fields as the desktop popup. Only a draft with Planning on
 * loads a session.
 */
export function useMobileRefineChat({
  organizationId,
  factoryId,
  factoryKey,
  factory,
  orderId,
  fixture,
  artifacts,
  canUpdate,
  modelSelect,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory?: FactoriesFactory | null;
  orderId: string;
  fixture: SplitRunFixture;
  artifacts: FactoriesWorkOrderArtifact[];
  canUpdate: boolean;
  modelSelect?: ReactNode;
}): MobileRefineChat {
  const refines = factoryPlanningEnabled(factory) && fixture.footer.kind === "draft";
  const fileUpload = useWorkOrderFileUpload({ organizationId, factoryId, orderId });
  const analysis = useAnalysisPlanningSession({
    organizationId,
    factoryId,
    workOrderId: orderId,
    enabled: refines && Boolean(organizationId && factoryId && orderId),
    canUpdate,
    isUploading: fileUpload.isUploading,
    uploadFiles: fileUpload.uploadFiles,
    analysisDelivered: analysisFirstResultDelivered({ checks: fixture.checks, artifacts }),
  });
  if (!refines) {
    return { loadFailed: false, sessionMissing: false };
  }
  if (!analysis.showChat && !analysis.queryError) {
    return { loadFailed: false, sessionMissing: !analysis.isLoading };
  }
  return {
    chat: draftStripAnalysis({ factory, organizationId, factoryKey, fixture, analysis }, modelSelect),
    loadFailed: Boolean(analysis.queryError),
    sessionMissing: false,
  };
}

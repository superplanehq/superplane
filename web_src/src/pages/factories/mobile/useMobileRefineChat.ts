import type { FactoriesFactory, FactoriesWorkOrderArtifact } from "@/api-client";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useWorkOrderFileUpload } from "@/hooks/useWorkOrderFileUpload";
import type { ReactNode } from "react";

import { analysisFirstResultDelivered } from "../lib/analysisOutcome";
import { factoryPlanningEnabled } from "../pages/planningSettingsModel";
import { draftAnalysisQueued, draftStripAnalysis } from "../pages/work-order-split-run/draftStripAnalysis";
import { SPLIT_RUN_ANALYZING_NOTE } from "../pages/work-order-split-run/splitRunFooter";
import type { IntentAnalysisChat } from "../pages/work-order-split-run/intentAnalysisChat";
import type { SplitRunFixture } from "../pages/work-order-split-run/splitRunMocks";
import { useAnalysisPlanningSession } from "../pages/work-order-split-run/useAnalysisPlanningSession";

export type MobileRefineChat = {
  /** Refine chat for a Planning draft, including analysis recovery. */
  chat?: IntentAnalysisChat;
  loadFailed: boolean;
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
  const analysisQueued = draftAnalysisQueued(fixture);
  const analysisActive = !analysisQueued && fixture.footer.note?.headline === SPLIT_RUN_ANALYZING_NOTE.headline;
  const { data: workspaceUsage } = useOrganizationWorkspaceUsage(refines ? organizationId : "", 30, true);
  const taskLimitReached =
    workspaceUsage?.activeTaskCount != null
      ? workspaceUsage.parallelTaskLimit != null && workspaceUsage.activeTaskCount >= workspaceUsage.parallelTaskLimit
      : fixture.taskLimitReached;
  const fileUpload = useWorkOrderFileUpload({ organizationId, factoryId, orderId });
  const analysis = useAnalysisPlanningSession({
    organizationId,
    factoryId,
    workOrderId: orderId,
    enabled: refines && Boolean(organizationId && factoryId && orderId),
    canUpdate,
    refinementCanStart: refines,
    analysisActive,
    analysisQueued,
    isUploading: fileUpload.isUploading,
    uploadFiles: fileUpload.uploadFiles,
    analysisDelivered: analysisFirstResultDelivered({ checks: fixture.checks, artifacts }),
  });
  if (!refines || (!analysis.showChat && analysis.isLoading && !analysis.queryError)) {
    return { loadFailed: false };
  }
  return {
    chat: draftStripAnalysis({ factory, organizationId, factoryKey, fixture, analysis, taskLimitReached }, modelSelect),
    loadFailed: Boolean(analysis.queryError),
  };
}

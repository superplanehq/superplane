import { useEffect, useRef } from "react";
import { useDraftActions } from "@/components/AgentSidebar/useDraftActions";
import { DraftActionsWidget } from "@/components/AgentSidebar/widgets/DraftActionsWidget";
import {
  buildAgentStagingAutoOpenKey,
  claimAgentStagingAutoOpen,
  releaseAgentStagingAutoOpen,
} from "@/pages/app/lib/agent-staging-auto-open";
import { isCanvasWorkflowTab, type CanvasPageHeaderMode } from "@/pages/app/viewState";
import type { AgentStagingReadyHandler } from "./useCanvasToolSidebarState";
import type { AgentMessage } from "./types";

type StagingActionsBarProps = {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  isEditing: boolean;
  outcomePassed?: boolean;
  onVersionPublished?: () => void;
  onAgentStagingReady?: AgentStagingReadyHandler;
  onAgentStagingCommit?: (commitMessage: string) => Promise<boolean>;
  liveCanvasVersionId?: string;
  headerMode?: CanvasPageHeaderMode;
  isRunInspectionMode?: boolean;
};

export function StagingActionsBar({
  messages,
  canvasId,
  organizationId,
  isEditing,
  outcomePassed,
  onVersionPublished,
  onAgentStagingReady,
  onAgentStagingCommit,
  liveCanvasVersionId,
  headerMode,
  isRunInspectionMode = false,
}: StagingActionsBarProps) {
  const { latestDraft: latestStaging, dismiss } = useDraftActions({
    messages,
    canvasId,
    organizationId,
    outcomePassed,
    onVersionPublished,
  });

  const openedStagingKeyRef = useRef<string | null>(null);
  const onAgentStagingReadyRef = useRef(onAgentStagingReady);
  onAgentStagingReadyRef.current = onAgentStagingReady;

  useEffect(() => {
    if (!latestStaging) {
      openedStagingKeyRef.current = null;
      return;
    }

    if (!onAgentStagingReadyRef.current || !liveCanvasVersionId) {
      return;
    }

    if (!isCanvasWorkflowTab(headerMode) || isRunInspectionMode) {
      return;
    }

    const stagingKey = buildAgentStagingAutoOpenKey(latestStaging.canvasId, latestStaging.message);
    if (openedStagingKeyRef.current === stagingKey || !claimAgentStagingAutoOpen(stagingKey)) {
      return;
    }

    let cancelled = false;

    void Promise.resolve(onAgentStagingReadyRef.current()).then((ready) => {
      if (cancelled || ready === false) {
        releaseAgentStagingAutoOpen(stagingKey);
        return;
      }
      openedStagingKeyRef.current = stagingKey;
    });

    return () => {
      cancelled = true;
    };
  }, [headerMode, isRunInspectionMode, latestStaging, liveCanvasVersionId]);

  if (!latestStaging) return null;

  return (
    <div className="border-t border-slate-200 bg-slate-50/80 px-3 py-2 dark:border-gray-800/70 dark:bg-gray-900">
      <div className="mx-auto w-full max-w-[800px]">
        <DraftActionsWidget
          versionId={latestStaging.versionId}
          message={latestStaging.message}
          canvasId={canvasId}
          organizationId={organizationId}
          isEditing={isEditing}
          onDismiss={dismiss}
          onViewStaging={onAgentStagingReady}
          onCommitStaging={onAgentStagingCommit}
        />
      </div>
    </div>
  );
}

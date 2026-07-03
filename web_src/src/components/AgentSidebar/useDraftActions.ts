import { useCallback, useEffect, useMemo, useState } from "react";
import { parseAgentContent, type DraftActionsSegment } from "./widgets/parser";
import type { AgentMessage } from "@/components/CanvasToolSidebar/types";
import { buildAgentStagingAutoOpenKey, releaseAgentStagingAutoOpen } from "@/pages/app/lib/agent-staging-auto-open";

type StagingSummaryResponse = {
  stagingSummary?: {
    hasStaging?: boolean;
  };
};

type UseDraftActionsArgs = {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  outcomePassed?: boolean;
  onVersionPublished?: () => void;
};

type UseDraftActionsResult = {
  latestDraft: DraftActionsSegment | null;
  dismiss: () => void;
};

/**
 * Manages the lifecycle of the staging-actions bar:
 * - Scans messages for the latest :::staging-actions segment
 * - Verifies staging still exists via API
 * - Listens to canvas websocket events to dismiss after commit/discard
 */
export function useDraftActions({
  messages,
  canvasId,
  organizationId,
  outcomePassed,
  onVersionPublished,
}: UseDraftActionsArgs): UseDraftActionsResult {
  const [dismissedCanvasIds, setDismissedCanvasIds] = useState<Set<string>>(() => new Set());
  const [autoDetectedDraft, setAutoDetectedDraft] = useState<DraftActionsSegment | null>(null);
  const [verifiedStaging, setVerifiedStaging] = useState<boolean | null>(null);

  const dismissCanvas = useCallback((targetCanvasId: string) => {
    setDismissedCanvasIds((current) => addDismissedCanvasId(current, targetCanvasId));
  }, []);

  const clearAutoDetectedDraft = useCallback((targetCanvasId?: string) => {
    setAutoDetectedDraft((current) => {
      if (!current) return null;
      if (targetCanvasId && current.canvasId !== targetCanvasId) return current;
      return null;
    });
  }, []);

  const latestDraft = useMemo(
    () => findLatestStagingAction(messages, canvasId, dismissedCanvasIds),
    [canvasId, dismissedCanvasIds, messages],
  );

  const effectiveDraft = latestDraft ?? autoDetectedDraft;

  useEffect(() => {
    if (!outcomePassed || latestDraft) {
      setAutoDetectedDraft(null);
      return;
    }

    let cancelled = false;

    async function detectStaging() {
      try {
        const hasStaging = await fetchCanvasHasStaging(canvasId, organizationId);
        if (cancelled || !hasStaging || dismissedCanvasIds.has(canvasId)) {
          return;
        }

        setAutoDetectedDraft({
          type: "staging-actions",
          canvasId,
          message: "Outcome complete — review staged changes",
        });
      } catch {
        // Ignore transient lookup failures and keep the bar hidden.
      }
    }

    void detectStaging();

    return () => {
      cancelled = true;
    };
  }, [canvasId, dismissedCanvasIds, latestDraft, organizationId, outcomePassed]);

  useEffect(() => {
    if (!effectiveDraft) {
      setVerifiedStaging(null);
      return;
    }

    const targetCanvasId = effectiveDraft.canvasId;
    let cancelled = false;

    async function verifyStaging() {
      try {
        const hasStaging = await fetchCanvasHasStaging(targetCanvasId, organizationId);
        if (cancelled) return;

        if (!hasStaging) {
          dismissCanvas(targetCanvasId);
        }
        setVerifiedStaging(hasStaging);
      } catch {
        if (!cancelled) {
          setVerifiedStaging(false);
        }
      }
    }

    void verifyStaging();

    return () => {
      cancelled = true;
    };
  }, [dismissCanvas, effectiveDraft, organizationId]);

  const dismiss = useCallback(() => {
    if (!effectiveDraft) return;

    releaseAgentStagingAutoOpen(buildAgentStagingAutoOpenKey(effectiveDraft.canvasId, effectiveDraft.message));
    dismissCanvas(effectiveDraft.canvasId);
    clearAutoDetectedDraft(effectiveDraft.canvasId);
    onVersionPublished?.();
  }, [clearAutoDetectedDraft, dismissCanvas, effectiveDraft, onVersionPublished]);

  if (!effectiveDraft || verifiedStaging !== true) {
    return { latestDraft: null, dismiss };
  }

  return { latestDraft: effectiveDraft, dismiss };
}

function findLatestStagingAction(
  messages: AgentMessage[],
  canvasId: string,
  dismissedCanvasIds: Set<string>,
): DraftActionsSegment | null {
  for (let index = messages.length - 1; index >= 0; index--) {
    const message = messages[index];
    if (message.role !== "assistant") {
      continue;
    }

    const segment = parseAgentContent(message.content).find(
      (candidate): candidate is DraftActionsSegment =>
        (candidate.type === "draft-actions" || candidate.type === "staging-actions") &&
        candidate.canvasId === canvasId &&
        !dismissedCanvasIds.has(candidate.canvasId),
    );

    if (segment) {
      return segment;
    }
  }

  return null;
}

function addDismissedCanvasId(current: Set<string>, canvasId: string): Set<string> {
  const next = new Set(current);
  next.add(canvasId);
  return next;
}

async function fetchCanvasHasStaging(canvasId: string, organizationId: string): Promise<boolean> {
  const data = await fetchJson<StagingSummaryResponse>(`/api/v1/canvases/${canvasId}/staging`, organizationId);
  return !!data?.stagingSummary?.hasStaging;
}

async function fetchJson<T>(url: string, organizationId: string): Promise<T | null> {
  const response = await fetch(url, {
    headers: { "x-organization-id": organizationId },
    credentials: "include",
  });

  if (!response.ok) {
    return null;
  }

  return (await response.json()) as T;
}

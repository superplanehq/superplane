import { useCallback, useEffect, useMemo, useState } from "react";
import type { AgentMode } from "./agentMode";
import { createSystemMessage } from "./systemMessages";
import { parseAgentContent, type DraftActionsSegment } from "./widgets/parser";
import type { AgentMessage } from "@/components/CanvasToolSidebar/types";
import type { useSendAgentChatMessage } from "@/hooks/useAgentChats";

const DRAFT_STATE = "STATE_DRAFT";
const PUBLISHED_STATE = "STATE_PUBLISHED";

type VersionMetadata = {
  id?: string;
  state?: string;
};

type CanvasVersionResponse = {
  version?: {
    metadata?: VersionMetadata;
  };
};

type CanvasVersionsResponse = {
  versions?: Array<{
    metadata?: VersionMetadata;
  }>;
};

type UseDraftActionsArgs = {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  chatId: string;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  agentMode: AgentMode;
  outcomePassed?: boolean;
  onVersionPublished?: () => void;
};

type UseDraftActionsResult = {
  latestDraft: DraftActionsSegment | null;
  dismiss: () => void;
};

/**
 * Manages the lifecycle of the draft-actions bar:
 * - Scans messages for the latest :::draft-actions segment
 * - Verifies the version is still a draft via API
 * - Listens to canvas:version-updated websocket events
 * - Sends system notifications to agent on publish/discard
 */
export function useDraftActions({
  messages,
  canvasId,
  organizationId,
  chatId,
  sendMutation,
  agentMode,
  outcomePassed,
  onVersionPublished,
}: UseDraftActionsArgs): UseDraftActionsResult {
  const [dismissedVersionIds, setDismissedVersionIds] = useState<Set<string>>(() => new Set());
  const [autoDetectedDraft, setAutoDetectedDraft] = useState<DraftActionsSegment | null>(null);
  const [verifiedDraft, setVerifiedDraft] = useState<boolean | null>(null);

  const dismissVersion = useCallback((versionId: string) => {
    setDismissedVersionIds((current) => addDismissedVersionId(current, versionId));
  }, []);

  const clearAutoDetectedDraft = useCallback((versionId?: string) => {
    setAutoDetectedDraft((current) => {
      if (!current) return null;
      if (versionId && current.versionId !== versionId) return current;
      return null;
    });
  }, []);

  const notifyAgent = useCallback(
    async (content: string) => {
      await sendMutation.mutateAsync({ chatId, content, mode: agentMode }).catch(() => {});
    },
    [agentMode, chatId, sendMutation],
  );

  const handleVersionPublished = useCallback(
    async (versionId: string) => {
      dismissVersion(versionId);
      clearAutoDetectedDraft(versionId);
      onVersionPublished?.();
      await notifyAgent(createSystemMessage(buildVersionChangeMessage("published", versionId)));
    },
    [clearAutoDetectedDraft, dismissVersion, notifyAgent, onVersionPublished],
  );

  const handleVersionDiscarded = useCallback(
    async (versionId: string) => {
      dismissVersion(versionId);
      clearAutoDetectedDraft(versionId);
      onVersionPublished?.();
      await notifyAgent(createSystemMessage(buildVersionChangeMessage("discarded", versionId)));
    },
    [clearAutoDetectedDraft, dismissVersion, notifyAgent, onVersionPublished],
  );

  const latestDraft = useMemo(
    () => findLatestDraftAction(messages, dismissedVersionIds),
    [dismissedVersionIds, messages],
  );

  const effectiveDraft = latestDraft ?? autoDetectedDraft;

  useEffect(() => {
    if (!outcomePassed || latestDraft) {
      setAutoDetectedDraft(null);
      return;
    }

    let cancelled = false;

    async function detectDraftVersion() {
      try {
        const data = await fetchCanvasVersions(canvasId, organizationId);
        if (cancelled || !data) return;

        const detectedDraft = findAutoDetectedDraft(data, dismissedVersionIds);
        setAutoDetectedDraft(detectedDraft);
      } catch {
        // Ignore transient lookup failures and keep the bar hidden.
      }
    }

    void detectDraftVersion();

    return () => {
      cancelled = true;
    };
  }, [canvasId, dismissedVersionIds, latestDraft, organizationId, outcomePassed]);

  useEffect(() => {
    if (!effectiveDraft) {
      setVerifiedDraft(null);
      return;
    }

    const versionId = effectiveDraft.versionId;
    let cancelled = false;

    async function verifyDraftVersion() {
      try {
        const data = await fetchCanvasVersion(canvasId, versionId, organizationId);
        if (cancelled || !data) return;

        const isDraft = data.version?.metadata?.state === DRAFT_STATE;
        if (!isDraft) {
          dismissVersion(versionId);
        }
        setVerifiedDraft(isDraft);
      } catch {
        if (!cancelled) {
          setVerifiedDraft(false);
        }
      }
    }

    void verifyDraftVersion();

    return () => {
      cancelled = true;
    };
  }, [canvasId, dismissVersion, effectiveDraft, organizationId]);

  useCanvasVersionUpdates({
    canvasId,
    dismissedVersionIds,
    dismissVersion,
    handleVersionDiscarded,
    handleVersionPublished,
    organizationId,
  });

  const dismiss = useCallback(() => {
    if (!effectiveDraft) return;

    dismissVersion(effectiveDraft.versionId);
    clearAutoDetectedDraft(effectiveDraft.versionId);
  }, [clearAutoDetectedDraft, dismissVersion, effectiveDraft]);

  if (!effectiveDraft || verifiedDraft !== true) {
    return { latestDraft: null, dismiss };
  }

  return { latestDraft: effectiveDraft, dismiss };
}

type UseCanvasVersionUpdatesArgs = {
  canvasId: string;
  organizationId: string;
  dismissedVersionIds: Set<string>;
  dismissVersion: (versionId: string) => void;
  handleVersionPublished: (versionId: string) => Promise<void>;
  handleVersionDiscarded: (versionId: string) => Promise<void>;
};

function useCanvasVersionUpdates({
  canvasId,
  organizationId,
  dismissedVersionIds,
  dismissVersion,
  handleVersionPublished,
  handleVersionDiscarded,
}: UseCanvasVersionUpdatesArgs): void {
  useEffect(() => {
    const listener = (event: Event) => {
      void processVersionUpdate({
        event,
        canvasId,
        organizationId,
        dismissedVersionIds,
        dismissVersion,
        handleVersionPublished,
        handleVersionDiscarded,
      });
    };

    window.addEventListener("canvas:version-updated", listener);
    return () => window.removeEventListener("canvas:version-updated", listener);
  }, [canvasId, dismissVersion, dismissedVersionIds, handleVersionDiscarded, handleVersionPublished, organizationId]);
}

type ProcessVersionUpdateArgs = UseCanvasVersionUpdatesArgs & {
  event: Event;
};

async function processVersionUpdate({
  event,
  canvasId,
  organizationId,
  dismissedVersionIds,
  dismissVersion,
  handleVersionPublished,
  handleVersionDiscarded,
}: ProcessVersionUpdateArgs): Promise<void> {
  const versionId = getUpdatedVersionId(event);
  if (!versionId || dismissedVersionIds.has(versionId)) return;

  try {
    const data = await fetchCanvasVersion(canvasId, versionId, organizationId);
    if (!data) {
      await handleVersionDiscarded(versionId);
      return;
    }

    const state = data.version?.metadata?.state;
    if (state === PUBLISHED_STATE) {
      await handleVersionPublished(versionId);
      return;
    }
    if (state && state !== DRAFT_STATE) {
      dismissVersion(versionId);
    }
  } catch {
    // Ignore transient websocket follow-up failures.
  }
}

function findLatestDraftAction(messages: AgentMessage[], dismissedVersionIds: Set<string>): DraftActionsSegment | null {
  for (let index = messages.length - 1; index >= 0; index--) {
    const message = messages[index];
    if (message.role === "user") {
      return null;
    }
    if (message.role !== "assistant") {
      continue;
    }

    const segment = parseAgentContent(message.content).find(
      (candidate): candidate is DraftActionsSegment =>
        candidate.type === "draft-actions" && !dismissedVersionIds.has(candidate.versionId),
    );

    if (segment) {
      return segment;
    }
  }

  return null;
}

function findAutoDetectedDraft(
  data: CanvasVersionsResponse,
  dismissedVersionIds: Set<string>,
): DraftActionsSegment | null {
  const draftVersion = data.versions?.find((version) => version.metadata?.state === DRAFT_STATE);
  const versionId = draftVersion?.metadata?.id;

  if (!versionId || dismissedVersionIds.has(versionId)) {
    return null;
  }

  return {
    type: "draft-actions",
    versionId,
    message: "Outcome complete — draft ready to publish",
  };
}

function addDismissedVersionId(current: Set<string>, versionId: string): Set<string> {
  const next = new Set(current);
  next.add(versionId);
  return next;
}

function getUpdatedVersionId(event: Event): string | null {
  const detail = (event as CustomEvent<{ versionId?: string }>).detail;
  return detail?.versionId ?? null;
}

function buildVersionChangeMessage(kind: "published" | "discarded", versionId: string): string {
  if (kind === "published") {
    return `User published draft version ${versionId}. Changes are now live. Re-read the canvas to see the current state.`;
  }

  return `User discarded draft version ${versionId}. Changes were NOT applied. The canvas is unchanged from the last published version.`;
}

async function fetchCanvasVersion(
  canvasId: string,
  versionId: string,
  organizationId: string,
): Promise<CanvasVersionResponse | null> {
  return await fetchJson<CanvasVersionResponse>(`/api/v1/canvases/${canvasId}/versions/${versionId}`, organizationId);
}

async function fetchCanvasVersions(canvasId: string, organizationId: string): Promise<CanvasVersionsResponse | null> {
  return await fetchJson<CanvasVersionsResponse>(`/api/v1/canvases/${canvasId}/versions`, organizationId);
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

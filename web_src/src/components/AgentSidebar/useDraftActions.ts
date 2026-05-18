import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { AgentMode } from "./agentMode";
import { parseAgentContent, type DraftActionsSegment } from "./widgets/parser";
import { createSystemMessage } from "./systemMessages";
import type { AgentMessage } from "@/components/CanvasToolSidebar/types";
import type { useSendAgentChatMessage } from "@/hooks/useAgentChats";

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
}: {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  chatId: string;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  agentMode: AgentMode;
  outcomePassed?: boolean;
  onVersionPublished?: () => void;
}): { latestDraft: DraftActionsSegment | null; dismiss: () => void } {
  const [, forceUpdate] = useState(0);
  const dismissedVersionIds = useRef(new Set<string>());
  const [verifiedDraft, setVerifiedDraft] = useState<boolean | null>(null);

  const notifyAgent = useCallback(
    (content: string) => {
      sendMutation.mutateAsync({ chatId, content, mode: agentMode }).catch(() => {});
    },
    [chatId, sendMutation, agentMode],
  );

  // Listen for canvas version changes — dismiss bar + notify agent
  useEffect(() => {
    const handler = (e: Event) => {
      const { versionId } = (e as CustomEvent).detail;
      if (!versionId || dismissedVersionIds.current.has(versionId)) return;
      fetch(`/api/v1/canvases/${canvasId}/versions/${versionId}`, {
        headers: { "x-organization-id": organizationId },
        credentials: "include",
      })
        .then((r) => {
          if (!r.ok) {
            dismissedVersionIds.current.add(versionId);
            setAutoDetectedDraft(null);
            forceUpdate((n) => n + 1);
            onVersionPublished?.();
            notifyAgent(
              createSystemMessage(
                `User discarded draft version ${versionId}. Changes were NOT applied. The canvas is unchanged from the last published version.`,
              ),
            );
            return null;
          }
          return r.json();
        })
        .then((data) => {
          if (!data) return;
          const state = data?.version?.metadata?.state;
          if (state === "STATE_PUBLISHED") {
            dismissedVersionIds.current.add(versionId);
            setAutoDetectedDraft(null);
            forceUpdate((n) => n + 1);
            onVersionPublished?.();
            notifyAgent(
              createSystemMessage(
                `User published draft version ${versionId}. Changes are now live. Re-read the canvas to see the current state.`,
              ),
            );
          } else if (state && state !== "STATE_DRAFT") {
            dismissedVersionIds.current.add(versionId);
            forceUpdate((n) => n + 1);
          }
        })
        .catch(() => {});
    };
    window.addEventListener("canvas:version-updated", handler);
    return () => window.removeEventListener("canvas:version-updated", handler);
  }, [canvasId, organizationId, notifyAgent, onVersionPublished]);

  const latestDraft = useMemo(() => {
    for (let i = messages.length - 1; i >= 0; i--) {
      const msg = messages[i];
      if (msg.role === "user") break;
      if (msg.role !== "assistant") continue;
      const segments = parseAgentContent(msg.content);
      for (const seg of segments) {
        if (seg.type === "draft-actions" && !dismissedVersionIds.current.has(seg.versionId)) return seg;
      }
    }
    return null;
  }, [messages]);

  // Auto-detect draft when outcome passes but agent didn't output :::draft-actions
  const [autoDetectedDraft, setAutoDetectedDraft] = useState<DraftActionsSegment | null>(null);
  useEffect(() => {
    if (!outcomePassed || latestDraft) {
      setAutoDetectedDraft(null);
      return;
    }
    let cancelled = false;
    fetch(`/api/v1/canvases/${canvasId}/versions`, {
      headers: { "x-organization-id": organizationId },
      credentials: "include",
    })
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (cancelled) return;
        const versions = data?.versions ?? [];
        const draft = versions.find((v: { metadata?: { state?: string } }) => v?.metadata?.state === "STATE_DRAFT");
        if (draft?.metadata?.id && !dismissedVersionIds.current.has(draft.metadata.id)) {
          setAutoDetectedDraft({
            type: "draft-actions",
            versionId: draft.metadata.id,
            message: "Outcome complete — draft ready to publish",
          });
        }
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [outcomePassed, latestDraft, canvasId, organizationId]);

  const effectiveDraft = latestDraft ?? autoDetectedDraft;

  // Verify the version is still a draft on mount / when version changes
  useEffect(() => {
    if (!effectiveDraft) {
      setVerifiedDraft(null);
      return;
    }
    let cancelled = false;
    fetch(`/api/v1/canvases/${canvasId}/versions/${effectiveDraft.versionId}`, {
      headers: { "x-organization-id": organizationId },
      credentials: "include",
    })
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (cancelled) return;
        const isDraft = data?.version?.metadata?.state === "STATE_DRAFT";
        if (!isDraft) dismissedVersionIds.current.add(effectiveDraft.versionId);
        setVerifiedDraft(isDraft);
      })
      .catch(() => {
        if (!cancelled) setVerifiedDraft(false);
      });
    return () => {
      cancelled = true;
    };
  }, [effectiveDraft, canvasId, organizationId]);

  const dismiss = useCallback(() => {
    if (effectiveDraft) {
      dismissedVersionIds.current.add(effectiveDraft.versionId);
      setAutoDetectedDraft(null);
      forceUpdate((n) => n + 1);
    }
  }, [effectiveDraft]);

  const verified = !effectiveDraft || verifiedDraft === false || verifiedDraft === null ? null : effectiveDraft;
  return { latestDraft: verified, dismiss };
}

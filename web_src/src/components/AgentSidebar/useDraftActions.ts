import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { AgentMessage } from "./types";
import type { AgentMode } from "./useAgentState";
import { parseAgentContent, type DraftActionsSegment } from "./widgets/parser";
import { createSystemMessage } from "./systemMessages";
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
}: {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  chatId: string;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  agentMode: AgentMode;
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
            forceUpdate((n) => n + 1);
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
            forceUpdate((n) => n + 1);
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
  }, [canvasId, organizationId, notifyAgent]);

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

  // Verify the version is still a draft on mount / when version changes
  useEffect(() => {
    if (!latestDraft) {
      setVerifiedDraft(null);
      return;
    }
    let cancelled = false;
    fetch(`/api/v1/canvases/${canvasId}/versions/${latestDraft.versionId}`, {
      headers: { "x-organization-id": organizationId },
      credentials: "include",
    })
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (cancelled) return;
        const isDraft = data?.version?.metadata?.state === "STATE_DRAFT";
        if (!isDraft) dismissedVersionIds.current.add(latestDraft.versionId);
        setVerifiedDraft(isDraft);
      })
      .catch(() => {
        if (!cancelled) setVerifiedDraft(false);
      });
    return () => {
      cancelled = true;
    };
  }, [latestDraft, canvasId, organizationId]);

  const dismiss = useCallback(() => {
    if (latestDraft) {
      dismissedVersionIds.current.add(latestDraft.versionId);
      forceUpdate((n) => n + 1);
    }
  }, [latestDraft]);

  const verified = !latestDraft || verifiedDraft === false || verifiedDraft === null ? null : latestDraft;
  return { latestDraft: verified, dismiss };
}

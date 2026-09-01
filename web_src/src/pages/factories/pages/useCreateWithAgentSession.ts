import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useCallback, useEffect, useRef, useState } from "react";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import {
  cancelCreateWithAgentEnd,
  emptyCreateWithAgentView,
  requestCreateWithAgentEnd,
  setCreateWithAgentComposer,
  updateCreateWithAgentDraft,
} from "./createWithAgentDemo";
import type { CreateWithAgentView } from "./createWithAgentTypes";
import {
  createPlanningSessionWorkOrder,
  describePlanningSession,
  endPlanningSession,
  sendPlanningSessionMessage,
  skipPlanningSessionDraft,
  startPlanningSession,
  updatePlanningSessionDraft,
} from "./planningSessionClient";
import { createWithAgentViewFromSession, type PlanningSessionPayload } from "./planningSessionView";

const POLL_MS = 1500;
const DRAFT_SAVE_MS = 400;

export function useCreateWithAgentSession(repository: string, organizationId: string, factoryId: string) {
  const [open, setOpen] = useState(false);
  const [sessionId, setSessionId] = useState("");
  const [view, setView] = useState<CreateWithAgentView>(() => emptyCreateWithAgentView(repository));
  const draftSaveTimer = useRef<number | undefined>(undefined);

  const applySession = useCallback((session: PlanningSessionPayload) => {
    setSessionId(session.id ?? "");
    setView((current) =>
      createWithAgentViewFromSession(session, {
        composer: current.composer,
        right: current.right,
        endConfirmOpen: current.endConfirmOpen,
      }),
    );
  }, []);

  useEffect(() => {
    if (!open || !sessionId || !organizationId || !factoryId) {
      return;
    }
    const poll = window.setInterval(() => {
      void describePlanningSession(organizationId, factoryId, sessionId)
        .then(applySession)
        .catch(() => undefined);
    }, POLL_MS);
    const endOnLeave = () => {
      void endPlanningSession(organizationId, factoryId, sessionId, { keepalive: true }).catch(() => undefined);
    };
    window.addEventListener("pagehide", endOnLeave);
    return () => {
      window.clearInterval(poll);
      window.removeEventListener("pagehide", endOnLeave);
    };
  }, [applySession, factoryId, open, organizationId, sessionId]);

  useEffect(() => {
    return () => {
      if (draftSaveTimer.current !== undefined) {
        window.clearTimeout(draftSaveTimer.current);
      }
    };
  }, []);

  const close = useCallback(() => {
    setOpen(false);
    setSessionId("");
    setView(emptyCreateWithAgentView(repository));
  }, [repository]);

  const start = useCallback(() => {
    setView(emptyCreateWithAgentView(repository));
    setSessionId("");
    setOpen(true);
    void startPlanningSession(organizationId, factoryId, repository)
      .then(applySession)
      .catch((error: unknown) => {
        showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedStart));
      });
  }, [applySession, factoryId, organizationId, repository]);

  const patchDraft = (title: string, description: string) => {
    setView((current) => updateCreateWithAgentDraft(current, { title, description }));
    if (!sessionId) {
      return;
    }
    if (draftSaveTimer.current !== undefined) {
      window.clearTimeout(draftSaveTimer.current);
    }
    draftSaveTimer.current = window.setTimeout(() => {
      void updatePlanningSessionDraft(organizationId, factoryId, sessionId, { title, description }).catch(
        (error: unknown) => {
          showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedDraft));
        },
      );
    }, DRAFT_SAVE_MS);
  };

  return {
    open,
    view,
    start,
    close,
    onComposerChange: (value: string) => setView((current) => setCreateWithAgentComposer(current, value)),
    onSend: () => {
      const text = view.composer.trim();
      if (!text || !sessionId) {
        return;
      }
      setView((current) => ({
        ...setCreateWithAgentComposer(current, ""),
        messages: [...current.messages, { id: `local-${current.messages.length + 1}`, kind: "text", role: "user", text }],
      }));
      void sendPlanningSessionMessage(organizationId, factoryId, sessionId, text)
        .then(applySession)
        .catch((error: unknown) => {
          showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedSend));
        });
    },
    onDraftTitleChange: (title: string) => {
      const description = view.right.kind === "draft" ? view.right.draft.description : "";
      patchDraft(title, description);
    },
    onDraftDescriptionChange: (description: string) => {
      const title = view.right.kind === "draft" ? view.right.draft.title : "";
      patchDraft(title, description);
    },
    onCreateDraft: () => {
      if (!sessionId) {
        return;
      }
      void createPlanningSessionWorkOrder(organizationId, factoryId, sessionId)
        .then(applySession)
        .catch((error: unknown) => {
          showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedCreate));
        });
    },
    onSkipDraft: () => {
      if (!sessionId) {
        return;
      }
      void skipPlanningSessionDraft(organizationId, factoryId, sessionId)
        .then(applySession)
        .catch((error: unknown) => {
          showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedSkip));
        });
    },
    onRequestClose: () => setView((current) => requestCreateWithAgentEnd(current)),
    onCancelEnd: () => setView((current) => cancelCreateWithAgentEnd(current)),
    onConfirmEnd: () => {
      if (sessionId) {
        void endPlanningSession(organizationId, factoryId, sessionId).finally(close);
        return;
      }
      close();
    },
  };
}

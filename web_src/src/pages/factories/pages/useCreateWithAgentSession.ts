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
const UNMOUNT_END_MS = 100;
const pendingUnmountEnds = new Map<string, number>();

function planningSessionHookKey(organizationId: string, factoryId: string) {
  return `${organizationId}:${factoryId}`;
}

function cancelScheduledPlanningSessionEnd(key: string) {
  const timer = pendingUnmountEnds.get(key);
  if (timer === undefined) {
    return;
  }
  window.clearTimeout(timer);
  pendingUnmountEnds.delete(key);
}

function schedulePlanningSessionEnd(key: string, end: () => void) {
  cancelScheduledPlanningSessionEnd(key);
  pendingUnmountEnds.set(
    key,
    window.setTimeout(() => {
      pendingUnmountEnds.delete(key);
      end();
    }, UNMOUNT_END_MS),
  );
}

export function useCreateWithAgentSession(repository: string, organizationId: string, factoryId: string) {
  const [open, setOpen] = useState(false);
  const [sessionId, setSessionId] = useState("");
  const [view, setView] = useState<CreateWithAgentView>(() => emptyCreateWithAgentView(repository));
  const draftSaveTimer = useRef<number | undefined>(undefined);
  const sessionIdRef = useRef("");
  const openRef = useRef(false);
  sessionIdRef.current = sessionId;
  openRef.current = open;

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

  const stopSession = useCallback(
    (id: string, options?: { keepalive?: boolean }) => {
      if (!id || !organizationId || !factoryId) {
        return;
      }
      const request = options
        ? endPlanningSession(organizationId, factoryId, id, options)
        : endPlanningSession(organizationId, factoryId, id);
      void request.catch(() => undefined);
    },
    [factoryId, organizationId],
  );

  useEffect(() => {
    const key = planningSessionHookKey(organizationId, factoryId);
    cancelScheduledPlanningSessionEnd(key);
    return () => {
      const id = sessionIdRef.current;
      if (!openRef.current || !id) {
        return;
      }
      schedulePlanningSessionEnd(key, () => {
        stopSession(id, { keepalive: true });
      });
    };
  }, [factoryId, organizationId, stopSession]);

  const close = useCallback(() => {
    const id = sessionIdRef.current;
    setOpen(false);
    setSessionId("");
    setView(emptyCreateWithAgentView(repository));
    stopSession(id);
  }, [repository, stopSession]);

  const start = useCallback(() => {
    stopSession(sessionIdRef.current);
    setView(emptyCreateWithAgentView(repository));
    setSessionId("");
    setOpen(true);
    void startPlanningSession(organizationId, factoryId, repository)
      .then(applySession)
      .catch((error: unknown) => {
        showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedStart));
      });
  }, [applySession, factoryId, organizationId, repository, stopSession]);

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

  const sendSessionText = (text: string, failedCopy: string) => {
    const body = text.trim();
    if (!body || !sessionId) {
      return;
    }
    setView((current) => ({
      ...setCreateWithAgentComposer(current, ""),
      survey: undefined,
      messages: [
        ...current.messages,
        { id: `local-${current.messages.length + 1}`, kind: "text", role: "user", text: body },
      ],
    }));
    void sendPlanningSessionMessage(organizationId, factoryId, sessionId, body)
      .then(applySession)
      .catch((error: unknown) => {
        showErrorToast(getApiErrorMessage(error, failedCopy));
      });
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
      sendSessionText(text, CREATE_WITH_AGENT_COPY.failedSend);
    },
    onSubmitSurvey: (text: string) => {
      if (!sessionId) {
        return;
      }
      sendSessionText(text, CREATE_WITH_AGENT_COPY.failedSurvey);
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
      close();
    },
  };
}

import { useCallback, useEffect, useState } from "react";

import {
  cancelCreateWithAgentEnd,
  createCreateWithAgentDraft,
  emptyCreateWithAgentView,
  markCreateWithAgentReady,
  requestCreateWithAgentEnd,
  selectCreateWithAgentCreated,
  sendCreateWithAgentMessage,
  setCreateWithAgentComposer,
  showCreateWithAgentList,
  skipCreateWithAgentDraft,
  updateCreateWithAgentDraft,
  workOnNewCreateWithAgent,
} from "./createWithAgentDemo";
import type { CreateWithAgentCreatedOrder, CreateWithAgentView } from "./createWithAgentTypes";
import {
  createPlanningSessionWorkOrder,
  describePlanningSession,
  endPlanningSession,
  heartbeatPlanningSession,
  sendPlanningSessionMessage,
  skipPlanningSessionDraft,
  startPlanningSession,
  updatePlanningSessionDraft,
} from "./planningSessionClient";
import { createWithAgentViewFromSession, type PlanningSessionPayload } from "./planningSessionView";

const READY_AFTER_MS = 700;
const POLL_MS = 1500;
const HEARTBEAT_MS = 15000;

export function useCreateWithAgentSession(repository: string, organizationId = "", factoryId = "") {
  const live = Boolean(organizationId && factoryId);
  const [open, setOpen] = useState(false);
  const [sessionId, setSessionId] = useState("");
  const [view, setView] = useState<CreateWithAgentView>(() => emptyCreateWithAgentView(repository));

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

  useDemoMachineReady(live, open, view.machineStatus, setView);
  useLivePlanningSessionSync({ live, open, sessionId, organizationId, factoryId, applySession });

  const start = useCallback(() => {
    setView(emptyCreateWithAgentView(repository));
    setSessionId("");
    setOpen(true);
    if (live) {
      void startPlanningSession(organizationId, factoryId, repository)
        .then(applySession)
        .catch(() => undefined);
    }
  }, [applySession, factoryId, live, organizationId, repository]);

  const close = useCallback(() => {
    setOpen(false);
    setSessionId("");
    setView(emptyCreateWithAgentView(repository));
  }, [repository]);

  return {
    open,
    view,
    start,
    close,
    onComposerChange: (value: string) => setView((current) => setCreateWithAgentComposer(current, value)),
    onSend: () => sendSessionMessage({ live, sessionId, organizationId, factoryId, view, setView, applySession }),
    onDraftTitleChange: (title: string) =>
      patchDraft({ live, sessionId, organizationId, factoryId, view, setView, title }),
    onDraftDescriptionChange: (description: string) =>
      patchDraft({ live, sessionId, organizationId, factoryId, view, setView, description }),
    onCreateDraft: () => {
      if (!live || !sessionId) {
        setView((current) => createCreateWithAgentDraft(current));
        return;
      }
      void createPlanningSessionWorkOrder(organizationId, factoryId, sessionId)
        .then(applySession)
        .catch(() => undefined);
    },
    onSkipDraft: () => {
      if (!live || !sessionId) {
        setView((current) => skipCreateWithAgentDraft(current));
        return;
      }
      void skipPlanningSessionDraft(organizationId, factoryId, sessionId)
        .then(applySession)
        .catch(() => undefined);
    },
    onWorkOnNew: () => setView((current) => workOnNewCreateWithAgent(current)),
    onSelectCreated: (orderId: string) => setView((current) => selectCreateWithAgentCreated(current, orderId)),
    onBackToList: () => setView((current) => showCreateWithAgentList(current)),
    onRequestClose: () => setView((current) => requestCreateWithAgentEnd(current)),
    onCancelEnd: () => setView((current) => cancelCreateWithAgentEnd(current)),
    onConfirmEnd: () => {
      if (live && sessionId) {
        void endPlanningSession(organizationId, factoryId, sessionId).finally(close);
        return;
      }
      close();
    },
    onOpenCreated: (_order: CreateWithAgentCreatedOrder) => undefined,
  };
}

function useDemoMachineReady(
  live: boolean,
  open: boolean,
  machineStatus: CreateWithAgentView["machineStatus"],
  setView: (update: (current: CreateWithAgentView) => CreateWithAgentView) => void,
) {
  useEffect(() => {
    if (!open || live || machineStatus !== "starting") {
      return;
    }
    const timer = window.setTimeout(() => {
      setView((current) => markCreateWithAgentReady(current));
    }, READY_AFTER_MS);
    return () => window.clearTimeout(timer);
  }, [live, machineStatus, open, setView]);
}

function useLivePlanningSessionSync(args: {
  live: boolean;
  open: boolean;
  sessionId: string;
  organizationId: string;
  factoryId: string;
  applySession: (session: PlanningSessionPayload) => void;
}) {
  const { live, open, sessionId, organizationId, factoryId, applySession } = args;
  useEffect(() => {
    if (!live || !open || !sessionId) {
      return;
    }
    const poll = window.setInterval(() => {
      void describePlanningSession(organizationId, factoryId, sessionId)
        .then(applySession)
        .catch(() => undefined);
    }, POLL_MS);
    const heartbeat = window.setInterval(() => {
      void heartbeatPlanningSession(organizationId, factoryId, sessionId)
        .then(applySession)
        .catch(() => undefined);
    }, HEARTBEAT_MS);
    const endOnLeave = () => {
      void endPlanningSession(organizationId, factoryId, sessionId, { keepalive: true }).catch(() => undefined);
    };
    window.addEventListener("pagehide", endOnLeave);
    return () => {
      window.clearInterval(poll);
      window.clearInterval(heartbeat);
      window.removeEventListener("pagehide", endOnLeave);
    };
  }, [applySession, factoryId, live, open, organizationId, sessionId]);
}

function sendSessionMessage(args: {
  live: boolean;
  sessionId: string;
  organizationId: string;
  factoryId: string;
  view: CreateWithAgentView;
  setView: (update: (current: CreateWithAgentView) => CreateWithAgentView) => void;
  applySession: (session: PlanningSessionPayload) => void;
}) {
  if (args.live && !args.sessionId) {
    return;
  }
  if (!args.live || !args.sessionId) {
    args.setView((current) => sendCreateWithAgentMessage(current));
    return;
  }
  const text = args.view.composer.trim();
  if (!text) {
    return;
  }
  args.setView((current) => ({
    ...setCreateWithAgentComposer(current, ""),
    messages: [...current.messages, { id: `local-${current.messages.length + 1}`, kind: "text", role: "user", text }],
  }));
  void sendPlanningSessionMessage(args.organizationId, args.factoryId, args.sessionId, text)
    .then(args.applySession)
    .catch(() => undefined);
}

function patchDraft(args: {
  live: boolean;
  sessionId: string;
  organizationId: string;
  factoryId: string;
  view: CreateWithAgentView;
  setView: (update: (current: CreateWithAgentView) => CreateWithAgentView) => void;
  title?: string;
  description?: string;
}) {
  const currentTitle = args.view.right.kind === "draft" ? args.view.right.draft.title : "";
  const currentDescription = args.view.right.kind === "draft" ? args.view.right.draft.description : "";
  const title = args.title ?? currentTitle;
  const description = args.description ?? currentDescription;
  args.setView((current) => updateCreateWithAgentDraft(current, { title, description }));
  if (args.live && args.sessionId) {
    void updatePlanningSessionDraft(args.organizationId, args.factoryId, args.sessionId, { title, description }).catch(
      () => undefined,
    );
  }
}

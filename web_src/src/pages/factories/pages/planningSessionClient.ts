import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import type { PlanningSessionPayload } from "./planningSessionView";

type SessionEnvelope = { session?: PlanningSessionPayload };

async function planningSessionRequest(
  organizationId: string,
  path: string,
  init: RequestInit,
): Promise<PlanningSessionPayload> {
  const headers = withOrganizationHeader({
    organizationId,
    headers: { Accept: "application/json", ...(init.body ? { "Content-Type": "application/json" } : {}), ...init.headers },
  }).headers as Record<string, string>;
  const response = await fetch(path, { ...init, headers, credentials: "include", keepalive: init.keepalive });
  if (!response.ok) {
    throw new Error("Planning session request failed");
  }
  const body = (await response.json()) as SessionEnvelope;
  if (!body.session) {
    throw new Error("Planning session is missing");
  }
  return body.session;
}

export function startPlanningSession(organizationId: string, factoryId: string, repository = "") {
  const trimmed = repository.trim();
  return planningSessionRequest(organizationId, `/api/v1/factories/${factoryId}/planning-sessions`, {
    method: "POST",
    body: JSON.stringify(trimmed ? { repository: trimmed } : {}),
  });
}

export function describePlanningSession(organizationId: string, factoryId: string, sessionId: string) {
  return planningSessionRequest(organizationId, `/api/v1/factories/${factoryId}/planning-sessions/${sessionId}`, {
    method: "GET",
  });
}

export function endPlanningSession(
  organizationId: string,
  factoryId: string,
  sessionId: string,
  options?: { keepalive?: boolean },
) {
  return planningSessionRequest(organizationId, `/api/v1/factories/${factoryId}/planning-sessions/${sessionId}/end`, {
    method: "POST",
    body: "{}",
    keepalive: options?.keepalive,
  });
}

export function sendPlanningSessionMessage(organizationId: string, factoryId: string, sessionId: string, text: string) {
  return planningSessionRequest(organizationId, `/api/v1/factories/${factoryId}/planning-sessions/${sessionId}/messages`, {
    method: "POST",
    body: JSON.stringify({ text }),
  });
}

export function updatePlanningSessionDraft(
  organizationId: string,
  factoryId: string,
  sessionId: string,
  draft: { title: string; description: string },
) {
  return planningSessionRequest(organizationId, `/api/v1/factories/${factoryId}/planning-sessions/${sessionId}/draft`, {
    method: "PATCH",
    body: JSON.stringify(draft),
  });
}

export function createPlanningSessionWorkOrder(organizationId: string, factoryId: string, sessionId: string) {
  return planningSessionRequest(organizationId, `/api/v1/factories/${factoryId}/planning-sessions/${sessionId}/create`, {
    method: "POST",
    body: "{}",
  });
}

export function skipPlanningSessionDraft(organizationId: string, factoryId: string, sessionId: string) {
  return planningSessionRequest(organizationId, `/api/v1/factories/${factoryId}/planning-sessions/${sessionId}/skip`, {
    method: "POST",
    body: "{}",
  });
}

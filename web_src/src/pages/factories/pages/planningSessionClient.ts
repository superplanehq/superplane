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
    headers: {
      Accept: "application/json",
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...init.headers,
    },
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

export async function findPlanningSessionByWorkOrder(
  organizationId: string,
  factoryId: string,
  workOrderId: string,
): Promise<PlanningSessionPayload | null> {
  const headers = withOrganizationHeader({
    organizationId,
    headers: { Accept: "application/json" },
  }).headers as Record<string, string>;
  const response = await fetch(`/api/v1/factories/${factoryId}/work-orders/${workOrderId}/planning-session`, {
    method: "GET",
    headers,
    credentials: "include",
  });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error("Planning session request failed");
  }
  const body = (await response.json()) as SessionEnvelope;
  return body.session ?? null;
}

export function sendPlanningSessionMessage(organizationId: string, factoryId: string, sessionId: string, text: string) {
  return planningSessionRequest(
    organizationId,
    `/api/v1/factories/${factoryId}/planning-sessions/${sessionId}/messages`,
    {
      method: "POST",
      body: JSON.stringify({ text }),
    },
  );
}

export function answerPlanningSessionSurvey(
  organizationId: string,
  factoryId: string,
  sessionId: string,
  text: string,
) {
  return planningSessionRequest(
    organizationId,
    `/api/v1/factories/${factoryId}/planning-sessions/${sessionId}/survey-answer`,
    {
      method: "POST",
      body: JSON.stringify({ text }),
    },
  );
}

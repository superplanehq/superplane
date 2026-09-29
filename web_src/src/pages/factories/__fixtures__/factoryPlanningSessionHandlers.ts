import type { FixtureResult } from "@/pages/home/__fixtures__/handlers";

import type { PlanningSessionPayload } from "../pages/planningSessionView";
import type { FactoriesFixture } from "./factoryPageResponses";

interface PlanningSessionRoute {
  pattern: RegExp;
  resolve: (match: RegExpExecArray, method: string, body: Record<string, unknown> | null) => FixtureResult;
}

const route = (pattern: string): RegExp => new RegExp(`^${pattern}$`);

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function findSessionById(fixture: FactoriesFixture, sessionId: string): PlanningSessionPayload | undefined {
  return Object.values(fixture.planningSessionsByWorkOrderId ?? {}).find((session) => session.id === sessionId);
}

function appendUserMessage(session: PlanningSessionPayload, text: string): PlanningSessionPayload {
  const messages = [
    ...(session.messages ?? []),
    {
      id: `storybook-user-${(session.messages ?? []).length + 1}`,
      role: "user",
      text,
      createdAt: new Date().toISOString(),
    },
  ];
  session.messages = messages;
  return session;
}

export function factoryPlanningSessionRoutes(fixture: FactoriesFixture): PlanningSessionRoute[] {
  return [
    {
      pattern: route("/api/v1/factories/([^/]+)/work-orders/([^/]+)/planning-session"),
      resolve: (match, method) => {
        if (method !== "GET") {
          return null;
        }
        const session = fixture.planningSessionsByWorkOrderId?.[match[2]];
        return session ? { json: { session } } : null;
      },
    },
    {
      pattern: route("/api/v1/factories/([^/]+)/planning-sessions/([^/]+)/messages"),
      resolve: (match, method, body) => {
        if (method !== "POST") {
          return null;
        }
        const session = findSessionById(fixture, match[2]);
        if (!session) {
          return null;
        }
        const text = stringValue(body?.text).trim();
        if (!text) {
          return { json: { session } };
        }
        return { json: { session: appendUserMessage(session, text) } };
      },
    },
    {
      pattern: route("/api/v1/factories/([^/]+)/planning-sessions/([^/]+)/survey-answer"),
      resolve: (match, method, body) => {
        if (method !== "POST") {
          return null;
        }
        const session = findSessionById(fixture, match[2]);
        if (!session) {
          return null;
        }
        const text = stringValue(body?.text).trim();
        if (text) {
          appendUserMessage(session, text);
        }
        session.survey = null;
        return { json: { session } };
      },
    },
  ];
}

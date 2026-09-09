import type { FactoriesFactoryPrFeedbackHandler } from "@/api-client";
import type { FixtureResult } from "@/pages/home/__fixtures__/handlers";

import type { FactoriesFixture } from "./factoryPageResponses";

export interface FactoryPRFeedbackRoute {
  pattern: RegExp;
  resolve: (match: RegExpExecArray, method: string, body: Record<string, unknown> | null, url: URL) => FixtureResult;
}

const HANDLER_NAME_BY_SOURCE: Record<string, string> = {
  SOURCE_PULL_REQUEST_DISCUSSION: "Address PR feedback",
  SOURCE_PULL_REQUEST_CHECKS: "Fix pull request checks",
};

function route(pattern: string): RegExp {
  return new RegExp(`^${pattern}$`);
}

function factoryHandlers(fixture: FactoriesFixture, factoryId: string): FactoriesFactoryPrFeedbackHandler[] {
  const existing = fixture.prFeedbackHandlersByFactoryId?.[factoryId];
  if (existing) {
    return existing;
  }
  const created: FactoriesFactoryPrFeedbackHandler[] = [];
  fixture.prFeedbackHandlersByFactoryId = { ...fixture.prFeedbackHandlersByFactoryId, [factoryId]: created };
  return created;
}

export function factoryPRFeedbackRoutes(fixture: FactoriesFixture): FactoryPRFeedbackRoute[] {
  return [
    {
      pattern: route("/api/v1/factories/([^/]+)/pr-feedback-handlers"),
      resolve: (match, method, body) => listOrCreatePRFeedbackHandlers(fixture, match[1], method, body),
    },
    {
      pattern: route("/api/v1/factories/([^/]+)/repository-status-checks"),
      resolve: (_match, method) =>
        method === "GET"
          ? {
              json: {
                repository: "acme/api",
                checks: [
                  { name: "lint", required: true },
                  { name: "unit", required: true },
                  {
                    name: "e2e",
                    required: false,
                    detailsUrl: "https://app.circleci.com/pipelines/github/acme/api/1",
                    suggestedIntegration: "circleci",
                  },
                ],
              },
            }
          : null,
    },
    {
      pattern: route("/api/v1/factories/([^/]+)/repository-review-bots"),
      resolve: (_match, method) =>
        method === "GET"
          ? {
              json: {
                repository: "acme/api",
                bots: [
                  { login: "coderabbitai", displayName: "coderabbitai[bot]" },
                  { login: "bugbot", displayName: "bugbot[bot]" },
                ],
              },
            }
          : null,
    },
  ];
}

function listOrCreatePRFeedbackHandlers(
  fixture: FactoriesFixture,
  factoryId: string,
  method: string,
  body: Record<string, unknown> | null,
): FixtureResult {
  const handlers = factoryHandlers(fixture, factoryId);
  if (method === "GET") {
    return { json: { handlers } };
  }
  if (method !== "POST") {
    return null;
  }

  const source = typeof body?.source === "string" ? body.source : "SOURCE_PULL_REQUEST_DISCUSSION";
  const name =
    typeof body?.name === "string" && body.name.trim()
      ? body.name.trim()
      : (HANDLER_NAME_BY_SOURCE[source] ?? "PR feedback");
  const handler: FactoriesFactoryPrFeedbackHandler = {
    id: `prfb-${source === "SOURCE_PULL_REQUEST_CHECKS" ? "checks" : "discussion"}-${handlers.length + 1}`,
    factoryId,
    canvasId: `app-${source === "SOURCE_PULL_REQUEST_CHECKS" ? "pr-checks" : "pr-discussion"}-${handlers.length + 1}`,
    name,
    source: source as FactoriesFactoryPrFeedbackHandler["source"],
    healthy: true,
  };
  handlers.push(handler);
  return { json: { handler } };
}

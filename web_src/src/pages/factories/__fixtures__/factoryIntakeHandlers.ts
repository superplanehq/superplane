import type { FactoriesFactoryIntake, FactoryIntakeSettings } from "@/api-client";
import type { FixtureResult } from "@/pages/home/__fixtures__/handlers";

import type { FactoriesFixture } from "./factoryPageResponses";

export interface FactoryIntakeRoute {
  pattern: RegExp;
  resolve: (match: RegExpExecArray, method: string, body: Record<string, unknown> | null, url: URL) => FixtureResult;
}

const INTAKE_NAME_BY_SOURCE: Record<string, string> = {
  SOURCE_GITHUB_ISSUES: "GitHub issues",
  SOURCE_SENTRY_EXCEPTIONS: "Sentry exceptions",
  SOURCE_PAGERDUTY_INCIDENTS: "PagerDuty incidents",
};

function route(pattern: string): RegExp {
  return new RegExp(`^${pattern}$`);
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function factoryIntakes(fixture: FactoriesFixture, factoryId: string): FactoriesFactoryIntake[] {
  const existing = fixture.intakesByFactoryId?.[factoryId];
  if (existing) {
    return existing;
  }

  const created: FactoriesFactoryIntake[] = [];
  fixture.intakesByFactoryId = { ...fixture.intakesByFactoryId, [factoryId]: created };
  return created;
}

export function factoryIntakeRoutes(fixture: FactoriesFixture): FactoryIntakeRoute[] {
  return [
    {
      pattern: route("/api/v1/factories/([^/]+)/intakes/([^/]+)/runs"),
      resolve: (match) => ({ json: { runs: fixture.intakeRunsByIntakeId?.[match[2]] ?? [] } }),
    },
    {
      pattern: route("/api/v1/factories/([^/]+)/intakes/([^/]+)"),
      resolve: (match, method, body) => updateOrDeleteFactoryIntake(fixture, match, method, body),
    },
    {
      pattern: route("/api/v1/factories/([^/]+)/intakes"),
      resolve: (match, method, body) => listOrCreateFactoryIntakes(fixture, match[1], method, body),
    },
  ];
}

function updateOrDeleteFactoryIntake(
  fixture: FactoriesFixture,
  match: RegExpExecArray,
  method: string,
  body: Record<string, unknown> | null,
): FixtureResult {
  const intakes = factoryIntakes(fixture, match[1]);
  const index = intakes.findIndex((intake) => intake.id === match[2]);
  if (index < 0) {
    return { json: {} };
  }
  if (method === "DELETE") {
    intakes.splice(index, 1);
    return { json: {} };
  }
  if (method !== "PATCH" && method !== "PUT") {
    return null;
  }

  const request = (body ?? {}) as { name?: unknown; settings?: FactoryIntakeSettings };
  const updated: FactoriesFactoryIntake = {
    ...intakes[index],
    ...(stringValue(request.name) ? { name: stringValue(request.name) } : {}),
    ...(request.settings ? { settings: { ...intakes[index].settings, ...request.settings } } : {}),
  };
  intakes[index] = updated;
  return { json: { intake: updated } };
}

function listOrCreateFactoryIntakes(
  fixture: FactoriesFixture,
  factoryId: string,
  method: string,
  body: Record<string, unknown> | null,
): FixtureResult {
  const intakes = factoryIntakes(fixture, factoryId);
  if (method !== "POST") {
    return { json: { intakes } };
  }

  const request = (body ?? {}) as { source?: unknown; name?: unknown; confidencePct?: unknown };
  const source = stringValue(request.source) || "SOURCE_GITHUB_ISSUES";
  const created: FactoriesFactoryIntake = {
    id: `storybook-intake-${intakes.length + 1}`,
    canvasId: `storybook-intake-canvas-${intakes.length + 1}`,
    factoryId,
    name: stringValue(request.name) || INTAKE_NAME_BY_SOURCE[source] || source,
    source: source as FactoriesFactoryIntake["source"],
    settings: {
      confidencePct: typeof request.confidencePct === "number" ? request.confidencePct : 65,
      labels: [],
      labelFilterMode: "LABEL_FILTER_MODE_INCLUDE",
      assignment: "ASSIGNMENT_ANY",
    },
    healthy: true,
  };
  intakes.push(created);
  return { json: { intake: created } };
}

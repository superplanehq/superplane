import type { CanvasesCanvasSummary } from "@/api-client";

/** Shared with AppPage stories so home → app continuity is obvious. */
export const HOME_ORGANIZATION_ID = "3ee1aa47-3a60-4c1f-b645-0b9859ab91f8";
export const SOFTWARE_FACTORY_APP_ID = "9725f25b-2947-4022-82f9-acb20a616bf6";

function makeCanvas(
  id: string,
  name: string,
  options: {
    description?: string;
    starred?: boolean;
    starredAt?: string;
    createdAt?: string;
    createdByName?: string;
  } = {},
): CanvasesCanvasSummary {
  return {
    id,
    name,
    description: options.description,
    createdAt: options.createdAt ?? "2026-05-05T00:00:00Z",
    createdBy: { name: options.createdByName ?? "Leonardo DiCaprio" },
    starred: options.starred,
    starredAt: options.starredAt,
  };
}

const canvases: CanvasesCanvasSummary[] = [
  makeCanvas(SOFTWARE_FACTORY_APP_ID, "Software Factory", {
    description: "Issue → plan → PR → CI babysitting for factory-labeled work.",
    starred: true,
    starredAt: "2026-07-16T12:00:00Z",
    createdAt: "2026-06-01T10:00:00Z",
  }),
  makeCanvas("app-pr-risk-review", "PR Risk Review", {
    description: "Scores pull requests and posts a risk summary.",
    createdAt: "2026-06-10T14:00:00Z",
  }),
  makeCanvas("app-docs-reviewer", "Docs Reviewer", {
    description: "Reviews documentation changes on open PRs.",
    createdAt: "2026-06-12T09:30:00Z",
  }),
  makeCanvas("app-superplane-saas", "SuperPlane SaaS", {
    description: "Production deployment pipeline console.",
    createdAt: "2026-05-20T08:00:00Z",
  }),
  makeCanvas("app-superplane-release", "SuperPlane Release", {
    description: "Release status, in-flight cuts, and history.",
    createdAt: "2026-05-22T11:15:00Z",
  }),
  makeCanvas("app-clean-code", "Clean Code Assessment", {
    description: "Grades PRs and posts a clean-code report.",
    createdAt: "2026-06-24T22:37:20Z",
  }),
];

export interface HomePageFixture {
  organizationId: string;
  organizationName: string;
  /** Organization slug shown on the org General settings tabs. Defaults to a slugified organizationName. */
  organizationSlug?: string;
  canvases: CanvasesCanvasSummary[];
  enabledExperimentalFeatures?: string[];
  factories?: Array<{ id: string; name: string; description?: string }>;
}

export const defaultHomePageFixture: HomePageFixture = {
  organizationId: HOME_ORGANIZATION_ID,
  organizationName: "SuperPlane",
  organizationSlug: "superplane",
  canvases,
};

/** Fresh org: no apps — HomePage redirects to the create/setup screen. */
export const emptyHomePageFixture: HomePageFixture = {
  organizationId: HOME_ORGANIZATION_ID,
  organizationName: "Acme",
  organizationSlug: "acme",
  canvases: [],
};

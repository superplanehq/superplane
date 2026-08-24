import { describe, expect, it } from "vitest";

import { findIngestionApp, installedIngestionFactoryIds } from "./ingestionApp";

describe("findIngestionApp", () => {
  it("finds the app by its bundled title", () => {
    const apps = [
      { id: "app-1", name: "Planning" },
      { id: "app-2", name: "Issue Ingestion" },
    ];

    expect(findIngestionApp(apps)?.id).toBe("app-2");
  });

  it("finds workspaces that still carry the old title", () => {
    const apps = [{ id: "app-1", name: "Issue Intake" }];

    expect(findIngestionApp(apps)?.id).toBe("app-1");
  });

  it("ignores the numeric suffix that onboarding adds to duplicate names", () => {
    const apps = [{ id: "app-1", name: "Issue Ingestion (2)" }];

    expect(findIngestionApp(apps)?.id).toBe("app-1");
  });

  it("returns nothing when the workspace has no ingestion app", () => {
    const apps = [{ id: "app-1", name: "PR Closure" }];

    expect(findIngestionApp(apps)).toBeUndefined();
  });
});

describe("installedIngestionFactoryIds", () => {
  it("finds both optional ingestion automations", () => {
    const apps = [
      { id: "app-1", name: "Issue Ingestion" },
      { id: "app-2", name: "Sentry Issue Ingestion" },
    ];

    expect([...installedIngestionFactoryIds(apps)]).toEqual(["issue-intake", "sentry-issue-intake"]);
  });
});

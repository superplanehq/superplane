import { describe, expect, it } from "bun:test";

import {
  canSaveIntakeConnection,
  factoryIntakeUpdateInput,
  filterIntakeConnections,
  intakeChooseConnectionCopy,
  intakeConnectLabel,
  intakeConnectionChanged,
  intakeConnectionComplete,
  intakeConnectionFromSource,
  intakeConnectionReturnPath,
  intakeHealthBanner,
  intakeProviderAppName,
  intakeReconnectLabel,
  intakeSourceAllowsRebind,
} from "./intakeConnectionModel";

describe("intakeConnectionModel", () => {
  it("allows rebind for Jira, Sentry, and Productive.io", () => {
    expect(intakeSourceAllowsRebind("jira-issues")).toBe(true);
    expect(intakeSourceAllowsRebind("sentry-exceptions")).toBe(true);
    expect(intakeSourceAllowsRebind("productive-tasks")).toBe(true);
    expect(intakeSourceAllowsRebind("github-issues")).toBe(false);
    expect(intakeSourceAllowsRebind("pagerduty-incidents")).toBe(false);
  });

  it("maps the provider app name", () => {
    expect(intakeProviderAppName("jira-issues")).toBe("jira");
    expect(intakeProviderAppName("sentry-exceptions")).toBe("sentry");
    expect(intakeProviderAppName("productive-tasks")).toBe("productive");
  });

  it("returns health banners for repair states", () => {
    expect(intakeHealthBanner("HEALTH_OK")).toBeUndefined();
    expect(intakeHealthBanner("HEALTH_MISSING_INTEGRATION")).toBe("This intake has no live connection.");
    expect(intakeHealthBanner("HEALTH_INTEGRATION_NOT_READY")).toBe("This connection cannot receive items.");
    expect(intakeHealthBanner("HEALTH_WEBHOOK_NOT_READY")).toBe("SuperPlane is still registering the Jira webhook.");
    expect(intakeHealthBanner("HEALTH_GRAPH_BROKEN")).toBe(
      "This automation cannot create tasks. Open the Automation tab to repair the steps.",
    );
  });

  it("detects a complete binding change", () => {
    expect(intakeConnectionComplete({ integrationId: "int-1", resourceId: "ENG" })).toBe(true);
    expect(intakeConnectionComplete({ integrationId: "int-1", resourceId: "" })).toBe(false);
    expect(
      intakeConnectionChanged(
        { integrationId: "int-1", resourceId: "ENG" },
        { integrationId: "int-2", resourceId: "ENG" },
      ),
    ).toBe(true);
  });

  it("filters connections by provider", () => {
    expect(
      filterIntakeConnections(
        [
          { metadata: { id: "jira-1", integrationName: "jira" } },
          { metadata: { id: "sentry-1", integrationName: "sentry" } },
        ],
        "jira-issues",
      ).map((integration) => integration.metadata?.id),
    ).toEqual(["jira-1"]);
  });

  it("labels connect and reconnect from the selected connection", () => {
    expect(intakeConnectLabel(false, "Jira")).toBe("Connect Jira");
    expect(intakeConnectLabel(true, "Jira")).toBe("Connect another");
    expect(intakeChooseConnectionCopy("jira-issues")).toBe("Choose the Jira site that SuperPlane will monitor.");
    expect(
      intakeReconnectLabel({
        metadata: { id: "jira-1", name: "Jira", integrationName: "jira" },
        status: { state: "error", stateDescription: "Authorization revoked, please reconnect the account" },
      }),
    ).toBe("Reconnect");
  });

  it("reads a live binding from the intake source", () => {
    expect(intakeConnectionFromSource({ integrationId: " int-1 ", resourceId: "ENG" })).toEqual({
      integrationId: "int-1",
      resourceId: "ENG",
    });
  });

  it("returns to intake General settings after a provider round trip", () => {
    expect(intakeConnectionReturnPath("org-1", "SP", "line-plan", "intake-1")).toBe(
      "/org-1/workspaces/sp/lines/line-plan?intake=1&intakeId=intake-1&settings=general",
    );
  });

  it("blocks save until a missing connection is complete", () => {
    expect(
      canSaveIntakeConnection(
        true,
        "HEALTH_MISSING_INTEGRATION",
        { integrationId: "", resourceId: "" },
        { integrationId: "int-1", resourceId: "" },
      ),
    ).toBe(false);
    expect(
      canSaveIntakeConnection(
        true,
        "HEALTH_MISSING_INTEGRATION",
        { integrationId: "", resourceId: "" },
        { integrationId: "int-1", resourceId: "ENG" },
      ),
    ).toBe(true);
  });

  it("sends the binding only when the connection changed", () => {
    const settings = { confidencePct: 65 };
    expect(
      factoryIntakeUpdateInput(
        "intake-1",
        settings,
        { integrationId: "int-1", resourceId: "ENG" },
        { integrationId: "int-1", resourceId: "ENG" },
        true,
      ),
    ).toEqual({ intakeId: "intake-1", settings });
    expect(
      factoryIntakeUpdateInput(
        "intake-1",
        settings,
        { integrationId: "int-1", resourceId: "ENG" },
        { integrationId: "int-2", resourceId: "OPS" },
        true,
      ),
    ).toEqual({
      intakeId: "intake-1",
      settings,
      integrationId: "int-2",
      resourceId: "OPS",
    });
  });
});

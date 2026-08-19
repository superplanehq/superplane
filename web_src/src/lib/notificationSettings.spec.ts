import { describe, expect, it } from "vitest";

import {
  defaultNotificationSettings,
  eventTypesFromToggles,
  filtersFromSettings,
  togglesFromAllScopeEventTypes,
  togglesFromEventTypes,
  workspaceScopeFromSettings,
} from "./notificationSettings";

describe("notificationSettings", () => {
  it("defaults to all workspaces", () => {
    const settings = defaultNotificationSettings();
    expect(settings.workspaces?.scope).toBe("WORKSPACE_SCOPE_ALL");
    expect(settings.workspaces?.filters).toEqual([]);
    expect(workspaceScopeFromSettings(undefined)).toBe("all");
  });

  it("treats a missing all-scope type list as every type on", () => {
    expect(togglesFromAllScopeEventTypes(undefined).TYPE_WORK_ORDER_ASSIGNED).toBe(true);
    expect(togglesFromAllScopeEventTypes([]).TYPE_WORK_ORDER_ARTIFACT_OWNED).toBe(true);
    expect(togglesFromAllScopeEventTypes(["TYPE_WORK_ORDER_ASSIGNED"]).TYPE_WORK_ORDER_COMMENT_OWNED).toBe(false);
  });

  it("treats a missing filtered type as off", () => {
    const toggles = togglesFromEventTypes(["TYPE_WORK_ORDER_ASSIGNED"]);
    expect(toggles.TYPE_WORK_ORDER_ASSIGNED).toBe(true);
    expect(toggles.TYPE_WORK_ORDER_COMMENT_OWNED).toBe(false);
  });

  it("round-trips enabled types into the API payload", () => {
    const toggles = togglesFromEventTypes(["TYPE_WORK_ORDER_ARTIFACT_OWNED"]);
    expect(eventTypesFromToggles(toggles)).toEqual(["TYPE_WORK_ORDER_ARTIFACT_OWNED"]);
  });

  it("ignores filters unless the scope is filtered", () => {
    expect(
      filtersFromSettings({
        workspaces: {
          scope: "WORKSPACE_SCOPE_ALL",
          filters: [{ workspaceId: "ws-1", eventTypes: ["TYPE_WORK_ORDER_ASSIGNED"] }],
        },
      }),
    ).toEqual([]);
    expect(
      filtersFromSettings({
        workspaces: {
          scope: "WORKSPACE_SCOPE_FILTERED",
          filters: [{ workspaceId: "ws-1", eventTypes: ["TYPE_WORK_ORDER_ASSIGNED"] }],
        },
      }),
    ).toEqual([{ workspaceId: "ws-1", eventTypes: ["TYPE_WORK_ORDER_ASSIGNED"] }]);
  });
});

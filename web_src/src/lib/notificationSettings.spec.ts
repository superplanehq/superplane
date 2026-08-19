import { describe, expect, it } from "vitest";

import {
  defaultNotificationSettings,
  defaultNotificationTypeToggles,
  eventTypesFromToggles,
  filtersFromSettings,
  NOTIFICATION_TYPE_OPTIONS,
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
    expect(defaultNotificationTypeToggles().TYPE_WORK_ORDER_MENTIONED).toBe(true);
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

describe("NOTIFICATION_TYPE_OPTIONS", () => {
  it("states each event label without requiring the tooltip", () => {
    const labelsByKey = Object.fromEntries(NOTIFICATION_TYPE_OPTIONS.map((option) => [option.key, option.label]));

    expect(labelsByKey).toEqual({
      TYPE_WORK_ORDER_ASSIGNED: "Added as a work order owner",
      TYPE_WORK_ORDER_COMMENT_OWNED: "Comments on work orders you own",
      TYPE_WORK_ORDER_COMMENT_CREATED: "Comments on work orders you created",
      TYPE_WORK_ORDER_STATUS_OWNED: "Status changes on work orders you own or created",
      TYPE_WORK_ORDER_ARTIFACT_OWNED: "New artifacts on work orders you own",
      TYPE_WORK_ORDER_MENTIONED: "Mentions in work order comments",
    });
  });

  it("keeps the two comment labels parallel so neither reads as authorship", () => {
    const owned = NOTIFICATION_TYPE_OPTIONS.find((option) => option.key === "TYPE_WORK_ORDER_COMMENT_OWNED");
    const created = NOTIFICATION_TYPE_OPTIONS.find((option) => option.key === "TYPE_WORK_ORDER_COMMENT_CREATED");

    expect(owned?.label.startsWith("Comments on work orders you ")).toBe(true);
    expect(created?.label.startsWith("Comments on work orders you ")).toBe(true);
  });
});

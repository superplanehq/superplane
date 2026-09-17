import { describe, expect, it } from "bun:test";

import {
  accountNotificationsFromSettings,
  defaultNotificationSettings,
  defaultNotificationTypeToggles,
  eventTypesFromToggles,
  filtersFromSettings,
  NOTIFICATION_SETTINGS_TYPES,
  NOTIFICATION_TYPE_OPTIONS,
  settingsFromAccountNotifications,
  togglesFromAllScopeEventTypes,
  togglesFromEventTypes,
  workspaceScopeFromChannel,
  workspaceScopeFromSettings,
} from "./notificationSettings";

describe("notificationSettings", () => {
  it("defaults to all workspaces and an off browser channel", () => {
    const settings = defaultNotificationSettings();
    expect(settings.workspaces?.scope).toBe("WORKSPACE_SCOPE_ALL");
    expect(settings.workspaces?.filters).toEqual([]);
    expect(settings.browser?.scope).toBe("WORKSPACE_SCOPE_NONE");
    expect(settings.browser?.showWhileViewing).toBe(true);
    expect(workspaceScopeFromSettings(undefined)).toBe("all");
    expect(workspaceScopeFromChannel(undefined, "none")).toBe("none");
    expect(defaultNotificationTypeToggles().TYPE_WORK_ORDER_AGENT_QUESTION).toBe(true);
  });

  it("treats a missing all-scope type list as every type on", () => {
    expect(togglesFromAllScopeEventTypes(undefined).TYPE_WORK_ORDER_STATUS_OWNED).toBe(true);
    expect(togglesFromAllScopeEventTypes([]).TYPE_WORK_ORDER_PLAN_READY).toBe(true);
    expect(togglesFromAllScopeEventTypes(["TYPE_WORK_ORDER_STATUS_OWNED"]).TYPE_WORK_ORDER_AGENT_QUESTION).toBe(false);
  });

  it("treats a missing filtered type as off", () => {
    const toggles = togglesFromEventTypes(["TYPE_WORK_ORDER_STATUS_OWNED"]);
    expect(toggles.TYPE_WORK_ORDER_STATUS_OWNED).toBe(true);
    expect(toggles.TYPE_WORK_ORDER_AGENT_QUESTION).toBe(false);
  });

  it("round-trips enabled types into the API payload", () => {
    const toggles = togglesFromEventTypes(["TYPE_WORK_ORDER_PLAN_READY"]);
    expect(eventTypesFromToggles(toggles)).toEqual(["TYPE_WORK_ORDER_PLAN_READY"]);
  });

  it("ignores filters unless the scope is filtered", () => {
    expect(
      filtersFromSettings({
        workspaces: {
          scope: "WORKSPACE_SCOPE_ALL",
          filters: [{ workspaceId: "ws-1", eventTypes: ["TYPE_WORK_ORDER_STATUS_OWNED"] }],
        },
      }),
    ).toEqual([]);
    expect(
      filtersFromSettings({
        workspaces: {
          scope: "WORKSPACE_SCOPE_FILTERED",
          filters: [{ workspaceId: "ws-1", eventTypes: ["TYPE_WORK_ORDER_STATUS_OWNED"] }],
        },
      }),
    ).toEqual([{ workspaceId: "ws-1", eventTypes: ["TYPE_WORK_ORDER_STATUS_OWNED"] }]);
  });

  it("maps API settings onto the account notifications form", () => {
    expect(accountNotificationsFromSettings({ workspaces: { scope: "WORKSPACE_SCOPE_NONE" } })).toMatchObject({
      emailEnabled: false,
      workspaceScope: "all",
      browserEnabled: false,
      browserShowWhileViewing: true,
    });
    expect(
      accountNotificationsFromSettings({
        workspaces: {
          scope: "WORKSPACE_SCOPE_FILTERED",
          filters: [{ workspaceId: "ws-1", eventTypes: ["TYPE_WORK_ORDER_STATUS_OWNED"] }],
        },
      }),
    ).toMatchObject({
      emailEnabled: true,
      workspaceScope: "selected",
      workspaceIds: ["ws-1"],
    });
    expect(
      settingsFromAccountNotifications({
        emailEnabled: false,
        workspaceScope: "all",
        workspaceIds: [],
        events: defaultNotificationTypeToggles(true),
        browserEnabled: false,
        browserWorkspaceScope: "all",
        browserWorkspaceIds: [],
        browserEvents: defaultNotificationTypeToggles(true),
        browserShowWhileViewing: true,
      }).workspaces?.scope,
    ).toBe("WORKSPACE_SCOPE_NONE");
  });

  it("round-trips browser channel fields", () => {
    const form = accountNotificationsFromSettings({
      workspaces: { scope: "WORKSPACE_SCOPE_ALL" },
      browser: {
        scope: "WORKSPACE_SCOPE_FILTERED",
        showWhileViewing: false,
        filters: [{ workspaceId: "ws-2", eventTypes: ["TYPE_WORK_ORDER_AGENT_QUESTION"] }],
      },
    });
    expect(form).toMatchObject({
      browserEnabled: true,
      browserWorkspaceScope: "selected",
      browserWorkspaceIds: ["ws-2"],
      browserShowWhileViewing: false,
    });
    expect(form.browserEvents.TYPE_WORK_ORDER_AGENT_QUESTION).toBe(true);
    expect(form.browserEvents.TYPE_WORK_ORDER_STATUS_OWNED).toBe(false);
    expect(settingsFromAccountNotifications(form).browser).toMatchObject({
      scope: "WORKSPACE_SCOPE_FILTERED",
      showWhileViewing: false,
    });
  });
});

describe("NOTIFICATION_TYPE_OPTIONS", () => {
  it("lists the four events a person can see and act on", () => {
    expect(NOTIFICATION_SETTINGS_TYPES).toEqual([
      "TYPE_WORK_ORDER_STATUS_OWNED",
      "TYPE_WORK_ORDER_STATUS_NOTE_OWNED",
      "TYPE_WORK_ORDER_AGENT_QUESTION",
      "TYPE_WORK_ORDER_PLAN_READY",
    ]);
    expect(NOTIFICATION_TYPE_OPTIONS.map((option) => option.key)).toEqual([...NOTIFICATION_SETTINGS_TYPES]);
  });

  it("states each event label without requiring the tooltip", () => {
    const labelsByKey = Object.fromEntries(NOTIFICATION_TYPE_OPTIONS.map((option) => [option.key, option.label]));

    expect(labelsByKey).toEqual({
      TYPE_WORK_ORDER_STATUS_OWNED: "Status changes on your tasks",
      TYPE_WORK_ORDER_STATUS_NOTE_OWNED: "Review requests on your tasks",
      TYPE_WORK_ORDER_AGENT_QUESTION: "Agent questions on your tasks",
      TYPE_WORK_ORDER_PLAN_READY: "Ready plans on your tasks",
    });
  });
});

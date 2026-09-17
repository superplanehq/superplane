import type {
  MeNotificationSettings,
  MeNotificationSettingsType,
  NotificationSettingsBrowser,
  NotificationSettingsWorkspaceFilter,
  NotificationSettingsWorkspaces,
} from "@/api-client";

export const NOTIFICATION_SETTINGS_TYPES = [
  "TYPE_WORK_ORDER_STATUS_OWNED",
  "TYPE_WORK_ORDER_STATUS_NOTE_OWNED",
  "TYPE_WORK_ORDER_AGENT_QUESTION",
  "TYPE_WORK_ORDER_PLAN_READY",
] as const satisfies readonly Exclude<MeNotificationSettingsType, "TYPE_UNSPECIFIED">[];

export type ConfigurableNotificationType = (typeof NOTIFICATION_SETTINGS_TYPES)[number];

export type NotificationTypeToggles = Record<ConfigurableNotificationType, boolean>;

export type WorkspaceScopeForm = "all" | "filtered" | "none";

export type AccountNotificationWorkspaceScope = "all" | "selected";

export interface AccountNotificationForm {
  emailEnabled: boolean;
  workspaceScope: AccountNotificationWorkspaceScope;
  workspaceIds: string[];
  events: NotificationTypeToggles;
  browserEnabled: boolean;
  browserWorkspaceScope: AccountNotificationWorkspaceScope;
  browserWorkspaceIds: string[];
  browserEvents: NotificationTypeToggles;
  browserShowWhileViewing: boolean;
}

export interface NotificationTypeOption {
  key: ConfigurableNotificationType;
  label: string;
  description: string;
}

export const NOTIFICATION_TYPE_OPTIONS: NotificationTypeOption[] = [
  {
    key: "TYPE_WORK_ORDER_STATUS_OWNED",
    label: "Status changes on your tasks",
    description: "A task you created opens, closes, or moves back to draft.",
  },
  {
    key: "TYPE_WORK_ORDER_STATUS_NOTE_OWNED",
    label: "Review requests on your tasks",
    description: "An automation flags a task you created as waiting on your review.",
  },
  {
    key: "TYPE_WORK_ORDER_AGENT_QUESTION",
    label: "Agent questions on your tasks",
    description: "The agent asks a question and waits for an answer.",
  },
  {
    key: "TYPE_WORK_ORDER_PLAN_READY",
    label: "Ready plans on your tasks",
    description: "Refinement finishes and the plan is ready.",
  },
];

export function defaultNotificationTypeToggles(enabled = true): NotificationTypeToggles {
  return Object.fromEntries(NOTIFICATION_SETTINGS_TYPES.map((type) => [type, enabled])) as NotificationTypeToggles;
}

export function defaultNotificationSettings(): MeNotificationSettings {
  return {
    workspaces: {
      scope: "WORKSPACE_SCOPE_ALL",
      filters: [],
    },
    browser: {
      scope: "WORKSPACE_SCOPE_NONE",
      filters: [],
      eventTypes: [],
      showWhileViewing: true,
    },
  };
}

export function workspaceScopeFromSettings(settings: MeNotificationSettings | undefined): WorkspaceScopeForm {
  return workspaceScopeFromChannel(settings?.workspaces);
}

export function workspaceScopeFromChannel(
  channel: Pick<NotificationSettingsWorkspaces, "scope"> | undefined,
  missing: WorkspaceScopeForm = "all",
): WorkspaceScopeForm {
  switch (channel?.scope) {
    case "WORKSPACE_SCOPE_FILTERED":
      return "filtered";
    case "WORKSPACE_SCOPE_NONE":
      return "none";
    case "WORKSPACE_SCOPE_ALL":
      return "all";
    default:
      return missing;
  }
}

export function eventTypesFromToggles(toggles: NotificationTypeToggles): ConfigurableNotificationType[] {
  return NOTIFICATION_SETTINGS_TYPES.filter((type) => toggles[type]);
}

export function togglesFromEventTypes(eventTypes: MeNotificationSettingsType[] | undefined): NotificationTypeToggles {
  const selected = new Set((eventTypes ?? []).filter(isConfigurableNotificationType));
  return Object.fromEntries(
    NOTIFICATION_SETTINGS_TYPES.map((type) => [type, selected.has(type)]),
  ) as NotificationTypeToggles;
}

export function togglesFromAllScopeEventTypes(
  eventTypes: MeNotificationSettingsType[] | undefined,
): NotificationTypeToggles {
  if (!eventTypes || eventTypes.length === 0) {
    return defaultNotificationTypeToggles(true);
  }
  return togglesFromEventTypes(eventTypes);
}

export function filtersFromSettings(
  settings: MeNotificationSettings | undefined,
): NotificationSettingsWorkspaceFilter[] {
  return filtersFromChannel(settings?.workspaces);
}

export function filtersFromChannel(
  channel: Pick<NotificationSettingsWorkspaces, "scope" | "filters"> | undefined,
): NotificationSettingsWorkspaceFilter[] {
  if (channel?.scope !== "WORKSPACE_SCOPE_FILTERED") {
    return [];
  }
  return channel.filters ?? [];
}

export function isConfigurableNotificationType(
  type: MeNotificationSettingsType | undefined,
): type is ConfigurableNotificationType {
  return NOTIFICATION_SETTINGS_TYPES.some((known) => known === type);
}

export function accountNotificationsFromSettings(
  settings: MeNotificationSettings | undefined,
): AccountNotificationForm {
  const email = channelFormFromSettings(settings?.workspaces, "all");
  const browser = channelFormFromSettings(settings?.browser, "none");
  return {
    emailEnabled: email.enabled,
    workspaceScope: email.workspaceScope,
    workspaceIds: email.workspaceIds,
    events: email.events,
    browserEnabled: browser.enabled,
    browserWorkspaceScope: browser.workspaceScope,
    browserWorkspaceIds: browser.workspaceIds,
    browserEvents: browser.events,
    browserShowWhileViewing: settings?.browser?.showWhileViewing ?? true,
  };
}

export function settingsFromAccountNotifications(form: AccountNotificationForm): MeNotificationSettings {
  return {
    workspaces: channelSettingsFromForm(form.emailEnabled, form.workspaceScope, form.workspaceIds, form.events),
    browser: {
      ...channelSettingsFromForm(
        form.browserEnabled,
        form.browserWorkspaceScope,
        form.browserWorkspaceIds,
        form.browserEvents,
      ),
      showWhileViewing: form.browserShowWhileViewing,
    },
  };
}

function channelFormFromSettings(
  channel: NotificationSettingsWorkspaces | NotificationSettingsBrowser | undefined,
  missingScope: WorkspaceScopeForm,
): {
  enabled: boolean;
  workspaceScope: AccountNotificationWorkspaceScope;
  workspaceIds: string[];
  events: NotificationTypeToggles;
} {
  const scope = workspaceScopeFromChannel(channel, missingScope);
  const filters = filtersFromChannel(channel);
  return {
    enabled: scope !== "none",
    workspaceScope: scope === "filtered" ? "selected" : "all",
    workspaceIds: filters.flatMap((filter) => (filter.workspaceId ? [filter.workspaceId] : [])),
    events:
      scope === "filtered"
        ? togglesFromEventTypes(filters[0]?.eventTypes)
        : togglesFromAllScopeEventTypes(channel?.eventTypes),
  };
}

function channelSettingsFromForm(
  enabled: boolean,
  workspaceScope: AccountNotificationWorkspaceScope,
  workspaceIds: string[],
  events: NotificationTypeToggles,
): NotificationSettingsWorkspaces {
  if (!enabled) {
    return { scope: "WORKSPACE_SCOPE_NONE", eventTypes: [], filters: [] };
  }
  if (workspaceScope === "selected") {
    return {
      scope: "WORKSPACE_SCOPE_FILTERED",
      eventTypes: [],
      filters: workspaceIds.map((workspaceId) => ({
        workspaceId,
        eventTypes: eventTypesFromToggles(events),
      })),
    };
  }
  return {
    scope: "WORKSPACE_SCOPE_ALL",
    eventTypes: eventTypesFromToggles(events),
    filters: [],
  };
}

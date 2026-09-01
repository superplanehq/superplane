import type {
  MeNotificationSettings,
  MeNotificationSettingsType,
  NotificationSettingsWorkspaceFilter,
} from "@/api-client";

export const NOTIFICATION_SETTINGS_TYPES = [
  "TYPE_WORK_ORDER_ASSIGNED",
  "TYPE_WORK_ORDER_COMMENT_OWNED",
  "TYPE_WORK_ORDER_COMMENT_CREATED",
  "TYPE_WORK_ORDER_STATUS_OWNED",
  "TYPE_WORK_ORDER_ARTIFACT_OWNED",
  "TYPE_WORK_ORDER_MENTIONED",
  "TYPE_WORK_ORDER_STATUS_NOTE_OWNED",
] as const satisfies readonly Exclude<MeNotificationSettingsType, "TYPE_UNSPECIFIED">[];

export type ConfigurableNotificationType = (typeof NOTIFICATION_SETTINGS_TYPES)[number];

export type NotificationTypeToggles = Record<ConfigurableNotificationType, boolean>;

export type WorkspaceScopeForm = "all" | "filtered" | "none";

export interface NotificationTypeOption {
  key: ConfigurableNotificationType;
  label: string;
  description: string;
}

export const NOTIFICATION_TYPE_OPTIONS: NotificationTypeOption[] = [
  {
    key: "TYPE_WORK_ORDER_ASSIGNED",
    label: "Added as a task owner",
    description: "You become an owner of a task.",
  },
  {
    key: "TYPE_WORK_ORDER_COMMENT_OWNED",
    label: "Comments on tasks you own",
    description: "Someone comments on a task you own.",
  },
  {
    key: "TYPE_WORK_ORDER_COMMENT_CREATED",
    label: "Comments on tasks you created",
    description: "Someone comments on a task you created.",
  },
  {
    key: "TYPE_WORK_ORDER_STATUS_OWNED",
    label: "Status changes on tasks you own or created",
    description: "A task you own or created opens, closes, or moves back to draft.",
  },
  {
    key: "TYPE_WORK_ORDER_ARTIFACT_OWNED",
    label: "New artifacts on tasks you own",
    description: "An artifact is added to a task you own.",
  },
  {
    key: "TYPE_WORK_ORDER_MENTIONED",
    label: "Mentions in task comments",
    description: "Someone mentions you in a task comment.",
  },
  {
    key: "TYPE_WORK_ORDER_STATUS_NOTE_OWNED",
    label: "Review requests on tasks you own or created",
    description: "An automation flags a task you own or created as waiting on your review.",
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
  };
}

export function workspaceScopeFromSettings(settings: MeNotificationSettings | undefined): WorkspaceScopeForm {
  switch (settings?.workspaces?.scope) {
    case "WORKSPACE_SCOPE_FILTERED":
      return "filtered";
    case "WORKSPACE_SCOPE_NONE":
      return "none";
    default:
      return "all";
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
  if (settings?.workspaces?.scope !== "WORKSPACE_SCOPE_FILTERED") {
    return [];
  }
  return settings.workspaces.filters ?? [];
}

export function isConfigurableNotificationType(
  type: MeNotificationSettingsType | undefined,
): type is ConfigurableNotificationType {
  return NOTIFICATION_SETTINGS_TYPES.some((known) => known === type);
}

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
    label: "Added as a work order owner",
    description: "You become an owner of a work order.",
  },
  {
    key: "TYPE_WORK_ORDER_COMMENT_OWNED",
    label: "Comments on work orders you own",
    description: "Someone comments on a work order you own.",
  },
  {
    key: "TYPE_WORK_ORDER_COMMENT_CREATED",
    label: "Comments on work orders you created",
    description: "Someone comments on a work order you created.",
  },
  {
    key: "TYPE_WORK_ORDER_STATUS_OWNED",
    label: "Status changes on work orders you own or created",
    description: "A work order you own or created opens, closes, or moves back to draft.",
  },
  {
    key: "TYPE_WORK_ORDER_ARTIFACT_OWNED",
    label: "New artifacts on work orders you own",
    description: "An artifact is added to a work order you own.",
  },
  {
    key: "TYPE_WORK_ORDER_MENTIONED",
    label: "Mentions in work order comments",
    description: "Someone mentions you in a work order comment.",
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

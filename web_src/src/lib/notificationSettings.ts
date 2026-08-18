import type { MeNotificationSettings, MeNotificationSettingsType, NotificationSettingsTypeToggle } from "@/api-client";

export const NOTIFICATION_SETTINGS_TYPES = [
  "TYPE_WORK_ORDER_ASSIGNED",
  "TYPE_WORK_ORDER_COMMENT_OWNED",
  "TYPE_WORK_ORDER_COMMENT_CREATED",
  "TYPE_WORK_ORDER_STATUS_OWNED",
  "TYPE_WORK_ORDER_ARTIFACT_OWNED",
] as const satisfies readonly Exclude<MeNotificationSettingsType, "TYPE_UNSPECIFIED">[];

export type ConfigurableNotificationType = (typeof NOTIFICATION_SETTINGS_TYPES)[number];

export type NotificationTypeToggles = Record<ConfigurableNotificationType, boolean>;

export function defaultNotificationTypeToggles(): NotificationTypeToggles {
  return Object.fromEntries(NOTIFICATION_SETTINGS_TYPES.map((type) => [type, true])) as NotificationTypeToggles;
}

export function defaultNotificationSettings(): MeNotificationSettings {
  return {
    enabled: true,
    workspaceScope: "WORKSPACE_SCOPE_ALL",
    factoryIds: [],
    types: notificationTypesFromToggles(defaultNotificationTypeToggles()),
  };
}

export function notificationTypeTogglesFromSettings(
  settings: MeNotificationSettings | undefined,
): NotificationTypeToggles {
  const toggles = defaultNotificationTypeToggles();
  for (const toggle of settings?.types ?? []) {
    if (!isConfigurableNotificationType(toggle.type) || toggle.enabled === undefined) {
      continue;
    }
    toggles[toggle.type] = toggle.enabled;
  }
  return toggles;
}

export function notificationTypesFromToggles(toggles: NotificationTypeToggles): NotificationSettingsTypeToggle[] {
  return NOTIFICATION_SETTINGS_TYPES.map((type) => ({ type, enabled: toggles[type] }));
}

export function isConfigurableNotificationType(
  type: MeNotificationSettingsType | undefined,
): type is ConfigurableNotificationType {
  return NOTIFICATION_SETTINGS_TYPES.some((known) => known === type);
}

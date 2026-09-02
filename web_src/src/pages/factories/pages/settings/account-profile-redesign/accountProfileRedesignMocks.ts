import { defaultNotificationTypeToggles, type NotificationTypeToggles } from "@/lib/notificationSettings";

export type AccountRedesignPageId = "profile" | "linked-accounts" | "security";

export type AccountRedesignWorkspaceScope = "all" | "selected";

export interface AccountRedesignNotifications {
  emailEnabled: boolean;
  workspaceScope: AccountRedesignWorkspaceScope;
  workspaceIds: string[];
  events: NotificationTypeToggles;
}

export const ACCOUNT_REDESIGN_NOTIFICATIONS: AccountRedesignNotifications = {
  emailEnabled: true,
  workspaceScope: "all",
  workspaceIds: [],
  events: defaultNotificationTypeToggles(true),
};

export type AccountRedesignSsoProvider = "github" | "google";

export interface AccountRedesignToken {
  id: string;
  name: string;
  createdAt: string;
  lastUsedAt?: string;
}

export interface AccountRedesignSsoAccount {
  provider: AccountRedesignSsoProvider;
  identity: string | null;
  email?: string | null;
}

export interface AccountRedesignProfile {
  name: string;
  email: string;
  userId: string;
  passwordSet: boolean;
  tokens: AccountRedesignToken[];
  ssoAccounts: AccountRedesignSsoAccount[];
  notifications: AccountRedesignNotifications;
}

export const ACCOUNT_REDESIGN_PROFILE: AccountRedesignProfile = {
  name: "Ada Lovelace",
  email: "ada@example.com",
  userId: "5f76536d-bc02-4f99-81e6-e159ac40ebbb",
  passwordSet: true,
  tokens: [],
  ssoAccounts: [
    { provider: "github", identity: "ada", email: "ada@example.com" },
    { provider: "google", identity: null, email: null },
  ],
  notifications: ACCOUNT_REDESIGN_NOTIFICATIONS,
};

export const ACCOUNT_REDESIGN_SECURE_PROFILE: AccountRedesignProfile = {
  ...ACCOUNT_REDESIGN_PROFILE,
  ssoAccounts: [
    { provider: "github", identity: "ada", email: "ada@users.noreply.github.com" },
    { provider: "google", identity: "ada@example.com", email: "ada@example.com" },
  ],
  tokens: [
    {
      id: "token-1",
      name: "CLI",
      createdAt: "Mar 12, 2026",
      lastUsedAt: "Today",
    },
    {
      id: "token-2",
      name: "Deploy script",
      createdAt: "Jan 4, 2026",
    },
  ],
};

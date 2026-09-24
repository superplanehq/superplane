import { defaultNotificationTypeToggles, type AccountNotificationForm } from "@/lib/notificationSettings";

export type AccountRedesignPageId = "profile" | "security";

export type AccountRedesignWorkspaceScope = "all" | "selected";

export type AccountRedesignNotifications = AccountNotificationForm;

export const ACCOUNT_REDESIGN_NOTIFICATIONS: AccountRedesignNotifications = {
  emailEnabled: true,
  workspaceScope: "all",
  workspaceIds: [],
  events: defaultNotificationTypeToggles(true),
  browserEnabled: false,
  browserWorkspaceScope: "all",
  browserWorkspaceIds: [],
  browserEvents: defaultNotificationTypeToggles(true),
  browserShowWhileViewing: true,
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
  passwordSet: boolean;
  linkedGithubUsername: string | null;
  tokens: AccountRedesignToken[];
  ssoAccounts: AccountRedesignSsoAccount[];
  notifications: AccountRedesignNotifications;
}

export const ACCOUNT_REDESIGN_PROFILE: AccountRedesignProfile = {
  name: "Ada Lovelace",
  email: "ada@example.com",
  passwordSet: true,
  linkedGithubUsername: null,
  tokens: [],
  ssoAccounts: [
    { provider: "github", identity: "ada", email: "ada@example.com" },
    { provider: "google", identity: null, email: null },
  ],
  notifications: ACCOUNT_REDESIGN_NOTIFICATIONS,
};

export const ACCOUNT_REDESIGN_SECURE_PROFILE: AccountRedesignProfile = {
  ...ACCOUNT_REDESIGN_PROFILE,
  linkedGithubUsername: "ada",
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

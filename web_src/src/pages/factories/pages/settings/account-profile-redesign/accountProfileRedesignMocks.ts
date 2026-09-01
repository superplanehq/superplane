import type { ThemePreference } from "@/lib/themePreference";

export type AccountRedesignPageId = "profile" | "security" | "notifications" | "preferences";

export interface AccountRedesignToken {
  id: string;
  name: string;
  createdAt: string;
  lastUsedAt?: string;
}

export interface AccountRedesignSession {
  id: string;
  device: string;
  location: string;
  lastSeen: string;
  isCurrent: boolean;
}

export interface AccountRedesignProfile {
  name: string;
  email: string;
  userId: string;
  avatarUrl: string | null;
  passwordSet: boolean;
  twoFactorEnabled: boolean;
  theme: ThemePreference;
  timezone: "auto" | "America/New_York" | "Europe/London" | "Asia/Tokyo";
  tokens: AccountRedesignToken[];
  sessions: AccountRedesignSession[];
}

export const ACCOUNT_REDESIGN_PROFILE: AccountRedesignProfile = {
  name: "Ada Lovelace",
  email: "ada@example.com",
  userId: "5f76536d-bc02-4f99-81e6-e159ac40ebbb",
  avatarUrl: null,
  passwordSet: true,
  twoFactorEnabled: false,
  theme: "system",
  timezone: "auto",
  tokens: [],
  sessions: [
    {
      id: "session-current",
      device: "Chrome on macOS",
      location: "São Paulo, BR",
      lastSeen: "Now",
      isCurrent: true,
    },
    {
      id: "session-other",
      device: "Safari on macOS",
      location: "São Paulo, BR",
      lastSeen: "2 hours ago",
      isCurrent: false,
    },
  ],
};

export const ACCOUNT_REDESIGN_SECURE_PROFILE: AccountRedesignProfile = {
  ...ACCOUNT_REDESIGN_PROFILE,
  twoFactorEnabled: true,
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

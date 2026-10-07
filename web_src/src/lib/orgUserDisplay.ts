import type { SuperplaneUsersUser } from "@/api-client";

import { getNameInitials } from "@/lib/nameInitials";

export const UNKNOWN_ORG_USER_NAME = "Unknown member";

export interface OrgUserDisplay {
  id: string;
  name: string;
  initials: string;
  avatarUrl?: string;
}

export type OrgUserDisplayLookup = (userId: string | undefined, fallbackName?: string) => OrgUserDisplay | null;

export function getUserInitials(name: string): string {
  return getNameInitials(name);
}

export function getOrgUserDisplayFromUser(user: SuperplaneUsersUser): OrgUserDisplay | null {
  const id = user.metadata?.id;
  if (!id) {
    return null;
  }

  const name = user.spec?.displayName?.trim() || user.metadata?.email?.trim() || UNKNOWN_ORG_USER_NAME;

  return {
    id,
    name,
    initials: getUserInitials(name),
    avatarUrl: user.status?.accountProviders?.[0]?.avatarUrl,
  };
}

export function buildOrgUserDisplayMap(users: SuperplaneUsersUser[]): Map<string, OrgUserDisplay> {
  const map = new Map<string, OrgUserDisplay>();

  for (const user of users) {
    const display = getOrgUserDisplayFromUser(user);
    if (display) {
      map.set(display.id, display);
    }
  }

  return map;
}

export function resolveOrgUserDisplay(
  usersById: Map<string, OrgUserDisplay>,
  userId: string | undefined,
  fallbackName?: string,
): OrgUserDisplay | null {
  if (!userId) {
    return null;
  }

  const fromMap = usersById.get(userId);
  if (fromMap) {
    return fromMap;
  }

  const name = fallbackName?.trim() || UNKNOWN_ORG_USER_NAME;

  return {
    id: userId,
    name,
    initials: getUserInitials(name),
  };
}

export function createOrgUserDisplayLookup(usersById: Map<string, OrgUserDisplay>): OrgUserDisplayLookup {
  return (userId, fallbackName) => resolveOrgUserDisplay(usersById, userId, fallbackName);
}

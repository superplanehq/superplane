import type { CanvasesUserRef } from "@/api-client";

export type UserDisplayProfile = {
  name: string;
  initials: string;
  avatarUrl?: string;
};

export function displayNameInitials(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) {
    return "?";
  }

  return trimmed
    .split(/\s+/)
    .map((part) => part[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

export function userRefDisplayProfile(
  owner: CanvasesUserRef | undefined,
  directory?: Map<string, UserDisplayProfile>,
): UserDisplayProfile {
  const ownerId = owner?.id?.trim();
  const fromDirectory = ownerId ? directory?.get(ownerId) : undefined;
  const name = fromDirectory?.name || owner?.name?.trim() || "Unknown";

  return {
    name,
    initials: fromDirectory?.initials || displayNameInitials(name),
    avatarUrl: fromDirectory?.avatarUrl,
  };
}

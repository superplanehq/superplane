import { useMemo } from "react";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { buildOrgUserDisplayMap } from "@/lib/orgUserDisplay";

export interface WorkOrderMentionOption {
  id: string;
  name: string;
  initials: string;
  avatarUrl?: string;
}

const MAX_MENTION_OPTIONS = 8;

/**
 * Organization members eligible for `@` mentioning, filtered by the text
 * typed after `@`. Shared by `WorkOrderMentionMenu` (rendering) and
 * `WorkOrderCommentComposer` (keyboard navigation + selection) so both stay
 * in sync on exactly the same list.
 */
export function useWorkOrderMentionOptions(organizationId: string, query: string): WorkOrderMentionOption[] {
  const { data: users = [] } = useOrganizationUsers(organizationId);

  return useMemo(() => {
    const displays = [...buildOrgUserDisplayMap(users).values()];
    const needle = query.trim().toLowerCase();
    const matches = needle ? displays.filter((user) => user.name.toLowerCase().includes(needle)) : displays;

    return matches
      .sort((a, b) => a.name.localeCompare(b.name))
      .slice(0, MAX_MENTION_OPTIONS)
      .map((user) => ({ id: user.id, name: user.name, initials: user.initials, avatarUrl: user.avatarUrl }));
  }, [users, query]);
}

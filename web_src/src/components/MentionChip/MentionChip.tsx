import { useMemo } from "react";
import { Avatar } from "@/components/Avatar/avatar";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { buildOrgUserDisplayMap, resolveOrgUserDisplay } from "@/lib/orgUserDisplay";

interface MentionChipFromLinkProps {
  userId: string;
  /** Display name embedded in the markdown link label, e.g. `@[Ada](user:1)`. */
  rawLabel?: string;
  organizationId: string;
}

/**
 * Renders a `user:<uuid>` markdown link (see `mentionComposer.ts`) as a
 * pill with the member's avatar + name, resolved live against
 * `useOrganizationUsers` so renamed/removed members stay accurate — falling
 * back to the name captured in the link at mention time otherwise. Mirrors
 * `NodeChipFromLink`'s pattern for `node:` links in `Markdown.tsx`.
 */
export function MentionChipFromLink({ userId, rawLabel, organizationId }: MentionChipFromLinkProps) {
  const { data: users = [] } = useOrganizationUsers(organizationId);

  const display = useMemo(
    () => resolveOrgUserDisplay(buildOrgUserDisplayMap(users), userId, rawLabel),
    [users, userId, rawLabel],
  );

  const name = display?.name ?? rawLabel ?? userId;

  return (
    <span
      data-testid="mention-chip"
      className="mx-0.5 inline-flex max-w-full items-center gap-1 rounded-full bg-blue-100 px-1.5 py-0.5 align-middle text-[12px] font-medium text-blue-700 dark:bg-blue-950 dark:text-blue-300"
    >
      <Avatar src={display?.avatarUrl} initials={display?.initials} alt={name} className="size-4 shrink-0" />
      <span className="truncate">@{name}</span>
    </span>
  );
}

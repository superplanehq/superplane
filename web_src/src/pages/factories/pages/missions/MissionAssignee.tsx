import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";

import { OrgUserReference } from "../../OrgUserReference";
import type { MissionAssignee as MissionAssigneeValue } from "./missionMocks";

export function MissionAssignee({
  organizationId,
  assignee,
}: {
  organizationId: string;
  assignee: MissionAssigneeValue;
}) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  return (
    <span data-testid="mission-assignee">
      <OrgUserReference
        display={resolveUser(assignee.id, assignee.name)}
        size="xs"
        nameClassName="text-[12px] text-muted-foreground"
      />
    </span>
  );
}

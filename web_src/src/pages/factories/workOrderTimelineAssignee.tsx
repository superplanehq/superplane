import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { Fragment, type ReactNode } from "react";
import type { WorkOrderTimelineEvent } from "./lib/workOrderTimelineEvents";
import { OrgUserReference } from "./OrgUserReference";

export function AssigneeChangeDescription({
  actorUserId,
  assigneeChange,
  resolveUserDisplay,
}: {
  actorUserId?: string;
  assigneeChange: NonNullable<WorkOrderTimelineEvent["assigneeChange"]>;
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  const { assignedUserIds, unassignedUserIds } = assigneeChange;
  const parts: ReactNode[] = [];

  const assignedDescription = buildAssignedChangeDescription(actorUserId, assignedUserIds, resolveUserDisplay);
  if (assignedDescription) {
    parts.push(<span key="assigned">{assignedDescription}</span>);
  }

  if (unassignedUserIds.length > 0) {
    parts.push(
      <span key="unassigned" className="inline-flex flex-wrap items-center gap-x-1 gap-y-1">
        removed <InlineUserList userIds={unassignedUserIds} resolveUserDisplay={resolveUserDisplay} /> as owner
      </span>,
    );
  }

  if (parts.length === 0) {
    return <span>updated owners</span>;
  }

  return (
    <>
      {parts.map((part, index) => (
        <Fragment key={index}>
          {index > 0 ? <span> and </span> : null}
          {part}
        </Fragment>
      ))}
    </>
  );
}

function buildAssignedChangeDescription(
  actorUserId: string | undefined,
  assignedUserIds: string[],
  resolveUserDisplay: OrgUserDisplayLookup,
): ReactNode | null {
  if (assignedUserIds.length === 0) {
    return null;
  }

  const assignedOthers = actorUserId ? assignedUserIds.filter((userId) => userId !== actorUserId) : assignedUserIds;
  const selfAssigned = Boolean(actorUserId && assignedUserIds.includes(actorUserId));

  if (selfAssigned && assignedOthers.length === 0) {
    return <span>took ownership</span>;
  }

  if (selfAssigned && assignedOthers.length > 0) {
    return (
      <span className="inline-flex flex-wrap items-center gap-x-1 gap-y-1">
        took ownership and assigned <InlineUserList userIds={assignedOthers} resolveUserDisplay={resolveUserDisplay} />{" "}
        as owner
      </span>
    );
  }

  return (
    <span className="inline-flex flex-wrap items-center gap-x-1 gap-y-1">
      assigned <InlineUserList userIds={assignedUserIds} resolveUserDisplay={resolveUserDisplay} /> as owner
    </span>
  );
}

function InlineUserList({
  userIds,
  resolveUserDisplay,
}: {
  userIds: string[];
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  if (userIds.length === 0) {
    return null;
  }

  if (userIds.length === 1) {
    return <OrgUserReference display={resolveUserDisplay(userIds[0])} size="sm" emphasizeName />;
  }

  if (userIds.length === 2) {
    return (
      <>
        <OrgUserReference display={resolveUserDisplay(userIds[0])} size="sm" emphasizeName />
        <span> and </span>
        <OrgUserReference display={resolveUserDisplay(userIds[1])} size="sm" emphasizeName />
      </>
    );
  }

  return (
    <>
      {userIds.slice(0, -1).map((userId, index) => (
        <Fragment key={userId}>
          {index > 0 ? <span>, </span> : null}
          <OrgUserReference display={resolveUserDisplay(userId)} size="sm" emphasizeName />
        </Fragment>
      ))}
      <span>, and </span>
      <OrgUserReference display={resolveUserDisplay(userIds[userIds.length - 1])} size="sm" emphasizeName />
    </>
  );
}

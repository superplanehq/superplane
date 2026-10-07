import { Fragment, type ReactElement } from "react";

import { Avatar } from "@/components/Avatar/avatar";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { getUserInitials } from "@/lib/orgUserDisplay";
import { mentionCandidateByName, splitMentionSegments, type WorkOrderMentionCandidate } from "@/lib/workOrderMentions";
import { cn } from "@/lib/utils";

export function WorkOrderMentionText({
  text,
  people,
  mentionClassName,
  capturePointer = false,
}: {
  text: string;
  people: WorkOrderMentionCandidate[];
  mentionClassName?: string;
  capturePointer?: boolean;
}) {
  const names = people.map((person) => person.name);
  return (
    <>
      {splitMentionSegments(text, names).map((segment, index) => (
        <Fragment key={index}>
          {segment.mention ? (
            <WorkOrderMentionToken
              text={segment.text}
              person={mentionCandidateByName(people, segment.text)}
              className={mentionClassName}
              capturePointer={capturePointer}
            />
          ) : (
            segment.text
          )}
        </Fragment>
      ))}
    </>
  );
}

export function WorkOrderPersonMention({
  person,
  className,
}: {
  person: Pick<WorkOrderMentionCandidate, "name" | "avatarUrl" | "email">;
  className?: string;
}) {
  return (
    <MentionHoverCard person={person}>
      <span
        className={cn("work-order-mention inline-flex items-center gap-1 px-1 align-middle", className)}
        data-testid="work-order-mention"
      >
        <Avatar src={person.avatarUrl} initials={getUserInitials(person.name)} alt="" className="size-3.5" />
        {person.name}
      </span>
    </MentionHoverCard>
  );
}

function WorkOrderMentionToken({
  text,
  person,
  className,
  capturePointer,
}: {
  text: string;
  person?: WorkOrderMentionCandidate;
  className?: string;
  capturePointer: boolean;
}) {
  const pill = (
    <span
      className={cn("work-order-mention", capturePointer && "pointer-events-auto", className)}
      data-testid="work-order-mention"
      onMouseDown={
        capturePointer
          ? (event) => {
              event.preventDefault();
            }
          : undefined
      }
    >
      {text}
    </span>
  );

  if (!person) {
    return pill;
  }

  return <MentionHoverCard person={person}>{pill}</MentionHoverCard>;
}

function MentionHoverCard({
  person,
  children,
}: {
  person: Pick<WorkOrderMentionCandidate, "name" | "avatarUrl" | "email">;
  children: ReactElement;
}) {
  return (
    <HoverCard openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>{children}</HoverCardTrigger>
      <HoverCardContent
        side="top"
        align="start"
        className="w-auto min-w-[14rem] p-3"
        data-testid="work-order-mention-tooltip"
      >
        <div className="flex items-center gap-2.5">
          <Avatar src={person.avatarUrl} initials={getUserInitials(person.name)} alt={person.name} className="size-8" />
          <span className="min-w-0">
            <span className="block truncate text-[13px] font-medium text-foreground">{person.name}</span>
            {person.email ? (
              <span className="block truncate text-[12px] text-muted-foreground">{person.email}</span>
            ) : null}
          </span>
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

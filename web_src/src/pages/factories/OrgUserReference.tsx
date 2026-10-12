import { Avatar } from "@/components/Avatar/avatar";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";
import { UNKNOWN_ORG_USER_NAME } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";
import { User, UserRoundX, UserRoundMinus } from "lucide-react";

const avatarSizeClass = {
  xs: "size-5",
  sm: "size-6",
  md: "size-8",
} as const;

interface OrgUserReferenceProps {
  display: OrgUserDisplay | null;
  size?: keyof typeof avatarSizeClass;
  showName?: boolean;
  emphasizeName?: boolean;
  nameClassName?: string;
  className?: string;
}

export function OrgUserReference({
  display,
  size = "sm",
  showName = true,
  emphasizeName = false,
  nameClassName,
  className,
}: OrgUserReferenceProps) {
  const resolvedDisplay: OrgUserDisplay = display ?? {
    id: "unknown",
    name: UNKNOWN_ORG_USER_NAME,
    initials: "?",
  };

  const memberStatus = resolvedDisplay.deleted ? "Deleted user" : resolvedDisplay.formerMember ? "Former member" : null;
  const StatusIcon = resolvedDisplay.deleted ? UserRoundX : UserRoundMinus;

  return (
    <span
      className={cn("inline-flex min-w-0 max-w-full items-center gap-1.5", className)}
      title={memberStatus ? `${resolvedDisplay.name} (${memberStatus.toLowerCase()})` : undefined}
    >
      <Avatar
        src={resolvedDisplay.avatarUrl}
        initials={resolvedDisplay.initials}
        alt={resolvedDisplay.name}
        className={cn("shrink-0", avatarSizeClass[size])}
      />
      {showName ? (
        <span className={cn("min-w-0 truncate text-foreground", emphasizeName && "font-semibold", nameClassName)}>
          {resolvedDisplay.name}
        </span>
      ) : null}
      {memberStatus ? (
        showName ? (
          <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
            {memberStatus}
          </span>
        ) : (
          <StatusIcon className="size-3 shrink-0 text-muted-foreground" role="img" aria-label={memberStatus} />
        )
      ) : null}
    </span>
  );
}

/** Dashed person mark used when a task has no owner. */
export function EmptyOwnerMark({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex size-5 shrink-0 items-center justify-center rounded-full border border-dashed border-muted-foreground/60 text-muted-foreground",
        className,
      )}
      data-testid="empty-owner-mark"
      title="No owner"
    >
      <User className="size-3" aria-hidden />
    </span>
  );
}
